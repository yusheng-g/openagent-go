// Command export implements `openagent export` — a fully offline command
// that reads the local SQLite session store and writes every session to a
// timestamped directory under the current working directory (or --output).
//
// Each session becomes one file named "<createdAt>_<title>.<ext>". Two
// formats are supported: json (default, lossless) and markdown (human
// readable). The command skips plugin loading, model initialization, and
// network access — it only touches config.Dir()/memory/memory.db.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/cmd/cli/config"
	"github.com/yusheng-g/openagent-go/session"
	sessionsqlite "github.com/yusheng-g/openagent-go/session/sqlite"
)

const (
	// timestampFormat is the compact, filesystem-safe time format used
	// for both the export directory name and per-session file names.
	timestampFormat = "20060102-150405"
	// maxFilenameLen caps the title portion of a file name, measured in
	// runes (not bytes), to avoid excessively long paths on some
	// filesystems. 100 runes keeps paths reasonable even with CJK titles.
	maxFilenameLen = 100
	// exportDirPrefix is prepended to the timestamped export directory
	// name so exports are visually distinct from other directories.
	exportDirPrefix = "export-"
	// untitledName is the fallback title used when a session has no
	// title or the sanitized title is empty.
	untitledName = "untitled"
)

// invalidFilenameChars matches characters that are unsafe in file names
// on any mainstream OS: path separators, wildcards, shell redirects,
// and control characters (newlines, tabs, etc.) that break file names
// or shell display.
var invalidFilenameChars = regexp.MustCompile(`[/\\:*?"<>|\x00-\x1f\x7f]`)

// collapseUnderscores matches runs of underscores so consecutive
// replacements (e.g. "a//b" → "a__b") collapse to a single separator.
var collapseUnderscores = regexp.MustCompile(`_+`)

// exportFormat selects the output format for a session export.
type exportFormat string

const (
	exportJSON     exportFormat = "json"
	exportMarkdown exportFormat = "markdown"
)

// exportPayload is the JSON structure for a single exported session.
// It bundles session metadata, the full message list, and any compressed
// context so the export is lossless and a future import command can
// reconstruct the session entirely.
type exportPayload struct {
	Session    session.SessionInfo          `json:"session"`
	Messages   []openagent.Message          `json:"messages"`
	Compressed *openagent.CompressedContext `json:"compressed"`
}

// buildExportCmd creates the `export` cobra subcommand. It accepts
// --format (-f) to choose json or markdown output and --output (-o) to
// override the destination directory (default: current working directory).
func buildExportCmd() *cobra.Command {
	var format string
	var outputDir string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export all sessions to a timestamped directory (offline)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cwd, _ := cmd.Flags().GetString("cwd"); cwd != "" {
				if err := os.Chdir(cwd); err != nil {
					return fmt.Errorf("--cwd: %w", err)
				}
			}
			quiet, _ := cmd.Flags().GetBool("quiet")
			return runExport(cmd.Context(), exportFormat(format), outputDir, quiet)
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "json", "Output format: json or markdown")
	cmd.Flags().StringVarP(&outputDir, "output", "o", "", "Output directory (default: current working directory)")

	return cmd
}

// runExport orchestrates the full export: opens the session stores,
// lists all sessions, creates a timestamped output directory, and writes
// one file per session. Progress is printed to stderr unless quiet is set.
func runExport(ctx context.Context, format exportFormat, outputDir string, quiet bool) error {
	if outputDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get cwd: %w", err)
		}
		outputDir = cwd
	}

	ms, store, err := openSessionStores()
	if err != nil {
		return err
	}
	defer ms.Close()
	defer store.Close()

	sessions, err := store.List(ctx)
	if err != nil {
		return fmt.Errorf("list sessions: %w", err)
	}

	if len(sessions) == 0 {
		if !quiet {
			fmt.Fprintln(os.Stderr, "No sessions found.")
		}
		return nil
	}

	dirName := exportDirPrefix + time.Now().Format(timestampFormat)
	exportDir := filepath.Join(outputDir, dirName)
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return fmt.Errorf("create export dir: %w", err)
	}

	if !quiet {
		fmt.Fprintf(os.Stderr, "Exporting %d session(s) to %s\n", len(sessions), exportDir)
	}

	for _, info := range sessions {
		if err := exportSession(ctx, ms, exportDir, format, info, quiet); err != nil {
			if !quiet {
				fmt.Fprintf(os.Stderr, "  SKIP %s: %v\n", info.ID, err)
			}
			continue
		}
	}

	if !quiet {
		fmt.Fprintf(os.Stderr, "Done: %s\n", exportDir)
	}
	return nil
}

// exportSession reads a single session's messages and compressed context
// from the store, renders them in the requested format, and writes the
// result to a file named "<createdAt>_<title>.<ext>" under dir.
func exportSession(ctx context.Context, ms *sessionsqlite.MessageStore, dir string, format exportFormat, info session.SessionInfo, quiet bool) error {
	count, err := ms.Count(ctx, info.ID)
	if err != nil {
		return fmt.Errorf("count messages: %w", err)
	}

	// RecentAfter with throughIndex=0 returns all messages in chronological
	// order without trimming leading tool messages (unlike Recent, which
	// drops orphaned tool results for model-context use).
	var messages []openagent.Message
	if count > 0 {
		messages, err = ms.RecentAfter(ctx, info.ID, 0, count)
		if err != nil {
			return fmt.Errorf("read messages: %w", err)
		}
	}

	compressed, err := ms.Compressed(ctx, info.ID)
	if err != nil {
		return fmt.Errorf("read compressed: %w", err)
	}

	var content string
	var ext string
	switch format {
	case exportMarkdown:
		ext = ".md"
		content = renderMarkdown(info, messages, compressed)
	default:
		ext = ".json"
		content, err = renderJSON(info, messages, compressed)
		if err != nil {
			return fmt.Errorf("render json: %w", err)
		}
	}

	filename := buildFilename(info, ext)
	path := filepath.Join(dir, filename)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	if !quiet {
		fmt.Fprintf(os.Stderr, "  Wrote %s\n", filename)
	}
	return nil
}

// renderJSON serializes a session, its messages, and compressed context
// into a pretty-printed JSON string. The compressed field is null when
// no compression context exists, keeping the export lossless.
func renderJSON(info session.SessionInfo, messages []openagent.Message, compressed *openagent.CompressedContext) (string, error) {
	payload := exportPayload{
		Session:    info,
		Messages:   messages,
		Compressed: compressed,
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// renderMarkdown renders a session as a human-readable Markdown document:
// a metadata header (ID, title, timestamps), an optional summary section
// when compressed context exists, then numbered sections for each message
// — user and assistant as top-level headings, tool calls and reasoning
// as sub-headings. Tool arguments are fenced as ```json blocks; tool
// results as plain ``` blocks.
func renderMarkdown(info session.SessionInfo, messages []openagent.Message, compressed *openagent.CompressedContext) string {
	var b strings.Builder

	b.WriteString("# Chat Transcript\n\n")
	fmt.Fprintf(&b, "- Session: %s\n", info.ID)
	if info.Title != "" {
		fmt.Fprintf(&b, "- Title: %s\n", info.Title)
	}
	if !info.CreatedAt.IsZero() {
		fmt.Fprintf(&b, "- Created: %s\n", info.CreatedAt.Format(time.RFC3339))
	}
	if !info.UpdatedAt.IsZero() {
		fmt.Fprintf(&b, "- Updated: %s\n", info.UpdatedAt.Format(time.RFC3339))
	}
	b.WriteString("\n")

	if compressed != nil && compressed.Summary != "" {
		b.WriteString("## Summary\n\n")
		b.WriteString(compressed.Summary)
		b.WriteString("\n\n")
	}

	for i, msg := range messages {
		switch msg.Role {
		case openagent.RoleUser:
			fmt.Fprintf(&b, "## %d. User\n\n%s\n\n", i+1, msg.Content)
		case openagent.RoleAssistant:
			fmt.Fprintf(&b, "## %d. Assistant\n\n%s\n\n", i+1, msg.Content)
			if msg.ReasoningContent != "" {
				fmt.Fprintf(&b, "### %d.%d. Thought\n\n> %s\n\n", i+1, 1, msg.ReasoningContent)
			}
			for j, tc := range msg.ToolCalls {
				fmt.Fprintf(&b, "### %d.%d. Tool: %s\n\n", i+1, j+2, tc.Function.Name)
				if tc.Function.Arguments != "" {
					b.WriteString("```json\n")
					b.WriteString(tc.Function.Arguments)
					b.WriteString("\n```\n\n")
				}
				if msg.Result != nil && msg.Result.Content != "" {
					b.WriteString("```\n")
					b.WriteString(msg.Result.Content)
					b.WriteString("\n```\n\n")
				}
			}
		case openagent.RoleTool:
			fmt.Fprintf(&b, "### %d. Tool Result\n\n", i+1)
			if msg.Content != "" {
				b.WriteString("```\n")
				b.WriteString(msg.Content)
				b.WriteString("\n```\n\n")
			}
		case openagent.RoleSystem:
			fmt.Fprintf(&b, "## %d. System\n\n%s\n\n", i+1, msg.Content)
		default:
			fmt.Fprintf(&b, "## %d. %s\n\n%s\n\n", i+1, msg.Role, msg.Content)
		}
	}

	return b.String()
}

// buildFilename constructs the per-session file name from the session's
// creation timestamp and sanitized title: "<timestamp>_<title><ext>".
// An empty title falls back to "untitled".
func buildFilename(info session.SessionInfo, ext string) string {
	ts := timestampName(info.CreatedAt)
	title := sanitizeFilename(info.Title)
	if title == "" {
		title = untitledName
	}
	return ts + "_" + title + ext
}

// timestampName formats a time as a compact, sortable string. A zero
// time falls back to the current time so sessions with missing creation
// timestamps still get a deterministic file name.
func timestampName(t time.Time) string {
	if t.IsZero() {
		return time.Now().Format(timestampFormat)
	}
	return t.Format(timestampFormat)
}

// sanitizeFilename replaces characters unsafe in file names with
// underscores, collapses consecutive underscores, trims surrounding
// whitespace and underscores, and truncates to maxFilenameLen runes.
// Truncation is rune-aware (not byte-aware) so multi-byte characters
// such as CJK text are never split mid-codepoint, which would produce
// invalid UTF-8.
func sanitizeFilename(s string) string {
	s = invalidFilenameChars.ReplaceAllString(s, "_")
	s = collapseUnderscores.ReplaceAllString(s, "_")
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "_.")
	if len(s) > maxFilenameLen {
		runes := []rune(s)
		if len(runes) > maxFilenameLen {
			runes = runes[:maxFilenameLen]
		}
		s = strings.TrimRight(string(runes), "_.")
	}
	return s
}

// openSessionStores opens the SQLite message store and metadata store
// at config.Dir()/memory/memory.db — the same path used by the server
// at runtime. The caller is responsible for closing both stores.
func openSessionStores() (*sessionsqlite.MessageStore, *sessionsqlite.Store, error) {
	memDir := filepath.Join(config.Dir(), "memory")
	path := filepath.Join(memDir, "memory.db")

	ms, err := sessionsqlite.NewMessageStore(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open message store: %w", err)
	}

	store, err := sessionsqlite.New(ms.DB())
	if err != nil {
		ms.Close()
		return nil, nil, fmt.Errorf("open session store: %w", err)
	}

	return ms, store, nil
}
