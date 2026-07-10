import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { ChatMessage, SSEEvent, PendingApproval, UsageInfo, SSEToolCall } from '@/types'
import { streamChat } from '@/api/sse'
import * as api from '@/api'

export const useChatStore = defineStore('chat', () => {
  const messages = ref<ChatMessage[]>([])
  const streaming = ref(false)
  const pendingApproval = ref<PendingApproval | null>(null)
  const usage = ref<UsageInfo | null>(null)
  const error = ref<string | null>(null)

  let abortController: AbortController | null = null
  let currentStreamMsg: ChatMessage | null = null
  let pendingThought: string = ''
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

  function sendMessage(
    sessionId: string,
    text: string,
    sessionType: 'single' | 'team' | 'plan',
    modelId?: string,
  ) {
    // Reset state
    error.value = null
    streaming.value = true
    pendingToolCalls.clear()
    currentStreamMsg = null
    pendingThought = ''

    // Add user message
    const userMsg: ChatMessage = {
      id: msgId(),
      role: 'user',
      content: text,
      timestamp: now(),
    }
    messages.value.push(userMsg)

    // Determine URL
    let url: string
    switch (sessionType) {
      case 'team': url = `/team/sessions/${sessionId}/chat`; break
      case 'plan': url = `/plan/sessions/${sessionId}/chat`; break
      default: url = `/sessions/${sessionId}/chat`
    }

    // Start SSE stream
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
        // Buffer reasoning tokens to attach to the next agent message.
        pendingThought += event.text || ''
        break
      }

      case 'text_delta': {
        if (!currentStreamMsg) {
          currentStreamMsg = {
            id: msgId(),
            role: 'agent',
            content: '',
            thoughtContent: pendingThought || undefined,
            agent: event.agent,
            timestamp: now(),
            isStreaming: true,
          }
          pendingThought = ''
          messages.value.push(currentStreamMsg)
        }
        currentStreamMsg.content += event.text || ''
        break
      }

      case 'tool_call': {
        // Tool calls clear pending thought without displaying it —
        // the reasoning before a tool call is the model's internal planning.
        pendingThought = ''
        const tc = event.tool_call
        if (tc) {
          const msg: ChatMessage = {
            id: msgId(),
            role: 'tool_call',
            content: tc.function.arguments || '',
            agent: event.agent,
            toolCall: tc,
            timestamp: now(),
          }
          messages.value.push(msg)
          pendingToolCalls.set(tc.id, msg)
        }
        break
      }

      case 'tool_progress': {
        // Update the matching tool call with progress
        if (event.tool_call_id) {
          const msg = pendingToolCalls.get(event.tool_call_id)
          if (msg) {
            msg.content += event.text || ''
          }
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
        pendingApproval.value = {
          toolCall: event.tool_call!,
          sessionId: '',
          sessionType: 'single',
        }
        break
      }

      case 'retrying': {
        const msg: ChatMessage = {
          id: msgId(),
          role: 'system',
          content: `Retrying: ${event.text || 'transient error'}`,
          timestamp: now(),
        }
        messages.value.push(msg)
        break
      }

      case 'aborted': {
        error.value = event.text || 'Aborted'
        streaming.value = false
        break
      }

      case 'done': {
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

      case 'agent_start':
        // Team agent turn begins — shown as a subtle label above its messages.
        // The text_delta events that follow carry the same agent name.
        break

      case 'agent_end':
        // Team agent turn ends. No visible marker needed.
        break

      case 'handoff': {
        // Handoff between team agents — show a subtle transition badge.
        const msg: ChatMessage = {
          id: msgId(),
          role: 'system',
          content: `${event.agent} → ${event.handoff_to}`,
          agent: event.agent,
          timestamp: now(),
        }
        messages.value.push(msg)
        break
      }

      case 'stage':
        // Forward to pipeline panel via a callback set by the view.
        if (onStageEvent) onStageEvent(event)
        break

      default:
        break
    }
  }

  async function approveTool(sessionId: string, sessionType: 'single' | 'team' | 'plan', allowed: boolean) {
    if (!pendingApproval.value) return
    try {
      switch (sessionType) {
        case 'single': await api.approveTool(sessionId, allowed); break
        case 'team': await api.approveTeamTool(sessionId, allowed); break
        case 'plan': await api.approvePlanTool(sessionId, allowed); break
      }
      pendingApproval.value = null
    } catch (e) {
      console.error('approveTool:', e)
    }
  }

  function clearChat() {
    abortController?.abort()
    messages.value = []
    streaming.value = false
    pendingApproval.value = null
    usage.value = null
    error.value = null
    pendingToolCalls.clear()
    currentStreamMsg = null
    pendingThought = ''
  }

  return {
    messages, streaming, pendingApproval, usage, error,
    sendMessage, approveTool, clearChat, setStageHandler,
  }
})
