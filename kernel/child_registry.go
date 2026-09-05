package kernel

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/agent"
)

// maxConcurrentSubAgents caps how many async sub-agents may run at once, so a
// model that fires off a large fan-out of explorer calls can't spawn an
// unbounded number of concurrent agent loops (each making API calls). New
// spawns past the cap are rejected with a clear error so the model waits for
// some to finish (it'll be notified via onExit).
const maxConcurrentSubAgents = 16

// childRegistry holds live resumable sub-agents for one Runtime (one session).
// Shared between subAgentTool (spawn) and sendTool (continue) via Deps.
// Children use in-memory stores — their conversations never touch the parent's
// on-disk SessionStore, keeping sub-agent sessions distinct from normal ones.
type childRegistry struct {
	mu          sync.Mutex
	counter     int
	live        map[string]*liveChild // keyed by agent_id (e.g. "explorer-1")
	activeAsync int                   // running async spawns, for the concurrency cap

	// onExit is called when an async sub-agent completes. The ACP layer
	// wires it to trigger an idle turn via the SDK mux's TriggerTurn — the
	// note becomes the prompt input, fully serialized with user turns via
	// sessionLocks. nil = no trigger (CLI one-shot, sync mode).
	onExitMu sync.RWMutex
	onExit   func(note string)
}

// SetOnExit registers the completion callback. Called by the ACP layer after
// the SDK mux injects a TurnTrigger. When set, subAgentTool.Execute runs
// children asynchronously (returns immediately, result delivered via onExit).
func (r *childRegistry) SetOnExit(fn func(note string)) {
	r.onExitMu.Lock()
	r.onExit = fn
	r.onExitMu.Unlock()
}

func (r *childRegistry) fireOnExit(note string) {
	r.onExitMu.RLock()
	fn := r.onExit
	r.onExitMu.RUnlock()
	if fn != nil {
		fn(note)
	}
}

// hasOnExit reports whether async completion is wired. When true, spawn runs
// asynchronously; when false, synchronously (blocking the parent turn).
func (r *childRegistry) hasOnExit() bool {
	r.onExitMu.RLock()
	defer r.onExitMu.RUnlock()
	return r.onExit != nil
}

// liveChild is one resumable sub-agent: a stable id, the child's session id
// (used as the SessionStore key), the agent config + deps it runs with (the
// deps carry a per-child memSessionStore so history accumulates across calls),
// and a mutex serializing execution on this child. toolFilter is held rather
// than pre-resolved so each runChild re-snapshots the parent's current tool
// set — the parent's tools may change between the spawn and a later continue
// (plan-mode transitions, dynamic injection).
type liveChild struct {
	id         string // agent_id shown to model (e.g. "explorer-1")
	sessionID  string // stable child session.ID ("sub-explorer-1")
	cfg        *agent.Agent
	deps       Deps                    // deps.SessionStore = child's memSessionStore; Compressor = same store; Tools resolved per-run
	toolFilter func() []openagent.Tool // nil = use deps.Tools as-is
	mu         sync.Mutex
	running    bool               // true while executing; concurrent calls error
	cancel     context.CancelFunc // cancels the async run; nil for sync spawns
}

// startAsync runs a child in a background goroutine and returns immediately.
// On completion it stores the result on the child, decrements the concurrency
// counter, clears running, then fires onExit. The ACP layer wires onExit to
// the SDK mux's TriggerTurn, which starts a fully-serialized idle turn carrying
// the <system-reminder> note — no concurrent turns, no parallel lock.
//
// The child's running flag is set atomically with the concurrency check under
// child.mu, so two concurrent calls to the same child can't both start. Returns
// an error if the child is already running OR the concurrency cap is reached.
func (r *childRegistry) startAsync(child *liveChild, session openagent.Session, task, description string) error {
	// Atomic check-and-set: child.running + concurrency cap under child.mu
	// and r.mu together. child.mu first (always same child), then r.mu (the
	// shared counter) — consistent lock order, no deadlock.
	child.mu.Lock()
	if child.running {
		child.mu.Unlock()
		return fmt.Errorf("sub-agent %s is still processing; wait for it to finish", child.id)
	}
	r.mu.Lock()
	if r.activeAsync >= maxConcurrentSubAgents {
		// List the running agent_ids so the model knows who to wait for.
		running := make([]string, 0, r.activeAsync)
		for id, c := range r.live {
			c.mu.Lock()
			if c.running {
				running = append(running, id)
			}
			c.mu.Unlock()
		}
		r.mu.Unlock()
		child.mu.Unlock()
		return fmt.Errorf("too many sub-agents running (%d/%d max) — wait for some to finish before launching more. Currently running: %s",
			r.activeAsync, maxConcurrentSubAgents, strings.Join(running, ", "))
	}
	r.activeAsync++
	child.running = true
	r.mu.Unlock()
	child.mu.Unlock()

	go func() {
		defer func() {
			child.mu.Lock()
			child.running = false
			child.mu.Unlock()
			r.mu.Lock()
			r.activeAsync--
			r.mu.Unlock()
		}()
		// Cancellable background context: the child outlives the parent
		// turn, but KillAll (session close) can cancel it. The child's own
		// MaxTurns bounds its runtime otherwise.
		bgCtx, cancel := context.WithCancel(context.Background())
		child.mu.Lock()
		child.cancel = cancel
		child.mu.Unlock()
		slog.Debug("subagent async start", "agent_id", child.id, "session_id", child.sessionID)
		output, err := runChild(bgCtx, child.cfg, child.resolveDeps(), session, task, nil, child.sessionID)
		slog.Debug("subagent async done", "agent_id", child.id, "output_len", len(output), "err", err)

		// If the child was cancelled by KillAll (session close), skip the
		// completion notification — the session is gone, triggerIdleTurn would
		// call OnPrompt on a non-existent session. Check BEFORE calling cancel()
		// (our own cancel), so a normal completion isn't mistaken for a KillAll.
		// KillAll cancels the context, runChild returns ctx.Err(), and we
		// don't fire onExit for a cancelled (not completed) run.
		if bgCtx.Err() != nil {
			cancel() // release context resources
			return
		}
		cancel() // release context resources

		stopReason := ""
		if err != nil {
			output = fmt.Sprintf("error: %v", err)
			stopReason = "error"
		}
		r.fireOnExit(FormatSubAgentNote(child.id, description, output, stopReason))
	}()
	return nil
}

// KillAll cancels every running async child and clears the registry. Called
// on session close so background goroutines don't outlive the session. After
// KillAll, the registry is empty — a sub_agent_send to any prior child id
// returns "not found".
func (r *childRegistry) KillAll() {
	r.mu.Lock()
	children := make([]*liveChild, 0, len(r.live))
	for _, c := range r.live {
		children = append(children, c)
	}
	r.live = make(map[string]*liveChild)
	r.activeAsync = 0
	r.mu.Unlock()

	for _, c := range children {
		c.mu.Lock()
		cancel := c.cancel
		c.mu.Unlock()
		if cancel != nil {
			cancel()
		}
	}
}

// resolveDeps materializes the child's tool set at run time from the parent's
// current snapshot (the parent's tools may have grown since the spawn).
func (c *liveChild) resolveDeps() Deps {
	deps := c.deps
	if c.toolFilter != nil {
		deps.Tools = c.toolFilter()
	}
	return deps
}

func newChildRegistry() *childRegistry {
	return &childRegistry{live: make(map[string]*liveChild)}
}

// spawn builds a fresh memSessionStore (with the parent's summarizer for
// compaction) + child deps, registers the child under a stable
// "sub-<name>-<n>" session id, and returns it. The returned liveChild.deps
// carries a non-nil SessionStore so the child's history persists across calls.
func (r *childRegistry) spawn(parentDeps Deps, cfg *agent.Agent, toolFilter func() []openagent.Tool) *liveChild {
	r.mu.Lock()
	r.counter++
	id := fmt.Sprintf("%s-%d", cfg.Name, r.counter)
	sessionID := fmt.Sprintf("sub-%s", id)
	r.mu.Unlock()

	// One in-memory store per child, backed by the parent's summarizer so the
	// child gets full compaction support (long delegations won't overflow). The
	// store implements both SessionStore and Compressor.
	store := newMemSessionStore(parentDeps.Summarizer)

	deps := parentDeps
	deps.Tools = nil // resolved per-run via toolFilter (see liveChild.resolveDeps)
	deps.SessionStore = store
	deps.Compressor = store
	// Sub-agent tool calls are NOT gated by the parent's approver — the child
	// runs with its own tool scope (read-only for explorer) and its own policy.
	// Inheriting the parent's manual-mode approver would prompt the user for
	// every read/grep the child does, defeating the point of delegation.
	deps.HumanApprover = nil
	deps.ApprovalMemory = nil

	child := &liveChild{
		id:         id,
		sessionID:  sessionID,
		cfg:        cfg,
		deps:       deps,
		toolFilter: toolFilter,
	}

	r.mu.Lock()
	r.live[id] = child
	r.mu.Unlock()
	return child
}

// get returns the live child for agentID, or (nil, false) if unknown.
func (r *childRegistry) get(id string) (*liveChild, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.live[id]
	return c, ok
}

// ── in-memory SessionStore + Compressor ──

// memSessionStore is an in-memory SessionStore+Compressor for sub-agent
// history. One instance per child; never persists to disk. Implements both
// interfaces so the child gets full compaction support (100-turn runs
// won't overflow the context window).
//
// The summarizer is the same one the parent uses (built from the same model)
// — only the storage backing changes (slice vs sqlite). When the parent has
// no compressor configured (nil), the child degrades to no-compaction rather
// than crashing: Append/Recent/RecentAfter still work, Compact is a no-op.
type memSessionStore struct {
	mu         sync.Mutex
	msgs       []openagent.Message                     // append-only, oldest first
	summarizer openagent.Summarizer                    // nil = no compaction
	compressed map[string]*openagent.CompressedContext // keyed by sessionID
}

// newMemSessionStore builds an in-memory store. summarizer is the parent's
// (shared) summarizer for compaction; nil means the child degrades to
// no-compaction (Append/Recent still work, Compact is a no-op).
func newMemSessionStore(summarizer openagent.Summarizer) *memSessionStore {
	return &memSessionStore{
		compressed: make(map[string]*openagent.CompressedContext),
		summarizer: summarizer,
	}
}

// ── SessionStore ──

func (m *memSessionStore) Append(_ context.Context, _ string, msg openagent.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgs = append(m.msgs, msg)
	return nil
}

func (m *memSessionStore) Recent(_ context.Context, _ string, n int, offset int) ([]openagent.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if n <= 0 || offset < 0 {
		return nil, nil
	}
	total := len(m.msgs)
	if offset >= total {
		return nil, nil
	}
	// offset skips from the END (most recent), matching sqlite's Recent semantics.
	start := total - offset - n
	if start < 0 {
		start = 0
	}
	end := total - offset
	if end > total {
		end = total
	}
	out := make([]openagent.Message, end-start)
	copy(out, m.msgs[start:end])
	return out, nil
}

func (m *memSessionStore) RecentAfter(_ context.Context, _ string, throughIndex, n int) ([]openagent.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if throughIndex < 0 {
		throughIndex = 0
	}
	if n <= 0 || throughIndex >= len(m.msgs) {
		return nil, nil
	}
	rest := m.msgs[throughIndex:]
	if n < len(rest) {
		rest = rest[:n]
	}
	out := make([]openagent.Message, len(rest))
	copy(out, rest)
	return out, nil
}

func (m *memSessionStore) Count(_ context.Context, _ string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.msgs), nil
}

func (m *memSessionStore) DeleteSession(_ context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// One store per child — clearing msgs + compressed is the whole session.
	m.msgs = nil
	delete(m.compressed, sessionID)
	return nil
}

// ── Compressor ──

// Compact compresses messages up to throughIndex into a summary, mirroring
// sqlite's logic (session/sqlite/message_store.go:246): load the previous
// CompressedContext marker, take all[lastIdx:safeIdx], summarize, store the
// new CompressedContext with ThroughIndex=safeIdx. Messages are never deleted.
func (m *memSessionStore) Compact(ctx context.Context, sessionID string, throughIndex int, messages []openagent.Message) error {
	if m.summarizer == nil {
		return nil // no summarizer configured: compaction unavailable
	}

	prev, err := m.Compressed(ctx, sessionID)
	if err != nil {
		// Unreadable marker: fall back to compressing from the start.
		prev = nil
	}
	lastIdx := 0
	if prev != nil {
		lastIdx = prev.ThroughIndex
	}
	if lastIdx >= throughIndex {
		return nil // nothing new to compress
	}

	// Use pre-fetched messages if provided, otherwise use the in-memory slice.
	all := messages
	if all == nil {
		m.mu.Lock()
		all = m.msgs
		m.mu.Unlock()
	}
	if len(all) == 0 || throughIndex <= 0 || throughIndex > len(all) {
		return nil
	}

	// Adjust to a safe boundary (don't cut tool_call/tool_result pairs).
	safeIdx := openagent.SafeCompressionBoundary(all, throughIndex)
	if safeIdx <= 0 || safeIdx > len(all) {
		return nil
	}

	if lastIdx < safeIdx {
		newMsgs := all[lastIdx:safeIdx]
		cc, sumErr := m.summarizer.Summarize(ctx, newMsgs, prev)
		if sumErr != nil {
			return sumErr
		}
		if cc != nil {
			cc.ThroughIndex = safeIdx
			m.mu.Lock()
			m.compressed[sessionID] = cc // replace previous (subsumes it)
			m.mu.Unlock()
		}
	}
	return nil
}

// Compressed returns the stored CompressedContext, or nil if none exists.
func (m *memSessionStore) Compressed(_ context.Context, sessionID string) (*openagent.CompressedContext, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cc := m.compressed[sessionID]
	if cc == nil {
		return nil, nil
	}
	// Return a copy so callers can't mutate the stored summary.
	out := *cc
	return &out, nil
}
