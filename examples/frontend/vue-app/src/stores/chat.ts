import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { ChatMessage, SSEEvent, PendingApproval, UsageInfo, SSEToolCall } from '@/types'
import { streamChat } from '@/api/sse'
import * as api from '@/api'

export const useChatStore = defineStore('chat', () => {
  const messages = ref<ChatMessage[]>([])
  const streaming = ref(false)
  const selectedModelId = ref<string>('')
  const availableModels = ref<Array<{ id: string; provider?: string }>>([])
  const pendingApproval = ref<PendingApproval | null>(null)
  // Queue for concurrent tool approvals (SSE events and HTTP responses
  // race on different channels). The exposed pendingApproval shows the
  // first item; approveTool pops it after the POST completes.
  const _approvalQueue: PendingApproval[] = []
  const usage = ref<UsageInfo | null>(null)
  const error = ref<string | null>(null)

  let abortController: AbortController | null = null
  let currentStreamMsg: ChatMessage | null = null
  let pendingThought: string = ''
  let thinkingMsg: ChatMessage | null = null
  let pendingToolCalls: Map<string, ChatMessage> = new Map()
  let onStageEvent: ((event: any) => void) | null = null

  function setStageHandler(fn: ((event: any) => void) | null) {
    onStageEvent = fn
  }

  function msgId() {
    return crypto.randomUUID()
  }

  function now() {
    return Date.now()
  }

  /** Push a message to the list and return its reactive proxy.
   *  Vue wraps objects pushed into a reactive array with a Proxy.
   *  All callers MUST mutate the returned proxy — not the raw object —
   *  otherwise Vue won't detect the change and won't re-render. */
  function pushMsg(msg: ChatMessage): ChatMessage {
    messages.value.push(msg)
    return messages.value[messages.value.length - 1]!
  }

  function sendMessage(
    sessionId: string,
    text: string,
    sessionType: 'single' | 'team' | 'plan',
  ) {
    const modelId = selectedModelId.value || undefined
    error.value = null
    streaming.value = true
    pendingApproval.value = null
    _approvalQueue.length = 0
    pendingToolCalls.clear()
    currentStreamMsg = null
    thinkingMsg = null
    pendingThought = ''

    pushMsg({
      id: msgId(),
      role: 'user',
      content: text,
      timestamp: now(),
    })

    let url: string
    switch (sessionType) {
      case 'team': url = `/team/sessions/${sessionId}/chat`; break
      case 'plan': url = `/plan/sessions/${sessionId}/chat`; break
      default: url = `/sessions/${sessionId}/chat`
    }

    abortController = new AbortController()
    streamChat(url, { message: text, modelId }, handleEvent, abortController.signal)
      .then(() => {
        streaming.value = false
        currentStreamMsg = null
      })
      .catch((e: Error) => {
        if (e.name === 'AbortError') return
        error.value = e.message
        streaming.value = false
        currentStreamMsg = null
      })
  }

  function handleEvent(event: SSEEvent) {
    switch (event.type) {
      case 'thought': {
        // Real-time reasoning display. Reasoning models spend 5+ seconds
        // generating reasoning_content before the final answer.
        // In team mode, agent_start creates thinkingMsg with the agent name
        // so the user immediately knows who is thinking. In single-agent
        // mode, thought is the first event — create thinkingMsg lazily.
        pendingThought += event.text || ''
        if (!currentStreamMsg) {
          if (!thinkingMsg) {
            thinkingMsg = pushMsg({
              id: msgId(),
              role: 'thought',
              content: '',
              agent: event.agent,
              timestamp: now(),
            })
          }
          thinkingMsg.content += event.text || ''
        }
        break
      }

      case 'text_delta': {
        // Convert thinkingMsg to agent message in-place so its position
        // relative to tool calls (which were pushed AFTER thinkingMsg)
        // is preserved. Pushing a new agent message at the end would
        // place it after tool calls — wrong visual order.
        if (!currentStreamMsg) {
          if (thinkingMsg) {
            thinkingMsg.role = 'agent'
            thinkingMsg.thoughtContent = pendingThought || undefined
            thinkingMsg.content = event.text || ''
            thinkingMsg.isStreaming = true
            currentStreamMsg = thinkingMsg
            thinkingMsg = null
          } else {
            currentStreamMsg = pushMsg({
              id: msgId(),
              role: 'agent',
              content: event.text || '',
              thoughtContent: pendingThought || undefined,
              agent: event.agent,
              timestamp: now(),
              isStreaming: true,
            })
          }
          pendingThought = ''
        } else {
          currentStreamMsg.content += event.text || ''
        }
        break
      }

      case 'tool_call': {
        // Keep thinkingMsg visible — tool calls don't interrupt the
        // agent's reasoning; they are part of the agent's working process.
        // Only text_delta replaces the Thinking collapse with final output.
        const tc = event.tool_call
        if (tc) {
          const proxy = pushMsg({
            id: msgId(),
            role: 'tool_call',
            content: tc.function.arguments || '',
            agent: event.agent,
            toolCall: tc,
            timestamp: now(),
          })
          pendingToolCalls.set(tc.id, proxy)
        }
        break
      }

      case 'tool_progress': {
        if (event.tool_call_id) {
          const msg = pendingToolCalls.get(event.tool_call_id)
          if (msg) msg.content += event.text || ''
        }
        break
      }

      case 'tool_result': {
        if (event.tool_call_id) {
          const msg = pendingToolCalls.get(event.tool_call_id)
          if (msg) {
            msg.role = 'tool_result'
            msg.content = event.text || msg.content
            pendingToolCalls.delete(event.tool_call_id)
          }
        }
        break
      }

      case 'tool_approval': {
        const approval: PendingApproval = {
          toolCall: event.tool_call!,
          sessionId: '',
          sessionType: 'single',
        }
        _approvalQueue.push(approval)
        // Show first in queue (the one the user must decide on).
        if (!pendingApproval.value) {
          pendingApproval.value = _approvalQueue[0]
        }
        break
      }

      case 'retrying': {
        pushMsg({
          id: msgId(),
          role: 'system',
          content: `Retrying: ${event.text || 'transient error'}`,
          timestamp: now(),
        })
        break
      }

      case 'aborted': {
        error.value = event.text || 'Aborted'
        streaming.value = false
        break
      }

      case 'done': {
        // If text_delta never arrived (e.g. tool denied, agent stopped),
        // thinkingMsg is still in the array — keep it visible so the user
        // can see what the agent was thinking. Normal path: thinkingMsg was
        // already converted to currentStreamMsg (set to null in text_delta).
        pendingThought = ''
        if (currentStreamMsg) {
          currentStreamMsg.isStreaming = false
          currentStreamMsg = null
        }
        if (event.prompt_tokens != null && event.context_window != null) {
          usage.value = {
            promptTokens: event.prompt_tokens,
            contextWindow: event.context_window,
          }
        }
        streaming.value = false
        break
      }

      case 'error': {
        error.value = event.text || 'Unknown error'
        streaming.value = false
        break
      }

      case 'agent_start': {
        // New agent taking over — reset streaming state from previous agent.
        if (thinkingMsg) {
          messages.value = messages.value.filter(m => m !== thinkingMsg)
          thinkingMsg = null
        }
        if (currentStreamMsg) {
          currentStreamMsg.isStreaming = false
          currentStreamMsg = null
        }
        pendingThought = ''
        // Team mode only: show who is about to act. The label appears
        // before thinking so the user never sees unlabelled reasoning.
        pushMsg({
          id: msgId(),
          role: 'agent_label',
          content: event.agent || '',
          agent: event.agent,
          timestamp: now(),
        })
        thinkingMsg = pushMsg({
          id: msgId(),
          role: 'thought',
          content: '',
          agent: event.agent,
          timestamp: now(),
        })
        break
      }

      case 'agent_end': {
        // Finalize the current agent's message so the next agent
        // starts a fresh bubble instead of appending to it.
        if (currentStreamMsg) {
          currentStreamMsg.isStreaming = false
          currentStreamMsg = null
        }
        break
      }

      case 'handoff': {
        // Reset per-agent streaming state so the next agent gets a
        // clean message bubble with its own name and thinking.
        if (thinkingMsg) {
          messages.value = messages.value.filter(m => m !== thinkingMsg)
          thinkingMsg = null
        }
        if (currentStreamMsg) {
          currentStreamMsg.isStreaming = false
          currentStreamMsg = null
        }
        pendingThought = ''
        pushMsg({
          id: msgId(),
          role: 'handoff',
          content: `${event.agent || '?'} → ${event.handoff_to || '?'}`,
          agent: event.agent,
          timestamp: now(),
        })
        break
      }

      case 'stage':
        if (onStageEvent) onStageEvent(event)
        break

      default:
        break
    }
  }

  async function approveTool(sessionId: string, sessionType: 'single' | 'team' | 'plan', allowed: boolean, feedback?: string) {
    if (!pendingApproval.value) return
    // Dequeue the current approval and advance to the next (or null).
    _approvalQueue.shift()
    pendingApproval.value = _approvalQueue[0] || null
    // POST asynchronously — the UI is already updated.
    try {
      switch (sessionType) {
        case 'single': await api.approveTool(sessionId, allowed, feedback); break
        case 'team': await api.approveTeamTool(sessionId, allowed, feedback); break
        case 'plan': await api.approvePlanTool(sessionId, allowed, feedback); break
      }
    } catch (e) {
      console.error('approveTool:', e)
    }
  }

  async function fetchModels() {
    if (availableModels.value.length > 0) return
    try {
      const data = await api.listModels()
      availableModels.value = data.models || []
    } catch { /* /models not available — use empty list */ }
  }

  function clearChat() {
    abortController?.abort()
    messages.value = []
    streaming.value = false
    pendingApproval.value = null
    _approvalQueue.length = 0
    usage.value = null
    error.value = null
    pendingToolCalls.clear()
    currentStreamMsg = null
    thinkingMsg = null
    pendingThought = ''
  }

  return {
    messages, streaming, pendingApproval, usage, error,
    selectedModelId, availableModels,
    sendMessage, approveTool, clearChat, setStageHandler, fetchModels,
  }
})
