package utils

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// These tests pin the invariants of the shared web-HTTP helpers directly at
// the utils layer. The tool package exercises them end-to-end through
// WebFetch/WebSearch; these unit tests guard the primitives in isolation so
// a future refactor of the tool layer can't silently drop coverage.

func TestIsPublicIP(t *testing.T) {
	bad := []string{"127.0.0.1", "10.0.0.1", "192.168.1.1", "172.16.0.1", "169.254.169.254", "::1", "0.0.0.0"}
	for _, s := range bad {
		if IsPublicIP(net.ParseIP(s)) {
			t.Errorf("%s should not be public", s)
		}
	}
	good := []string{"8.8.8.8", "1.1.1.1", "203.0.113.1"}
	for _, s := range good {
		if !IsPublicIP(net.ParseIP(s)) {
			t.Errorf("%s should be public", s)
		}
	}
	if IsPublicIP(nil) {
		t.Error("nil IP must not be public")
	}
}

func TestIsPublicIPBlocksCGNAT(t *testing.T) {
	// RFC 6598 100.64.0.0/10 — not covered by Go's IsPrivate.
	for _, s := range []string{"100.64.0.1", "100.127.255.254", "100.64.10.20"} {
		if IsPublicIP(net.ParseIP(s)) {
			t.Errorf("%s (CGNAT) must not be public", s)
		}
	}
	for _, s := range []string{"100.63.0.1", "100.128.0.1"} {
		if !IsPublicIP(net.ParseIP(s)) {
			t.Errorf("%s is outside CGNAT, should remain public", s)
		}
	}
}

func TestIsLoopbackHost(t *testing.T) {
	cases := []struct {
		host string
		want bool
	}{
		{"127.0.0.1", true},
		{"127.0.0.1:8080", true},
		{"localhost", true},
		{"localhost:9000", true},
		{"[::1]", true},
		{"[::1]:8080", true},
		{"[::ffff:127.0.0.1]", true},
		{"8.8.8.8", false},
		{"example.com", false},
		{"10.0.0.1", false}, // private but not loopback
		{"", false},
	}
	for _, c := range cases {
		if got := IsLoopbackHost(c.host); got != c.want {
			t.Errorf("IsLoopbackHost(%q) = %v, want %v", c.host, got, c.want)
		}
	}
}

func TestValidateRequestURL(t *testing.T) {
	good := []string{"http://example.com/", "https://example.com/a?b=c"}
	for _, u := range good {
		if _, err := ValidateRequestURL(u); err != nil {
			t.Errorf("%s should validate, got %v", u, err)
		}
	}
	bad := []string{
		"ftp://example.com/",            // bad scheme
		"http://evil@169.254.169.254/",  // userinfo
		"http://",                       // no host
		"://broken",                     // unparseable
	}
	for _, u := range bad {
		if _, err := ValidateRequestURL(u); err == nil {
			t.Errorf("%s should be rejected", u)
		}
	}
}

func TestResolveAndCheckSSRF(t *testing.T) {
	// IP literals short-circuit DNS — deterministic, no network.
	for _, s := range []string{"169.254.169.254", "127.0.0.1", "10.0.0.1", "::1"} {
		err := ResolveAndCheck(s)
		if err == nil || !errors.Is(err, ErrSSRF) {
			t.Errorf("%s must be rejected as SSRF, got %v", s, err)
		}
	}
	// A public IP literal passes.
	if err := ResolveAndCheck("8.8.8.8"); err != nil {
		t.Errorf("8.8.8.8 should pass, got %v", err)
	}
}

func TestScrubURL(t *testing.T) {
	cases := []struct{ in, want string }{
		{"https://example.com/a?token=secret", "https://example.com/a"},
		{"https://example.com/a#frag", "https://example.com/a"},
		{"https://example.com/a?x=1#f", "https://example.com/a"},
		{"https://user:pass@example.com/a", "https://user:pass@example.com/a"}, // userinfo retained (scrub only drops query+fragment)
		// "not a url" parses "successfully" in Go (space → %20), so ScrubURL
		// round-trips it through url.String rather than returning it verbatim.
		// This is the documented behavior (best-effort; malformed is unlikely
		// to carry a usable secret), so we assert the round-trip output.
		{"not a url", "not%20a%20url"},
	}
	for _, c := range cases {
		if got := ScrubURL(c.in); got != c.want {
			t.Errorf("ScrubURL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestWrapUntrusted(t *testing.T) {
	if got := WrapUntrusted(""); got != "" {
		t.Errorf("empty input should return empty, got %q", got)
	}
	got := WrapUntrusted("hello")
	if !strings.HasPrefix(got, "[Untrusted web content]\n") || !strings.HasSuffix(got, "[/Untrusted web content]") {
		t.Errorf("output must be bracketed, got %q", got)
	}
	if !strings.Contains(got, "hello") {
		t.Errorf("content lost, got %q", got)
	}
}

func TestReadErrorSnippet(t *testing.T) {
	// Reads up to 2 KiB then stops; larger bodies are truncated, not errored.
	big := strings.Repeat("x", 5000)
	snip := ReadErrorSnippet(strings.NewReader(big))
	if len(snip) > 2048 {
		t.Errorf("snippet should be <= 2048 bytes, got %d", len(snip))
	}
	if ReadErrorSnippet(strings.NewReader("")) != "" {
		t.Error("empty body should yield empty snippet")
	}
}

func TestDrainAndClose(t *testing.T) {
	// Serve a body, drain+close it — should not panic and should allow the
	// test server to shut down cleanly (the real contract: connection reuse).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("payload"))
	}))
	defer srv.Close()
	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	DrainAndClose(resp.Body) // must not panic
}
