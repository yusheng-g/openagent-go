// Package rest provides a reusable HTTP REST API for openagent-go agents.
//
// Create a Handler and register it on an http.ServeMux:
//
//	agent := openagent.NewAgent("assistant",
//	    openagent.WithModel(model),
//	    openagent.WithMemory(mem),
//	)
//	handler := rest.NewHandler(agent)
//
//	mux := http.NewServeMux()
//	handler.Register(mux)
//	http.ListenAndServe(":8080", mux)
//
// The handler exposes session CRUD, SSE streaming chat, and tool approval.
// It uses Go 1.22+ pattern routing (method-based paths).
//
// Sessions are stored in memory (lost on restart). Message history persists
// via the agent's configured Memory backend.
package rest

import (
	"encoding/json"
	"time"
)

// ── Session ──

// SessionInfo is the public representation of a conversation session.
type SessionInfo struct {
	ID        string    `json:"id"`
	Title     string    `json:"title,omitempty"`
	AgentName string    `json:"agentName"`
	ModelID   string    `json:"modelId,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ── Request / Response types ── [satirize]

// CreateSessionRequest is the optional body for POST /sessions.
// All fields are optional — the handler defaults to the base agent's name.
type CreateSessionRequest struct {
	AgentName string `json:"agentName,omitempty"`
	Title     string `json:"title,omitempty"`
	ModelID   string `json:"modelId,omitempty"` // override model per session
}

// ChatRequest is the body for POST /sessions/{id}/chat.
// Model overrides the session model for this single request (e.g. for
// switching between gpt-4o and gpt-4 within the same conversation).
type ChatRequest struct {
	Message string `json:"message"`
	ModelID string `json:"modelId,omitempty"` // override model for this turn
}

// ApproveRequest is the body for POST /sessions/{id}/approve.
type ApproveRequest struct {
	Allowed bool `json:"allowed"`
}

// ── SSE event ──

// SSEEvent is the JSON payload for a single Server-Sent Event.
// It is serialised as "data: <json>\n\n".
type SSEEvent struct {
	Type string `json:"type"`

	// text_delta, tool_result, retrying, error
	Text string `json:"text,omitempty"`

	// tool_call, tool_approval
	ToolCall *SSEToolCall `json:"tool_call,omitempty"`

	// tool_result
	ToolCallID string `json:"tool_call_id,omitempty"`

	// done
	FinalOutput   string `json:"final_output,omitempty"`
	PromptTokens  int    `json:"prompt_tokens,omitempty"`
	ContextWindow int    `json:"context_window,omitempty"`

	// agent_start, agent_end (team mode)
	Agent string `json:"agent,omitempty"`

	// handoff (team mode)
	HandoffTo string `json:"handoff_to,omitempty"`

	// step_start, step_done, step_failed (plan mode)
	StepID string `json:"step_id,omitempty"`

	// error detail (agent_end, retrying, error)
	Error string `json:"error,omitempty"`

	// stage (pipeline visualization)
	Stage json.RawMessage `json:"stage,omitempty"`
}

// SSEToolCall mirrors an LLM function-call tool invocation.
type SSEToolCall struct {
	ID       string              `json:"id"`
	Function SSEToolCallFunction `json:"function"`
}

// SSEToolCallFunction holds the tool name and JSON-encoded arguments.
type SSEToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}
