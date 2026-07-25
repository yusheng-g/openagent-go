package tool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"

	openagent "github.com/yusheng-g/openagent-go"
)

// webTimeout is the per-request deadline for web tools. Generous enough for
// slow pages, short enough that a hung host doesn't stall the agent loop.
const webTimeout = 30 * time.Second

// webMaxBody caps how many bytes a web tool will read off the wire. Pages
// larger than this are truncated (WebFetch) or rejected (WebSearch). Bounds
// memory so a hostile/huge URL can't OOM the process.
const webMaxBody = 5 * 1024 * 1024 // 5 MiB

// webUserAgent identifies the agent to upstream servers. Some sites block
// the default Go UA; a bespoke one is more widely accepted.
const webUserAgent = "openagent-webfetch/1.0 (+https://github.com/yusheng-g/openagent-go)"

// webFetchName / webSearchName are the tool names exposed to the model and
// used as error-prefix stems. Centralized so the Definition name and the
// error prefix can't drift apart.
const (
	webFetchName  = "webfetch"
	webSearchName = "websearch"
)

// Default output caps for WebFetch / WebSearch. Exposed as named constants
// rather than bare ints so the Description text and the clamp logic share
// one source of truth.
const (
	defaultMaxChars   = 65536
	defaultMaxResults = 8
	maxResultsCap     = 20
)

// readBoundedBody reads at most max bytes from r. Used to cap memory when a
// hostile/huge URL would otherwise stream unbounded bytes into the parser.
// Returns the bytes read (len <= max) and whether the underlying stream had
// more (truncated). A read error mid-stream is returned as err.
func readBoundedBody(r io.Reader, max int64) ([]byte, bool, error) {
	limited := io.LimitReader(r, max+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, false, err
	}
	truncated := int64(len(data)) > max
	if truncated {
		data = data[:max]
	}
	return data, truncated, nil
}

// upgradeHTTPS rewrites an http:// URL to https://. Loopback hosts are left
// alone so httptest servers (http://127.0.0.1:port, http://[::1]:port) keep
// working in tests. (Note: WebFetch still SSRF-validates the dial; the
// loopback exemption here is only about the http→https rewrite, not about
// bypassing network policy — loopback is blocked at dial time in production,
// but tests need plain http to reach httptest.)
func upgradeHTTPS(rawURL string) string {
	if !strings.HasPrefix(rawURL, "http://") {
		return rawURL
	}
	rest := rawURL[len("http://"):]
	host, _, ok := strings.Cut(rest, "/")
	if !ok {
		host = rest
	}
	if isLoopbackHost(host) {
		return rawURL
	}
	return "https://" + rest
}

// htmlToText extracts visible text from an HTML document. Block-level
// elements get a trailing newline; whitespace runs are collapsed. Script,
// style, noscript, and the entire <head> (title/meta/link) are dropped so
// head metadata doesn't pollute the visible-text output.
func htmlToText(r io.Reader) (string, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			switch c.Type {
			case html.TextNode:
				b.WriteString(c.Data)
			case html.ElementNode:
				switch c.Data {
				case "script", "style", "noscript", "head", "title", "meta", "link", "base":
					continue
				}
				walk(c)
				if isBlock(c.Data) {
					b.WriteByte('\n')
				}
			}
		}
	}
	walk(doc)

	// Collapse whitespace runs to single spaces, keep newlines. Read the
	// builder's bytes once — calling b.String() inside the loop would copy
	// the whole buffer on every iteration (O(n²)).
	raw := b.String()
	var out strings.Builder
	out.Grow(len(raw))
	inSpace := false
	for i := 0; i < len(raw); i++ {
		switch ch := raw[i]; ch {
		case '\n':
			out.WriteByte('\n')
			inSpace = false
		case ' ', '\t', '\r':
			if !inSpace {
				out.WriteByte(' ')
				inSpace = true
			}
		default:
			out.WriteByte(ch)
			inSpace = false
		}
	}
	return strings.TrimSpace(out.String()), nil
}

// isBlock reports whether tag is block-level — emitting a newline after it
// keeps paragraphs and list items on separate lines instead of one blob.
func isBlock(tag string) bool {
	switch tag {
	case "p", "div", "br", "li", "ul", "ol", "tr", "blockquote",
		"h1", "h2", "h3", "h4", "h5", "h6",
		"section", "article", "header", "footer", "nav", "main":
		return true
	}
	return false
}

// ── WebFetch ──

// WebFetch fetches a URL and returns its visible text, truncated.
// http:// is upgraded to https:// (loopback exempt for tests). Implements
// [openagent.Tool] and [openagent.SelfApproving] — network reads are
// treated as safe, like read/ls/grep. SSRF is blocked at the dial layer
// (see webhttp.go): private/loopback/link-local IPs are refused, including
// cloud metadata endpoints.
type WebFetch struct {
	client *http.Client // injectable for tests; defaults to sharedClient()
}

// NewWebFetch creates a WebFetch tool with the shared SSRF-safe HTTP client.
func NewWebFetch() *WebFetch { return &WebFetch{client: sharedClient()} }

// withClient returns a WebFetch that uses the given client. Untested callers
// must not use this — it exists so tests can point at an httptest server
// (loopback) without weakening the production SSRF policy.
func (t *WebFetch) withClient(c *http.Client) *WebFetch { return &WebFetch{client: c} }

func (t *WebFetch) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name: webFetchName,
		Description: "Fetch a URL and return the page as plain text (HTML stripped to text). " +
			"HTTP is upgraded to HTTPS. Output is truncated to max_chars (default 65536). " +
			"Use for reading documentation, articles, or any web page content. " +
			"Internal/private/loopback addresses are blocked (SSRF protection).",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"url":       {"type": "string",  "description": "URL to fetch (http:// auto-upgraded to https://)"},
				"max_chars": {"type": "integer", "description": "Maximum characters to return (default: 65536)"}
			},
			"required": ["url"]
		}`),
	}
}

func (t *WebFetch) CanSelfApprove(_ json.RawMessage) bool { return true }

func (t *WebFetch) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		URL      string `json:"url"`
		MaxChars int    `json:"max_chars"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("%s: %w", webFetchName, err)
	}
	if params.URL == "" {
		return "", fmt.Errorf("%s: url is required", webFetchName)
	}
	if params.MaxChars <= 0 {
		params.MaxChars = defaultMaxChars
	}

	// Entry URL policy: scheme + no userinfo. SSRF IP check happens at dial
	// time and on every redirect hop (see webhttp.go).
	if _, err := validateRequestURL(params.URL); err != nil {
		return "", fmt.Errorf("%s: %w", webFetchName, err)
	}

	url := upgradeHTTPS(params.URL)

	release, err := acquireWebSlot(ctx)
	if err != nil {
		return "", fmt.Errorf("%s: %w", webFetchName, err)
	}
	defer release()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("%s: %w", webFetchName, err)
	}
	req.Header.Set("User-Agent", webUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s: %w", webFetchName, err)
	}
	defer drainAndClose(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet := readErrorSnippet(resp.Body)
		return "", fmt.Errorf("%s: HTTP %d for %s: %s", webFetchName, resp.StatusCode, scrubURL(params.URL), snippet)
	}

	// Bound memory: read at most webMaxBody. If the page is larger, we parse
	// what we have (HTML parser tolerates truncated input) rather than failing
	// or OOMing. A truncation marker is appended below.
	body, bodyTruncated, err := readBoundedBody(resp.Body, webMaxBody)
	if err != nil {
		return "", fmt.Errorf("%s: %w", webFetchName, err)
	}
	text, err := htmlToText(bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("%s: %w", webFetchName, err)
	}

	var header string
	if resp.Request != nil && resp.Request.URL != nil && resp.Request.URL.String() != url {
		// Redirected — surface the final (scrubbed) URL so the model knows where
		// the content came from, without leaking any query/fragment credentials.
		header = "[redirected to " + scrubURL(resp.Request.URL.String()) + "]\n"
	}

	// Truncate by rune to avoid cutting a multi-byte UTF-8 sequence in half
	// (which would produce invalid UTF-8 / mojibake in the model's input).
	runes := []rune(text)
	truncated := bodyTruncated
	if len(runes) > params.MaxChars {
		runes = runes[:params.MaxChars]
		truncated = true
	}
	result := header + string(runes)
	if truncated {
		result += "\n…[truncated]"
	}
	return result, nil
}
