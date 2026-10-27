package session

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Runtime manages the session lifecycle: create, save, restore, and
// checkpoint. It composes the metadata store ([Store]) with the message
// store ([SessionStore]) so the application layer has one explicit API
// instead of juggling two stores.
//
// A checkpoint is a recoverable restore point: the session metadata plus
// the message count at that point. Restore after a crash returns the
// checkpointed messages (the conversation up to the last checkpoint),
// which the kernel can resume from.
type Runtime interface {
	// Create initializes a new session for a user. The session is empty
	// until Save or Checkpoint.
	Create(ctx context.Context, userID string) (SessionInfo, error)
	// Save persists session metadata (title, cwd, meta, ...).
	Save(ctx context.Context, info SessionInfo) error
	// Get returns session metadata, or (nil, nil) when the session does
	// not exist (or metadata storage is disabled).
	Get(ctx context.Context, id string) (*SessionInfo, error)
	// List returns all session metadata (nil when metadata storage is
	// disabled).
	List(ctx context.Context) ([]SessionInfo, error)
	// Restore loads a session and its messages. ok=false when the
	// session does not exist.
	Restore(ctx context.Context, id string) (SessionInfo, []MessageRef, bool, error)
	// Checkpoint persists metadata + the current message count as a
	// recoverable restore point.
	Checkpoint(ctx context.Context, id string) error
	// Fork copies a session under a new ID: messages, the compressed
	// summary, and metadata (minus per-session counters). The source
	// session is left untouched.
	Fork(ctx context.Context, srcID string) (SessionInfo, error)
	// Delete removes metadata and messages for a session.
	Delete(ctx context.Context, id string) error
}

// MessageRef is a lightweight message reference (content + index) used by
// Restore to report checkpointed conversation length.
type MessageRef struct {
	Index   int64
	Content string
}

// DefaultRuntime is the standard session lifecycle implementation over a
// metadata Store and a message SessionStore. A nil store disables that
// half (metadata-only or message-only sessions).
type DefaultRuntime struct {
	Meta     Store
	Messages SessionStore
}

// NewRuntime creates a DefaultRuntime.
func NewRuntime(meta Store, msgs SessionStore) *DefaultRuntime {
	return &DefaultRuntime{Meta: meta, Messages: msgs}
}

func newSessionID() string {
	return "sess-" + uuid.NewString()[:12]
}

// Create implements Runtime.
func (r *DefaultRuntime) Create(ctx context.Context, userID string) (SessionInfo, error) {
	now := time.Now()
	info := SessionInfo{
		ID:        newSessionID(),
		CreatedAt: now,
		UpdatedAt: now,
		Meta:      map[string]any{},
	}
	if userID != "" {
		info.Meta["user_id"] = userID
	}
	if r.Meta != nil {
		if err := r.Meta.Save(ctx, info); err != nil {
			return SessionInfo{}, fmt.Errorf("session create: %w", err)
		}
	}
	return info, nil
}

// Save implements Runtime.
func (r *DefaultRuntime) Save(ctx context.Context, info SessionInfo) error {
	if r.Meta == nil {
		return nil
	}
	info.UpdatedAt = time.Now()
	return r.Meta.Save(ctx, info)
}

// Get implements Runtime: metadata only (nil when disabled/not found).
func (r *DefaultRuntime) Get(ctx context.Context, id string) (*SessionInfo, error) {
	if r.Meta == nil {
		return nil, nil
	}
	return r.Meta.Get(ctx, id)
}

// List implements Runtime: all session metadata (nil when disabled).
func (r *DefaultRuntime) List(ctx context.Context) ([]SessionInfo, error) {
	if r.Meta == nil {
		return nil, nil
	}
	return r.Meta.List(ctx)
}

// Restore implements Runtime: metadata + all messages after the last
// checkpoint (full recovery — no cap). checkpoint_msgs marks how many
// messages the last Checkpoint covered; everything after it is the
// un-recovered remainder. Without a checkpoint the whole conversation is
// returned.
func (r *DefaultRuntime) Restore(ctx context.Context, id string) (SessionInfo, []MessageRef, bool, error) {
	var info SessionInfo
	if r.Meta != nil {
		got, err := r.Meta.Get(ctx, id)
		if err != nil {
			return SessionInfo{}, nil, false, fmt.Errorf("session restore: %w", err)
		}
		if got == nil {
			return SessionInfo{}, nil, false, nil
		}
		info = *got
	}
	var refs []MessageRef
	if r.Messages != nil {
		// Remainder after the checkpoint (JSON numbers decode as float64).
		start := 0
		if v, ok := info.Meta["checkpoint_msgs"]; ok {
			if f, ok := v.(float64); ok {
				start = int(f)
			}
		}
		total, err := r.Messages.Count(ctx, id)
		if err != nil {
			return SessionInfo{}, nil, false, fmt.Errorf("session restore count: %w", err)
		}
		remain := total - start
		if remain < 0 {
			remain = 0
		}
		if remain > 0 {
			msgs, err := r.Messages.Recent(ctx, id, remain, 0)
			if err != nil {
				return SessionInfo{}, nil, false, fmt.Errorf("session restore messages: %w", err)
			}
			for _, m := range msgs {
				refs = append(refs, MessageRef{Index: m.Index, Content: m.Content})
			}
		}
	}
	return info, refs, true, nil
}

// Checkpoint implements Runtime: save metadata and mark the current
// message count in the metadata (crash-recovery restore point).
func (r *DefaultRuntime) Checkpoint(ctx context.Context, id string) error {
	if r.Meta == nil {
		return nil
	}
	info, err := r.Meta.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("session checkpoint: %w", err)
	}
	if info == nil {
		return fmt.Errorf("session checkpoint: session %q not found", id)
	}
	count := int64(0)
	if r.Messages != nil {
		if n, err := r.Messages.Count(ctx, id); err == nil {
			count = int64(n)
		}
	}
	if info.Meta == nil {
		info.Meta = map[string]any{}
	}
	info.Meta["checkpoint_msgs"] = count
	info.Meta["checkpoint_at"] = time.Now().Format(time.RFC3339)
	info.UpdatedAt = time.Now()
	return r.Meta.Save(ctx, *info)
}

// Fork implements Runtime: copy the source session under a new ID. The
// message history and the compressed-summary layer (when the backend
// supports copying it) are duplicated; metadata is copied except for the
// keys that must not carry over (checkpoint markers, counters, creation
// stamps), and the fork is stamped with its origin.
func (r *DefaultRuntime) Fork(ctx context.Context, srcID string) (SessionInfo, error) {
	var src SessionInfo
	if r.Meta != nil {
		got, err := r.Meta.Get(ctx, srcID)
		if err != nil {
			return SessionInfo{}, fmt.Errorf("session fork: %w", err)
		}
		if got == nil {
			return SessionInfo{}, fmt.Errorf("session fork: session %q not found", srcID)
		}
		src = *got
	}

	now := time.Now()
	dst := SessionInfo{
		ID:                    newSessionID(),
		Cwd:                   src.Cwd,
		Title:                 forkedTitle(src.Title),
		AdditionalDirectories: src.AdditionalDirectories,
		CreatedAt:             now,
		UpdatedAt:             now,
		Meta:                  forkMeta(src.Meta),
	}
	dst.SetMeta("forked_from", srcID)
	dst.SetMeta("forked_at", now.Format(time.RFC3339))

	if r.Messages != nil {
		if err := r.copyMessages(ctx, srcID, dst.ID); err != nil {
			return SessionInfo{}, err
		}
		if c, ok := r.Messages.(CompressedCopier); ok {
			if err := c.CopyCompressed(ctx, srcID, dst.ID); err != nil {
				return SessionInfo{}, fmt.Errorf("session fork compressed: %w", err)
			}
		}
	}
	if r.Meta != nil {
		if err := r.Meta.Save(ctx, dst); err != nil {
			return SessionInfo{}, fmt.Errorf("session fork: %w", err)
		}
	}
	return dst, nil
}

func forkedTitle(src string) string {
	if src == "" {
		return ""
	}
	// A fork of a fork must not accumulate suffixes: strip a trailing
	// " (forked)" before appending, so lineage depth stays invisible.
	src = strings.TrimSuffix(src, " (forked)")
	return src + " (forked)"
}

// forkMeta copies session metadata, dropping keys that must not carry
// over to a fork: checkpoint markers, per-session counters, and creation
// stamps (the fork gets fresh values of its own). The keys are the
// cross-package contract of "per-session state that a fork must reset" —
// keep this list in sync with the writers (runtime.go Checkpoint,
// acp/server.go saveTotalTokens / session meta created_at).
var forkMetaResetKeys = map[string]struct{}{
	"checkpoint_msgs": {},
	"checkpoint_at":   {},
	"total_tokens":    {},
	"created_at":      {},
}

func forkMeta(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src)+2)
	for k, v := range src {
		if _, drop := forkMetaResetKeys[k]; drop {
			continue
		}
		dst[k] = v
	}
	return dst
}

// copyMessages duplicates the full conversation, oldest first, so the
// fork's stored order matches the source exactly.
func (r *DefaultRuntime) copyMessages(ctx context.Context, srcID, dstID string) error {
	total, err := r.Messages.Count(ctx, srcID)
	if err != nil {
		return fmt.Errorf("session fork count: %w", err)
	}
	if total == 0 {
		return nil
	}
	msgs, err := r.Messages.RecentAfter(ctx, srcID, 0, total)
	if err != nil {
		return fmt.Errorf("session fork read: %w", err)
	}
	for _, msg := range msgs {
		if err := r.Messages.Append(ctx, dstID, msg); err != nil {
			return fmt.Errorf("session fork append: %w", err)
		}
	}
	return nil
}

// Delete implements Runtime: metadata + messages.
func (r *DefaultRuntime) Delete(ctx context.Context, id string) error {
	if r.Meta != nil {
		if err := r.Meta.Delete(ctx, id); err != nil {
			return fmt.Errorf("session delete: %w", err)
		}
	}
	if r.Messages != nil {
		if err := r.Messages.DeleteSession(ctx, id); err != nil {
			return fmt.Errorf("session delete messages: %w", err)
		}
	}
	return nil
}
