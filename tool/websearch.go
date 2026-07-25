package tool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	openagent "github.com/yusheng-g/openagent-go"
)

// tavilyURL is the Tavily Search endpoint. Authenticated via the
// X-Tavily-Access-Mode: keyless header — no API key required.
// Tavily aggregates web search results and returns clean JSON, making it
// ideal for agent use without the captcha/anti-bot problems of scraping
// HTML search pages directly.
const (
	tavilyURL          = "https://api.tavily.com/search"
	tavilyAccessHeader = "X-Tavily-Access-Mode"
	tavilyAccessMode   = "keyless"
)

// tavilyKeyEnv is the environment variable holding an optional Tavily API
// key. When set, requests authenticate with Authorization: Bearer <key>
// (higher rate limits). When unset, requests use keyless mode (free,
// rate-limited, no account). See https://docs.tavily.com/documentation/keyless.
const tavilyKeyEnv = "TAVILY_API_KEY"

// setTavilyAuth applies the appropriate auth header to a Tavily request:
// Bearer token if TAVILY_API_KEY is set, keyless header otherwise. Both
// produce identical response schemas per Tavily docs, so callers need no
// other change to switch between them.
func setTavilyAuth(req *http.Request) {
	if key := os.Getenv(tavilyKeyEnv); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
		return
	}
	req.Header.Set(tavilyAccessHeader, tavilyAccessMode)
}

// tavilyResult is one hit in Tavily's response.
type tavilyResult struct {
	URL     string  `json:"url"`
	Title   string  `json:"title"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

// tavilyResponse is the shape of Tavily's search response.
type tavilyResponse struct {
	Results []tavilyResult `json:"results"`
	Answer  string        `json:"answer"`
}

// WebSearch searches the web via Tavily (keyless mode) and returns titles,
// URLs, and snippets. Implements [openagent.Tool] and [openagent.SelfApproving].
type WebSearch struct {
	client *http.Client // injectable for tests; defaults to sharedClient()
}

// NewWebSearch creates a WebSearch tool with the shared SSRF-safe HTTP client.
func NewWebSearch() *WebSearch { return &WebSearch{client: sharedClient()} }

// withClient returns a WebSearch that uses the given client. For tests only.
func (t *WebSearch) withClient(c *http.Client) *WebSearch { return &WebSearch{client: c} }

func (t *WebSearch) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name: webSearchName,
		Description: "Search the web and return titles, URLs, and snippets. " +
			"Use for finding current information, documentation, or recent events. " +
			"Backed by Tavily — no API key required (set TAVILY_API_KEY env var for higher rate limits).",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"query":        {"type": "string",  "description": "Search query"},
				"max_results":  {"type": "integer", "description": "Maximum results to return (default: 8, max: 20)"}
			},
			"required": ["query"]
		}`),
	}
}

func (t *WebSearch) CanSelfApprove(_ json.RawMessage) bool { return true }

func (t *WebSearch) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	return webSearchAt(ctx, tavilyURL, t.client, args)
}

// tavilyHost is the only host webSearchAt will POST to in production. The
// endpoint parameter exists solely so tests can redirect to an httptest
// server; a production call with a different host is a misconfiguration or
// injection and is rejected. Tests bypass this by using a 127.0.0.1 endpoint.
const tavilyHost = "api.tavily.com"

// webSearchAt is the core search logic against an explicit endpoint. Split
// out so tests can point at an httptest server instead of the real Tavily.
// The endpoint must be the Tavily host in production; loopback is allowed
// for tests. client is the HTTP client to use (sharedClient() in prod).
func webSearchAt(ctx context.Context, endpoint string, client *http.Client, args json.RawMessage) (string, error) {
	var params struct {
		Query      string `json:"query"`
		MaxResults int    `json:"max_results"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("%s: %w", webSearchName, err)
	}
	if params.Query == "" {
		return "", fmt.Errorf("%s: query is required", webSearchName)
	}
	if params.MaxResults <= 0 {
		params.MaxResults = defaultMaxResults
	}
	if params.MaxResults > maxResultsCap {
		params.MaxResults = maxResultsCap
	}

	// Guard against endpoint injection: only the real Tavily host or a
	// loopback test server is allowed. This keeps the test hook from becoming
	// an SSRF vector if webSearchAt is ever called with a tunable endpoint.
	epURL, err := validateRequestURL(endpoint)
	if err != nil {
		return "", fmt.Errorf("%s: %w", webSearchName, err)
	}
	if h := epURL.Hostname(); h != tavilyHost && !isLoopbackHost(h) {
		return "", fmt.Errorf("%s: endpoint host %q not allowed", webSearchName, h)
	}

	// Tavily takes the desired count in the JSON body.
	body, err := json.Marshal(map[string]any{
		"query":       params.Query,
		"max_results": params.MaxResults,
	})
	if err != nil {
		return "", fmt.Errorf("%s: %w", webSearchName, err)
	}

	release, err := acquireWebSlot(ctx)
	if err != nil {
		return "", fmt.Errorf("%s: %w", webSearchName, err)
	}
	defer release()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("%s: %w", webSearchName, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	setTavilyAuth(req)
	req.Header.Set("User-Agent", webUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s: %w", webSearchName, err)
	}
	defer drainAndClose(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet := readErrorSnippet(resp.Body)
		return "", fmt.Errorf("%s: HTTP %d: %s", webSearchName, resp.StatusCode, snippet)
	}

	// Bound memory: a hostile/buggy endpoint could stream unbounded JSON.
	respBody, _, err := readBoundedBody(resp.Body, webMaxBody)
	if err != nil {
		return "", fmt.Errorf("%s: %w", webSearchName, err)
	}
	var tr tavilyResponse
	if err := json.Unmarshal(respBody, &tr); err != nil {
		return "", fmt.Errorf("%s: %w", webSearchName, err)
	}
	if len(tr.Results) == 0 {
		return "No results found.", nil
	}

	var b strings.Builder
	if tr.Answer != "" {
		b.WriteString(tr.Answer)
		b.WriteString("\n\n")
	}
	for i, r := range tr.Results {
		fmt.Fprintf(&b, "%d. %s\n   %s\n", i+1, r.Title, r.URL)
		if r.Content != "" {
			b.WriteString("   ")
			b.WriteString(r.Content)
			b.WriteByte('\n')
		}
	}
	return strings.TrimSpace(b.String()), nil
}
