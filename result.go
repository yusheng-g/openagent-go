package openagent

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/yusheng-g/openagent-go/version"
)

// ToolResult is the structured outcome of a [Tool] execution.
//
// Content is the display text shown to the model (truncated when the raw
// result exceeds the runtime result policy threshold — see
// [ApplyResultPolicy]). JSON carries optional structured data for tools
// that produce it (e.g. parsed plan output). Metadata holds arbitrary
// key/value observations (exit code, duration, mime type, ...).
//
// When a result is too large for the model context, the runtime saves the
// raw output to disk, sets Truncated and FileRef, and replaces Content
// with a short pointer the model can read or grep on demand.
type ToolResult struct {
	Content   string          `json:"content,omitempty"`
	JSON      json.RawMessage `json:"json,omitempty"`
	Metadata  map[string]any  `json:"metadata,omitempty"`
	Truncated bool            `json:"truncated,omitempty"`
	FileRef   string          `json:"file_ref,omitempty"`
	Error     *ToolError      `json:"error,omitempty"`
}

// ToolError is a structured tool failure. Retryable marks errors that the
// runtime may retry with backoff (P3 wires the retry policy); Code is an
// optional machine-readable error code.
type ToolError struct {
	Message   string `json:"message"`
	Retryable bool   `json:"retryable,omitempty"`
	Code      string `json:"code,omitempty"`
}

// ErrorResult constructs a ToolResult carrying a structured error.
func ErrorResult(err error, retryable bool, code string) *ToolResult {
	return &ToolResult{Error: &ToolError{
		Message:   err.Error(),
		Retryable: retryable,
		Code:      code,
	}}
}

// AsError returns the tool error as an error value, or nil for a successful
// result. Handy for call sites that want the classic (result, error) shape.
func (r *ToolResult) AsError() error {
	if r == nil || r.Error == nil {
		return nil
	}
	return &toolErrorValue{err: r.Error}
}

// IsErr reports whether the result carries an error.
func (r *ToolResult) IsErr() bool { return r != nil && r.Error != nil }

// toolErrorValue adapts ToolError to the error interface.
type toolErrorValue struct{ err *ToolError }

func (e *toolErrorValue) Error() string { return e.err.Message }

// ── Runtime result policy ──

// artifactFraction is the percentage of the model's context window a single
// tool result may consume before the runtime saves it to disk. Mirrors the
// former hooks/artifact threshold so behavior stays consistent.
const artifactFraction = 5

// ArtifactRoot returns the platform-appropriate artifact directory:
// Linux/macOS /tmp/<version.Name> (default /tmp/openagent), Windows
// %TEMP%\<version.Name>. Runtime result truncation saves oversized output
// here. tool.ArtifactRoot delegates here so the two stay structurally
// identical — there is a single source of truth for the tmp root.
func ArtifactRoot() string {
	return filepath.Join(os.TempDir(), version.SafeName())
}

// ResultPolicy decides how raw tool output becomes the final [ToolResult]
// the model sees. nil ResultPolicy = no truncation.
//
// Implementations must be safe for concurrent use. The runner applies the
// policy after hooks have run (so redaction happens first) and before the
// result enters memory.
type ResultPolicy interface {
	// Apply truncates/saves result in place. Returns the same pointer.
	Apply(ctx context.Context, session Session, result *ToolResult) *ToolResult
}

// DefaultResultPolicy truncates oversized tool results by saving the raw
// output to disk under <ArtifactRoot()>/sess-<sessionID>/ and replacing
// Content with a short pointer. The threshold is token-based, measured
// with the same tokenizer the runner uses for context-window trimming, so
// the two lines agree on one ruler. A result that survives this policy
// (≤ threshold tokens) will not, by itself, push the next turn past the
// window.
//
// The session directory layout (sess-<sessionID>) mirrors the process
// manager's per-session output dir so all session-scoped ephemeral state
// can be cleaned together.
type DefaultResultPolicy struct {
	// ModelID is the tokenizer model id (default "gpt-4").
	ModelID string
	// Window is the context window in tokens; 0 = fall back to the
	// session model's ContextWindow, then to 128 KB.
	Window int
}

// Apply implements [ResultPolicy].
func (p *DefaultResultPolicy) Apply(ctx context.Context, session Session, result *ToolResult) *ToolResult {
	if result == nil || result.Content == "" || result.Error != nil {
		return result
	}

	cw := p.Window
	modelID := p.ModelID
	if modelID == "" {
		modelID = "gpt-4"
	}
	if cw <= 0 && session.Model != nil {
		if w := session.Model.ContextWindow(); w > 0 {
			cw = w
		}
		if tm, ok := session.Model.(TokenizerModeler); ok {
			if name := tm.TokenizerModel(); name != "" {
				modelID = name
			}
		}
	}
	if cw <= 0 {
		cw = 128 * 1024
	}

	threshold := cw * artifactFraction / 100
	if CountTokens(modelID, result.Content) <= threshold {
		return result
	}

	// Guard against the artifact-of-artifact cascade: when "read" targets
	// a path under ArtifactRoot() and the result exceeds the threshold,
	// truncate IN PLACE instead of spilling to a new artifact file.
	//
	// Why in-place and not "skip truncation entirely": the artifact file
	// can be much larger than the model's input limit (a 576 KB artifact
	// vs a 307 K-char API cap). Skipping truncation lets the full content
	// into the prompt, which the model API rejects with a 400 "prompt
	// length exceeds" — the model cannot read the file at all. In-place
	// truncation returns a token-bounded preview (≤ threshold) plus a
	// pointer to the SAME artifact file (FileRef), so the model can page
	// through with read line=N+1. No new artifact file is created, so the
	// cascade (artifact A → read A → artifact B → ...) is broken.
	//
	// Why only "read" (not "grep"): see artifactReadPath. grep's FileRef
	// would point at the grepped file, not the match list — the model
	// could not page correctly. grep overflow is also rare and
	// self-terminating.
	if artifactPath, startLine, hasPrefix := p.artifactReadPath(result.Metadata); artifactPath != "" {
		origBytes := len(result.Content)
		preview, lines := truncatePreview(modelID, result.Content, threshold)
		// truncatePreview counts every '\n' in the truncated Content, but
		// read inserts a "[lines N-M, total, bytes]:\n" summary prefix when
		// line>1 or limit>0 (tool/file.go). That prefix line is NOT file
		// data — counting it would make the continuation hint advance past
		// real content by one line per page, permanently skipping a line
		// each turn (off-by-one pagination). Subtract it when present.
		dataLines := lines
		if hasPrefix && dataLines > 0 {
			dataLines--
		}
		continueLine := startLine + dataLines
		result.Content = preview + fmt.Sprintf(
			"\n... [%d lines shown of a larger artifact file (starting at line %d); to continue, call read with path=%s line=%d]",
			dataLines, startLine, artifactPath, continueLine)
		result.Truncated = true
		result.FileRef = artifactPath
		if result.Metadata == nil {
			result.Metadata = map[string]any{}
		}
		// Match the spill path semantics: artifact_bytes is the ORIGINAL
		// output size, not the truncated preview+hint length, so
		// observers/logs tracking true output size agree across paths.
		result.Metadata["artifact_bytes"] = origBytes
		return result
	}

	dir := filepath.Join(ArtifactRoot(), "sess-"+SanitizeName(session.ID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		// Truncation failed: log instead of silently flooding the model
		// with the raw oversized output.
		slog.Warn("artifact dir create failed; oversized result passed through", "session", session.ID, "error", err)
		return result
	}
	path := filepath.Join(dir, "artifact-"+randHex(8)+".txt")
	raw := result.Content
	// Break overlong single lines (minified JSON, base64, logs without
	// newlines): read/grep cap a single line at 1MB (bufio.Scanner), so
	// an unwrapped megabyte line would make the artifact unreadable and
	// the model could never inspect what it was told to read.
	raw = wrapLongLines(raw, maxArtifactLine)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		slog.Warn("artifact write failed; oversized result passed through", "session", session.ID, "error", err)
		return result
	}

	origBytes := len(result.Content)
	sizeKB := (len(raw) + 1023) / 1024
	lines := strings.Count(raw, "\n") + 1
	result.Content = "Output was too large for the context window, so it was saved to " + path + " (" + strconv.Itoa(sizeKB) + " KB, " + strconv.Itoa(lines) + " lines). Use read with a line range or grep to inspect specific parts."
	result.Truncated = true
	result.FileRef = path
	if result.Metadata == nil {
		result.Metadata = map[string]any{}
	}
	result.Metadata["artifact_bytes"] = origBytes
	return result
}

// artifactReadPath returns the absolute artifact-file path, the 1-based start
// line of the read, and whether read will have inserted its "[lines ...]:"
// summary prefix (true when line>1 or limit>0), when the tool call that
// produced this result is a "read" targeting a path under ArtifactRoot().
// It returns ("",0,false) otherwise. The runtime stamps "tool_name" and
// "tool_args" into result.Metadata before calling Apply; when absent (e.g. a
// third-party policy or a tool that clears metadata) the guard returns
// ("",0,false) and truncation proceeds as usual.
//
// Only "read" is guarded, not "grep":
//   - The cascade's real trigger is read (shell big output → artifact A →
//     read A → artifact B → ...). grep output is matching lines, which is
//     a subset of the file and almost always under threshold; a wide
//     grep (".*") that does overflow naturally terminates (each step's
//     output ⊂ the file), unlike read's identity-copy cascade.
//   - read's FileRef semantics are exact after in-place truncation: the
//     truncated Content is a prefix of the file (from the start line)
//     and FileRef points at that same file, so the model can continue
//     with read line=startLine+linesShown. grep's FileRef would point
//     at the file it grepped, not at the match list — misleading the
//     model if it tries to continue.
//
// hasPrefix is returned so the caller can subtract the prefix line from the
// line count when computing the continuation line number — read inserts the
// prefix unconditionally when line>1 or limit>0, and counting it as file
// data would advance the hint past real content (off-by-one per page).
//
// Path resolution uses filepath.Abs so relative paths like
// "../../tmp/openagent/..." are caught — the old hooks/artifact guard
// used a raw strings.HasPrefix that missed relative paths.
func (p *DefaultResultPolicy) artifactReadPath(meta map[string]any) (path string, startLine int, hasPrefix bool) {
	if meta == nil {
		return "", 0, false
	}
	name, _ := meta["tool_name"].(string)
	if name != "read" {
		return "", 0, false
	}
	raw, _ := meta["tool_args"].(json.RawMessage)
	if len(raw) == 0 {
		return "", 0, false
	}
	var params struct {
		Path  string `json:"path"`
		Line  int    `json:"line"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		return "", 0, false
	}
	if params.Path == "" {
		return "", 0, false
	}
	abs, err := filepath.Abs(filepath.Clean(params.Path))
	if err != nil {
		return "", 0, false
	}
	root := ArtifactRoot()
	if strings.HasPrefix(abs, root+string(filepath.Separator)) {
		// read's Line is 1-based; 0/unspecified means 1. Normalize so
		// the continuation hint (startLine + linesShown) is correct.
		start := params.Line
		if start <= 0 {
			start = 1
		}
		// read inserts its "[lines N-M, total, bytes]:\n" prefix exactly
		// when line>1 or limit>0 (tool/file.go). Mirror that condition so
		// the caller can discount the prefix line from the data line count.
		return abs, start, params.Line > 1 || params.Limit > 0
	}
	return "", 0, false
}

// truncatePreview returns a prefix of s that fits within maxTokens (measured
// with the same tokenizer the runner uses), cut at a line boundary so the
// caller can report an exact "shown N lines, continue at line N+1" hint.
// Also returns the number of complete lines in the preview.
//
// Used by Apply when "read" targets an existing artifact file: instead of
// spilling the content to a NEW artifact file (which would restart the
// cascade), we give the model a bounded preview of the SAME file and let
// it page with read line=N+1. The preview is capped at the result-policy
// threshold (context window × artifactFraction %), which is far below the
// model's input limit, so the prompt never overflows.
//
// Algorithm: estimate a byte cutoff from the token ratio, snap left to the
// previous '\n' (line boundary), then verify with CountTokens. If still
// over budget (the ratio estimate is coarse for non-ASCII), halve and
// re-snap until it fits — at most ~log2(len) iterations.
func truncatePreview(modelID, s string, maxTokens int) (preview string, lines int) {
	if maxTokens <= 0 || len(s) == 0 {
		return "", 0
	}
	if CountTokens(modelID, s) <= maxTokens {
		// Fast path: entire content fits. NOTE: this branch is currently
		// unreachable from Apply, which pre-filters at result.go:147
		// (CountTokens <= threshold → early return) before calling
		// truncatePreview. The +1 here vs the no-+1 in the truncation
		// path below is intentional but semantically inconsistent: this
		// path counts the trailing line without a '\n' (the whole content
		// is shown), while the truncation path counts only complete lines
		// (cut at a '\n', no trailing partial line). If a future caller
		// reaches this path, reconcile the lines semantic — the caller
		// uses `lines` to compute a continuation line number where the
		// no-+1 "complete lines only" meaning is required.
		return s, strings.Count(s, "\n") + 1
	}
	// Initial estimate: bytes ≈ tokens × (len/totalTokens), with 10%
	// headroom for the ratio's coarseness on multi-byte text.
	totalTokens := CountTokens(modelID, s)
	if totalTokens <= 0 {
		totalTokens = 1
	}
	cutoff := len(s) * maxTokens / totalTokens * 9 / 10
	if cutoff < 1 {
		cutoff = 1
	}
	// Snap to the previous line boundary (keep complete lines only).
	snapToLine := func(c int) int {
		if c >= len(s) {
			c = len(s)
		}
		if idx := strings.LastIndexByte(s[:c], '\n'); idx >= 0 {
			return idx + 1 // first byte after the newline
		}
		return 0 // no line boundary found
	}
	cutoff = snapToLine(cutoff)
	// Verify and halve if the estimate was too high.
	for cutoff > 0 && CountTokens(modelID, s[:cutoff]) > maxTokens {
		cutoff = snapToLine(cutoff / 2)
	}
	if cutoff > 0 {
		return s[:cutoff], strings.Count(s[:cutoff], "\n")
	}
	// No line boundary found in the budget: the content is a single line
	// longer than the token budget (minified JSON, base64, a huge log
	// line). Returning ("",0) would give the model "0 lines shown" and
	// force a re-read of the same line → infinite loop. Fall back to a
	// byte-level cut so the preview is non-empty and the model can make
	// progress. The cut may split a token/multibyte rune — we accept
	// that over showing nothing, because the alternative is a livelock
	// where the file content is never visible at all.
	//
	// Byte-level fallback: find the largest byte count ≤ the initial
	// estimate that fits the token budget, without requiring a newline.
	cutoff = len(s) * maxTokens / totalTokens * 9 / 10
	if cutoff < 1 {
		cutoff = 1
	}
	if cutoff > len(s) {
		cutoff = len(s)
	}
	for cutoff > 0 && CountTokens(modelID, s[:cutoff]) > maxTokens {
		cutoff /= 2
	}
	if cutoff == 0 {
		cutoff = 1 // never return empty for non-empty input
	}
	// Report 1 line: there is no newline, but the model sees one logical
	// line of content (truncated). The continuation hint will re-suggest
	// the same start line — read has no byte-offset paging, so the model
	// re-reads the same line and truncatePreview again takes the prefix.
	// This is not a clean page (the line repeats on re-read), but it
	// avoids the livelock and the content is at least visible.
	return s[:cutoff], 1
}

// maxArtifactLine is the line length cap applied when spilling an
// oversized result to disk. tool/read and tool/grep scan line-by-line
// with a 1MB per-line cap; rune-counting keeps the byte size ≤ 4× the
// rune count, so 32K runes stay well inside the 1MB bound even for
// 4-byte (emoji) content. 32K runes (~32-128KB) is also a comfortable
// read unit for the model — a single 128K-rune line would still be a
// wall of text. A var (not const) so tests can shrink it.
var maxArtifactLine = 32 * 1024

// wrapMarker is appended at every artificial break so the model can tell
// a wrapped continuation from a newline that was actually in the original
// output — without it, a minified blob read from the artifact would look
// like real line breaks and could be misparsed (e.g. treated as separate
// log lines).
const wrapMarker = " [line wrapped; continues below]"

// wrapLongLines inserts '\n' every maxLine runes inside lines that run
// longer than maxLine, marking each artificial break with wrapMarker.
// Content-preserving for display purposes: minified JSON / base64 /
// newline-less logs read the same with a marked break every ~32K runes.
func wrapLongLines(s string, maxLine int) string {
	if maxLine <= 0 {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + (len(s)/maxLine+1)*(len(wrapMarker)+1))
	count := 0
	for _, r := range s {
		b.WriteRune(r)
		// Both '\n' and '\r' terminate a line: '\r'-only line endings
		// (old Mac, some tool output) and Windows '\r\n' must not be
		// counted as content — otherwise a '\r'-separated blob reads as
		// one huge line and gets falsely wrapped (with a continuation
		// marker the model would misread as a single long line).
		if r == '\n' || r == '\r' {
			count = 0
			continue
		}
		count++
		if count >= maxLine {
			b.WriteString(wrapMarker)
			b.WriteByte('\n')
			count = 0
		}
	}
	return b.String()
}

// SanitizeName replaces path separators (and NUL) with '_' so a hostile
// or malformed session id cannot escape its session directory. Export for
// callers that build session-scoped paths (REST cleanup, artifacts).
func SanitizeName(name string) string {
	return strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' || r == 0 {
			return '_'
		}
		return r
	}, name)
}

// artifactSeq disambiguates artifact names when crypto/rand fails (all-zero
// names would collide and overwrite each other).
var artifactSeq atomic.Uint64

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte(strconv.FormatInt(time.Now().UnixNano(), 10))) + "-" + strconv.FormatUint(artifactSeq.Add(1), 10)
	}
	return hex.EncodeToString(b)
}
