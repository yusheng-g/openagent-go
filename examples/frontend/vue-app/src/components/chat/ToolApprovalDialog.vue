<template>
  <n-modal :show="true" :mask-closable="false" title="Tool Approval">
    <div class="approval-body">
      <div class="tool-name">{{ toolCall.function.name }}</div>
      <pre class="tool-args">{{ formattedArgs }}</pre>

      <div class="feedback-row">
        <n-input
          v-model:value="feedback"
          type="textarea"
          :autosize="{ minRows: 1, maxRows: 3 }"
          placeholder="Reason for denying (optional — the model will see this)"
        />
      </div>

      <div class="approval-actions">
        <n-button type="error" @click="handleResolve(false)">Deny</n-button>
        <n-button type="primary" @click="handleResolve(true)">Allow</n-button>
      </div>
    </div>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { NModal, NButton, NInput } from 'naive-ui'
import type { SSEToolCall } from '@/types'

const props = defineProps<{ toolCall: SSEToolCall }>()
const emit = defineEmits<{ resolve: [allowed: boolean, feedback?: string] }>()

const feedback = ref('')

function handleResolve(allowed: boolean) {
  const text = (feedback.value || '').trim()
  emit('resolve', allowed, text || undefined)
}

const formattedArgs = computed(() => {
  try {
    return JSON.stringify(JSON.parse(props.toolCall.function.arguments), null, 2)
  } catch {
    return props.toolCall.function.arguments
  }
})
</script>

<style scoped>
.approval-body {
  padding: 16px;
  min-width: 420px;
}

.tool-name {
  font-size: 1.1em;
  font-weight: 600;
  margin-bottom: 12px;
}

.tool-args {
  background: rgba(0,0,0,0.15);
  padding: 12px;
  border-radius: 6px;
  max-height: 300px;
  overflow-y: auto;
  font-size: 0.85em;
  white-space: pre-wrap;
  word-break: break-word;
}

.feedback-row {
  margin-top: 12px;
}

.approval-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 16px;
}
</style>
