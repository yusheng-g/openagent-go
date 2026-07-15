import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { ChatMessage, SSEEvent, PendingApproval, UsageInfo, SSEToolCall } from '@/types'
import { streamChat } from '@/api/sse'
import * as api from '@/api'

// ── Per-session state ──
// Each session has its own message list, streaming state, and SSE controller.
// Switching sessions preserves the old session's state (including live stream)
// so you can switch back mid-generation and pick up where you left off.

interface SessionPane {
  messages: ChatMessage[]
  streaming: boolean
  usage: UsageInfo | null
  error: string | null
  abortController: AbortController | null
  currentStreamMsg: ChatMessage | null
  thinkingMsg: ChatMessage | null
  pendingThought: string
  pendingToolCalls: Map<string, ChatMessage>
  pendingApproval: PendingApproval | null
  _approvalQueue: PendingApproval[]
  _fetched: boolean // true after initial message load
}

export const useChatStore = defineStore('chat', () => {
  // ── Global (cross-session) state ──
  // selectedModelKey stores "provider:modelId" (composite key from registry).
  // When provider is empty the key is just ":modelId".
  const selectedModelKey = ref<string>('')
  const availableModels = ref<Array<{ id: string; provider?: string }>>([])
  // Human-readable label for the current model selection.
  const selectedModelLabel = computed(() => {
    const key = selectedModelKey.value
    const m = availableModels.value.find(m => (m.provider || '') + ':' + m.id === key)
    if (m) return m.provider ? `${m.id} (${m.provider})` : m.id
    return key.replace(/^:/, '') // strip leading colon if any
  })
  const contextWindow = ref(0)
  const messageCount = ref(0)
  const currentSessionId = ref<string | null>(null)
  const currentSessionType = ref<'single' | 'team' | 'plan'>('single')
  let onStageEvent: ((event: any) => void) | null = null

  // ── Per-session pane cache ──
  const _panes = new Map<string, SessionPane>()

  function makePane(): SessionPane {
    return {
      messages: [],
      streaming: false,
      usage: null,
      error: null,
      abortController: null,
      currentStreamMsg: null,
      thinkingMsg: null,
      pendingThought: '',
      pendingToolCalls: new Map(),
      pendingApproval: null,
      _approvalQueue: [],
      _fetched: false,
    }
  }

  function getPane(sid: string): SessionPane {
    let p = _panes.get(sid)
    if (!p) {
      p = makePane()
      _panes.set(sid, p)
    }
    return p
  }

  // ── Active-session views (what the UI reads) ──
  // These are plain reactive refs — on session switch they are synced
  // from the target pane's state. They must stay as plain refs (not
  // computed) because handleEvent mutates them via pushMsg etc.
  const messages = ref<ChatMessage[]>([])
  const streaming = ref(false)
  const usage = ref<UsageInfo | null>(null)
  const error = ref<string | null>(null)
  const pendingApproval = ref<PendingApproval | null>(null)
  // Runtime helpers (not exported, but swapped on session switch)
  let _abortController: AbortController | null = null
  let _currentStreamMsg: ChatMessage | null = null
  let _thinkingMsg: ChatMessage | null = null
  let _pendingThought = ''
  let _pendingToolCalls: Map<string, ChatMessage> = new Map()
  let _approvalQueue: PendingApproval[] = []

  /** Save current active state back into its pane. */
  function saveActive() {
    const sid = currentSessionId.value
    if (!sid) return
    const p = getPane(sid)
    p.messages = messages.value
    p.streaming = streaming.value
    p.usage = usage.value
    p.error = error.value
    p.abortController = _abortController
    p.currentStreamMsg = _currentStreamMsg
    p.thinkingMsg = _thinkingMsg
    p.pendingThought = _pendingThought
    p.pendingToolCalls = _pendingToolCalls
    p.pendingApproval = pendingApproval.value
    p._approvalQueue = _approvalQueue
  }

  /** Restore target session's state into the active refs.
   *  After restoreActive, pane.messages IS the reactive proxy backed by
   *  the ref — so mutations via pane.messages.push() trigger Vue reactivity. */
  function restoreActive(sid: string) {
    const p = getPane(sid)
    messages.value = p.messages
    p.messages = messages.value // reactive proxy, not raw array
    streaming.value = p.streaming
    usage.value = p.usage
    error.value = p.error
    _abortController = p.abortController
    _currentStreamMsg = p.currentStreamMsg
    _thinkingMsg = p.thinkingMsg
    _pendingThought = p.pendingThought
    _pendingToolCalls = p.pendingToolCalls
    pendingApproval.value = p.pendingApproval
    _approvalQueue = p._approvalQueue
  }

  // ── Helpers ──

  function msgId() { return crypto.randomUUID() }
  function now() { return Date.now() }

  function pushMsg(msg: ChatMessage): ChatMessage {
    messages.value.push(msg)
    return messages.value[messages.value.length - 1]!
  }

  function setStageHandler(fn: ((event: any) => void) | null) {
    onStageEvent = fn
  }

  // ── Session switching ──

  /** Activate a session — preserve current, restore target.
   *  Does NOT kill the previous session's SSE stream. */
  function activateSession(sid: string, type?: 'single' | 'team' | 'plan') {
    if (currentSessionId.value === sid) return
    if (type) currentSessionType.value = type
    saveActive()
    currentSessionId.value = sid
    restoreActive(sid)
    // First activation: load history from backend.
    const p = getPane(sid)
    if (!p._fetched) {
      p._fetched = true
      fetchSessionDetail(sid, type)
      fetchMessages(sid, type)
    }
  }

  // ── Chat ──

  function sendMessage(
    sessionId: string,
    text: string,
    sessionType: 'single' | 'team' | 'plan',
  ) {
    currentSessionType.value = sessionType
    // Parse "provider:modelId" key into separate fields for the API.
    const key = selectedModelKey.value || availableModels.value[0]?.id || ''
    const colon = key.indexOf(':')
    const modelProvider = colon >= 0 ? key.slice(0, colon) : ''
    const modelId = colon >= 0 ? key.slice(colon + 1) : key

    // Capture the pane for this session — SSE events will push here.
    // saveActive already wrote the current state, so the pane is fresh.
    const pane = getPane(sessionId)

    pane.error = null
    pane.streaming = true
    pane.pendingApproval = null
    pane._approvalQueue.length = 0
    pane.pendingToolCalls.clear()
    pane.currentStreamMsg = null
    pane.thinkingMsg = null
    pane.pendingThought = ''

    // If this is the active session, sync pane state into active refs
    // so the UI sees the reset. messages.value must share its reactive
    // identity with pane.messages so SSE events (which push to
    // pane.messages) update the UI.
    if (sessionId === currentSessionId.value) {
      messages.value = pane.messages
      pane.messages = messages.value // reactive proxy
      error.value = null
      streaming.value = true
      pendingApproval.value = null
      _approvalQueue.length = 0
      _pendingToolCalls.clear()
      _currentStreamMsg = null
      _thinkingMsg = null
      _pendingThought = ''
    }

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

    pane.abortController?.abort()
    pane.abortController = new AbortController()
    if (sessionId === currentSessionId.value) {
      _abortController = pane.abortController
    }

    // SSE handler pushes directly to the pane, not the active refs.
    // This way stream events always land in the right session even
    // if the user switches to another tab mid-stream.
    const handler = makeHandler(pane)

    streamChat(url, { message: text, modelId, provider: modelProvider }, handler, pane.abortController!.signal)
      .then(() => {
        pane.streaming = false
        pane.currentStreamMsg = null
        if (sessionId === currentSessionId.value) {
          streaming.value = false
          _currentStreamMsg = null
        }
      })
      .catch((e: Error) => {
        if (e.name === 'AbortError') return
        pane.error = e.message
        pane.streaming = false
        pane.currentStreamMsg = null
        if (sessionId === currentSessionId.value) {
          error.value = e.message
          streaming.value = false
          _currentStreamMsg = null
        }
      })
  }

  // ── SSE event handler factory ──
  // Every push into p.messages MUST read back the last element to get
  // Vue's reactive Proxy — raw object references don't trigger reactivity.

  function panePush(p: SessionPane, msg: ChatMessage): ChatMessage {
    p.messages.push(msg)
    return p.messages[p.messages.length - 1]!
  }

  function makeHandler(p: SessionPane) {
    return function handleEvent(event: SSEEvent) {
      switch (event.type) {
        case 'thought': {
          p.pendingThought += event.text || ''
          if (!p.currentStreamMsg) {
            if (!p.thinkingMsg) {
              p.thinkingMsg = panePush(p, {
                id: msgId(),
                role: 'thought',
                content: '',
                agent: event.agent,
                timestamp: now(),
              })
            }
            p.thinkingMsg.content += event.text || ''
          }
          break
        }

        case 'text_delta': {
          if (!p.currentStreamMsg) {
            if (p.thinkingMsg) {
              p.thinkingMsg.role = 'agent'
              p.thinkingMsg.thoughtContent = p.pendingThought || undefined
              p.thinkingMsg.content = event.text || ''
              p.thinkingMsg.isStreaming = true
              p.currentStreamMsg = p.thinkingMsg
              p.thinkingMsg = null
            } else {
              p.currentStreamMsg = panePush(p, {
                id: msgId(),
                role: 'agent',
                content: event.text || '',
                thoughtContent: p.pendingThought || undefined,
                agent: event.agent,
                timestamp: now(),
                isStreaming: true,
              })
            }
            p.pendingThought = ''
          } else {
            p.currentStreamMsg.content += event.text || ''
          }
          break
        }

        case 'tool_call': {
          const tc = event.tool_call
          if (tc) {
            const proxy = panePush(p, {
              id: msgId(),
              role: 'tool_call',
              content: tc.function.arguments || '',
              agent: event.agent,
              toolCall: tc,
              toolCallId: tc.id,
              timestamp: now(),
            })
            p.pendingToolCalls.set(tc.id, proxy)
          }
          break
        }

        case 'tool_progress': {
          if (event.tool_call_id) {
            const msg = p.pendingToolCalls.get(event.tool_call_id)
            if (msg) msg.content += event.text || ''
          }
          break
        }

        case 'tool_result': {
          if (event.tool_call_id) {
            const msg = p.pendingToolCalls.get(event.tool_call_id)
            if (msg) {
              msg.role = 'tool_result'
              msg.content = event.text || msg.content
              p.pendingToolCalls.delete(event.tool_call_id)
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
          p._approvalQueue.push(approval)
          if (!p.pendingApproval) p.pendingApproval = p._approvalQueue[0]
          if (currentSessionId.value && getPane(currentSessionId.value) === p) {
            pendingApproval.value = p.pendingApproval
          }
          break
        }

        case 'retrying': {
          panePush(p, {
            id: msgId(),
            role: 'system',
            content: `Retrying: ${event.text || 'transient error'}`,
            timestamp: now(),
          })
          break
        }

        case 'aborted': {
          p.error = event.text || 'Aborted'
          p.streaming = false
          if (currentSessionId.value && getPane(currentSessionId.value) === p) {
            error.value = p.error
            streaming.value = false
          }
          break
        }

        case 'done': {
          p.pendingThought = ''
          if (p.currentStreamMsg) {
            p.currentStreamMsg.isStreaming = false
            p.currentStreamMsg = null
          }
          if (event.prompt_tokens != null && event.context_window != null) {
            p.usage = { promptTokens: event.prompt_tokens, contextWindow: event.context_window }
          }
          p.streaming = false
          if (currentSessionId.value && getPane(currentSessionId.value) === p) {
            if (event.prompt_tokens != null && event.context_window != null) usage.value = p.usage
            streaming.value = false
          }
          break
        }

        case 'error': {
          p.error = event.text || 'Unknown error'
          p.streaming = false
          if (currentSessionId.value && getPane(currentSessionId.value) === p) {
            error.value = p.error
            streaming.value = false
          }
          break
        }

        case 'agent_start': {
          // Team mode: label who acts next. Remove old thinking
          // placeholder in-place (splice, not filter) to preserve
          // reactive identity with messages.value.
          if (p.thinkingMsg) {
            const idx = p.messages.indexOf(p.thinkingMsg)
            if (idx >= 0) p.messages.splice(idx, 1)
          }
          p.thinkingMsg = null
          if (p.currentStreamMsg) { p.currentStreamMsg.isStreaming = false; p.currentStreamMsg = null }
          p.pendingThought = ''
          panePush(p, { id: msgId(), role: 'agent_label', content: event.agent || '', agent: event.agent, timestamp: now() })
          p.thinkingMsg = panePush(p, { id: msgId(), role: 'thought', content: '', agent: event.agent, timestamp: now() })
          break
        }

        case 'agent_end': {
          if (p.currentStreamMsg) { p.currentStreamMsg.isStreaming = false; p.currentStreamMsg = null }
          break
        }

        case 'handoff': {
          if (p.thinkingMsg) {
            const idx = p.messages.indexOf(p.thinkingMsg)
            if (idx >= 0) p.messages.splice(idx, 1)
          }
          p.thinkingMsg = null
          if (p.currentStreamMsg) { p.currentStreamMsg.isStreaming = false; p.currentStreamMsg = null }
          p.pendingThought = ''
          panePush(p, { id: msgId(), role: 'handoff', content: `${event.agent || '?'} → ${event.handoff_to || '?'}`, agent: event.agent, timestamp: now() })
          break
        }

        case 'stage':
          if (onStageEvent) onStageEvent(event)
          break

        default:
          break
      }
    }
  }

  // ── Approval ──

  async function approveTool(sessionId: string, sessionType: 'single' | 'team' | 'plan', allowed: boolean, feedback?: string) {
    if (!pendingApproval.value) return
    _approvalQueue.shift()
    pendingApproval.value = _approvalQueue[0] || null
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

  // ── Data fetching ──

  async function fetchModels() {
    if (availableModels.value.length > 0) return
    try {
      const data = await api.listModels()
      availableModels.value = data.models || []
      if (availableModels.value.length > 0 && !selectedModelKey.value) {
        const first = availableModels.value[0]
        selectedModelKey.value = (first.provider || '') + ':' + first.id
      }
    } catch { /* /models not available */ }
  }

  async function fetchSessionDetail(sessionId: string, type?: 'single' | 'team' | 'plan') {
    const t = type || currentSessionType.value
    try {
      const fn = t === 'team' ? api.getTeamSessionDetail :
                t === 'plan' ? api.getPlanSessionDetail :
                api.getSessionDetail
      const d = await fn(sessionId)
      contextWindow.value = d.contextWindow || 0
      messageCount.value = d.messageCount || 0
      // Restore session model, or use first available.  Prevents stale
      // key leaking from a previous session into one that never selected a model.
      if (d.modelId) {
        selectedModelKey.value = (d.provider ? d.provider + ':' : '') + d.modelId
      } else if (availableModels.value.length > 0) {
        const first = availableModels.value[0]
        selectedModelKey.value = (first.provider || '') + ':' + first.id
      }
    } catch { /* ignore */ }
  }

  async function fetchMessages(sessionId: string, type?: 'single' | 'team' | 'plan') {
    const t = type || currentSessionType.value
    // Capture the session we're fetching for — stale fetch from a
    // previous session must not overwrite the current chat.
    const reqSession = sessionId
    try {
      const fn = t === 'team' ? api.listTeamMessages :
                t === 'plan' ? api.listPlanMessages :
                api.listMessages
      const msgs = await fn(sessionId, 100)
      // Switched sessions while loading — drop stale result.
      if (reqSession !== currentSessionId.value) return
      const converted: ChatMessage[] = []
      for (const m of msgs) {
        // An assistant message with tool_calls carries both a visible
        // response AND function invocations. Split into two entries
        // so the content renders as an agent bubble and the tool call
        // renders as a separate card — matching SSE stream behaviour.
        if (m.role === 'assistant' && m.tool_calls?.length > 0) {
          if (m.content) {
            converted.push({
              id: `${sessionId}-${converted.length}`,
              role: 'agent',
              content: m.content,
              thoughtContent: m.reasoning_content || undefined,
              agent: m.name || undefined,
              timestamp: Date.now() - converted.length,
            })
          }
          for (const tc of m.tool_calls) {
            converted.push({
              id: `${sessionId}-${converted.length}`,
              role: 'tool_call',
              content: tc.function.arguments,
              toolCallId: tc.id,
              agent: m.name || undefined,
              toolCall: {
                id: tc.id,
                function: { name: tc.function.name, arguments: tc.function.arguments },
              },
              timestamp: Date.now() - converted.length,
            })
          }
          continue
        }
        const role = m.role === 'user' ? 'user' :
                     m.role === 'assistant' ? 'agent' :
                     m.role === 'tool' ? 'tool_result' : m.role
        converted.push({
          id: `${sessionId}-${converted.length}`,
          role: role as any,
          content: m.content || '',
          thoughtContent: m.reasoning_content || undefined,
          toolCallId: m.tool_call_id || undefined,
          agent: m.name || undefined,
          timestamp: Date.now() - converted.length,
        })
      }
      // Don't overwrite if a live SSE stream is active for this session.
      if (!streaming.value) {
        messages.value = converted
        // Update pane cache as well.
        const p = getPane(sessionId)
        p.messages = messages.value // reactive proxy, not raw
      }
    } catch { /* ignore */ }
  }

  /** Clear the current session's chat — abort stream, reset state. */
  function clearChat() {
    _abortController?.abort()
    _abortController = null
    messages.value = []
    streaming.value = false
    pendingApproval.value = null
    _approvalQueue.length = 0
    usage.value = null
    error.value = null
    _pendingToolCalls.clear()
    _currentStreamMsg = null
    _thinkingMsg = null
    _pendingThought = ''

    // Also clear the pane so a subsequent activateSession doesn't restore stale state.
    const sid = currentSessionId.value
    if (sid) {
      _panes.delete(sid)
    }
  }

  return {
    messages, streaming, pendingApproval, usage, error,
    selectedModelKey, selectedModelLabel, availableModels, contextWindow, messageCount,
    currentSessionId, currentSessionType,
    sendMessage, approveTool, clearChat, setStageHandler, fetchModels, fetchSessionDetail,
    fetchMessages, activateSession,
  }
})
