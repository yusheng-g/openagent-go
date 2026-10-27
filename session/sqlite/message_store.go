package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	_ "modernc.org/sqlite" // driver registration — this package opens "sqlite" DSNs itself

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/session"
)

// MessageStore implements session.SessionStore and session.Compressor over
// SQLite: the conversation (messages table + FTS5 index) and its compressed
// summaries. Durable knowledge (MemoryProvider) lives in
// provider/memory/sqlite over the same database file — this package is the
// session domain only.
type MessageStore struct {
	db         *sql.DB
	summarizer openagent.Summarizer
}

// NewMessageStore opens a SQLite database at path and runs migrations.
// Enables WAL mode, foreign keys, and a 5s busy timeout for concurrent
// safety. DB() shares the connection with the metadata Store and the
// knowledge provider.
func NewMessageStore(path string) (*MessageStore, error) {
	dsn := path + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite message store: open: %w", err)
	}
	m := &MessageStore{db: db}
	if err := m.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return m, nil
}

// DB returns the underlying *sql.DB so callers can share the connection
// (session metadata Store, knowledge provider).
func (m *MessageStore) DB() *sql.DB { return m.db }

// WithSummarizer enables compaction. nil (default) disables it. The runtime
// triggers compaction via Compact when the working set exceeds the token
// budget.
func (m *MessageStore) WithSummarizer(s openagent.Summarizer) *MessageStore {
	m.summarizer = s
	return m
}

// Close releases the database connection.
func (m *MessageStore) Close() error { return m.db.Close() }

// ── session.SessionStore ──

var _ session.SessionStore = (*MessageStore)(nil)

// DeleteSession removes all conversation data for the given session
// (messages, FTS5 entries, compressed summaries). FTS5 entries are removed
// first since they lack foreign key constraints.
func (m *MessageStore) DeleteSession(ctx context.Context, sessionID string) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite delete session: %w", err)
	}
	defer tx.Rollback()

	// Delete FTS5 entries first (no foreign key).
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM messages_fts WHERE rowid IN
		 (SELECT id FROM messages WHERE session_id = ?)`,
		sessionID,
	); err != nil {
		return fmt.Errorf("sqlite delete session fts: %w", err)
	}

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM compressed WHERE session_id = ?`, sessionID,
	); err != nil {
		return fmt.Errorf("sqlite delete session compressed: %w", err)
	}

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM messages WHERE session_id = ?`, sessionID,
	); err != nil {
		return fmt.Errorf("sqlite delete session messages: %w", err)
	}

	return tx.Commit()
}

// Count returns the total number of messages for a session.
func (m *MessageStore) Count(ctx context.Context, sessionID string) (int, error) {
	var count int
	err := m.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM messages WHERE session_id = ?`, sessionID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("sqlite count: %w", err)
	}
	return count, nil
}

// Append adds a message to the conversation and indexes it in FTS5.
func (m *MessageStore) Append(ctx context.Context, sessionID string, msg openagent.Message) error {
	toolCallsJSON, _ := json.Marshal(msg.ToolCalls)
	if toolCallsJSON == nil {
		toolCallsJSON = []byte("[]")
	}

	contentPartsJSON, _ := json.Marshal(msg.ContentParts)
	if contentPartsJSON == nil {
		contentPartsJSON = []byte("[]")
	}

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite append: %w", err)
	}
	defer tx.Rollback()

	// created_at: store RFC3339 UTC to match the sessions table format.
	// A nil CreatedAt (legacy message never stamped, or test fixture)
	// serializes as the empty string — scanMessages parses '' back to nil,
	// which omitempty then keeps off the wire.
	var createdAt string
	if msg.CreatedAt != nil {
		createdAt = msg.CreatedAt.UTC().Format(time.RFC3339)
	}
	res, err := tx.ExecContext(ctx,
		`INSERT INTO messages (session_id, role, name, content, content_parts, tool_calls, tool_call_id, reasoning_content, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sessionID, msg.Role, msg.Name, msg.Content, string(contentPartsJSON), string(toolCallsJSON), msg.ToolCallID, msg.ReasoningContent, createdAt,
	)
	if err != nil {
		return fmt.Errorf("sqlite append: %w", err)
	}

	id, _ := res.LastInsertId()

	// FTS5 index.
	if msg.Content != "" {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO messages_fts (rowid, content) VALUES (?, ?)`, id, msg.Content,
		); err != nil {
			return fmt.Errorf("sqlite fts: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sqlite append commit: %w", err)
	}
	return nil
}

// Recent returns up to n most recent messages, skipping the offset most
// recent, oldest first. offset=0 returns the latest n.
func (m *MessageStore) Recent(ctx context.Context, sessionID string, n int, offset int) ([]openagent.Message, error) {
	// Fetch most recent messages in reverse-chronological order,
	// then reverse to chronological. Fetch 2×n so we can trim
	// incomplete tool_call/tool_result pairs at boundaries.
	fetchN := n*2 + offset
	if fetchN < 20 {
		fetchN = 20
	}
	rows, err := m.db.QueryContext(ctx,
		`SELECT id, role, name, content, content_parts, tool_calls, tool_call_id, reasoning_content, created_at
		 FROM messages WHERE session_id = ?
		 ORDER BY id DESC LIMIT ?`,
		sessionID, fetchN,
	)
	if err != nil {
		return nil, fmt.Errorf("sqlite recent: %w", err)
	}
	defer rows.Close()

	msgs, err := scanMessages(rows)
	if err != nil {
		return nil, err
	}

	// Reverse to chronological order (oldest first).
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}

	// Trim leading tool messages. A tool result without its preceding
	// assistant message (which carried the tool_call) is orphaned and
	// provides no useful context to the model.
	for len(msgs) > 0 && msgs[0].Role == openagent.RoleTool {
		msgs = msgs[1:]
	}

	// Skip 'offset' most recent messages, then return up to n.
	if offset > 0 && len(msgs) > offset {
		msgs = msgs[:len(msgs)-offset]
	} else if offset > 0 {
		msgs = nil
	}
	if n > 0 && len(msgs) > n {
		msgs = msgs[len(msgs)-n:]
	}

	return msgs, nil
}

// RecentAfter returns up to n messages after the throughIndex-th message
// (0 = from the start), oldest first. OFFSET is linear in SQLite but the
// win is skipping the JSON deserialization of already-summarized history.
func (m *MessageStore) RecentAfter(ctx context.Context, sessionID string, throughIndex, n int) ([]openagent.Message, error) {
	if throughIndex < 0 {
		throughIndex = 0
	}
	if n <= 0 {
		return nil, nil
	}
	rows, err := m.db.QueryContext(ctx,
		`SELECT id, role, name, content, content_parts, tool_calls, tool_call_id, reasoning_content, created_at
		 FROM messages WHERE session_id = ?
		 ORDER BY id ASC LIMIT ? OFFSET ?`,
		sessionID, n, throughIndex,
	)
	if err != nil {
		return nil, fmt.Errorf("sqlite recent_after: %w", err)
	}
	defer rows.Close()
	return scanMessages(rows)
}

// ── session.Compressor ──

var _ session.Compressor = (*MessageStore)(nil)

// CopyCompressed implements session.CompressedCopier: duplicate the
// source's compressed summary under the fork's ID, replacing anything the
// destination already has (delete-then-insert honors the interface
// contract even though forks start fresh today). No rows are written
// when the source has no summary (INSERT … SELECT matches nothing).
func (m *MessageStore) CopyCompressed(ctx context.Context, srcID, dstID string) error {
	_, err := m.db.ExecContext(ctx,
		`DELETE FROM compressed WHERE session_id = ?`,
		dstID,
	)
	if err != nil {
		return fmt.Errorf("sqlite copy compressed: %w", err)
	}
	_, err = m.db.ExecContext(ctx,
		`INSERT INTO compressed (session_id, data)
		 SELECT ?, data FROM compressed WHERE session_id = ?`,
		dstID, srcID,
	)
	if err != nil {
		return fmt.Errorf("sqlite copy compressed: %w", err)
	}
	return nil
}

// Compact compresses messages up to throughIndex into a summary. The
// runtime calls this when the working set exceeds the token budget.
// Compression is incremental (rolling): new overflow messages are
// summarized together with the previous CompressedContext. Original
// messages are NEVER deleted.
func (m *MessageStore) Compact(ctx context.Context, sessionID string, throughIndex int, messages []openagent.Message) error {
	if m.summarizer == nil {
		return nil
	}

	// Load previous compression marker.
	prev, err := m.Compressed(ctx, sessionID)
	if err != nil {
		// Corrupt/unreadable history: fall back to compressing from the
		// start rather than silently skipping the marker.
		slog.Warn("previous compression marker unreadable", "session", sessionID, "error", err)
	}
	lastIdx := 0
	if prev != nil {
		lastIdx = prev.ThroughIndex
	}

	if lastIdx >= throughIndex {
		return nil // nothing new to compress
	}

	// Use pre-fetched messages if available, otherwise query.
	var all []openagent.Message
	if messages != nil && throughIndex <= len(messages) {
		all = messages
	} else {
		var count int
		if err := m.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM messages WHERE session_id = ?`, sessionID,
		).Scan(&count); err != nil {
			return fmt.Errorf("sqlite compact: %w", err)
		}
		if count == 0 || throughIndex <= 0 || throughIndex > count {
			return nil
		}
		fetchCount := throughIndex + 20
		if fetchCount > count {
			fetchCount = count
		}
		rows, err := m.db.QueryContext(ctx,
			`SELECT id, role, name, content, content_parts, tool_calls, tool_call_id, reasoning_content, created_at
			 FROM messages WHERE session_id = ?
			 ORDER BY id ASC LIMIT ?`,
			sessionID, fetchCount,
		)
		if err != nil {
			return fmt.Errorf("sqlite compact: %w", err)
		}
		all, err = scanMessages(rows)
		rows.Close()
		if err != nil {
			return fmt.Errorf("sqlite compact: scan messages: %w", err)
		}
	}

	if len(all) == 0 || throughIndex > len(all) {
		return nil
	}

	// Adjust to safe boundary (don't cut tool_call/tool_result pairs).
	safeIdx := openagent.SafeCompressionBoundary(all, throughIndex)
	if safeIdx <= 0 || safeIdx > len(all) {
		return nil
	}

	// Only compress newly overflowed messages.
	if lastIdx < safeIdx {
		newMsgs := all[lastIdx:safeIdx]
		cc, sumErr := m.summarizer.Summarize(ctx, newMsgs, prev)
		if sumErr != nil {
			return sumErr
		}
		if cc != nil {
			cc.ThroughIndex = safeIdx
			if err := m.storeCompressed(ctx, sessionID, cc); err != nil {
				return fmt.Errorf("sqlite compact: store compressed: %w", err)
			}
		}
	}

	return nil
}

// Compressed returns the stored CompressedContext, or nil if none exists.
func (m *MessageStore) Compressed(ctx context.Context, sessionID string) (*openagent.CompressedContext, error) {
	var summaryJSON []byte
	err := m.db.QueryRowContext(ctx,
		`SELECT data FROM compressed WHERE session_id = ? ORDER BY id DESC LIMIT 1`,
		sessionID,
	).Scan(&summaryJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("sqlite compressed: %w", err)
	}
	var cc openagent.CompressedContext
	if err := json.Unmarshal(summaryJSON, &cc); err != nil {
		return nil, fmt.Errorf("sqlite compressed: %w", err)
	}
	return &cc, nil
}

func (m *MessageStore) storeCompressed(ctx context.Context, sessionID string, cc *openagent.CompressedContext) error {
	b, _ := json.Marshal(cc)

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Replace the previous compressed entry for this session — the new
	// summary subsumes the old one. Without this, compressed rows accumulate
	// indefinitely.
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM compressed WHERE session_id = ?`, sessionID,
	); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO compressed (session_id, data) VALUES (?, ?)`,
		sessionID, string(b),
	); err != nil {
		return err
	}

	return tx.Commit()
}

// ── migration ──

func (m *MessageStore) migrate() error {
	_, err := m.db.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id       TEXT    NOT NULL,
			role             TEXT    NOT NULL,
			name             TEXT    NOT NULL DEFAULT '', -- dead column (legacy OpenAI name); kept to avoid a schema migration
			content          TEXT    NOT NULL DEFAULT '',
			content_parts    TEXT    NOT NULL DEFAULT '',
			tool_calls       TEXT    NOT NULL DEFAULT '[]',
			tool_call_id     TEXT    NOT NULL DEFAULT '',
			reasoning_content TEXT   NOT NULL DEFAULT '',
			created_at        TEXT    NOT NULL DEFAULT ''  -- RFC3339 UTC; '' = legacy row pre-column
		);
		CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id, id);

		CREATE TABLE IF NOT EXISTS compressed (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			data       TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_compressed_session ON compressed(session_id, id);
	`)
	if err != nil {
		return fmt.Errorf("sqlite message store migrate: %w", err)
	}

	// created_at is a later addition. CREATE TABLE IF NOT EXISTS is a no-op
	// on a pre-existing table, so databases created before this column need
	// an explicit ALTER TABLE ADD COLUMN. SQLite fills existing rows with
	// the DEFAULT ('' → zero time upstream → omitted on the wire). Probe
	// via pragma_table_info so fresh databases (column already in CREATE
	// TABLE) and already-migrated databases both skip the ALTER.
	var hasCreatedAt int
	if err := m.db.QueryRow(
		`SELECT COUNT(*) FROM pragma_table_info('messages') WHERE name='created_at'`,
	).Scan(&hasCreatedAt); err != nil {
		return fmt.Errorf("sqlite message store migrate probe created_at: %w", err)
	}
	if hasCreatedAt == 0 {
		if _, err := m.db.Exec(
			`ALTER TABLE messages ADD COLUMN created_at TEXT NOT NULL DEFAULT ''`,
		); err != nil {
			return fmt.Errorf("sqlite message store migrate add created_at: %w", err)
		}
	}

	// FTS5 index with the trigram tokenizer so search matches arbitrary
	// substrings — including CJK — instead of only whole tokens. The default
	// unicode61 tokenizer treats a run of CJK characters as one token, so CJK
	// queries match nothing. Legacy unicode61 tables are rebuilt in place;
	// the messages table is the source of truth, so re-indexing is safe.
	var createSQL string
	switch err := m.db.QueryRow(
		`SELECT sql FROM sqlite_master WHERE type='table' AND name='messages_fts'`,
	).Scan(&createSQL); {
	case err == sql.ErrNoRows:
		// Table absent — created below.
	case err != nil:
		return fmt.Errorf("sqlite message store migrate fts: %w", err)
	case !strings.Contains(createSQL, "trigram"):
		if _, err := m.db.Exec(`DROP TABLE messages_fts`); err != nil {
			return fmt.Errorf("sqlite message store migrate fts drop: %w", err)
		}
	}

	if _, err := m.db.Exec(
		`CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(content, tokenize='trigram')`,
	); err != nil {
		return fmt.Errorf("sqlite message store migrate fts create: %w", err)
	}

	// Backfill any messages not yet indexed (fresh/rebuilt table, or rows
	// from a pre-FTS schema). Idempotent via the NOT IN guard.
	if _, err := m.db.Exec(
		`INSERT INTO messages_fts (rowid, content)
		 SELECT id, content FROM messages
		 WHERE content != '' AND id NOT IN (SELECT rowid FROM messages_fts)`,
	); err != nil {
		return fmt.Errorf("sqlite message store migrate fts backfill: %w", err)
	}
	return nil
}

// ── helpers ──

func scanMessages(rows *sql.Rows) ([]openagent.Message, error) {
	var msgs []openagent.Message
	for rows.Next() {
		var id int64
		var role, name, content, contentParts, toolCalls, toolCallID, reasoningContent, createdAt string
		if err := rows.Scan(&id, &role, &name, &content, &contentParts, &toolCalls, &toolCallID, &reasoningContent, &createdAt); err != nil {
			return nil, err
		}
		msg := rowToMessage(role, name, content, contentParts, toolCalls, toolCallID, reasoningContent)
		msg.Index = id
		// created_at is '' for legacy rows (pre-column or never stamped);
		// leave nil so omitempty drops it on the wire.
		if createdAt != "" {
			if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
				msg.CreatedAt = &t
			}
		}
		msgs = append(msgs, msg)
	}
	return msgs, rows.Err()
}

func rowToMessage(role, name, content, contentParts, toolCalls, toolCallID, reasoningContent string) openagent.Message {
	msg := openagent.Message{
		Role:             openagent.Role(role),
		Name:             name,
		Content:          content,
		ToolCallID:       toolCallID,
		ReasoningContent: reasoningContent,
	}
	if contentParts != "" {
		var parts []openagent.ContentPart
		if json.Unmarshal([]byte(contentParts), &parts) == nil {
			msg.ContentParts = parts
		}
	}
	if toolCalls != "" && toolCalls != "[]" {
		var calls []openagent.ToolCall
		if json.Unmarshal([]byte(toolCalls), &calls) == nil {
			msg.ToolCalls = calls
		}
	}
	return msg
}
