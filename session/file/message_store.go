package file

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/session"
)

// MessageStore implements session.SessionStore and session.Compressor with
// JSONL files on disk (one JSON object per line per session), zero external
// dependencies. Durable knowledge (MemoryProvider) lives in
// provider/memory/file in the same directory.
type MessageStore struct {
	dir        string
	mu         sync.RWMutex
	summarizer openagent.Summarizer // nil = compaction is a no-op
	nextIdx    map[string]int64     // sessionID → next message Index (0 = unseeded)
}

// NewMessageStore creates a store at dir. Directory is created if missing.
func NewMessageStore(dir string) (*MessageStore, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("file message store: %w", err)
	}
	return &MessageStore{dir: dir, nextIdx: make(map[string]int64)}, nil
}

// WithSummarizer enables compaction. nil (default) disables it. The runtime
// triggers compaction via Compact when the working set exceeds the token
// budget.
func (m *MessageStore) WithSummarizer(s openagent.Summarizer) *MessageStore {
	m.summarizer = s
	return m
}

// ── session.SessionStore ──

var _ session.SessionStore = (*MessageStore)(nil)

// Close implements io.Closer. The file-based implementation opens and closes
// files per-operation, so it holds no persistent resources. Returns nil.
func (m *MessageStore) Close() error { return nil }

// DeleteSession removes the session's JSONL file and compressed file.
// It is safe to call on a session that doesn't exist (no error).
func (m *MessageStore) DeleteSession(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	ctxErr := ctx.Err() // capture before I/O — ctx state after removal is irrelevant
	_ = os.Remove(m.sessionPath(sessionID))
	_ = os.Remove(m.compressedPath(sessionID))
	delete(m.nextIdx, sessionID)
	return ctxErr
}

// Count returns the total number of messages for a session.
func (m *MessageStore) Count(ctx context.Context, sessionID string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	n, err := m.countLinesLocked(sessionID)
	return int(n), err
}

// Append writes a message to the session's JSONL file.
func (m *MessageStore) Append(ctx context.Context, sessionID string, msg openagent.Message) error {
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
			return fmt.Errorf("file message store append: %w", err)
		}
		m.nextIdx[sessionID] = n + 1
	}
	msg.Index = m.nextIdx[sessionID]
	m.nextIdx[sessionID]++

	f, err := os.OpenFile(m.sessionPath(sessionID), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("file message store append: %w", err)
	}
	defer f.Close()

	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("file message store append: %w", err)
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		return fmt.Errorf("file message store append: %w", err)
	}
	return nil
}

// Recent returns up to n messages for a session, skipping the offset most
// recent, oldest first. offset=0 returns the latest n.
func (m *MessageStore) Recent(ctx context.Context, sessionID string, n int, offset int) ([]openagent.Message, error) {
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

// RecentAfter returns up to n messages after the throughIndex-th message
// (0 = from the start), oldest first.
func (m *MessageStore) RecentAfter(ctx context.Context, sessionID string, throughIndex, n int) ([]openagent.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if throughIndex < 0 {
		throughIndex = 0
	}
	if n <= 0 {
		return nil, nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	all, err := m.readAllLocked(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if throughIndex >= len(all) {
		return nil, nil
	}
	end := throughIndex + n
	if end > len(all) {
		end = len(all)
	}
	return all[throughIndex:end], nil
}

// ── session.Compressor ──

var _ session.Compressor = (*MessageStore)(nil)

// CopyCompressed implements session.CompressedCopier: duplicate the
// source's compressed-summary file under the fork's ID. A no-op when
// the source has none (readCompressed returns nil, nil).
func (m *MessageStore) CopyCompressed(ctx context.Context, srcID, dstID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cc, err := m.readCompressed(srcID)
	if err != nil || cc == nil {
		return err
	}
	return m.writeCompressed(dstID, cc)
}

// Compact compresses messages up to throughIndex into a summary. The
// runtime calls this when the working set exceeds the token budget.
// Compression is incremental (rolling): new overflow messages are
// summarized together with the previous CompressedContext. Original
// messages are NEVER deleted.
//
// messages is an optional pre-fetched slice to avoid a redundant read.
// When nil, the backend fetches messages internally.
func (m *MessageStore) Compact(ctx context.Context, sessionID string, throughIndex int, messages []openagent.Message) error {
	if m.summarizer == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	// Phase 1 (locked): read messages + previous compression state.
	// The LLM Summarize call happens outside the lock (Phase 2) so it
	// doesn't block all sessions for the duration of the model call.
	m.mu.Lock()
	all := messages
	var err error
	if all == nil || throughIndex > len(all) {
		all, err = m.readAllLocked(ctx, sessionID)
		if err != nil {
			m.mu.Unlock()
			return err
		}
	}

	if len(all) == 0 || throughIndex <= 0 || throughIndex > len(all) {
		m.mu.Unlock()
		return nil
	}

	// Adjust to safe boundary (don't cut tool_call/tool_result pairs).
	safeIdx := openagent.SafeCompressionBoundary(all, throughIndex)
	if safeIdx <= 0 {
		m.mu.Unlock()
		return nil
	}

	// Load previous compression marker for incremental compression.
	prev, _ := m.readCompressed(sessionID)
	lastIdx := 0
	if prev != nil {
		lastIdx = prev.ThroughIndex
	}
	m.mu.Unlock()

	// Phase 2 (unlocked): only compress newly overflowed messages.
	if lastIdx >= safeIdx {
		return nil
	}
	newMsgs := all[lastIdx:safeIdx]
	cc, err := m.summarizer.Summarize(ctx, newMsgs, prev)
	if err != nil {
		return fmt.Errorf("file compact: summarize: %w", err)
	}
	if cc == nil {
		return nil
	}
	cc.ThroughIndex = safeIdx

	// Phase 3 (locked): write the compressed result.
	m.mu.Lock()
	if err := m.writeCompressed(sessionID, cc); err != nil {
		m.mu.Unlock()
		return fmt.Errorf("file compact: write: %w", err)
	}
	m.mu.Unlock()
	return nil
}

// Compressed returns the stored CompressedContext, or nil if none exists.
func (m *MessageStore) Compressed(ctx context.Context, sessionID string) (*openagent.CompressedContext, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.readCompressed(sessionID)
}

// ── Internal ──

func (m *MessageStore) sessionPath(sessionID string) string {
	safe := strings.ReplaceAll(sessionID, "/", "_")
	safe = strings.ReplaceAll(safe, string(os.PathSeparator), "_")
	return filepath.Join(m.dir, safe+".jsonl")
}

func (m *MessageStore) compressedPath(sessionID string) string {
	safe := strings.ReplaceAll(sessionID, "/", "_")
	safe = strings.ReplaceAll(safe, string(os.PathSeparator), "_")
	return filepath.Join(m.dir, safe+".compressed.json")
}

// readAllLocked reads all messages from the JSONL file. Caller must hold m.mu.
func (m *MessageStore) readAllLocked(ctx context.Context, sessionID string) ([]openagent.Message, error) {
	f, err := os.Open(m.sessionPath(sessionID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("file message store read: %w", err)
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
			// A corrupt line is data loss — log it instead of silently
			// dropping it.
			slog.Warn("skipping corrupt message line", "session", sessionID, "error", err)
			continue
		}
		if msg.Index == 0 {
			// Pre-index data (old files): assign sequential indices so
			// compaction offsets stay consistent.
			msg.Index = fallback
		}
		msgs = append(msgs, msg)
	}
	return msgs, scanner.Err()
}

// countLinesLocked returns the number of complete lines in the session
// file. A trailing partial line with no '\n' (a crashed/corrupt append) is
// not counted. Caller must hold m.mu.
func (m *MessageStore) countLinesLocked(sessionID string) (int64, error) {
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

func (m *MessageStore) writeCompressed(sessionID string, cc *openagent.CompressedContext) error {
	b, err := json.Marshal(cc)
	if err != nil {
		return err
	}
	// Atomic write (tmp + rename, same as the sessions.json flush): a
	// crash mid-write must not leave a truncated compressed file that
	// silently reads back as "no history".
	path := m.compressedPath(sessionID)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (m *MessageStore) readCompressed(sessionID string) (*openagent.CompressedContext, error) {
	b, err := os.ReadFile(m.compressedPath(sessionID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("file message store compressed read: %w", err)
	}
	var cc openagent.CompressedContext
	if err := json.Unmarshal(b, &cc); err != nil {
		// Corrupt history must not be silently reset to nothing.
		slog.Warn("corrupted compressed summary, ignoring", "session", sessionID, "error", err)
		return nil, nil
	}
	return &cc, nil
}
