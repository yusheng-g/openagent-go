// Package sqlite implements the durable knowledge provider
// (context.MemoryProvider) with SQLite: the knowledge table plus its
// vector index. It opens the same database file as session/sqlite
// (WAL supports multiple connections) — this package is the memory
// domain only; conversation storage lives in the session domain.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"

	_ "modernc.org/sqlite"

	openagent "github.com/yusheng-g/openagent-go"
	ctxpkg "github.com/yusheng-g/openagent-go/context"
)

// Memory implements context.MemoryProvider backed by SQLite: durable
// knowledge items (preferences, facts, lessons) scoped by
// user/project/session, with semantic vector recall when an embedder is
// configured.
type Memory struct {
	db       *sql.DB
	embedder openagent.Embedder
}

// New opens the knowledge database at path and runs migrations. WAL mode
// and a 30s busy timeout make concurrent access from the session store
// safe — the session store's startup migrations can hold the write lock
// for several seconds on this shared database file (the old 5s timeout
// tripped with SQLITE_BUSY on concurrent cold starts).
func New(path string) (*Memory, error) {
	dsn := path + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(30000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite knowledge: open: %w", err)
	}
	m := &Memory{db: db}
	if err := m.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return m, nil
}

// WithEmbedder enables semantic vector recall. nil (default) falls back to
// keyword matching.
func (m *Memory) WithEmbedder(e openagent.Embedder) *Memory {
	m.embedder = e
	return m
}

// UpdateEmbedder refreshes the baseURL, apiKey, and model of the configured
// embedder in place. Used by runtime_set_embedding_config to refresh
// credentials without restarting. No-op when no embedder is configured or
// the embedder does not expose an Update method.
func (m *Memory) UpdateEmbedder(baseURL, apiKey, model string) {
	if u, ok := m.embedder.(interface{ Update(string, string, string) }); ok {
		u.Update(baseURL, apiKey, model)
	}
}

// Close releases the database connection.
func (m *Memory) Close() error { return m.db.Close() }

// ── context.MemoryProvider ──

var _ ctxpkg.MemoryProvider = (*Memory)(nil)

// Recall returns durable knowledge matching the query, scoped by user
// (and project/session when set). Semantic (vector) recall first, keyword
// LIKE fallback.
func (m *Memory) Recall(ctx context.Context, scope ctxpkg.ContextScope, query string, limit int) ([]ctxpkg.MemoryEntry, error) {
	return m.knowledgeRecall(ctx, scope, query, limit)
}

// Store persists a durable knowledge item under the given scope, and
// indexes its embedding when an embedder is configured. A non-empty
// Topic upserts: the existing item with the same (scope, topic) is
// replaced (the extractor's "update" operation); empty topic appends.
func (m *Memory) Store(ctx context.Context, scope ctxpkg.ContextScope, item ctxpkg.MemoryItem) error {
	meta, err := json.Marshal(item.Meta)
	if err != nil {
		meta = []byte("{}")
	}

	var id int64
	if item.Topic != "" {
		// Upsert by (scope_user, scope_project, scope_session, topic) — the
		// session is part of the key so an update in one session can never
		// overwrite (and re-scope) another session's entry.
		res, err := m.db.ExecContext(ctx,
			`UPDATE knowledge SET kind = ?, content = ?, meta = ?
			 WHERE scope_user = ? AND scope_project = ? AND scope_session = ? AND topic = ?`,
			item.Kind, item.Content, string(meta),
			scope.UserID, scope.ProjectID, scope.SessionID, item.Topic,
		)
		if err != nil {
			return fmt.Errorf("sqlite knowledge store: %w", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("sqlite knowledge store: %w", err)
		}
		if n > 0 {
			// Updated in place: look up the id to refresh its vector.
			err := m.db.QueryRowContext(ctx,
				`SELECT id FROM knowledge WHERE scope_user = ? AND scope_project = ? AND scope_session = ? AND topic = ?`,
				scope.UserID, scope.ProjectID, scope.SessionID, item.Topic,
			).Scan(&id)
			if err != nil {
				return fmt.Errorf("sqlite knowledge store: %w", err)
			}
			slog.Debug("knowledge stored (update)", "id", id, "kind", item.Kind, "topic", item.Topic)
			m.indexEmbedding(ctx, id, item.Content, knowledgeSubject(item))
			return nil
		}
	}

	res, err := m.db.ExecContext(ctx,
		`INSERT INTO knowledge (scope_user, scope_project, scope_session, kind, content, meta, topic)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		scope.UserID, scope.ProjectID, scope.SessionID, item.Kind, item.Content, string(meta), item.Topic,
	)
	if err != nil {
		return fmt.Errorf("sqlite knowledge store: %w", err)
	}
	id, _ = res.LastInsertId()

	if id > 0 {
		slog.Debug("knowledge stored (insert)", "id", id, "kind", item.Kind, "topic", item.Topic)
		m.indexEmbedding(ctx, id, item.Content, knowledgeSubject(item))
	}
	return nil
}

// indexEmbedding refreshes the vector index for a knowledge row.
// Best-effort: an absent embedder is a no-op, but a failure is logged —
// a stale vector would silently return outdated content on recall. The
// DecisionExtractor event reports the outcome of this extraction step so a
// consumer can tell a stored-with-vector from a stored-without-vector or a
// failed-embedding (the row is persisted either way — this is best-effort).
func (m *Memory) indexEmbedding(ctx context.Context, id int64, content, subject string) {
	obs := openagent.DecisionObserverFromContext(ctx)
	ri := openagent.RunInfoFromContext(ctx)
	// This leaf has no session param, so stamp SessionID directly from ctx —
	// the kernel's WithSession (run.go) plumbs the session down through
	// context/runtime Build → MemoryProvider.Store. The free ObserveDecision
	// helper is NOT in the call path (it takes a RunObserver; obs here is the
	// DecisionObserver extracted from ctx), so SessionID must be set here, not
	// left for the helper. CallID stays blank: extraction is not a tool call.
	sessionID := ""
	if s, ok := openagent.SessionFromContext(ctx); ok {
		sessionID = s.ID
	}
	emit := func(outcome string, detail map[string]any) {
		if obs != nil {
			obs.ObserveDecision(ctx, openagent.DecisionEvent{
				Layer: openagent.DecisionExtractor, Outcome: outcome, Subject: subject,
				Detail: detail, RunID: ri.RunID, TurnID: ri.TurnID,
				ParentRunID: ri.ParentRunID, SessionID: sessionID,
			})
		}
	}
	if m.embedder == nil || id <= 0 {
		emit(openagent.OutcomeSkipped, map[string]any{"id": id, "reason": "no embedder or no row id"})
		return
	}
	vec, err := m.embedder.Embed(ctx, content)
	if err != nil {
		emit(openagent.OutcomeFailed, map[string]any{"id": id, "error": err.Error()})
		return
	}
	buf := floatsToBytes(vec)
	if _, err := m.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO knowledge_vectors (knowledge_id, embedding) VALUES (?, ?)`,
		id, buf,
	); err != nil {
		emit(openagent.OutcomeFailed, map[string]any{"id": id, "dim": len(vec), "error": err.Error()})
	} else {
		emit(openagent.OutcomeStored, map[string]any{"id": id, "dim": len(vec)})
	}
}

// knowledgeRecall is the recall pipeline: vector-first when an embedder is
// configured, keyword LIKE fallback otherwise.
func (m *Memory) knowledgeRecall(ctx context.Context, scope ctxpkg.ContextScope, query string, limit int) ([]ctxpkg.MemoryEntry, error) {
	if limit <= 0 {
		limit = 5
	}
	// DecisionObserver + RunInfo are plumbed through ctx by the context
	// runtime (Build) — a leaf provider emits without holding a struct
	// field. nil = silent (no observer, or ctx not prepared by a run).
	obs := openagent.DecisionObserverFromContext(ctx)
	ri := openagent.RunInfoFromContext(ctx)
	// Same stamping policy as indexEmbedding: ParentRunID direct (joins
	// team/orchestrator), SessionID stamped directly from ctx (the free
	// ObserveDecision helper is NOT in the call path — it takes a RunObserver;
	// obs here is the DecisionObserver from ctx — so SessionID must be set
	// here, not left for the helper). WithSession plumbs session down via
	// context/runtime Build → MemoryProvider.Recall. CallID blank (recall is
	// not a tool call).
	sessionID := ""
	if s, ok := openagent.SessionFromContext(ctx); ok {
		sessionID = s.ID
	}
	emit := func(layer, outcome string, detail map[string]any) {
		if obs != nil {
			obs.ObserveDecision(ctx, openagent.DecisionEvent{
				Layer: layer, Outcome: outcome, Subject: query,
				Detail: detail, RunID: ri.RunID, TurnID: ri.TurnID,
				ParentRunID: ri.ParentRunID, SessionID: sessionID,
			})
		}
	}
	// Semantic recall first: cosine-ranked by the configured embedder. All
	// three outcomes emit so a consumer can tell a tried-and-missed vector
	// path (fell through to keyword) from an unconfigured one (no embedder
	// = no event at all).
	if m.embedder != nil {
		entries, err := m.knowledgeVectorRecall(ctx, scope, query, limit)
		if err != nil {
			emit(openagent.DecisionVectorRecall, openagent.OutcomeFailed, map[string]any{"error": err.Error()})
		} else if len(entries) > 0 {
			emit(openagent.DecisionVectorRecall, openagent.OutcomeHit, map[string]any{"count": len(entries), "top_score": entries[0].Score})
			return entries, nil
		} else {
			emit(openagent.DecisionVectorRecall, openagent.OutcomeMiss, nil)
		}
	}
	// Fallback: keyword LIKE matching.
	words := knowledgeKeywords(query)
	if len(words) == 0 {
		emit(openagent.DecisionKeywordRecall, openagent.OutcomeMiss, map[string]any{"reason": "no keywords"})
		return nil, nil
	}
	// Build OR-ed LIKE clauses for the keywords.
	clauses := make([]string, 0, len(words))
	args := []any{scope.UserID, scope.ProjectID}
	if scope.SessionID != "" {
		args = append(args, scope.SessionID)
	}
	for _, w := range words {
		clauses = append(clauses, "LOWER(content) LIKE ?")
		args = append(args, "%"+w+"%")
	}
	args = append(args, limit)

	sessionClause := ""
	if scope.SessionID != "" {
		sessionClause = " AND (scope_session = ? OR scope_session = '')"
	}
	rows, err := m.db.QueryContext(ctx, `
		SELECT kind, content, meta, topic
		FROM knowledge
		WHERE (scope_user = ? OR scope_user = '')
		  AND (scope_project = ? OR scope_project = '')`+sessionClause+`
		  AND (`+strings.Join(clauses, " OR ")+`)
		ORDER BY id DESC
		LIMIT ?`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("sqlite knowledge recall: %w", err)
	}
	defer rows.Close()

	var entries []ctxpkg.MemoryEntry
	for rows.Next() {
		var kind, content, meta, topic string
		if err := rows.Scan(&kind, &content, &meta, &topic); err != nil {
			return nil, err
		}
		score := 0.5
		for _, w := range words {
			if strings.HasPrefix(strings.ToLower(content), w) {
				score = 1.0
				break
			}
		}
		entries = append(entries, ctxpkg.MemoryEntry{
			Kind:    ctxpkg.MemoryKind(kind),
			Content: content,
			Topic:   topic,
			Score:   score,
		})
	}
	if err := rows.Err(); err != nil {
		emit(openagent.DecisionKeywordRecall, openagent.OutcomeFailed, map[string]any{"error": err.Error()})
		return nil, err
	}
	if len(entries) == 0 {
		emit(openagent.DecisionKeywordRecall, openagent.OutcomeMiss, nil)
	} else {
		emit(openagent.DecisionKeywordRecall, openagent.OutcomeHit, map[string]any{"count": len(entries)})
	}
	return entries, nil
}

// knowledgeVectorRecall ranks knowledge items by cosine similarity to the
// query embedding.
func (m *Memory) knowledgeVectorRecall(ctx context.Context, scope ctxpkg.ContextScope, query string, limit int) ([]ctxpkg.MemoryEntry, error) {
	// Query side: use the embedder's query variant when an embedder
	// implements EmbedQuery (e.g. one that applies a query instruction
	// prefix), else fall back to plain Embed. The current OpenAI
	// embedder does not implement EmbedQuery, so the fallback path is
	// the live one.
	var qvec []float64
	var err error
	if qe, ok := m.embedder.(interface {
		EmbedQuery(ctx context.Context, text string) ([]float64, error)
	}); ok {
		qvec, err = qe.EmbedQuery(ctx, query)
	} else {
		qvec, err = m.embedder.Embed(ctx, query)
	}
	if err != nil {
		return nil, err
	}

	sessionClause := ""
	args := []any{scope.UserID, scope.ProjectID}
	if scope.SessionID != "" {
		sessionClause = " AND (k.scope_session = ? OR k.scope_session = '')"
		args = append(args, scope.SessionID)
	}

	// Rank over ALL scoped candidates: truncating in SQL (ORDER BY id
	// DESC LIMIT) would return the newest N rows, not the most similar N —
	// the knowledge store is small, so similarity ranking happens in Go.
	rows, err := m.db.QueryContext(ctx, `
		SELECT k.id, k.kind, k.content, k.topic, v.embedding
		FROM knowledge k
		JOIN knowledge_vectors v ON v.knowledge_id = k.id
		WHERE (k.scope_user = ? OR k.scope_user = '')
		  AND (k.scope_project = ? OR k.scope_project = '')`+sessionClause, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type scored struct {
		entry ctxpkg.MemoryEntry
		sim   float64
	}
	var hits []scored
	for rows.Next() {
		var id int64
		var kind, content, topic string
		var blob []byte
		if err := rows.Scan(&id, &kind, &content, &topic, &blob); err != nil {
			return nil, err
		}
		vec := bytesToFloats(blob)
		hits = append(hits, scored{
			entry: ctxpkg.MemoryEntry{Kind: ctxpkg.MemoryKind(kind), Content: content, Topic: topic},
			sim:   cosineSim(qvec, vec),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].sim > hits[j].sim })
	if len(hits) > limit {
		hits = hits[:limit]
	}
	out := make([]ctxpkg.MemoryEntry, 0, len(hits))
	for _, h := range hits {
		e := h.entry
		e.Score = h.sim
		out = append(out, e)
	}
	return out, nil
}

// knowledgeSubject returns the DecisionEvent subject for a stored item:
// the topic when set (the extractor's "update" key), else the kind. Gives
// an observer a stable handle on what was extracted/indexed.
func knowledgeSubject(item ctxpkg.MemoryItem) string {
	if item.Topic != "" {
		return item.Topic
	}
	return string(item.Kind)
}

// knowledgeKeywords tokenizes a recall query into keywords: latin
// alphanumeric runs (3+ chars) plus CJK ideographs (kept individually —
// splitting on them would shred Chinese queries into empty keywords and
// the keyword fallback would always miss). Mirrors the extractor's
// tokenizer.
func knowledgeKeywords(q string) []string {
	fields := strings.FieldsFunc(strings.ToLower(q), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r >= '一' && r <= '鿿')
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if len(f) >= 3 {
			out = append(out, f)
		}
	}
	return out
}

// ── migration ──

func (m *Memory) migrate() error {
	_, err := m.db.Exec(`
		CREATE TABLE IF NOT EXISTS knowledge (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			scope_user    TEXT NOT NULL DEFAULT '',
			scope_project TEXT NOT NULL DEFAULT '',
			scope_session TEXT NOT NULL DEFAULT '',
			kind          TEXT NOT NULL DEFAULT 'fact',
			content       TEXT NOT NULL,
			meta          TEXT NOT NULL DEFAULT '{}',
			topic         TEXT NOT NULL DEFAULT '',
			created_at    TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE INDEX IF NOT EXISTS idx_knowledge_scope ON knowledge(scope_user, scope_project, scope_session);

		CREATE TABLE IF NOT EXISTS knowledge_vectors (
			knowledge_id INTEGER PRIMARY KEY REFERENCES knowledge(id) ON DELETE CASCADE,
			embedding    BLOB NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("sqlite knowledge migrate: %w", err)
	}
	return nil
}

// ── helpers ──

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

// cosineSim computes cosine similarity of two float64 vectors. Vectors of
// different lengths are incomparable (a dimension mismatch means the
// stored embedding came from a different model) — return 0 rather than a
// meaningless partial-prefix score.
func cosineSim(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
