// TypeScript interfaces mirroring the REST API types from rest/types.go and plan/types.go.

// ── REST API ──

export interface SessionInfo {
  id: string
  title?: string
  agentName: string
  modelId?: string
  createdAt: string
  updatedAt: string
}

export interface CreateSessionRequest {
  agentName?: string
  title?: string
  modelId?: string
}

export interface ChatRequest {
  message: string
  modelId?: string
}

export interface ApproveRequest {
  allowed: boolean
  feedback?: string
}

// ── SSE Event (canonical REST format) ──

export interface SSEToolCall {
  id: string
  function: {
    name: string
    arguments: string // JSON string
  }
}

export interface SSEEvent {
  type: string
  text?: string
  tool_call?: SSEToolCall
  tool_call_id?: string
  final_output?: string
  prompt_tokens?: number
  context_window?: number
  agent?: string
  handoff_to?: string
  step_id?: string
  error?: string
  stage?: any // raw JSON object for pipeline panel (json.RawMessage in Go)
}

// ── Chat message (UI model) ──

export type MessageRole = 'user' | 'agent' | 'thought' | 'tool_call' | 'tool_result' | 'system' | 'handoff' | 'agent_label'

export interface ChatMessage {
  id: string
  role: MessageRole
  content: string
  thoughtContent?: string  // reasoning that preceded this message (deepseek-r1, o1)
  agent?: string
  toolCall?: SSEToolCall
  toolCallId?: string
  stepId?: string
  timestamp: number
  isStreaming?: boolean
}

// ── Plan types ──

export interface StepDef {
  id: string
  agent: string
  task: string
  depends_on?: string[]
  final?: boolean
  max_retries?: number
}

export interface PlanDef {
  goal: string
  steps: StepDef[]
}

export type StepStatus = 'pending' | 'running' | 'done' | 'failed' | 'skipped'

export interface StepState {
  status: StepStatus
  output: string
  summary: string
  error?: string
  toolCalls: { name: string; args: string; result?: string }[]
}

// ── Team types ──

export interface AgentInfo {
  name: string
  description: string
  type: 'internal' | 'external'
}

// ── Tool approval ──

export interface PendingApproval {
  toolCall: SSEToolCall
  sessionId: string
  sessionType: 'single' | 'team' | 'plan'
}

// ── Usage ──

export interface UsageInfo {
  promptTokens: number
  contextWindow: number
}
