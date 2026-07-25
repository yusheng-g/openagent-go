package tool

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
)

// ── SSRF defense ──
//
// WebFetch accepts a model-supplied URL and dials it. Without explicit
// network-boundary checks that is a Server-Side Request Forgery vector:
// the model could fetch http://169.254.169.254/... (cloud IAM metadata),
// http://10.0.0.1/admin, http://127.0.0.1:6379, etc., and exfiltrate the
// response. Because CanSelfApprove=true, no human gates the call.
//
// The defense is layered:
//   1. validateRequestURL: scheme ∈ {http,https}, no userinfo, host non-empty.
//   2. resolveAndCheck: resolve host → IPs, every IP must be public
//      (reject loopback/private/link-local/multicast/unspecified).
//   3. DialContext re-validates the dial-time IP — defeats DNS rebinding,
//      where the first lookup (validation) returns a public IP and the
//      second lookup (actual dial) returns 169.254.169.254.
//   4. CheckRedirect runs validateRequestURL + resolveAndCheck on every
//      hop — a public URL that 301→http://169.254.169.254/ is rejected.
//
// isPublicIP covers the address classes we refuse to dial. 169.254.169.254
// is link-local unicast (IsLinkLocalUnicast), so it's blocked here.

// errSSRF is returned for any URL that would dial a non-public target.
// Distinct error type so tests can assert "blocked by SSRF policy" rather
// than a generic network error.
var errSSRF = errors.New("url resolves to a non-public address (SSRF blocked)")

// isPublicIP reports whether ip is safe to dial: not loopback, not RFC 1918 /
// ULA, not RFC 6598 CGNAT (100.64.0.0/10 — not covered by Go's IsPrivate),
// not link-local (covers 169.254.x.x cloud metadata), not multicast,
// not unspecified. nil → false.
func isPublicIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return false
	}
	// RFC 6598 carrier-grade NAT: 100.64.0.0/10. Go's IsPrivate() predates
	// RFC 6598 and does not cover it; in CGNAT/SD-WAN environments this range
	// is internal and a real SSRF bypass, so reject it explicitly.
	if ip4 := ip.To4(); ip4 != nil && ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
		return false
	}
	return true
}

// isLoopbackHost reports whether host (the authority portion of a URL, before
// the path) refers to a loopback address, with or without a port, IPv4,
// IPv6-literal, or IPv4-mapped-IPv6 form. Uses net.ParseIP so ::ffff:127.0.0.1
// and bare ::1 are recognized, not just string-equality. Shared by upgradeHTTPS
// (webfetch) and the webSearchAt endpoint guard (websearch).
func isLoopbackHost(host string) bool {
	// IPv6 literal: http://[::1]:port/ → host is "[::1]:port" or "[::1]"
	if strings.HasPrefix(host, "[") {
		if end := strings.IndexByte(host, ']'); end > 0 {
			ip := net.ParseIP(host[1:end])
			return ip != nil && ip.IsLoopback()
		}
		return false
	}
	// host or host:port
	h, _, _ := strings.Cut(host, ":")
	if ip := net.ParseIP(h); ip != nil {
		return ip.IsLoopback()
	}
	return h == "localhost"
}

// resolveAndCheck looks up host and requires every resolved IP to be public.
// host may be a hostname or an IP literal. Returns errSSRF (wrapped with
// detail) if any address is non-public; a lookup error is returned as-is.
func resolveAndCheck(host string) error {
	// IP literal: skip DNS, evaluate directly. net.ParseIP handles IPv4,
	// IPv6, and IPv4-mapped IPv6 (::ffff:127.0.0.1).
	if ip := net.ParseIP(host); ip != nil {
		if !isPublicIP(ip) {
			return fmt.Errorf("%w: %s", errSSRF, ip)
		}
		return nil
	}
	ips, err := net.DefaultResolver.LookupIPAddr(context.Background(), host)
	if err != nil {
		return err
	}
	for _, ip := range ips {
		if !isPublicIP(ip.IP) {
			return fmt.Errorf("%w: %s → %s", errSSRF, host, ip.IP)
		}
	}
	return nil
}

// validateRequestURL enforces the entry-level URL policy: allowed scheme,
// no userinfo (bypass via http://evil@169.254.169.254/), non-empty host.
// Returns the parsed *url.URL on success.
func validateRequestURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("scheme %q not allowed (want http or https)", u.Scheme)
	}
	if u.User != nil {
		return nil, errors.New("userinfo (user:pass@) not allowed in URL")
	}
	if u.Host == "" {
		return nil, errors.New("URL missing host")
	}
	return u, nil
}

// maxRedirects caps redirect chains. Go's default is 10; we tighten to 5.
const maxRedirects = 5

// safeCheckRedirect is the per-hop policy: bound chain length and re-run
// the full URL + SSRF check on each target. This closes the "public URL
// 301 → internal IP" bypass.
func safeCheckRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return fmt.Errorf("too many redirects (>%d)", maxRedirects)
	}
	if _, err := validateRequestURL(req.URL.String()); err != nil {
		return fmt.Errorf("redirect target rejected: %w", err)
	}
	if err := resolveAndCheck(req.URL.Hostname()); err != nil {
		return fmt.Errorf("redirect target rejected: %w", err)
	}
	return nil
}

// safeDialContext wraps the dialer so the IP is re-validated at the moment
// of dialing. This is the DNS-rebinding defense: even if resolveAndCheck
// during request setup saw a public IP, the dial-time lookup is checked
// again against the address actually being dialed.
func safeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	if err := resolveAndCheck(host); err != nil {
		return nil, err
	}
	return safeDialer.DialContext(ctx, network, addr)
}

var safeDialer = &net.Dialer{Timeout: 10 * time.Second}

// ── shared singleton HTTP client ──
//
// One client per process — its Transport owns the connection pool, so
// keep-alive and TLS sessions are reused across requests. Constructing a
// client per call (the old behavior) discarded the pool and accumulated
// idle transports under GC pressure.

var (
	sharedHTTPClientOnce sync.Once
	sharedHTTPClient     *http.Client
)

// sharedClient returns the process-wide safe HTTP client, creating it once.
// Tests that need to point at an httptest server pass its URL directly to
// the tool (the client's per-request URL, not a per-test client), so the
// singleton is safe to share.
func sharedClient() *http.Client {
	sharedHTTPClientOnce.Do(func() {
		sharedHTTPClient = &http.Client{
			Timeout:       webTimeout,
			CheckRedirect: safeCheckRedirect,
			Transport: &http.Transport{
				DialContext:           safeDialContext,
				MaxIdleConns:          100,
				MaxIdleConnsPerHost:   10,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 15 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		}
	})
	return sharedHTTPClient
}

// ── concurrency limit ──
//
// Without a bound, a single agent turn that fans out many webfetch calls
// (the runner can issue tool calls in parallel) opens dozens of simultaneous
// connections and 5 MiB read buffers. A weighted semaphore caps in-flight
// web-tool calls process-wide.

const webConcurrency = 8

var webSem = semaphore.NewWeighted(webConcurrency)

// acquireWebSlot blocks until a concurrency slot is free or ctx is cancelled.
// Release with the returned func. Callers MUST release exactly once.
func acquireWebSlot(ctx context.Context) (release func(), err error) {
	if err := webSem.Acquire(ctx, 1); err != nil {
		return nil, err
	}
	var once sync.Once
	return func() {
		once.Do(func() { webSem.Release(1) })
	}, nil
}

// ── URL scrubbing for logs/errors ──
//
// URLs may carry credentials in query/fragment (?access_token=…). Echoing
// the raw URL into an error message or the model's context leaks them,
// violating the project rule "Do not include secrets in user-facing output".
// scrubURL keeps scheme+host+path, drops query and fragment.

func scrubURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw // best-effort; malformed is unlikely to carry a usable secret
	}
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

// drainAndClose reads any remaining body bytes then closes, so the
// underlying TCP connection can be returned to the pool for keep-alive
// reuse. Calling on an already-drained body is a no-op. Use in defer.
func drainAndClose(body io.ReadCloser) {
	_, _ = io.Copy(io.Discard, body)
	_ = body.Close()
}

// readErrorSnippet reads up to 2 KiB of an error response body for
// inclusion in the error message — aids debugging without unbounded reads.
func readErrorSnippet(r io.Reader) string {
	buf := make([]byte, 2048)
	n, _ := r.Read(buf)
	return strings.TrimSpace(string(buf[:n]))
}
