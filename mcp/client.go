package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"

	openagent "github.com/yusheng-g/openagent-go"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Client connects to an external MCP server and imports its tools.
//
// Create with [NewClient], connect to a server, then use [Session.Tools]
// to get openagent.Tool wrappers:
//
//	client := mcp.NewClient("my-agent", "1.0.0")
//	session, _ := client.ConnectStdio(ctx, "my-mcp-server")
//	tools, _ := session.Tools(ctx)
//	agent := openagent.NewAgent("bot", openagent.WithTools(tools...))
type Client struct {
	inner *mcpsdk.Client
}

// NewClient creates an MCP [Client] with the given implementation identity.
func NewClient(name, version string) *Client {
	return &Client{
		inner: mcpsdk.NewClient(&mcpsdk.Implementation{
			Name: name, Version: version,
		}, nil),
	}
}

// ConnectStdio connects to an MCP server by spawning it as a subprocess and
// communicating over stdin/stdout. command is the executable; args are its
// arguments. The process inherits the parent environment.
//
// The context is used to start the process. Call [Session.Close] to terminate
// the process and clean up.
func (c *Client) ConnectStdio(ctx context.Context, command string, args ...string) (*Session, error) {
	return c.connectStdioInternal(ctx, command, args, nil)
}

// ConnectStdioWithEnv is like ConnectStdio but appends additional
// environment variables (name=value pairs) to the spawned process.
func (c *Client) ConnectStdioWithEnv(ctx context.Context, command string, args []string, env []string) (*Session, error) {
	return c.connectStdioInternal(ctx, command, args, env)
}

func (c *Client) connectStdioInternal(ctx context.Context, command string, args []string, env []string) (*Session, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	if len(env) > 0 {
		cmd.Env = append(cmd.Environ(), env...)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp stdout pipe: %w", err)
	}
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("mcp start %q: %w", command, err)
	}

	transport := &mcpsdk.IOTransport{Reader: stdout, Writer: stdin}
	sess, err := c.inner.Connect(ctx, transport, nil)
	if err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		if stderr := stderrBuf.String(); stderr != "" {
			return nil, fmt.Errorf("mcp connect stdio: %w\nstderr:\n%s", err, stderr)
		}
		return nil, fmt.Errorf("mcp connect stdio: %w", err)
	}

	return &Session{inner: sess, cmd: cmd, stdin: stdin, stderrBuf: &stderrBuf}, nil
}

// ConnectHTTP connects to an MCP server over HTTP/SSE.
func (c *Client) ConnectHTTP(ctx context.Context, endpoint string) (*Session, error) {
	transport := &mcpsdk.StreamableClientTransport{
		Endpoint: endpoint,
	}
	sess, err := c.inner.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("mcp connect http: %w", err)
	}
	return &Session{inner: sess}, nil
}

// ConnectTransport connects using a custom MCP transport.
// The transport can connect to any MCP server (stdio subprocess,
// HTTP endpoint, in-memory, etc.).
func (c *Client) ConnectTransport(ctx context.Context, transport mcpsdk.Transport) (*Session, error) {
	sess, err := c.inner.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("mcp connect: %w", err)
	}
	return &Session{inner: sess}, nil
}

// Session is an active MCP connection. Use [Session.Tools] to import tools.
type Session struct {
	inner *mcpsdk.ClientSession
	cmd   *exec.Cmd // non-nil for stdio connections

	// stdin is the write end of the process's stdin pipe. Closed in Close()
	// to signal EOF before killing the process.
	stdin io.WriteCloser

	// stderrBuf captures the process's stderr output for diagnostics.
	stderrBuf *bytes.Buffer
}

// Tools lists tools available on the MCP server and returns them as
// openagent.Tool wrappers. Each tool's Execute method calls the MCP
// server via CallTool.
func (s *Session) Tools(ctx context.Context) ([]openagent.Tool, error) {
	resp, err := s.inner.ListTools(ctx, &mcpsdk.ListToolsParams{})
	if err != nil {
		return nil, fmt.Errorf("mcp list tools: %w", err)
	}

	tools := make([]openagent.Tool, 0, len(resp.Tools))
	for _, t := range resp.Tools {
		def, err := ToFunctionDefinition(*t)
		if err != nil {
			return nil, err
		}
		tools = append(tools, &mcpToolAdapter{
			session: s.inner,
			def:     def,
		})
	}
	return tools, nil
}

// ListTools returns the raw MCP tool definitions (without wrapping as
// openagent.Tool). Use this if you only need metadata.
func (s *Session) ListTools(ctx context.Context) ([]mcpsdk.Tool, error) {
	resp, err := s.inner.ListTools(ctx, &mcpsdk.ListToolsParams{})
	if err != nil {
		return nil, err
	}
	tools := make([]mcpsdk.Tool, len(resp.Tools))
	for i, t := range resp.Tools {
		tools[i] = *t
	}
	return tools, nil
}

// Close terminates the MCP session. For stdio connections, stdin is closed
// first to signal EOF, then the subprocess is terminated.
func (s *Session) Close() error {
	// Close stdin to signal EOF before killing the process.
	if s.stdin != nil {
		_ = s.stdin.Close()
	}

	err := s.inner.Close()
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		_ = s.cmd.Wait()
	}
	return err
}

// Stderr returns the captured stderr output of the subprocess.
// Empty string if nothing was written to stderr. Use this for
// diagnostics when the process fails or behaves unexpectedly.
func (s *Session) Stderr() string {
	if s.stderrBuf == nil {
		return ""
	}
	return s.stderrBuf.String()
}

// ── mcpToolAdapter ──

// mcpToolAdapter wraps an MCP tool as an openagent.Tool.
type mcpToolAdapter struct {
	session *mcpsdk.ClientSession
	def     openagent.FunctionDefinition
}

func (a *mcpToolAdapter) Definition() openagent.FunctionDefinition {
	return a.def
}

func (a *mcpToolAdapter) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	// Unmarshal arguments to map[string]any (MCP CallToolParams expects this).
	var v map[string]any
	if len(args) > 0 {
		if err := json.Unmarshal(args, &v); err != nil {
			return "", fmt.Errorf("mcp: unmarshal args: %w", err)
		}
	}

	result, err := a.session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      a.def.Name,
		Arguments: v,
	})
	if err != nil {
		return "", fmt.Errorf("mcp call tool %q: %w", a.def.Name, err)
	}

	// Extract text content from the result.
	text := extractText(result.Content)
	if result.IsError {
		return text, fmt.Errorf("%s", text)
	}
	return text, nil
}

// extractText concatenates all TextContent from an MCP result.
func extractText(contents []mcpsdk.Content) string {
	var out string
	for _, c := range contents {
		if tc, ok := c.(*mcpsdk.TextContent); ok {
			if out != "" {
				out += "\n"
			}
			out += tc.Text
		}
	}
	return out
}


