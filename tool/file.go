package tool

import (
	"bufio"
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
		Description: "Read a file from the given path. Use line+limit to read a specific line range — combine with grep to locate a line number first, then read the surrounding context.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path":  {"type": "string",  "description": "File path"},
				"line":  {"type": "integer", "description": "Start line (1-based, default: 1). Use with limit to read a specific range."},
				"limit": {"type": "integer", "description": "Max lines to read (default: all remaining). Use with line to read a window around a grep hit."}
			},
			"required": ["path"]
		}`),
	}
}

func (t *ReadFile) IsReadOnly() bool { return true }

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
	return isWithinWorkspace(t.workDir, abs) || isWithinArtifactDir(abs)
}

func (t *ReadFile) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Path  string `json:"path"`
		Line  int    `json:"line"`  // 1-based, 0 = default (1)
		Limit int    `json:"limit"` // 0 = default (all)
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("read: %w", err)
	}
	if params.Path == "" {
		return "", fmt.Errorf("read: path is required")
	}
	if params.Line < 0 {
		params.Line = 0
	}
	if params.Line == 0 {
		params.Line = 1
	}

	abs, err := validatePath(t.workDir, params.Path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("read: file not found: %s", params.Path)
		}
		return "", fmt.Errorf("read: %w", err)
	}

	file, err := os.Open(abs)
	if err != nil {
		return "", fmt.Errorf("read: %w", err)
	}
	defer file.Close()

	// Binary detection: peek first 512 bytes for null bytes.
	peek := make([]byte, 512)
	n, _ := file.Read(peek)
	if n > 0 && isBinary(peek[:n]) {
		return fmt.Sprintf("[binary file: %s, %d bytes, type: %s]",
			params.Path, info.Size(), detectType(peek[:n])), nil
	}
	// Rewind to beginning.
	file.Seek(0, 0)

	var (
		out       strings.Builder
		lineNum   int
		lineCount int
		hitOffset bool
	)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // 1 MB max line

	for scanner.Scan() {
		lineNum++
		if lineNum < params.Line {
			continue
		}
		hitOffset = true
		out.WriteString(scanner.Text())
		out.WriteByte('\n')
		lineCount++
		if params.Limit > 0 && lineCount >= params.Limit {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read: %w", err)
	}

	if !hitOffset {
		return fmt.Sprintf("[line %d is beyond end of file (%d lines)]", params.Line, lineNum), nil
	}

	result := out.String()
	if params.Line > 1 || params.Limit > 0 {
		prefix := fmt.Sprintf("[lines %d-%d, %d total, %d bytes]:\n",
			params.Line, params.Line+lineCount-1, lineNum, info.Size())
		result = prefix + result
	}
	return result, nil
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
	if n >= 4 && data[0] == 0x7f && data[1] == 'E' && data[2] == 'L' && data[3] == 'F' {
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
		Description: "Write content to a file. Creates parent directories as needed.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path":    {"type": "string", "description": "File path"},
				"content": {"type": "string", "description": "Content to write to the file"}
			},
			"required": ["path", "content"]
		}`),
	}
}

func (t *WriteFile) CanSelfApprove(args json.RawMessage) bool {
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
	return isWithinWorkspace(t.workDir, abs) || isWithinArtifactDir(abs)
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
		Description: "List files and directories at the given path.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {"type": "string", "description": "Directory path"}
			},
			"required": ["path"]
		}`),
	}
}

func (t *ListDir) IsReadOnly() bool { return true }

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
	return isWithinWorkspace(t.workDir, abs) || isWithinArtifactDir(abs)
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

// ── EditFile ──

// EditFile performs exact string replacement in a file.
type EditFile struct {
	workDir string
}

func NewEditFile(workDir string) *EditFile {
	abs, _ := filepath.Abs(workDir)
	return &EditFile{workDir: abs}
}

func (t *EditFile) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        "edit",
		Description: "Replace a string in a file. Finds old_text and replaces it with new_text. When replace_all is false (default), only the first match is replaced. Returns an error when old_text is not unique — use replace_all or make old_text more specific.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path":        {"type": "string", "description": "File path"},
				"old_text":    {"type": "string", "description": "Text to find and replace"},
				"new_text":    {"type": "string", "description": "Replacement text"},
				"replace_all": {"type": "boolean", "description": "Replace all occurrences (default: false)"}
			},
			"required": ["path", "old_text", "new_text"]
		}`),
	}
}

func (t *EditFile) CanSelfApprove(args json.RawMessage) bool {
	var params struct {
		Path string `json:"path"`
	}
	json.Unmarshal(args, &params)
	if params.Path == "" {
		return false
	}
	abs, err := validatePath(t.workDir, params.Path)
	if err != nil {
		return false
	}
	return isWithinWorkspace(t.workDir, abs) || isWithinArtifactDir(abs)
}

func (t *EditFile) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Path       string `json:"path"`
		OldText    string `json:"old_text"`
		NewText    string `json:"new_text"`
		ReplaceAll bool   `json:"replace_all"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("edit: %w", err)
	}
	if params.Path == "" {
		return "", fmt.Errorf("edit: path is required")
	}
	if params.OldText == "" {
		return "", fmt.Errorf("edit: old_text is required")
	}

	abs, err := validatePath(t.workDir, params.Path)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("edit: file not found: %s", params.Path)
		}
		return "", fmt.Errorf("edit: %w", err)
	}

	content := string(data)
	count := strings.Count(content, params.OldText)
	if count == 0 {
		return "", fmt.Errorf("edit: old_text not found in %s", params.Path)
	}
	if !params.ReplaceAll && count > 1 {
		return "", fmt.Errorf("edit: old_text found %d times in %s — set replace_all to true or make old_text more specific", count, params.Path)
	}

	n := 1
	if params.ReplaceAll {
		n = count
	}
	newContent := strings.Replace(content, params.OldText, params.NewText, n)
	if err := os.WriteFile(abs, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("edit: %w", err)
	}

	if params.ReplaceAll {
		return fmt.Sprintf("Replaced %d occurrences in %s", count, params.Path), nil
	}
	return fmt.Sprintf("Replaced in %s", params.Path), nil
}
