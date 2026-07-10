<template>
  <div class="stat">
    <div class="stat-title">Monitor</div>
    <div class="stat-list">
      <div
        v-for="s in stageList"
        :key="s.key"
        class="stat-row"
        :class="s.status"
      >
        <span class="stat-dot" :style="{ background: s.color }"></span>
        <span class="stat-label">{{ s.label }}</span>
        <span v-if="s.detail" class="stat-extra">{{ s.detail }}</span>
        <span class="stat-time">{{ s.duration }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'

const STAGES = [
  'memory.fetch', 'guard.in', 'prompt.build',
  'model.call', 'guard.out', 'tool.execute', 'memory.append',
]
const LABELS: Record<string, string> = {
  'memory.fetch': 'Fetch', 'guard.in': 'Guard', 'prompt.build': 'Prompt',
  'model.call': 'Model', 'guard.out': 'Check', 'tool.execute': 'Tools',
  'memory.append': 'Save',
}

const COLORS: Record<string, string> = {
  pending: 'rgba(255,255,255,0.14)',
  active: '#818cf8',
  done: '#34d399',
  error: '#f87171',
}

interface StageState { status: string; duration: string; detail: string }
const stages = ref<Map<string, StageState>>(new Map())

function handleStage(data: any) {
  const key = data.name as string
  const phase = data.phase as string
  if (phase === 'enter') {
    let detail = ''
    if (data.detail?.tool) detail = data.detail.tool as string
    stages.value.set(key, { status: 'active', duration: '', detail })
    stages.value = new Map(stages.value)
  } else {
    const ms = data.duration_ms as number
    const err = data.err as string | undefined
    stages.value.set(key, {
      status: err ? 'error' : 'done',
      duration: ms != null ? `${ms}ms` : '',
      detail: stages.value.get(key)?.detail || '',
    })
    stages.value = new Map(stages.value)
  }
}

function reset() { stages.value = new Map() }
defineExpose({ handleStage, reset })

const stageList = computed(() =>
  STAGES.map(k => {
    const s = stages.value.get(k)
    return {
      key: k, label: LABELS[k] || k,
      status: s?.status || 'pending',
      duration: s?.duration || '',
      detail: s?.detail || '',
      color: COLORS[s?.status || 'pending'],
    }
  })
)
</script>

<style scoped>
.stat {
  margin: 8px 10px 10px;
  border-radius: 10px;
  background: rgba(255,255,255,0.025);
  border: 1px solid rgba(255,255,255,0.06);
  flex-shrink: 0;
  user-select: none;
}
.stat-title {
  padding: 10px 14px 4px;
  font-size: 0.65em;
  font-weight: 600;
  opacity: 0.4;
  letter-spacing: 0.07em;
  text-transform: uppercase;
}
.stat-list { padding: 2px 14px 10px; display: flex; flex-direction: column; gap: 1px; }
.stat-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 3px 6px;
  border-radius: 5px;
  font-size: 0.73em;
  transition: background 0.2s, opacity 0.2s;
}
.stat-row.pending { opacity: 0.35; }
.stat-row.active {
  opacity: 1;
  background: rgba(129,140,248,0.1);
}
.stat-row.done { opacity: 0.6; }
.stat-row.error { opacity: 0.85; }
.stat-dot {
  width: 7px; height: 7px;
  border-radius: 50%;
  flex-shrink: 0;
  transition: background 0.3s;
}
.stat-label { flex: 1; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.stat-extra {
  font-size: 0.85em;
  opacity: 0.35;
  max-width: 70px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.stat-row.active .stat-extra { opacity: 0.6; }
.stat-time {
  font-size: 0.85em;
  opacity: 0.4;
  flex-shrink: 0;
  font-variant-numeric: tabular-nums;
  min-width: 34px;
  text-align: right;
}
</style>
