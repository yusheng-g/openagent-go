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
    modelId?: string,
  ) {
    error.value = null
    streaming.value = true
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
        pendingThought += event.text || ''
        if (!currentStreamMsg) {
          if (!thinkingMsg) {
            thinkingMsg = pushMsg({
              id: msgId(),
              role: 'thought',
              content: '',
              timestamp: now(),
            })
          }
          thinkingMsg.content += event.text || ''
        }
        break
      }

      case 'text_delta': {
        // Replace the thinking indicator with the agent message.
        if (thinkingMsg) {
          messages.value = messages.value.filter(m => m !== thinkingMsg)
          thinkingMsg = null
        }
        if (!currentStreamMsg) {
          currentStreamMsg = pushMsg({
            id: msgId(),
            role: 'agent',
            content: '',
            thoughtContent: pendingThought || undefined,
            agent: event.agent,
            timestamp: now(),
            isStreaming: true,
          })
          pendingThought = ''
        }
        currentStreamMsg.content += event.text || ''
        break
      }

      case 'tool_call': {
        if (thinkingMsg) {
          messages.value = messages.value.filter(m => m !== thinkingMsg)
          thinkingMsg = null
        }
        pendingThought = ''
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
        pendingApproval.value = {
          toolCall: event.tool_call!,
          sessionId: '',
          sessionType: 'single',
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
        if (thinkingMsg) {
          messages.value = messages.value.filter(m => m !== thinkingMsg)
          thinkingMsg = null
        }
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
        break

      case 'agent_end':
        break

      case 'handoff': {
        pushMsg({
          id: msgId(),
          role: 'system',
          content: `${event.agent} → ${event.handoff_to}`,
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
    thinkingMsg = null
    pendingThought = ''
  }

  return {
    messages, streaming, pendingApproval, usage, error,
    sendMessage, approveTool, clearChat, setStageHandler,
  }
})
