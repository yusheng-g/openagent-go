// REST API client for session CRUD + chat + approval endpoints.
// All three modes (single, team, plan) share the same pattern.

import type { SessionInfo, CreateSessionRequest, ChatRequest, ApproveRequest, AgentInfo } from '@/types'

const BASE = ''

async function request<T>(url: string, opts?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${url}`, {
    headers: { 'Content-Type': 'application/json' },
    ...opts,
  })
  if (!res.ok) {
    const text = await res.text().catch(() => '')
    throw new Error(`${res.status}: ${text}`)
  }
  if (res.status === 204) return undefined as T
  return res.json()
}

// ── Single-agent sessions ──

export function listSessions(): Promise<SessionInfo[]> {
  return request<SessionInfo[]>('/sessions')
}

export function createSession(body?: CreateSessionRequest): Promise<SessionInfo> {
  return request<SessionInfo>('/sessions', {
    method: 'POST',
    body: body ? JSON.stringify(body) : undefined,
  })
}

export function getSession(id: string): Promise<SessionInfo> {
  return request<SessionInfo>(`/sessions/${id}`)
}

export function deleteSession(id: string): Promise<void> {
  return request<void>(`/sessions/${id}`, { method: 'DELETE' })
}

export function approveTool(sessionId: string, allowed: boolean, feedback?: string): Promise<{ status: string }> {
  const body: ApproveRequest = { allowed }
  if (feedback) body.feedback = feedback
  return request(`/sessions/${sessionId}/approve`, {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

// ── Team sessions ──

export function listTeamSessions(): Promise<SessionInfo[]> {
  return request<SessionInfo[]>('/team/sessions')
}

export function createTeamSession(body?: CreateSessionRequest): Promise<SessionInfo> {
  return request<SessionInfo>('/team/sessions', {
    method: 'POST',
    body: body ? JSON.stringify(body) : undefined,
  })
}

export function deleteTeamSession(id: string): Promise<void> {
  return request<void>(`/team/sessions/${id}`, { method: 'DELETE' })
}

export function listTeamAgents(sessionId: string): Promise<{ agents: AgentInfo[] }> {
  return request(`/team/sessions/${sessionId}/agents`)
}

export function addTeamAgent(sessionId: string, name: string, description: string, instructions: string): Promise<{ status: string }> {
  return request(`/team/sessions/${sessionId}/agents`, {
    method: 'POST',
    body: JSON.stringify({ name, description, instructions }),
  })
}

export function removeTeamAgent(sessionId: string, name: string): Promise<void> {
  return request<void>(`/team/sessions/${sessionId}/agents?name=${encodeURIComponent(name)}`, {
    method: 'DELETE',
  })
}

export function approveTeamTool(sessionId: string, allowed: boolean, feedback?: string): Promise<{ status: string }> {
  const body: ApproveRequest = { allowed }
  if (feedback) body.feedback = feedback
  return request(`/team/sessions/${sessionId}/approve`, {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

// ── Plan sessions ──

export function listPlanSessions(): Promise<SessionInfo[]> {
  return request<SessionInfo[]>('/plan/sessions')
}

export function createPlanSession(body?: CreateSessionRequest): Promise<SessionInfo> {
  return request<SessionInfo>('/plan/sessions', {
    method: 'POST',
    body: body ? JSON.stringify(body) : undefined,
  })
}

export function deletePlanSession(id: string): Promise<void> {
  return request<void>(`/plan/sessions/${id}`, { method: 'DELETE' })
}

export function approvePlanTool(sessionId: string, allowed: boolean, feedback?: string): Promise<{ status: string }> {
  const body: ApproveRequest = { allowed }
  if (feedback) body.feedback = feedback
  return request(`/plan/sessions/${sessionId}/approve`, {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export async function generatePlan(
  sessionId: string,
  goal: string,
  onThinking: (text: string) => void,
): Promise<any> {
  const response = await fetch(`${BASE}/plan/sessions/${sessionId}/generate`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ goal }),
  })
  if (!response.ok) throw new Error(`HTTP ${response.status}`)

  // Read SSE stream: plan_thinking events while LLM generates,
  // then a final plan_generated event with the PlanDef JSON.
  const reader = response.body!.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) break

    buffer += decoder.decode(value, { stream: true })
    const lines = buffer.split('\n')
    buffer = lines.pop() || ''

    for (const line of lines) {
      if (!line.startsWith('data: ')) continue
      let evt: any
      try {
        evt = JSON.parse(line.slice(6))
      } catch {
        continue // skip malformed frames
      }
      if (evt.type === 'plan_thinking') {
        onThinking(evt.text || '')
      } else if (evt.type === 'plan_generated') {
        return JSON.parse(evt.text)
      } else if (evt.type === 'plan_error') {
        throw new Error(evt.error || 'Plan generation failed')
      }
    }
  }

  throw new Error('Plan generation stream ended without result')
}

export function getPlan(sessionId: string): Promise<any> {
  return request(`/plan/sessions/${sessionId}/plan`)
}

export function updatePlan(sessionId: string, def: any): Promise<any> {
  return request(`/plan/sessions/${sessionId}/plan`, {
    method: 'PUT',
    body: JSON.stringify(def),
  })
}

export function executePlan(sessionId: string): Promise<{ status: string }> {
  return request(`/plan/sessions/${sessionId}/execute`, { method: 'POST' })
}

export function cancelPlan(sessionId: string): Promise<{ status: string }> {
  return request(`/plan/sessions/${sessionId}/cancel`, { method: 'POST' })
}

export function retryPlanStep(sessionId: string, stepId: string): Promise<{ status: string }> {
  return request(`/plan/sessions/${sessionId}/steps/${stepId}/retry`, { method: 'POST' })
}

export function listModels(): Promise<{ models: Array<{ id: string; provider?: string }> }> {
  return request('/models')
}

export function replan(sessionId: string, feedback: string): Promise<{ status: string }> {
  return request(`/plan/sessions/${sessionId}/replan`, {
    method: 'POST',
    body: JSON.stringify({ feedback }),
  })
}
