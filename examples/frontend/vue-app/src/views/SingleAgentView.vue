<template>
  <MessageStream
    :messages="chat.messages"
    :usage="chat.usage"
    :error="chat.error"
    :disabled="chat.streaming || !sessionId"
    @send="handleSend"
  />
  <ToolApprovalDialog
    v-if="chat.pendingApproval"
    :tool-call="chat.pendingApproval.toolCall"
    @resolve="handleApprove"
  />
</template>

<script setup lang="ts">
import { computed, watch, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import { useChatStore } from '@/stores/chat'
import { useSessionsStore } from '@/stores/sessions'
import MessageStream from '@/components/chat/MessageStream.vue'
import ToolApprovalDialog from '@/components/chat/ToolApprovalDialog.vue'

const route = useRoute()
const chat = useChatStore()
const sessions = useSessionsStore()

const sessionId = computed(() => (route.params.sessionId as string) || sessions.currentSessionId)

watch(sessionId, () => { chat.clearChat() })
onBeforeUnmount(() => { chat.clearChat() })

function handleSend(text: string) {
  const sid = sessionId.value
  if (!sid) return
  chat.sendMessage(sid, text, 'single')
}
function handleApprove(allowed: boolean) {
  const sid = sessionId.value
  if (!sid) return
  chat.approveTool(sid, 'single', allowed)
}
</script>
