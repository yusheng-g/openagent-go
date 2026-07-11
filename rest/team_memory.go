package rest

import (
	"context"
	"sort"

	openagent "github.com/yusheng-g/openagent-go"
)

// teamAgentMemory splits persistence across agent-private and team-shared
// memory so each agent's internal work (tool calls, tool results) stays
// private while user messages, handoffs, and text output are shared.
//
// Private: keyed by sessionID + "::" + agentName
// Shared:  keyed by sessionID (the team session)
//
// All methods are safe for concurrent use — the underlying Memory
// implementations (SQLite WAL, file RWMutex) handle that.
type teamAgentMemory struct {
	agentName string
	shared    openagent.Memory
	private   openagent.Memory // same underlying store, different key prefix
}

func newTeamAgentMemory(agentName string, shared openagent.Memory) *teamAgentMemory {
	return &teamAgentMemory{
		agentName: agentName,
		shared:    shared,
		private:   shared, // same store; private ops use prefixed sessionID
	}
}

// privateKey returns the agent-scoped session key.
func (m *teamAgentMemory) privateKey(sessionID string) string {
	return sessionID + "::" + m.agentName
}

// ── openagent.Memory interface ──

func (m *teamAgentMemory) Append(ctx context.Context, sessionID string, msg openagent.Message) error {
	// Tool results and assistant messages that carry tool_calls are
	// internal agent work — store in private memory.
	if msg.Role == openagent.RoleTool {
		return m.private.Append(ctx, m.privateKey(sessionID), msg)
	}
	if msg.Role == openagent.RoleAssistant && len(msg.ToolCalls) > 0 {
		return m.private.Append(ctx, m.privateKey(sessionID), msg)
	}
	// Everything else — user messages, text-only assistant output,
	// system messages, handoffs — goes to shared memory.
	return m.shared.Append(ctx, sessionID, msg)
}

func (m *teamAgentMemory) Recent(ctx context.Context, sessionID string, n int) ([]openagent.Message, error) {
	shared, _ := m.shared.Recent(ctx, sessionID, n)
	priv, _ := m.private.Recent(ctx, m.privateKey(sessionID), n)

	// Concatenate: shared (narrative) first, then private (own work).
	// This gives the agent: "here's the conversation so far, and here's
	// what you did last time." The runner's prefix (tr.runMessages)
	// provides exact chronological ordering within the current run.
	result := make([]openagent.Message, 0, len(shared)+len(priv))
	result = append(result, shared...)
	result = append(result, priv...)
	if len(result) > n {
		result = result[len(result)-n:]
	}
	return result, nil
}

func (m *teamAgentMemory) Count(ctx context.Context, sessionID string) (int, error) {
	sharedN, _ := m.shared.Count(ctx, sessionID)
	privN, _ := m.private.Count(ctx, m.privateKey(sessionID))
	return sharedN + privN, nil
}

func (m *teamAgentMemory) Compact(ctx context.Context, sessionID string, throughIndex int, messages []openagent.Message) error {
	// Compaction targets the shared narrative — private memory is small.
	return m.shared.Compact(ctx, sessionID, throughIndex, messages)
}

func (m *teamAgentMemory) Compressed(ctx context.Context, sessionID string) (*openagent.CompressedContext, error) {
	return m.shared.Compressed(ctx, sessionID)
}

func (m *teamAgentMemory) Search(ctx context.Context, sessionID, query string, limit int) ([]openagent.SearchResult, error) {
	sharedResults, _ := m.shared.Search(ctx, sessionID, query, limit)
	privResults, _ := m.private.Search(ctx, m.privateKey(sessionID), query, limit)

	if len(sharedResults)+len(privResults) == 0 {
		return nil, nil
	}

	// Merge and sort by score descending.
	all := make([]openagent.SearchResult, 0, len(sharedResults)+len(privResults))
	all = append(all, sharedResults...)
	all = append(all, privResults...)
	sort.Slice(all, func(i, j int) bool {
		return all[i].Score > all[j].Score
	})
	if len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}

func (m *teamAgentMemory) DeleteSession(ctx context.Context, sessionID string) error {
	// Delete agent-private data. Shared data is deleted by the handler.
	return m.private.DeleteSession(ctx, m.privateKey(sessionID))
}

func (m *teamAgentMemory) Close() error {
	// The shared Memory is owned by the handler — don't close it here.
	return nil
}
