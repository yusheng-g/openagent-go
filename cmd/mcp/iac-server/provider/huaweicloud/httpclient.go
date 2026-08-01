package huaweicloud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	opentool "github.com/yusheng-g/openagent-go/tool"
	"github.com/yusheng-g/openagent-go/utils"
)

// maxInlineBody is the maximum response body size returned inline to the LLM.
// Larger bodies are saved to the artifact directory and a file path is
// returned instead, so the LLM can use read/grep to inspect on demand
// without consuming context window.
const maxInlineBody = 8 * 1024

// HTTPRequest is an openagent.Tool that lets the server-side LLM make
// authenticated HTTP requests to HuaweiCloud APIs. The tool handles
// SDK-HMAC-SHA256 signing automatically — the LLM never sees AK/SK.
//
// The LLM provides method, url, optional headers, and optional body.
// The tool signs the request with the configured credentials, sends it,
// and returns the response status, headers, and body as JSON.
type HTTPRequest struct {
	ak            string
	sk            string
	securityToken string
	client        *http.Client
}

// NewHTTPRequest creates an HTTPRequest tool with the given credentials.
// Credentials are read from the environment at construction time (by the
// caller, typically HuaweiCloud.HTTPRequest()) and never exposed to the LLM.
func NewHTTPRequest(ak, sk, securityToken string) *HTTPRequest {
	return &HTTPRequest{
		ak:            ak,
		sk:            sk,
		securityToken: securityToken,
		client: &http.Client{
			// No client-level Timeout — the per-request deadline comes from
			// context.WithTimeout in Execute (default 30s, clamped to [1,120]s
			// via the timeout parameter). A fixed client timeout would cap the
			// parameter silently.
			// Disable redirects — BSS API should respond directly. Following
			// redirects could leak signed requests to unexpected hosts.
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// Definition returns the tool's function definition.
func (t *HTTPRequest) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name: "http_request",
		Description: "Send an HTTP request to a HuaweiCloud API. " +
			"SDK-HMAC-SHA256 authentication is handled automatically — do NOT pass credentials. " +
			"Returns {status, headers, body} as JSON. " +
			"Use this to call BSS pricing APIs and other HuaweiCloud service APIs.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"method":  {"type": "string", "description": "HTTP method: GET, POST, PUT, DELETE, etc."},
				"url":     {"type": "string", "description": "Full URL, e.g. https://bss.myhuaweicloud.com/v2/bills/ratings/on-demand-resources"},
				"headers": {"type": "object", "description": "Optional extra headers (e.g. Content-Type). Do NOT pass Authorization or x-sdk-date — they are auto-signed.", "additionalProperties": {"type": "string"}},
				"body":    {"type": "string", "description": "Optional request body (e.g. JSON string for POST requests)"},
				"timeout": {"type": "integer", "description": "Request timeout in seconds (default: 30, min: 1, max: 120)", "default": 30, "minimum": 1, "maximum": 120}
			},
			"required": ["method", "url"]
		}`),
	}
}

// CanSelfApprove returns true — server-side execution needs no approval.
func (t *HTTPRequest) CanSelfApprove(_ json.RawMessage) bool { return true }

// Execute sends the HTTP request with automatic signing and returns the response.
func (t *HTTPRequest) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Method  string            `json:"method"`
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
		Body    string            `json:"body"`
		Timeout int               `json:"timeout"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("http_request: %w", err)
	}
	if params.URL == "" {
		return "", fmt.Errorf("http_request: url is required")
	}
	if t.ak == "" || t.sk == "" {
		return "", fmt.Errorf("http_request: credentials not configured — set HW_ACCESS_KEY and HW_SECRET_KEY")
	}
	method := strings.ToUpper(params.Method)
	if method == "" {
		method = "GET"
	}

	// Parse the URL into endpoint, path, and query for signing.
	parsed, err := url.Parse(params.URL)
	if err != nil {
		return "", fmt.Errorf("http_request: parse url: %w", err)
	}

	// Build query map for signing (first value of each key).
	query := make(map[string]string)
	for k, v := range parsed.Query() {
		if len(v) > 0 {
			query[k] = v[0]
		}
	}

	// Endpoint is scheme://host (no path/query).
	endpoint := fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)

	// Request body bytes.
	var bodyBytes []byte
	if params.Body != "" {
		bodyBytes = []byte(params.Body)
	}

	// Sign the request.
	signedHeaders := Sign(method, endpoint, parsed.Path, query, bodyBytes, t.ak, t.sk, t.securityToken)

	// Clamp timeout.
	timeout := 30 * time.Second
	if params.Timeout > 0 {
		timeout = time.Duration(params.Timeout) * time.Second
		if timeout < 1*time.Second {
			timeout = 1 * time.Second
		}
		if timeout > 120*time.Second {
			timeout = 120 * time.Second
		}
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build the request.
	var bodyReader io.Reader
	if len(bodyBytes) > 0 {
		bodyReader = strings.NewReader(params.Body)
	}
	req, err := http.NewRequestWithContext(ctx, method, params.URL, bodyReader)
	if err != nil {
		return "", fmt.Errorf("http_request: create request: %w", err)
	}

	// Apply signed headers.
	for k, v := range signedHeaders {
		req.Header.Set(k, v)
	}

	// Apply user-provided headers (they win for non-signing headers).
	// Signing headers (Authorization, host, x-sdk-date, x-security-token)
	// are kept from Sign() — user cannot override them.
	signingKeys := map[string]bool{
		"Authorization":    true,
		"Host":             true,
		"X-Sdk-Date":       true,
		"X-Security-Token": true,
	}
	for k, v := range params.Headers {
		if signingKeys[http.CanonicalHeaderKey(k)] {
			continue // skip — signing headers are managed by Sign()
		}
		req.Header.Set(k, v)
	}

	// Default Content-Type for requests with a body.
	if len(bodyBytes) > 0 && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Send.
	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http_request: %w", err)
	}
	defer utils.DrainAndClose(resp.Body)

	// Read response body (bounded to a generous limit for safety).
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024+1))
	if err != nil {
		return "", fmt.Errorf("http_request: read response: %w", err)
	}

	// Collect response headers.
	respHeaders := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			respHeaders[k] = v[0]
		}
	}

	result := map[string]any{
		"status":  resp.StatusCode,
		"headers": respHeaders,
	}

	// Small body: return inline. Large body: save to artifact file and
	// return a path so the LLM can use read/grep to inspect on demand.
	if len(respBody) <= maxInlineBody {
		result["body"] = string(respBody)
	} else {
		dir := filepath.Join(opentool.ArtifactRoot(), "iac-server")
		_ = os.MkdirAll(dir, 0755)
		name := fmt.Sprintf("http_%d.json", time.Now().UnixNano())
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, respBody, 0644); err != nil {
			// Fallback: return inline truncated.
			result["body"] = string(respBody[:maxInlineBody])
			result["truncated"] = true
		} else {
			sizeKB := (len(respBody) + 1023) / 1024
			result["body"] = fmt.Sprintf("(response body saved to %s, %d KB — use read or grep to inspect)", path, sizeKB)
			result["body_path"] = path
		}
	}

	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("http_request: marshal result: %w", err)
	}
	return string(data), nil
}
