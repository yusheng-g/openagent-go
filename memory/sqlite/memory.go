// Package sqlite implements openagent.Memory with SQLite.
//
// Features:
//   - FTS5 full-text search (always enabled)
//   - Vector semantic search when configured via WithEmbedder
//   - Automatic schema migration on open
//
// Usage:
//
//	mem, err := sqlite.New("/path/to/memory.db")
//	mem.WithEmbedder(openaiEmbedder) // optional
//	agent := openagent.NewAgent("bot", openagent.WithMemory(mem))
package sqlite

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode"

	_ "modernc.org/sqlite"

	openagent "github.com/yusheng-g/openagent-go"
)

// Memory implements openagent.Memory backed by SQLite.
type Memory struct {
	db            *sql.DB
	embedder      openagent.Embedder
	summarizer    openagent.Summarizer
	maxVectorScan int // max rows to scan for vector similarity, default 2000
}

// New opens a SQLite database at path and runs migrations.
// Enables WAL mode, foreign keys, and a 5s busy timeout for concurrent safety.
func New(path string) (*Memory, error) {
	dsn := path + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open: %w", err)
	}
	m := &Memory{db: db, maxVectorScan: 2000}
	if err := m.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return m, nil
}

// DB returns the underlying *sql.DB so callers can share the connection
// (e.g., for co-located session metadata storage).
func (m *Memory) DB() *sql.DB { return m.db }

// WithEmbedder enables semantic (vector) search.
func (m *Memory) WithEmbedder(e openagent.Embedder) *Memory {
	m.embedder = e
	return m
}

// WithSummarizer enables compaction. nil (default) disables it. The Runner
// triggers compaction via Compact() when the working set exceeds the token budget.
func (m *Memory) WithSummarizer(s openagent.Summarizer) *Memory {
	m.summarizer = s
	return m
}

// WithMaxVectorScan sets the max rows to scan for vector similarity search. Default 2000.
// Higher values improve recall at the cost of latency and memory. Set to 0 to remove the limit
// (load all vectors — not recommended for large sessions).
func (m *Memory) WithMaxVectorScan(n int) *Memory {
	m.maxVectorScan = n
	return m
}

// Close releases the database connection.
func (m *Memory) Close() error { return m.db.Close() }

// DeleteSession removes all data for the given session from messages,
// compressed, and vectors tables. FTS5 entries are removed first since
// they lack foreign key constraints.
func (m *Memory) DeleteSession(ctx context.Context, sessionID string) error {
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

	// Vectors and compressed have foreign keys but delete explicitly for clarity.
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM vectors WHERE message_id IN
		 (SELECT id FROM messages WHERE session_id = ?)`,
		sessionID,
	); err != nil {
		return fmt.Errorf("sqlite delete session vectors: %w", err)
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

// ── openagent.Memory ──

// Count returns the total number of messages for a session.
func (m *Memory) Count(ctx context.Context, sessionID string) (int, error) {
	var count int
	err := m.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM messages WHERE session_id = ?`, sessionID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("sqlite count: %w", err)
	}
	return count, nil
}

func (m *Memory) Append(ctx context.Context, sessionID string, msg openagent.Message) error {
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

	res, err := tx.ExecContext(ctx,
		`INSERT INTO messages (session_id, role, name, content, content_parts, tool_calls, tool_call_id, reasoning_content)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sessionID, msg.Role, msg.Name, msg.Content, string(contentPartsJSON), string(toolCallsJSON), msg.ToolCallID, msg.ReasoningContent,
	)
	if err != nil {
		return fmt.Errorf("sqlite append: %w", err)
	}

	id, _ := res.LastInsertId()

	// FTS5 index
	if msg.Content != "" {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO messages_fts (rowid, content) VALUES (?, ?)`, id, msg.Content,
		); err != nil {
			return fmt.Errorf("sqlite fts: %w", err)
		}
	}

	// Vector index (best-effort)
	if m.embedder != nil && msg.Content != "" {
		vec, err := m.embedder.Embed(ctx, msg.Content)
		if err == nil {
			buf := floatsToBytes(vec)
			_, _ = tx.ExecContext(ctx,
				`INSERT OR REPLACE INTO vectors (message_id, embedding) VALUES (?, ?)`, id, buf,
			)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sqlite append commit: %w", err)
	}
	return nil
}

func (m *Memory) Recent(ctx context.Context, sessionID string, n int, offset int) ([]openagent.Message, error) {
	// Fetch most recent messages in reverse-chronological order,
	// then reverse to chronological. Fetch 2×n so we can trim
	// incomplete tool_call/tool_result pairs at boundaries.
	fetchN := n*2 + offset
	if fetchN < 20 {
		fetchN = 20
	}
	rows, err := m.db.QueryContext(ctx,
		`SELECT id, role, name, content, content_parts, tool_calls, tool_call_id, reasoning_content
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

// Compact compresses messages up to throughIndex into a summary. The Runner
// calls this when the working set exceeds the token budget. Compression is
// incremental (rolling): new overflow messages are summarized together with
// the previous CompressedContext. Original messages are NEVER deleted.
func (m *Memory) Compact(ctx context.Context, sessionID string, throughIndex int, messages []openagent.Message) error {
	if m.summarizer == nil {
		return nil
	}

	// Load previous compression marker.
	prev, _ := m.Compressed(ctx, sessionID)
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
			`SELECT id, role, name, content, content_parts, tool_calls, tool_call_id, reasoning_content
			 FROM messages WHERE session_id = ?
			 ORDER BY id ASC LIMIT ?`,
			sessionID, fetchCount,
		)
		if err != nil {
			return fmt.Errorf("sqlite compact: %w", err)
		}
		all, _ = scanMessages(rows)
		rows.Close()
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
			m.storeCompressed(ctx, sessionID, cc)
		}
	}

	return nil
}

func (m *Memory) Compressed(ctx context.Context, sessionID string) (*openagent.CompressedContext, error) {
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

func (m *Memory) storeCompressed(ctx context.Context, sessionID string, cc *openagent.CompressedContext) error {
	b, _ := json.Marshal(cc)

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Replace the previous compressed entry for this session — the new
	// summary subsumes the old one. Without this, compressed rows accumulate
	// indefinitely (BUGS.md #38l).
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

func (m *Memory) Search(ctx context.Context, sessionID, query string, limit int) ([]openagent.SearchResult, error) {
	if m.embedder != nil {
		if results, err := m.vectorSearch(ctx, sessionID, query, limit); err == nil {
			return results, nil
		}
	}
	return m.ftsSearch(ctx, sessionID, query, limit)
}

// ── Search backends ──

func (m *Memory) vectorSearch(ctx context.Context, sessionID, query string, limit int) ([]openagent.SearchResult, error) {
	qVec, err := m.embedder.Embed(ctx, query)
	if err != nil {
		return nil, err
	}

	rows, err := m.db.QueryContext(ctx,
		`SELECT v.embedding, m.id, m.role, m.name, m.content, m.content_parts, m.tool_calls, m.tool_call_id, reasoning_content
		 FROM vectors v
		 JOIN messages m ON v.message_id = m.id
		 WHERE m.session_id = ?
		 ORDER BY m.id DESC
		 LIMIT ?`,
		sessionID, m.maxVectorScan,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type scored struct {
		msg   openagent.Message
		score float64
	}
	var candidates []scored

	for rows.Next() {
		var raw []byte
		var id int64
		var role, name, content, contentParts, toolCalls, toolCallID, reasoningContent string
		if err := rows.Scan(&raw, &id, &role, &name, &content, &contentParts, &toolCalls, &toolCallID, &reasoningContent); err != nil {
			continue
		}
		vec := bytesToFloats(raw)
		score := cosineSimilarity(qVec, vec)
		msg := rowToMessage(role, name, content, contentParts, toolCalls, toolCallID, reasoningContent)
		msg.Index = id
		candidates = append(candidates, scored{msg: msg, score: score})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Sort descending by score
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	if limit > len(candidates) {
		limit = len(candidates)
	}
	results := make([]openagent.SearchResult, limit)
	for i := 0; i < limit; i++ {
		results[i] = openagent.SearchResult{
			Message: candidates[i].msg,
			Score:   candidates[i].score,
		}
	}
	return results, nil
}

func (m *Memory) ftsSearch(ctx context.Context, sessionID, query string, limit int) ([]openagent.SearchResult, error) {
	if limit <= 0 || strings.TrimSpace(query) == "" {
		return nil, nil
	}

	// Build a keyword query: each whitespace-separated token has
	// leading/trailing punctuation trimmed (so "colour?" matches "colour"),
	// is quoted as an FTS5 phrase, and is OR-joined. Results are ranked by
	// BM25 (ORDER BY rank), so messages sharing more — and rarer — tokens
	// surface first. OR (not implicit AND) is used because the query is
	// often natural language (the recall_memory tool passes the model's
	// query); AND would require every token present and return nothing when
	// the query has words not in any stored message.
	//
	// The trigram tokenizer (see migrate) only matches tokens of ≥3
	// characters, so shorter tokens are dropped. When no usable token
	// remains, fall back to a LIKE substring scan.
	if q := buildFTSQuery(query); q != "" {
		rows, err := m.db.QueryContext(ctx,
			`SELECT m.id, m.role, m.name, m.content, m.content_parts, m.tool_calls, m.tool_call_id, reasoning_content
			 FROM messages_fts f
			 JOIN messages m ON f.rowid = m.id
			 WHERE m.session_id = ? AND messages_fts MATCH ?
			 ORDER BY rank
			 LIMIT ?`,
			sessionID, q, limit,
		)
		if err == nil {
			msgs, scanErr := scanMessages(rows)
			rows.Close()
			if scanErr != nil {
				return nil, scanErr
			}
			return toSearchResults(msgs), nil
		}
		// FTS5 errored (unexpected query shape) — fall back to LIKE.
	}
	return m.likeSearch(ctx, sessionID, query, limit)
}

// buildFTSQuery turns a free-text query into a safe FTS5 expression. Each
// whitespace-separated token has leading/trailing punctuation trimmed, is
// quoted as a phrase, and is OR-joined. Tokens shorter than 3 characters are
// dropped (the trigram tokenizer cannot match them). Returns "" when no usable
// token remains.
func buildFTSQuery(query string) string {
	var parts []string
	for _, tok := range strings.Fields(query) {
		tok = strings.TrimFunc(tok, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		})
		if len([]rune(tok)) < 3 {
			continue
		}
		// Escape embedded double-quotes per FTS5 phrase rules.
		parts = append(parts, `"`+strings.ReplaceAll(tok, `"`, `""`)+`"`)
	}
	return strings.Join(parts, " OR ")
}

// likeSearch is the substring fallback used when the FTS query is empty (all
// tokens too short) or errors out. It does a case-insensitive LIKE scan.
func (m *Memory) likeSearch(ctx context.Context, sessionID, query string, limit int) ([]openagent.SearchResult, error) {
	rows, err := m.db.QueryContext(ctx,
		`SELECT id, role, name, content, content_parts, tool_calls, tool_call_id, reasoning_content
		 FROM messages
		 WHERE session_id = ? AND content LIKE ? ESCAPE '\'
		 ORDER BY id DESC
		 LIMIT ?`,
		sessionID, "%"+likeEscape(query)+"%", limit,
	)
	if err != nil {
		return nil, fmt.Errorf("sqlite fts: %w", err)
	}
	defer rows.Close()
	msgs, err := scanMessages(rows)
	if err != nil {
		return nil, err
	}
	return toSearchResults(msgs), nil
}

func likeEscape(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

func toSearchResults(msgs []openagent.Message) []openagent.SearchResult {
	results := make([]openagent.SearchResult, len(msgs))
	for i, msg := range msgs {
		results[i] = openagent.SearchResult{Message: msg, Score: 1.0}
	}
	return results
}

// ── Schema ──

func (m *Memory) migrate() error {
	_, err := m.db.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id       TEXT    NOT NULL,
			role             TEXT    NOT NULL,
			name             TEXT    NOT NULL DEFAULT '',
			content          TEXT    NOT NULL DEFAULT '',
			content_parts    TEXT    NOT NULL DEFAULT '',
			tool_calls       TEXT    NOT NULL DEFAULT '[]',
			tool_call_id     TEXT    NOT NULL DEFAULT '',
			reasoning_content TEXT   NOT NULL DEFAULT '',
			turn             INTEGER NOT NULL DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id, id);

		CREATE TABLE IF NOT EXISTS vectors (
			message_id INTEGER PRIMARY KEY REFERENCES messages(id) ON DELETE CASCADE,
			embedding  BLOB NOT NULL
		);

		CREATE TABLE IF NOT EXISTS compressed (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			data       TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_compressed_session ON compressed(session_id, id);
	`)
	if err != nil {
		return fmt.Errorf("sqlite migrate: %w", err)
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
		return fmt.Errorf("sqlite migrate fts: %w", err)
	case !strings.Contains(createSQL, "trigram"):
		if _, err := m.db.Exec(`DROP TABLE messages_fts`); err != nil {
			return fmt.Errorf("sqlite migrate fts drop: %w", err)
		}
	}

	if _, err := m.db.Exec(
		`CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(content, tokenize='trigram')`,
	); err != nil {
		return fmt.Errorf("sqlite migrate fts create: %w", err)
	}

	// Backfill any messages not yet indexed (fresh/rebuilt table, or rows
	// from a pre-FTS schema). Idempotent via the NOT IN guard.
	if _, err := m.db.Exec(
		`INSERT INTO messages_fts (rowid, content)
		 SELECT id, content FROM messages
		 WHERE content != '' AND id NOT IN (SELECT rowid FROM messages_fts)`,
	); err != nil {
		return fmt.Errorf("sqlite migrate fts backfill: %w", err)
	}
	return nil
}

// ── Helpers ──

func scanMessages(rows *sql.Rows) ([]openagent.Message, error) {
	var msgs []openagent.Message
	for rows.Next() {
		var id int64
		var role, name, content, contentParts, toolCalls, toolCallID, reasoningContent string
		if err := rows.Scan(&id, &role, &name, &content, &contentParts, &toolCalls, &toolCallID, &reasoningContent); err != nil {
			return nil, err
		}
		msg := rowToMessage(role, name, content, contentParts, toolCalls, toolCallID, reasoningContent)
		msg.Index = id
		msgs = append(msgs, msg)
	}
	return msgs, rows.Err()
}

func rowToMessage(role, name, content, contentParts, toolCalls, toolCallID, reasoningContent string) openagent.Message {
	msg := openagent.Message{
		Role:             openagent.Role(role),
		Name:             name,
		Content:          content,
		ReasoningContent: reasoningContent,
		ToolCallID:       toolCallID,
	}
	if contentParts != "" && contentParts != "[]" {
		json.Unmarshal([]byte(contentParts), &msg.ContentParts)
	}
	if toolCalls != "" && toolCalls != "[]" {
		json.Unmarshal([]byte(toolCalls), &msg.ToolCalls)
	}
	return msg
}

func floatsToBytes(v []float64) []byte {
	buf := make([]byte, len(v)*8)
	for i, f := range v {
		binary.LittleEndian.PutUint64(buf[i*8:], math.Float64bits(f))
	}
	return buf
}

func bytesToFloats(b []byte) []float64 {
	v := make([]float64, len(b)/8)
	for i := range v {
		v[i] = math.Float64frombits(binary.LittleEndian.Uint64(b[i*8:]))
	}
	return v
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
