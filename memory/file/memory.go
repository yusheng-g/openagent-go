// Package file implements openagent.Memory with JSONL files on disk.
// Zero external dependencies — uses only the standard library.
//
// Each session is stored as a JSONL file (one JSON object per line).
// Search uses case-insensitive substring matching on message content.
//
// Usage:
//
//	mem, err := file.New("/path/to/memory/dir")
//	mem.WithSummarizer(openai.NewSummarizer(...))  // optional, enables compaction
//	agent := openagent.NewAgent("bot", openagent.WithMemory(mem))
package file

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	openagent "github.com/yusheng-g/openagent-go"
)

// Memory implements openagent.Memory backed by JSONL files.
type Memory struct {
	dir        string
	mu         sync.RWMutex
	summarizer openagent.Summarizer // nil = compaction is a no-op
	nextIdx    map[string]int64     // sessionID → next message Index (0 = unseeded)
}

// New creates a Memory store at dir. Directory is created if missing.
func New(dir string) (*Memory, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("file memory: %w", err)
	}
	return &Memory{dir: dir, nextIdx: make(map[string]int64)}, nil
}

// WithSummarizer enables compaction. nil (default) disables it. The Runner
// triggers compaction via Compact() when the working set exceeds the token budget.
func (m *Memory) WithSummarizer(s openagent.Summarizer) *Memory {
	m.summarizer = s
	return m
}

// ── openagent.Memory ──

// Close implements io.Closer. The file-based implementation opens and closes
// files per-operation, so it holds no persistent resources. Returns nil.
func (m *Memory) Close() error { return nil }

// DeleteSession removes the session's JSONL file and compressed file.
// It is safe to call on a session that doesn't exist (no error).
func (m *Memory) DeleteSession(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	ctxErr := ctx.Err() // capture before I/O — ctx state after removal is irrelevant
	_ = os.Remove(m.sessionPath(sessionID))
	_ = os.Remove(m.compressedPath(sessionID))
	delete(m.nextIdx, sessionID)
	return ctxErr
}

// Count returns the total number of messages for a session.
func (m *Memory) Count(ctx context.Context, sessionID string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	n, err := m.countLinesLocked(sessionID)
	return int(n), err
}

// Append writes a message to the session's JSONL file.
func (m *Memory) Append(ctx context.Context, sessionID string, msg openagent.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Seed the index counter on first append to this session (e.g. after
	// restart). Subsequent appends use the in-memory counter — no per-append
	// file scan.
	if m.nextIdx[sessionID] == 0 {
		n, err := m.countLinesLocked(sessionID)
		if err != nil {
			return fmt.Errorf("file memory append: %w", err)
		}
		m.nextIdx[sessionID] = n + 1
	}
	msg.Index = m.nextIdx[sessionID]
	m.nextIdx[sessionID]++

	f, err := os.OpenFile(m.sessionPath(sessionID), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("file memory append: %w", err)
	}
	defer f.Close()

	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("file memory append: %w", err)
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		return fmt.Errorf("file memory append: %w", err)
	}
	return nil
}

// Recent returns up to n messages for a session, skipping the offset most
// recent, oldest first. offset=0 returns the latest n.
func (m *Memory) Recent(ctx context.Context, sessionID string, n int, offset int) ([]openagent.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	all, err := m.readAllLocked(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if offset > 0 && len(all) > offset {
		all = all[:len(all)-offset]
	} else if offset > 0 {
		return nil, nil
	}
	if len(all) <= n {
		return all, nil
	}
	return all[len(all)-n:], nil
}

// Compact compresses messages up to throughIndex into a summary. The Runner
// calls this when the working set exceeds the token budget. Compression is
// incremental (rolling): new overflow messages are summarized together with
// the previous CompressedContext. Original messages are NEVER deleted.
//
// messages is an optional pre-fetched slice to avoid a redundant read.
// When nil, the backend fetches messages internally.
func (m *Memory) Compact(ctx context.Context, sessionID string, throughIndex int, messages []openagent.Message) error {
	if m.summarizer == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	all := messages
	var err error
	if all == nil || throughIndex > len(all) {
		all, err = m.readAllLocked(ctx, sessionID)
		if err != nil {
			return err
		}
	}

	if len(all) == 0 || throughIndex <= 0 || throughIndex > len(all) {
		return nil
	}

	// Adjust to safe boundary (don't cut tool_call/tool_result pairs).
	safeIdx := openagent.SafeCompressionBoundary(all, throughIndex)
	if safeIdx <= 0 {
		return nil
	}

	// Load previous compression marker for incremental compression.
	prev, _ := m.readCompressed(sessionID)
	lastIdx := 0
	if prev != nil {
		lastIdx = prev.ThroughIndex
	}

	// Only compress newly overflowed messages.
	if lastIdx < safeIdx {
		newMsgs := all[lastIdx:safeIdx]
		cc, err := m.summarizer.Summarize(ctx, newMsgs, prev)
		if err == nil && cc != nil {
			cc.ThroughIndex = safeIdx
			m.writeCompressed(sessionID, cc)
		}
	}

	return nil
}

// Compressed returns the stored CompressedContext, or nil if none exists.
func (m *Memory) Compressed(ctx context.Context, sessionID string) (*openagent.CompressedContext, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.readCompressed(sessionID)
}

// Search finds messages containing query as a case-insensitive substring.
// This is the simplest possible search — no tokenization hacks.
func (m *Memory) Search(ctx context.Context, sessionID, query string, limit int) ([]openagent.SearchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	all, err := m.readAllLocked(ctx, sessionID)
	m.mu.RUnlock()
	if err != nil {
		return nil, err
	}

	q := strings.ToLower(query)

	type scored struct {
		idx   int
		score float64
	}
	var matches []scored

	for i, msg := range all {
		pos := strings.Index(strings.ToLower(msg.Content), q)
		if pos < 0 {
			continue
		}
		// Earlier match → higher score, normalized to ~[0,1]
		score := 1.0 / (1.0 + float64(pos))
		matches = append(matches, scored{idx: i, score: score})
	}

	// Sort by score descending
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].score > matches[i].score {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	if limit > len(matches) {
		limit = len(matches)
	}

	results := make([]openagent.SearchResult, limit)
	for i := 0; i < limit; i++ {
		m := matches[i]
		results[i] = openagent.SearchResult{
			Message: all[m.idx],
			Score:   m.score,
		}
	}
	return results, nil
}

// ── Internal ──

func (m *Memory) sessionPath(sessionID string) string {
	safe := strings.ReplaceAll(sessionID, "/", "_")
	safe = strings.ReplaceAll(safe, string(os.PathSeparator), "_")
	return filepath.Join(m.dir, safe+".jsonl")
}

func (m *Memory) compressedPath(sessionID string) string {
	safe := strings.ReplaceAll(sessionID, "/", "_")
	safe = strings.ReplaceAll(safe, string(os.PathSeparator), "_")
	return filepath.Join(m.dir, safe+".compressed.json")
}

// readAllLocked reads all messages from the JSONL file. Caller must hold m.mu.
func (m *Memory) readAllLocked(ctx context.Context, sessionID string) ([]openagent.Message, error) {
	f, err := os.Open(m.sessionPath(sessionID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("file memory read: %w", err)
	}
	defer f.Close()

	var msgs []openagent.Message
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var fallback int64
	for scanner.Scan() {
		if len(msgs)%100 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		fallback++
		var msg openagent.Message
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}
		if msg.Index == 0 {
			msg.Index = fallback // old files without persisted Index
		}
		msgs = append(msgs, msg)
	}
	return msgs, scanner.Err()
}

// countLinesLocked returns the number of lines in the session's JSONL file.
// Caller must hold m.mu.
//
// We count by newline, not via bufio.Scanner: a Scanner without an explicit
// Buffer inherits the 64KB default token cap, so a single JSONL line larger
// than that (e.g. an assistant message embedding a large artifact) would
// trip bufio.ErrTooLong. Since Append always writes a trailing '\n', a line is
// one record regardless of size; ReadString chunks oversized lines safely and
// only counts a complete line on the '\n' terminator. A trailing partial line
// with no '\n' (a crashed/corrupt append) is not counted.
func (m *Memory) countLinesLocked(sessionID string) (int64, error) {
	f, err := os.Open(m.sessionPath(sessionID))
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	defer f.Close()

	var count int64
	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 && line[len(line)-1] == '\n' {
			count++
		}
		if err != nil {
			if err == io.EOF {
				return count, nil
			}
			return count, err
		}
	}
}

func (m *Memory) writeCompressed(sessionID string, cc *openagent.CompressedContext) error {
	b, err := json.Marshal(cc)
	if err != nil {
		return err
	}
	return os.WriteFile(m.compressedPath(sessionID), b, 0644)
}

func (m *Memory) readCompressed(sessionID string) (*openagent.CompressedContext, error) {
	b, err := os.ReadFile(m.compressedPath(sessionID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("file memory compressed read: %w", err)
	}
	var cc openagent.CompressedContext
	if err := json.Unmarshal(b, &cc); err != nil {
		return nil, nil
	}
	return &cc, nil
}
