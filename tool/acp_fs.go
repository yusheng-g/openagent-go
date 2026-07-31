package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	openagent "github.com/yusheng-g/openagent-go"
	openacp "github.com/yusheng-g/openagent-go/acp/sdk"
)

// ACPReadFile reads a file from the client's filesystem via fs/read_text_file.
// This is an Agent→Client RPC — the agent asks the client to read a file.
type ACPReadFile struct {
	client    openacp.ClientRequester
	sessionID openacp.SessionId
}

func NewACPReadFile(client openacp.ClientRequester, sid openacp.SessionId) *ACPReadFile {
	return &ACPReadFile{client: client, sessionID: sid}
}

func (t *ACPReadFile) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        "read_client_file",
		Description: "Read a file from the client's filesystem. Use this when the file is on the user's machine rather than the agent's workspace. Path must be absolute.",
		Parameters: json.RawMessage(`{
  "type": "object",
  "properties": {
    "path":    { "type": "string", "description": "Absolute path to the file to read." },
    "line":    { "type": "integer", "description": "Optional 1-based line number to start reading from." },
    "limit":   { "type": "integer", "description": "Optional maximum number of lines to read." }
  },
  "required": ["path"]
}`),
	}
}

func (t *ACPReadFile) IsReadOnly() bool { return true }

func (t *ACPReadFile) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Path  string `json:"path"`
		Line  *int   `json:"line"`
		Limit *int   `json:"limit"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("read_client_file: invalid arguments: %w", err)
	}
	if strings.TrimSpace(params.Path) == "" {
		return "", fmt.Errorf("read_client_file: path is required")
	}

	resp, err := t.client.ReadTextFile(ctx, openacp.ReadTextFileRequest{
		SessionID: t.sessionID,
		Path:      params.Path,
		Line:      params.Line,
		Limit:     params.Limit,
	})
	if err != nil {
		return "", fmt.Errorf("read_client_file: %w", err)
	}
	return resp.Content, nil
}

// ACPWriteFile writes content to a file on the client's filesystem via
// fs/write_text_file. This is an Agent→Client RPC.
type ACPWriteFile struct {
	client    openacp.ClientRequester
	sessionID openacp.SessionId
}

func NewACPWriteFile(client openacp.ClientRequester, sid openacp.SessionId) *ACPWriteFile {
	return &ACPWriteFile{client: client, sessionID: sid}
}

func (t *ACPWriteFile) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        "write_client_file",
		Description: "Write content to a file on the client's filesystem. Use this when the file needs to be written to the user's machine rather than the agent's workspace. Path must be absolute.",
		Parameters: json.RawMessage(`{
  "type": "object",
  "properties": {
    "path":    { "type": "string", "description": "Absolute path to the file to write." },
    "content": { "type": "string", "description": "Text content to write to the file." }
  },
  "required": ["path","content"]
}`),
	}
}

func (t *ACPWriteFile) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("write_client_file: invalid arguments: %w", err)
	}
	if strings.TrimSpace(params.Path) == "" {
		return "", fmt.Errorf("write_client_file: path is required")
	}

	_, err := t.client.WriteTextFile(ctx, openacp.WriteTextFileRequest{
		SessionID: t.sessionID,
		Path:      params.Path,
		Content:   params.Content,
	})
	if err != nil {
		return "", fmt.Errorf("write_client_file: %w", err)
	}
	return "File written successfully.", nil
}
