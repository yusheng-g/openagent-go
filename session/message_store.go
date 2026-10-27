// Package session defines the runtime session model: short-term conversation
// storage ([SessionStore]) and token-budget compression ([Compressor]).
//
// These two interfaces are the P2 split of the former monolithic
// openagent.Memory interface — see the Context Architecture design.
// Implementations live in session/sqlite and session/file; durable
// knowledge (MemoryProvider) lives in provider/memory, over the same
// physical storage (zero schema migration).
package session

import (
	"context"

	openagent "github.com/yusheng-g/openagent-go"
)

// SessionStore persists the current conversation — the short-lived working
// state of a session (user input, assistant output, tool exchanges). It is
// the "what is happening right now" layer, keyed by sessionID.
//
// Implementations must be safe for concurrent use. A nil SessionStore means
// no persistence: each run starts fresh.
type SessionStore interface {
	// Append adds a message to the conversation history.
	Append(ctx context.Context, sessionID string, msg openagent.Message) error

	// Recent returns up to n most recent messages, skipping the first
	// offset messages from the end. offset=0 returns the latest n.
	Recent(ctx context.Context, sessionID string, n int, offset int) ([]openagent.Message, error)

	// RecentAfter returns up to n messages after the throughIndex-th
	// message (0 = from the start), oldest first. Messages are never
	// deleted, so the post-summary increment is the only part the model
	// sees — the summary covers up to throughIndex. throughIndex is the
	// session-relative position (Compressed.ThroughIndex semantics), not
	// a backend row id.
	RecentAfter(ctx context.Context, sessionID string, throughIndex, n int) ([]openagent.Message, error)

	// Count returns the total number of stored messages for a session.
	Count(ctx context.Context, sessionID string) (int, error)

	// DeleteSession removes all data for the given session.
	DeleteSession(ctx context.Context, sessionID string) error
}

// Compressor owns the compressed-summary layer: history compression and
// token budget control. It answers "what has already happened, in brief"
// and is fed by the Context Runtime when the working set exceeds the
// token budget.
//
// The ThroughIndex contract must be honored: Compressed returns the last
// summary with its ThroughIndex marker; Compact only compresses messages
// after that index (incremental/rolling compression). Original messages
// are NEVER deleted.
type Compressor interface {
	// Compact compresses messages up to throughIndex into a summary.
	// messages is an optional pre-fetched slice; when nil the backend
	// fetches messages internally.
	Compact(ctx context.Context, sessionID string, throughIndex int, messages []openagent.Message) error

	// Compressed returns the stored CompressedContext, or nil if none exists.
	Compressed(ctx context.Context, sessionID string) (*openagent.CompressedContext, error)
}

// CompressedCopier is an optional SessionStore capability: copying the
// compressed-summary layer from one session to another. Backends that
// implement it let Runtime.Fork carry the summary across, so the fork
// resumes with the same context budget as its source. Backends that
// don't are skipped (the fork re-compresses from scratch when needed).
type CompressedCopier interface {
	// CopyCompressed copies srcID's compressed context to dstID,
	// replacing anything dstID already has. A no-op when srcID has none.
	CopyCompressed(ctx context.Context, srcID, dstID string) error
}
