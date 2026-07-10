<template>
  <n-modal :show="true" :mask-closable="false" title="Tool Approval">
    <div class="approval-body">
      <div class="tool-name">{{ toolCall.function.name }}</div>
      <pre class="tool-args">{{ formattedArgs }}</pre>
      <div class="approval-actions">
        <n-button type="error" @click="$emit('resolve', false)">Deny</n-button>
        <n-button type="primary" @click="$emit('resolve', true)">Allow</n-button>
      </div>
    </div>
  </n-modal>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NModal, NButton } from 'naive-ui'
import type { SSEToolCall } from '@/types'

const props = defineProps<{ toolCall: SSEToolCall }>()
defineEmits<{ resolve: [allowed: boolean] }>()

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
  min-width: 400px;
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
.approval-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 16px;
}
</style>
