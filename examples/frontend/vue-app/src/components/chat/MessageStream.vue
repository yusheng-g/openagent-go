<template>
  <div class="chat-view">
    <div class="msg-area" ref="scrollRef">
      <n-empty v-if="messages.length === 0" description="Send a message to get started" class="empty-state" />
      <template v-for="msg in messages" :key="msg.id">
        <!-- System -->
        <div v-if="msg.role === 'system'" class="sys-msg">{{ msg.content }}</div>

        <!-- Thought (deprecated standalone — now inline in agent bubble) -->
        <n-collapse v-else-if="msg.role === 'thought'" class="msg-thought">
          <n-collapse-item title="Thinking...">
            <MarkdownContent :content="msg.content" />
          </n-collapse-item>
        </n-collapse>

        <!-- Tool call -->
        <n-collapse v-else-if="msg.role === 'tool_call'" class="msg-tool">
          <n-collapse-item :title="toolTitle(msg)">
            <pre class="tool-body">{{ toolArgs(msg) }}</pre>
          </n-collapse-item>
        </n-collapse>

        <!-- Tool result -->
        <n-collapse v-else-if="msg.role === 'tool_result'" class="msg-tool">
          <n-collapse-item :title="toolResultTitle(msg)">
            <div class="tool-body"><MarkdownContent :content="truncate(msg.content)" /></div>
          </n-collapse-item>
        </n-collapse>

        <!-- Agent -->
        <div v-else-if="msg.role === 'agent'" class="msg-agent">
          <div v-if="msg.agent" class="agent-label">{{ msg.agent }}</div>
          <div v-if="msg.thoughtContent" class="thought-inline">
            <n-collapse>
              <n-collapse-item title="Thinking...">
                <div class="thought-text">{{ msg.thoughtContent }}</div>
              </n-collapse-item>
            </n-collapse>
          </div>
          <div class="agent-body">
            <MarkdownContent :content="msg.content" />
            <span v-if="msg.isStreaming" class="cursor">▌</span>
          </div>
        </div>

        <!-- User -->
        <div v-else-if="msg.role === 'user'" class="msg-user">
          <div class="user-body">{{ msg.content }}</div>
        </div>
      </template>

      <div v-if="error" class="error-msg">{{ error }}</div>
    </div>

    <UsageBar :usage="usage" />

    <div class="input-area">
      <n-input
        v-model:value="inputText"
        type="textarea"
        :autosize="{ minRows: 1, maxRows: 5 }"
        placeholder="Type a message... (Enter to send, Shift+Enter for newline)"
        :disabled="disabled"
        @keydown="onKeydown"
      />
      <n-button type="primary" :disabled="!inputText.trim() || disabled" @click="send" class="send-btn">Send</n-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick, watch } from 'vue'
import { NEmpty, NCollapse, NCollapseItem, NInput, NButton } from 'naive-ui'
import type { ChatMessage, UsageInfo } from '@/types'
import MarkdownContent from '@/components/common/MarkdownContent.vue'
import UsageBar from '@/components/chat/UsageBar.vue'

const props = defineProps<{
  messages: ChatMessage[]
  usage: UsageInfo | null
  error: string | null
  disabled?: boolean
}>()

const emit = defineEmits<{ send: [text: string] }>()

const inputText = ref('')
const scrollRef = ref<HTMLElement | null>(null)

watch(() => props.messages.length, () => {
  nextTick(() => {
    if (scrollRef.value) scrollRef.value.scrollTop = scrollRef.value.scrollHeight
  })
})

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); send() }
}
function send() {
  const t = inputText.value.trim()
  if (!t || props.disabled) return
  emit('send', t)
  inputText.value = ''
}

function toolTitle(m: ChatMessage): string {
  const tc = m.toolCall
  if (!tc) return 'Tool call'
  try {
    const a = JSON.parse(tc.function.arguments)
    const p = Object.entries(a).slice(0,2).map(([k,v]) => `${k}=${JSON.stringify(v)}`).join(', ')
    return `${tc.function.name}(${p})`
  } catch { return tc.function.name }
}
function toolArgs(m: ChatMessage): string {
  if (!m.toolCall) return ''
  try { return JSON.stringify(JSON.parse(m.toolCall.function.arguments), null, 2) } catch { return m.toolCall.function.arguments }
}
function toolResultTitle(m: ChatMessage): string {
  return m.toolCall ? `${m.toolCall.function.name} result` : 'Tool result'
}
function truncate(s: string): string {
  return s.length > 10000 ? s.slice(0, 10000) + '\n\n... (truncated)' : s
}
</script>

<style scoped>
.chat-view {
  height: 100%; display: flex; flex-direction: column; overflow: hidden;
}

.msg-area {
  flex: 1; overflow-y: auto; padding: 16px 0;
  display: flex; flex-direction: column; gap: 6px;
}
.empty-state { margin-top: 80px; }

.sys-msg {
  text-align: center; font-size: 0.7em; opacity: 0.3;
  padding: 2px 12px;
}

.thought-inline { margin-bottom: 8px; opacity: 0.6; font-size: 0.85em; }
.thought-text { font-size: 0.9em; white-space: pre-wrap; word-break: break-word; max-height: 200px; overflow-y: auto; }
.msg-thought, .msg-tool { margin: 4px 16px; }
.tool-body {
  font-size: 0.82em; white-space: pre-wrap; word-break: break-word;
  background: rgba(255,255,255,0.04); padding: 12px; border-radius: 6px;
  max-height: 350px; overflow-y: auto;
}

.msg-agent { padding: 4px 16px; max-width: 85%; align-self: flex-start; }
.agent-label {
  font-size: 0.72em; font-weight: 700; opacity: 0.55; margin-bottom: 2px;
  text-transform: uppercase; letter-spacing: 0.04em;
}
.agent-body { font-size: 0.92em; line-height: 1.6; }
.cursor { animation: blink 1s step-end infinite; }
@keyframes blink { 50% { opacity: 0; } }

.msg-user { padding: 4px 16px; max-width: 75%; align-self: flex-end; }
.user-body {
  background: #2563eb; color: #fff;
  padding: 10px 16px; border-radius: 18px 18px 4px 18px;
  font-size: 0.92em; line-height: 1.5;
}

.input-area {
  display: flex; gap: 8px; padding: 12px 16px;
  border-top: 1px solid rgba(255,255,255,0.08);
  align-items: flex-end; flex-shrink: 0;
}
.send-btn { flex-shrink: 0; }

.error-msg {
  background: rgba(239, 68, 68, 0.12); color: #ef4444;
  padding: 10px 16px; margin: 4px 16px; border-radius: 6px; font-size: 0.85em;
}
</style>
