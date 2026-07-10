<template>
  <div class="usage-bar" v-if="usage && usage.contextWindow">
    <n-progress
      type="line"
      :percentage="pct"
      :color="color"
      :height="6"
      :border-radius="3"
      :show-indicator="false"
      style="flex:1"
    />
    <span class="usage-text">{{ usage.promptTokens.toLocaleString() }} / {{ usage.contextWindow.toLocaleString() }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NProgress } from 'naive-ui'
import type { UsageInfo } from '@/types'

const props = defineProps<{ usage: UsageInfo | null }>()

const pct = computed(() => {
  if (!props.usage?.contextWindow) return 0
  return Math.round((props.usage.promptTokens / props.usage.contextWindow) * 100)
})
const color = computed(() => pct.value > 90 ? '#ef4444' : pct.value > 70 ? '#f59e0b' : '#22c55e')
</script>

<style scoped>
.usage-bar { display: flex; align-items: center; gap: 10px; padding: 8px 16px; border-top: 1px solid rgba(255,255,255,0.08); }
.usage-text { font-size: 0.78em; opacity: 0.55; white-space: nowrap; font-variant-numeric: tabular-nums; }
</style>
