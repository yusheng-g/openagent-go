package tool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	openagent "github.com/yusheng-g/openagent-go"
)

// validatePath resolves p against workDir into a safe absolute path.
// Accepts both relative paths (joined with workDir) and absolute paths.
// Resolves symlinks but does NOT enforce workspace boundaries —
// that policy belongs to the [openagent.Approver].
func validatePath(workDir, p string) (string, error) {
	var abs string
	var err error
	if filepath.IsAbs(p) {
		abs = p
	} else {
		abs = filepath.Join(workDir, p)
	}
	abs, err = filepath.Abs(abs)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}
	// Resolve symlinks to prevent /workspace/link → /etc escapes.
	real, err := filepath.EvalSymlinks(abs)
	if err == nil {
		abs = real
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("invalid path: %w", err)
	}
	return abs, nil
}

// ── ReadFile ──

// isWithinWorkspace reports whether resolved (absolute, symlink-resolved) is
// within workDir. Returns false if resolved escapes the workspace boundary.
func isWithinWorkspace(workDir, resolved string) bool {
	return resolved == workDir || strings.HasPrefix(resolved, workDir+string(os.PathSeparator))
}

// ReadFile reads a file from the sandbox workspace.
type ReadFile struct {
	workDir string
}

func NewReadFile(workDir string) *ReadFile {
	abs, _ := filepath.Abs(workDir)
	return &ReadFile{workDir: abs}
}

func (t *ReadFile) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        "read",
		Description: "Read a file from the workspace. Path relative to workspace root.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {"type": "string", "description": "File path relative to workspace root"}
			},
			"required": ["path"]
		}`),
	}
}

func (t *ReadFile) CanSelfApprove(args json.RawMessage) bool {
	var params struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &params); err != nil || params.Path == "" {
		return false
	}
	abs, err := validatePath(t.workDir, params.Path)
	if err != nil {
		return false
	}
	return isWithinWorkspace(t.workDir, abs)
}

func (t *ReadFile) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("read: %w", err)
	}
	if params.Path == "" {
		return "", fmt.Errorf("read: path is required")
	}

	abs, err := validatePath(t.workDir, params.Path)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("read: file not found: %s", params.Path)
		}
		if os.IsPermission(err) {
			return "", fmt.Errorf("read: permission denied: %s", params.Path)
		}
		return "", fmt.Errorf("read: %w", err)
	}

	// Binary detection: check first 512 bytes for null bytes.
	// If the file is binary, return a clear message so the model doesn't
	// try to parse garbage or mistakenly treat it as a directory.
	if isBinary(data) {
		return fmt.Sprintf("[binary file: %s, %d bytes, type: %s]",
			params.Path, len(data), detectType(data)), nil
	}

	const maxSize = 100 * 1024 // 100KB
	if len(data) > maxSize {
		return string(data[:maxSize]) + fmt.Sprintf("\n... [truncated, %d bytes total]", len(data)), nil
	}
	return string(data), nil
}

func isBinary(data []byte) bool {
	n := len(data)
	if n > 512 {
		n = 512
	}
	for _, b := range data[:n] {
		if b == 0 {
			return true
		}
	}
	return false
}

func detectType(data []byte) string {
	n := len(data)
	if n > 64 {
		n = 64
	}
	for _, b := range data[:n] {
		if b < 9 || (b > 13 && b < 32) && b != 27 {
			return "binary data"
		}
	}
	if data[0] == 0x7f && data[1] == 'E' && data[2] == 'L' && data[3] == 'F' {
		return "ELF executable"
	}
	return "unknown binary"
}

// ── WriteFile ──

// WriteFile writes content to a file in the sandbox workspace.
type WriteFile struct {
	workDir string
}

func NewWriteFile(workDir string) *WriteFile {
	abs, _ := filepath.Abs(workDir)
	return &WriteFile{workDir: abs}
}

func (t *WriteFile) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        "write",
		Description: "Write content to a file in the workspace. Creates parent directories as needed.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path":    {"type": "string", "description": "File path relative to workspace root"},
				"content": {"type": "string", "description": "Content to write to the file"}
			},
			"required": ["path", "content"]
		}`),
	}
}

func (t *WriteFile) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("write: %w", err)
	}
	if params.Path == "" {
		return "", fmt.Errorf("write: path is required")
	}

	const maxSize = 10 * 1024 * 1024 // 10MB
	if len(params.Content) > maxSize {
		return "", fmt.Errorf("write: content too large (%d bytes, max %d)", len(params.Content), maxSize)
	}

	abs, err := validatePath(t.workDir, params.Path)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		return "", fmt.Errorf("write: %w", err)
	}
	if err := os.WriteFile(abs, []byte(params.Content), 0644); err != nil {
		return "", fmt.Errorf("write: %w", err)
	}

	info, _ := os.Stat(abs)
	size := int64(0)
	if info != nil {
		size = info.Size()
	}
	return fmt.Sprintf("Wrote %s (%d bytes)", params.Path, size), nil
}

// ── ListDir ──

// ListDir lists directory contents in the sandbox workspace.
type ListDir struct {
	workDir string
}

func NewListDir(workDir string) *ListDir {
	abs, _ := filepath.Abs(workDir)
	return &ListDir{workDir: abs}
}

func (t *ListDir) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        "ls",
		Description: "List files and directories in the workspace. Path relative to workspace root (default: root).",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {"type": "string", "description": "Directory path relative to workspace root (default: root)"}
			}
		}`),
	}
}

func (t *ListDir) CanSelfApprove(args json.RawMessage) bool {
	var params struct {
		Path string `json:"path"`
	}
	json.Unmarshal(args, &params)
	if params.Path == "" {
		return true // default to workspace root is safe
	}
	abs, err := validatePath(t.workDir, params.Path)
	if err != nil {
		return false
	}
	return isWithinWorkspace(t.workDir, abs)
}

func (t *ListDir) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Path string `json:"path"`
	}
	json.Unmarshal(args, &params)

	dir, err := validatePath(t.workDir, params.Path)
	if err != nil {
		// Empty path defaults to workspace root.
		if params.Path == "" {
			dir = t.workDir
		} else {
			return "", err
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("ls: %w", err)
	}

	type fileEntry struct {
		Name  string
		Size  int64
		IsDir bool
	}

	var files []fileEntry
	for _, e := range entries {
		info, err := e.Info()
		size := int64(0)
		if err == nil {
			size = info.Size()
		}
		files = append(files, fileEntry{e.Name(), size, e.IsDir()})
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	var b strings.Builder
	if params.Path != "" {
		b.WriteString(params.Path + ":\n")
	}
	for _, f := range files {
		if f.IsDir {
			b.WriteString(fmt.Sprintf("  %s/\n", f.Name))
		} else {
			b.WriteString(fmt.Sprintf("  %s  (%d)\n", f.Name, f.Size))
		}
	}
	if len(files) == 0 {
		b.WriteString("  (empty)\n")
	}
	return b.String(), nil
}
