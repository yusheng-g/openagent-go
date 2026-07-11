<template>
  <div class="dag" ref="dagRef">
    <svg v-if="arrows.length > 0"
      class="dag-svg"
      :style="{ width: boxW + 'px', height: boxH + 'px' }"
      :viewBox="`0 0 ${boxW} ${boxH}`"
    >
      <defs>
        <marker id="dag-arr" markerWidth="8" markerHeight="6" refX="8" refY="3" orient="auto">
          <polygon points="0 0, 8 3, 0 6" fill="#6b7280" />
        </marker>
      </defs>
      <path v-for="a in arrows" :key="a.key"
        :d="a.d"
        :stroke="arrowColor(a.status)"
        stroke-width="1.5"
        fill="none"
        :opacity="a.active ? 0.7 : 0.25"
        marker-end="url(#dag-arr)" />
    </svg>

    <div class="dag-cols" ref="colsRef">
      <div v-for="(col, ci) in columns" :key="ci" class="dag-col">
        <div
          v-for="step in col" :key="step.id"
          :ref="el => nodeRefs[step.id] = (el as HTMLElement | null)"
          class="dag-node"
          :class="[stepState(step.id).status, { selected: selectedId === step.id }]"
          @click="selectedId = selectedId === step.id ? null : step.id"
        >
          <div class="node-head">
            <span class="node-dot" :class="stepState(step.id).status" />
            <span class="node-agent">{{ step.agent }}</span>
            <n-tag v-if="step.final" type="warning" size="tiny" :bordered="false">FINAL</n-tag>
            <n-tag :type="statusTag(stepState(step.id).status)" size="tiny" :bordered="false">
              {{ stepState(step.id).status }}
            </n-tag>
          </div>
          <div class="node-task">{{ step.task }}</div>

          <div v-if="stepState(step.id).status === 'running' && stepState(step.id).output" class="node-glimpse">
            <pre class="glimpse-text">{{ stepState(step.id).output.slice(-300) }}</pre>
          </div>
          <div v-else-if="stepState(step.id).status === 'done' && stepState(step.id).summary" class="node-glimpse done">
            {{ stepState(step.id).summary }}
          </div>
          <div v-else-if="stepState(step.id).status === 'failed' && stepState(step.id).error" class="node-glimpse err">
            {{ stepState(step.id).error }}
          </div>

          <div v-if="step.depends_on?.length" class="node-deps">
            ← {{ step.depends_on.join(', ') }}
          </div>
        </div>
      </div>
    </div>

    <!-- Modal overlay -->
    <Teleport to="body">
      <div v-if="selectedId && selectedStep" class="dag-modal-mask" @click.self="selectedId = null">
        <div class="dag-modal">
          <div class="modal-head">
            <span class="modal-id">{{ selectedId }}</span>
            <span class="modal-agent">{{ selectedStep.agent }} · {{ stepState(selectedId).status }}</span>
            <n-button size="tiny" quaternary @click="selectedId = null">✕</n-button>
          </div>
          <div class="modal-body">
            <div class="modal-task-text">{{ selectedStep.task }}</div>

            <n-collapse :default-expanded-names="['output']">
              <n-collapse-item name="output" title="Output">
                <pre class="modal-output">{{ stepState(selectedId).output || '(no output yet)' }}</pre>
              </n-collapse-item>
              <n-collapse-item v-if="stepState(selectedId).summary" name="summary" title="Summary">
                <div class="modal-summary">{{ stepState(selectedId).summary }}</div>
              </n-collapse-item>
              <n-collapse-item v-if="stepState(selectedId).error" name="error" title="Error">
                <div class="modal-error">{{ stepState(selectedId).error }}</div>
              </n-collapse-item>
              <n-collapse-item v-if="stepState(selectedId).toolCalls.length" name="tools" :title="'Tool calls (' + stepState(selectedId).toolCalls.length + ')'">
                <div v-for="(tc, i) in stepState(selectedId).toolCalls" :key="i" class="modal-tool">
                  <span class="mt-name">{{ tc.name }}</span>
                  <pre class="mt-args">{{ tc.args }}</pre>
                  <pre v-if="tc.result" class="mt-result">{{ tc.result }}</pre>
                </div>
              </n-collapse-item>
            </n-collapse>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { NTag, NButton, NCollapse, NCollapseItem } from 'naive-ui'
import type { StepDef, StepState, StepStatus } from '@/types'

const props = defineProps<{
  steps: StepDef[]
  stepState: (id: string) => StepState
}>()

const selectedId = ref<string | null>(null)
const dagRef = ref<HTMLElement | null>(null)
const colsRef = ref<HTMLElement | null>(null)
const nodeRefs: Record<string, HTMLElement | null> = {}
const boxW = ref(0)
const boxH = ref(0)

interface Arrow { key: string; d: string; status: StepStatus; active: boolean }

const selectedStep = computed(() => {
  if (!selectedId.value) return null
  return props.steps.find(s => s.id === selectedId.value) || null
})

// ── Topological layering ──

const layers = computed(() => {
  const layer = new Map<string, number>()
  const inDeg = new Map<string, number>()
  const depTo = new Map<string, string[]>()

  for (const s of props.steps) {
    if (!inDeg.has(s.id)) inDeg.set(s.id, 0)
    for (const d of s.depends_on || []) {
      if (!depTo.has(d)) depTo.set(d, [])
      depTo.get(d)!.push(s.id)
      inDeg.set(s.id, (inDeg.get(s.id) || 0) + 1)
    }
  }

  const queue: string[] = []
  for (const s of props.steps) {
    if ((inDeg.get(s.id) || 0) === 0) queue.push(s.id)
  }

  const remaining = new Map(inDeg)
  while (queue.length > 0) {
    const batch = [...queue]
    queue.length = 0
    for (const id of batch) {
      for (const next of depTo.get(id) || []) {
        layer.set(next, Math.max(layer.get(next) || 0, (layer.get(id) || 0) + 1))
        remaining.set(next, (remaining.get(next) || 1) - 1)
        if (remaining.get(next) === 0) queue.push(next)
      }
    }
  }

  for (const s of props.steps) {
    if (!layer.has(s.id)) layer.set(s.id, 0)
  }
  return layer
})

const columns = computed(() => {
  const cols = new Map<number, StepDef[]>()
  for (const s of props.steps) {
    const l = layers.value.get(s.id) || 0
    if (!cols.has(l)) cols.set(l, [])
    cols.get(l)!.push(s)
  }
  return [...cols.entries()].sort((a, b) => a[0] - b[0]).map(e => e[1])
})

// ── SVG arrows ──

function computeArrows(): Arrow[] {
  if (!dagRef.value) return []
  const dagRect = dagRef.value.getBoundingClientRect()
  boxW.value = dagRect.width
  boxH.value = dagRect.height

  const arrows: Arrow[] = []
  for (const step of props.steps) {
    const deps = step.depends_on || []
    if (deps.length === 0) continue
    const toEl = nodeRefs[step.id]
    if (!toEl) continue
    const toRect = toEl.getBoundingClientRect()
    const toX = toRect.left - dagRect.left
    const toY = toRect.top - dagRect.top + toRect.height / 2

    for (const depId of deps) {
      const fromEl = nodeRefs[depId]
      if (!fromEl) continue
      const fromRect = fromEl.getBoundingClientRect()
      const fromX = fromRect.right - dagRect.left
      const fromY = fromRect.top - dagRect.top + fromRect.height / 2

      const dx = toX - fromX
      const cp = Math.max(Math.abs(dx) * 0.55, 24)
      const d = `M ${fromX},${fromY} C ${fromX + cp},${fromY} ${toX - cp},${toY} ${toX},${toY}`

      arrows.push({
        key: `${depId}→${step.id}`,
        d,
        status: props.stepState(depId).status,
        active: props.stepState(depId).status === 'done',
      })
    }
  }
  return arrows
}

const arrows = ref<Arrow[]>([])

function recalc() {
  requestAnimationFrame(() => {
    requestAnimationFrame(() => { arrows.value = computeArrows() })
  })
}

let observer: ResizeObserver | null = null

onMounted(() => {
  recalc()
  if (dagRef.value) {
    observer = new ResizeObserver(recalc)
    observer.observe(dagRef.value)
  }
})

onBeforeUnmount(() => observer?.disconnect())

watch(() => props.steps.map(s => {
  const st = props.stepState(s.id)
  return `${st.status}:${st.output.length}:${st.summary.length}`
}), recalc, { deep: false })

function statusTag(s: StepStatus): 'default' | 'info' | 'warning' | 'error' | 'success' {
  switch (s) {
    case 'running': return 'warning'
    case 'done': return 'success'
    case 'failed': return 'error'
    default: return 'default'
  }
}

function arrowColor(s: StepStatus): string {
  switch (s) {
    case 'done': return '#22c55e'
    case 'running': return '#f59e0b'
    case 'failed': return '#ef4444'
    default: return '#6b7280'
  }
}
</script>

<style scoped>
.dag { position: relative; padding: 20px 0 40px; min-height: 200px; }

.dag-svg { position: absolute; top: 0; left: 0; pointer-events: none; z-index: 0; }

.dag-cols {
  display: flex; gap: 40px;
  position: relative; z-index: 1; padding: 0 20px;
}
.dag-col {
  display: flex; flex-direction: column; gap: 14px;
  min-width: 200px; max-width: 260px; flex: 0 0 auto;
}

/* ── Node card ── */

.dag-node {
  background: rgba(255,255,255,0.03);
  border: 1px solid rgba(255,255,255,0.08);
  border-left: 3px solid transparent;
  border-radius: 10px;
  padding: 12px 14px;
  cursor: pointer;
  transition: border-color 0.2s, background 0.2s;
}
.dag-node:hover { background: rgba(255,255,255,0.06); border-color: rgba(255,255,255,0.14); }
.dag-node.selected { border-color: rgba(255,255,255,0.22); }

.dag-node.pending  { border-left-color: rgba(255,255,255,0.12); }
.dag-node.running  { border-left-color: #f59e0b; }
.dag-node.done     { border-left-color: #22c55e; }
.dag-node.failed   { border-left-color: #ef4444; }

.dag-node.running { background: rgba(245,158,11,0.06); border-color: rgba(245,158,11,0.2); }

.node-head { display: flex; align-items: center; gap: 8px; margin-bottom: 4px; }

.node-dot { width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0; }
.node-dot.pending  { background: rgba(255,255,255,0.12); }
.node-dot.running  { background: #f59e0b; animation: pulse 1.2s ease-in-out infinite; }
.node-dot.done     { background: #22c55e; }
.node-dot.failed   { background: #ef4444; }

@keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.35; } }

.node-agent {
  font-size: 0.68em; font-weight: 700; opacity: 0.5;
  text-transform: uppercase; letter-spacing: 0.04em; flex: 1;
}
.node-task { font-size: 0.84em; line-height: 1.4; color: rgba(255,255,255,0.7); }

/* Glimpse */
.node-glimpse { margin-top: 6px; }
.node-glimpse.err { color: #f87171; font-size: 0.78em; }
.glimpse-text {
  font-size: 0.73em; white-space: pre-wrap; word-break: break-word;
  max-height: 100px; overflow: hidden;
  color: rgba(255,255,255,0.4); line-height: 1.35; opacity: 0.7;
}
.node-glimpse.done {
  font-size: 0.78em; color: rgba(255,255,255,0.4); line-height: 1.4;
  display: -webkit-box; -webkit-line-clamp: 3; -webkit-box-orient: vertical; overflow: hidden;
}
.node-deps {
  margin-top: 6px; padding-top: 4px;
  border-top: 1px solid rgba(255,255,255,0.05);
  font-size: 0.65em; opacity: 0.22;
}
</style>

<!-- Modal styles — NOT scoped because <Teleport to="body"> moves elements
     outside the component tree, where scoped data attributes are lost. -->
<style>
.dag-modal-mask {
  position: fixed; inset: 0; z-index: 2000;
  background: rgba(0,0,0,0.5);
  display: flex; align-items: center; justify-content: center;
}
.dag-modal {
  background: var(--n-color, #1e1e24);
  border: 1px solid rgba(255,255,255,0.1);
  border-radius: 14px;
  width: min(640px, 90vw);
  max-height: 80vh;
  display: flex; flex-direction: column;
  box-shadow: 0 8px 48px rgba(0,0,0,0.6);
}

.modal-head {
  display: flex; align-items: center; gap: 10px;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(255,255,255,0.08);
  flex-shrink: 0;
}
.modal-id {
  font-size: 0.78em; font-weight: 700;
  opacity: 0.65; text-transform: uppercase; letter-spacing: 0.04em;
}
.modal-agent { font-size: 0.72em; opacity: 0.4; flex: 1; }

.modal-body { padding: 20px; overflow-y: auto; flex: 1; }

.modal-task-text {
  font-size: 0.88em; line-height: 1.5; margin-bottom: 16px;
  color: rgba(255,255,255,0.65);
  padding: 12px; background: rgba(255,255,255,0.04); border-radius: 8px;
}

.modal-output {
  font-size: 0.82em; white-space: pre-wrap; word-break: break-word;
  max-height: 400px; overflow-y: auto;
  padding: 0; margin: 0;
  line-height: 1.55; color: rgba(255,255,255,0.6);
}
.modal-summary { font-size: 0.86em; color: rgba(255,255,255,0.55); line-height: 1.5; }
.modal-error  { font-size: 0.86em; color: #f87171; }

.modal-tool { margin: 8px 0; }
.mt-name {
  font-size: 0.68em; font-weight: 600; opacity: 0.5;
  text-transform: uppercase; letter-spacing: 0.03em;
}
.mt-args {
  font-size: 0.74em; white-space: pre-wrap; word-break: break-word;
  margin-top: 4px; color: rgba(255,255,255,0.35);
  max-height: 100px; overflow-y: auto;
}
.mt-result {
  font-size: 0.74em; white-space: pre-wrap; word-break: break-word;
  margin-top: 4px; color: rgba(255,255,255,0.45);
  max-height: 200px; overflow-y: auto;
}
</style>
