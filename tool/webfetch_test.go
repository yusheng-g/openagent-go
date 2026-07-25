package tool

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// ── test helpers ──
//
// These helpers live in webfetch_test.go (the package's main test file) so
// all tool tests share them. They are test-only by virtue of the _test.go
// suffix — never compiled into production. happy-path tests use newTestClient
// to reach httptest loopback servers without the production SSRF dial guard;
// the SSRF tests below use NewWebFetch()/sharedClient() directly to prove the
// guard is enforced.

// newTestClient is a test-only HTTP client that permits loopback/private
// targets, so happy-path unit tests can reach httptest servers (127.0.0.1).
// Redirect chain length is still bounded; SSRF IP checks are skipped here
// because real network-boundary attacks have dedicated tests below that use
// the production sharedClient() and assert the blocks.
func newTestClient() *http.Client {
	return &http.Client{
		Timeout: webTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return errors.New("too many redirects")
			}
			return nil
		},
		Transport: &http.Transport{
			DialContext: (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		},
	}
}

// testDialContext is a dialer that permits any destination. It exists so the
// full-redirect-chain SSRF test can reach its httptest entry server (loopback)
// while still exercising the production CheckRedirect policy on redirect
// targets.
func testDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, network, addr)
}

// newRedirectServer returns an httptest server that 301-redirects every
// request to target. Used by SSRF redirect-chain tests.
func newRedirectServer(target string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	}))
}

// mustGetReq builds a *http.Request for redirect-check tests (URL only).
func mustGetReq(rawurl string) *http.Request {
	req, err := http.NewRequest(http.MethodGet, rawurl, nil)
	if err != nil {
		panic(err)
	}
	return req
}

// parseIP is a test helper for isPublicIP table tests.
func parseIP(s string) net.IP { return net.ParseIP(s) }

// ── upgradeHTTPS ──

func TestUpgradeHTTPS(t *testing.T) {
	cases := []struct{ in, want string }{
		{"http://example.com/a", "https://example.com/a"},
		{"https://example.com/a", "https://example.com/a"},
		{"http://127.0.0.1:8080/x", "http://127.0.0.1:8080/x"},        // loopback exempt
		{"http://localhost:9000/x", "http://localhost:9000/x"},        // loopback exempt
		{"http://localhost/x", "http://localhost/x"},                  // loopback exempt
		{"ftp://example.com", "ftp://example.com"},                    // non-http untouched
		{"http://example.com", "https://example.com"},                 // no path
	}
	for _, c := range cases {
		if got := upgradeHTTPS(c.in); got != c.want {
			t.Errorf("upgradeHTTPS(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// ── htmlToText ──

func TestHTMLToText(t *testing.T) {
	html := `<!DOCTYPE html><html><body>
		<h1>Title</h1><p>Hello <b>world</b>.</p>
		<script>ignore this</script>
		<ul><li>a</li><li>b</li></ul>
		</body></html>`
	got, err := htmlToText(strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "Title") || !strings.Contains(got, "Hello world.") {
		t.Errorf("missing expected text: %q", got)
	}
	if strings.Contains(got, "ignore this") {
		t.Errorf("script content leaked into text: %q", got)
	}
	if !strings.Contains(got, "a") || !strings.Contains(got, "b") {
		t.Errorf("list items missing: %q", got)
	}
	t.Logf("✅ htmlToText:\n%s", got)
}

// ── WebFetch happy path ──

func TestWebFetchBasic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><h1>Hi</h1><p>page body</p></body></html>`))
	}))
	defer srv.Close()

	f := NewWebFetch().withClient(newTestClient())
	out, err := f.Execute(context.Background(), []byte(`{"url":"`+srv.URL+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Hi") || !strings.Contains(out, "page body") {
		t.Errorf("missing page text: %s", out)
	}
	t.Logf("✅ webfetch basic:\n%s", out)
}

func TestWebFetchTruncates(t *testing.T) {
	big := strings.Repeat("x", 1000)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><body><p>` + big + `</p></body></html>`))
	}))
	defer srv.Close()

	f := NewWebFetch().withClient(newTestClient())
	out, err := f.Execute(context.Background(), []byte(`{"url":"`+srv.URL+`","max_chars":50}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "[truncated]") {
		t.Errorf("expected truncation marker: %s", out)
	}
	if len(out) > 200 { // 50 chars + marker + slack
		t.Errorf("output not truncated: %d bytes", len(out))
	}
	t.Logf("✅ webfetch truncated:\n%s", out)
}

func TestWebFetchNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	f := NewWebFetch().withClient(newTestClient())
	_, err := f.Execute(context.Background(), []byte(`{"url":"`+srv.URL+`"}`))
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !strings.HasPrefix(err.Error(), "webfetch:") {
		t.Errorf("error should have webfetch: prefix: %v", err)
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error should mention status 404: %v", err)
	}
	t.Logf("✅ webfetch 404: %v", err)
}

func TestWebFetchSelfApproves(t *testing.T) {
	f := NewWebFetch().withClient(newTestClient())
	if !f.CanSelfApprove(nil) {
		t.Error("WebFetch should self-approve")
	}
}

func TestWebFetchRedirectShown(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><body><p>final content</p></body></html>`))
	}))
	defer target.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusMovedPermanently)
	}))
	defer srv.Close()

	f := NewWebFetch().withClient(newTestClient())
	out, err := f.Execute(context.Background(), []byte(`{"url":"`+srv.URL+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "final content") {
		t.Errorf("missing redirected body: %s", out)
	}
	if !strings.Contains(out, "[redirected to") {
		t.Errorf("missing redirect notice: %s", out)
	}
	t.Logf("✅ webfetch redirect:\n%s", out)
}

// ── boundary & error cases ──

func TestWebFetchEmptyURL(t *testing.T) {
	f := NewWebFetch().withClient(newTestClient())
	_, err := f.Execute(context.Background(), []byte(`{"url":""}`))
	if err == nil || !strings.Contains(err.Error(), "url is required") {
		t.Errorf("expected url-required error, got: %v", err)
	}
}

func TestWebFetchMalformedArgs(t *testing.T) {
	f := NewWebFetch().withClient(newTestClient())
	_, err := f.Execute(context.Background(), []byte(`{not json`))
	if err == nil || !strings.HasPrefix(err.Error(), "webfetch:") {
		t.Errorf("expected webfetch: error prefix, got: %v", err)
	}
}

func TestWebFetchDefaultMaxChars(t *testing.T) {
	// Omitting max_chars should default to 65536 and NOT truncate a small page.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><body><p>small page</p></body></html>`))
	}))
	defer srv.Close()

	f := NewWebFetch().withClient(newTestClient())
	out, err := f.Execute(context.Background(), []byte(`{"url":"`+srv.URL+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "[truncated]") {
		t.Errorf("small page should not be truncated: %s", out)
	}
	if !strings.Contains(out, "small page") {
		t.Errorf("missing body: %s", out)
	}
}

func TestWebFetchUTF8TruncationSafe(t *testing.T) {
	// A page of multi-byte chars truncated mid-sequence must still be valid UTF-8.
	// "中" is 3 bytes; max_chars=2 runes should yield 2 runes + marker, no mojibake.
	page := strings.Repeat("中", 100)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><body><p>` + page + `</p></body></html>`))
	}))
	defer srv.Close()

	f := NewWebFetch().withClient(newTestClient())
	out, err := f.Execute(context.Background(), []byte(`{"url":"`+srv.URL+`","max_chars":2}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "[truncated]") {
		t.Errorf("expected truncation marker: %s", out)
	}
	// out should be valid UTF-8 — if we sliced mid-byte, this would catch it.
	if !utf8.ValidString(out) {
		t.Errorf("truncated output is not valid UTF-8: %q", out)
	}
}

func TestWebFetchLargeBodyGracefulTruncation(t *testing.T) {
	// A response larger than webMaxBody should parse what we have, not error.
	// We can't easily exceed 5 MiB in a unit test cheaply, but we can verify
	// the readBoundedBody helper truncates instead of erroring.
	big := strings.Repeat("a", 100)
	got, truncated, err := readBoundedBody(strings.NewReader(big), 50)
	if err != nil {
		t.Fatal(err)
	}
	if !truncated {
		t.Error("expected truncated=true")
	}
	if len(got) != 50 {
		t.Errorf("expected 50 bytes, got %d", len(got))
	}
}

func TestWebFetchContextCancel(t *testing.T) {
	// Server that never responds; ctx cancelled before call should fail fast.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	f := NewWebFetch().withClient(newTestClient())
	_, err := f.Execute(ctx, []byte(`{"url":"`+srv.URL+`"}`))
	if err == nil {
		t.Error("expected error on cancelled context")
	}
}

func TestWebFetchNonHTMLBody(t *testing.T) {
	// Plain text (no tags) should pass through htmlToText largely intact.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("just plain text\nno tags here"))
	}))
	defer srv.Close()

	f := NewWebFetch().withClient(newTestClient())
	out, err := f.Execute(context.Background(), []byte(`{"url":"`+srv.URL+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "just plain text") {
		t.Errorf("plain text body lost: %s", out)
	}
}

func TestWebFetchOversizedBodyGracefulTruncation(t *testing.T) {
	// Serve a response larger than webMaxBody (5 MiB). WebFetch must parse
	// what it has and append the truncation marker — NOT error, NOT OOM.
	// We wrap the payload in a <p> so htmlToText yields real content.
	oversized := []byte("<html><body><p>" + strings.Repeat("A", 6*1024*1024) + "</p></body></html>")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write(oversized)
	}))
	defer srv.Close()

	f := NewWebFetch().withClient(newTestClient())
	out, err := f.Execute(context.Background(), []byte(`{"url":"`+srv.URL+`","max_chars":1000}`))
	if err != nil {
		t.Fatalf("oversized body should not error, got: %v", err)
	}
	if !strings.Contains(out, "[truncated]") {
		t.Errorf("oversized body should be marked truncated: %s", out[:200])
	}
	// Output must be valid UTF-8 despite truncation.
	if !utf8.ValidString(out) {
		t.Errorf("truncated oversized output is not valid UTF-8")
	}
	t.Logf("✅ oversized body (%d bytes) -> output %d bytes, truncated", len(oversized), len(out))
}

// ── SSRF defense ──
//
// These tests use NewWebFetch() (production sharedClient() with the SSRF dial
// guard) to prove the network boundary is enforced: private/loopback/link-local
// IPs are refused before any connection is dialed. No real network I/O happens
// for IP literals — resolveAndCheck short-circuits on net.ParseIP and returns
// errSSRF immediately.

func TestSSRFBlocksCloudMetadata(t *testing.T) {
	f := NewWebFetch()
	_, err := f.Execute(context.Background(), []byte(`{"url":"http://169.254.169.254/latest/meta-data/"}`))
	if err == nil {
		t.Fatal("169.254.169.254 must be blocked")
	}
	if !errors.Is(err, errSSRF) {
		t.Errorf("expected errSSRF, got: %v", err)
	}
}

func TestSSRFBlocksLoopback(t *testing.T) {
	f := NewWebFetch()
	for _, u := range []string{
		"http://127.0.0.1:6379/",
		"http://localhost:8080/",
		"http://[::1]:8080/",
	} {
		_, err := f.Execute(context.Background(), []byte(`{"url":"`+u+`"}`))
		if err == nil || !strings.Contains(err.Error(), "SSRF") {
			t.Errorf("%s should be SSRF-blocked, got: %v", u, err)
		}
	}
}

func TestSSRFBlocksPrivateRanges(t *testing.T) {
	f := NewWebFetch()
	for _, u := range []string{
		"http://10.0.0.1/",
		"http://192.168.1.1/admin",
		"http://172.16.0.1/",
	} {
		_, err := f.Execute(context.Background(), []byte(`{"url":"`+u+`"}`))
		if err == nil || !strings.Contains(err.Error(), "SSRF") {
			t.Errorf("%s should be SSRF-blocked, got: %v", u, err)
		}
	}
}

func TestSSRFBlocksUserinfo(t *testing.T) {
	// http://evil@169.254.169.254/ — userinfo must be rejected at entry,
	// before any dial.
	f := NewWebFetch()
	_, err := f.Execute(context.Background(), []byte(`{"url":"http://evil@169.254.169.254/"}`))
	if err == nil {
		t.Fatal("userinfo URL must be rejected")
	}
	if !strings.Contains(err.Error(), "userinfo") {
		t.Errorf("expected userinfo rejection, got: %v", err)
	}
}

func TestSSRFBlocksBadScheme(t *testing.T) {
	f := NewWebFetch()
	_, err := f.Execute(context.Background(), []byte(`{"url":"ftp://example.com/"}`))
	if err == nil || !strings.Contains(err.Error(), "scheme") {
		t.Errorf("ftp scheme must be rejected, got: %v", err)
	}
}

func TestSSRFBlocksDecimalIPOne(t *testing.T) {
	// 2130706433 == 127.0.0.1 in decimal. net.ParseIP rejects this form, so
	// it falls through to DNS lookup, which fails ("no such host") rather
	// than resolving to 127.0.0.1 on this host. The assertion is therefore
	// "this non-standard IP form does not succeed" — a defense-in-depth check.
	// It does NOT prove SSRF interception (the IP never resolves to loopback
	// here); a host whose resolver canonicalizes decimal IPs would need
	// resolveAndCheck to catch the resulting 127.0.0.1, which is covered by
	// TestSSRFBlocksLoopback.
	f := NewWebFetch()
	_, err := f.Execute(context.Background(), []byte(`{"url":"http://2130706433/"}`))
	if err == nil {
		t.Fatal("decimal-IP 127.0.0.1 must not succeed")
	}
	t.Logf("decimal-IP rejected (defense-in-depth): %v", err)
}

func TestSSRFRedirectToInternalBlocked(t *testing.T) {
	// A public URL that 301 → http://169.254.169.254/ must be blocked at
	// the redirect hop. We assert this two ways:
	//
	// (a) Direct: safeCheckRedirect rejects an internal target — the per-hop
	//     guard, no network involved.
	for _, u := range []string{"http://169.254.169.254/", "http://10.0.0.1/"} {
		if err := safeCheckRedirect(mustGetReq(u), nil); err == nil {
			t.Errorf("safeCheckRedirect must reject %s", u)
		}
	}

	// (b) Full redirect chain: an httptest server 301 → 169.254.169.254.
	//     We use a client with the production CheckRedirect (the SSRF policy
	//     under test) but a permissive dialer for the *entry* host only,
	//     so the request can reach the test server. The redirect target
	//     is still validated by the production CheckRedirect and must fail.
	srv := newRedirectServer("http://169.254.169.254/latest/meta-data/")
	defer srv.Close()
	client := &http.Client{
		Timeout:       webTimeout,
		CheckRedirect: safeCheckRedirect, // production policy
		Transport:     &http.Transport{DialContext: testDialContext},
	}
	resp, err := client.Get(srv.URL)
	if err == nil {
		resp.Body.Close()
		t.Fatal("redirect to 169.254.169.254 must be blocked, got success")
	}
	if !strings.Contains(err.Error(), "SSRF") {
		t.Errorf("expected SSRF rejection on redirect, got: %v", err)
	}
}

func TestSSRFPublicHostAllowedByCheckRedirect(t *testing.T) {
	// A real public hostname should pass the redirect check (we don't dial
	// here, only resolve+validate). api.tavily.com is public.
	if err := safeCheckRedirect(mustGetReq("https://api.tavily.com/search"), nil); err != nil {
		t.Errorf("public host should pass redirect check, got: %v", err)
	}
}

func TestIsPublicIP(t *testing.T) {
	bad := []string{"127.0.0.1", "10.0.0.1", "192.168.1.1", "172.16.0.1", "169.254.169.254", "::1", "0.0.0.0"}
	for _, s := range bad {
		if isPublicIP(parseIP(s)) {
			t.Errorf("%s should not be public", s)
		}
	}
	good := []string{"8.8.8.8", "1.1.1.1", "203.0.113.1"}
	for _, s := range good {
		if !isPublicIP(parseIP(s)) {
			t.Errorf("%s should be public", s)
		}
	}
}

func TestSSRFBlocksCGNAT(t *testing.T) {
	// RFC 6598 100.64.0.0/10 — carrier-grade NAT. Go's IsPrivate() does NOT
	// cover it, so it must be rejected explicitly. In CGNAT/SD-WAN environments
	// this range is internal and a real SSRF bypass if allowed.
	for _, s := range []string{"100.64.0.1", "100.127.255.254", "100.64.10.20"} {
		if isPublicIP(parseIP(s)) {
			t.Errorf("%s (CGNAT 100.64.0.0/10) must not be public", s)
		}
	}
	// Boundaries: 100.63.x and 100.128.x are NOT CGNAT, should stay public.
	for _, s := range []string{"100.63.0.1", "100.128.0.1"} {
		if !isPublicIP(parseIP(s)) {
			t.Errorf("%s is outside CGNAT, should remain public", s)
		}
	}
}
