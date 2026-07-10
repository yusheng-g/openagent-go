import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { PlanDef, StepState, SSEEvent } from '@/types'
import { connectSSE } from '@/api/sse'
import * as api from '@/api'

export const usePlanStore = defineStore('plan', () => {
  const planDef = ref<PlanDef | null>(null)
  const steps = ref<Record<string, StepState>>({})
  const executing = ref(false)
  const planError = ref<string | null>(null)
  const waitingRetry = ref<string | null>(null)
  const planDone = ref(false)

  let eventCleanup: (() => void) | null = null

  function initSteps(def: PlanDef) {
    const map: Record<string, StepState> = {}
    for (const step of def.steps) {
      map[step.id] = {
        status: 'pending',
        output: '',
        summary: '',
        toolCalls: [],
      }
    }
    steps.value = map
  }

  async function generatePlan(sessionId: string, goal: string, onThinking: (text: string) => void) {
    planError.value = null
    try {
      const def = await api.generatePlan(sessionId, goal, onThinking)
      planDef.value = def
      initSteps(def)
      planDone.value = false
    } catch (e: any) {
      planError.value = e.message
    }
  }

  async function executePlan(sessionId: string) {
    if (!planDef.value) return
    planError.value = null
    waitingRetry.value = null
    planDone.value = false

    // Subscribe to events BEFORE triggering execution
    eventCleanup?.()
    eventCleanup = connectSSE(
      `/plan/sessions/${sessionId}/events`,
      handlePlanEvent,
      (err) => console.error('plan SSE error:', err),
    )

    try {
      await api.executePlan(sessionId)
      executing.value = true
    } catch (e: any) {
      planError.value = e.message
      eventCleanup?.()
    }
  }

  function handlePlanEvent(event: SSEEvent) {
    switch (event.type) {
      case 'step_start': {
        if (event.step_id && steps.value[event.step_id]) {
          steps.value[event.step_id].status = 'running'
        }
        break
      }

      case 'step_text_delta': {
        if (event.step_id && steps.value[event.step_id]) {
          steps.value[event.step_id].output += event.text || ''
        }
        break
      }

      case 'step_tool_call': {
        if (event.step_id && steps.value[event.step_id]) {
          steps.value[event.step_id].toolCalls.push({
            name: event.tool_call?.function.name || 'unknown',
            args: event.tool_call?.function.arguments || '',
          })
        }
        break
      }

      case 'step_tool_progress': {
        if (event.step_id) {
          const step = steps.value[event.step_id]
          if (step) {
            step.output += event.text || ''
          }
        }
        break
      }

      case 'step_tool_result': {
        if (event.step_id) {
          const step = steps.value[event.step_id]
          if (step) {
            const lastCall = step.toolCalls[step.toolCalls.length - 1]
            if (lastCall) {
              lastCall.result = event.text || ''
            }
            step.output += `\nResult: ${event.text || ''}`
          }
        }
        break
      }

      case 'step_done': {
        if (event.step_id && steps.value[event.step_id]) {
          steps.value[event.step_id].status = 'done'
          steps.value[event.step_id].summary = event.text || ''
        }
        break
      }

      case 'step_failed': {
        if (event.step_id && steps.value[event.step_id]) {
          steps.value[event.step_id].status = 'failed'
          steps.value[event.step_id].error = event.error || ''
        }
        break
      }

      case 'plan_waiting_retry': {
        waitingRetry.value = event.step_id || null
        executing.value = false
        break
      }

      case 'plan_done': {
        executing.value = false
        planDone.value = true
        eventCleanup?.()
        break
      }

      case 'plan_error': {
        planError.value = event.error || 'Plan error'
        executing.value = false
        eventCleanup?.()
        break
      }

      case 'plan_cancelled': {
        executing.value = false
        eventCleanup?.()
        break
      }
    }
  }

  async function cancelExecution(sessionId: string) {
    try {
      await api.cancelPlan(sessionId)
      executing.value = false
      eventCleanup?.()
    } catch (e: any) {
      console.error('cancelPlan:', e)
    }
  }

  async function retryStep(sessionId: string, stepId: string) {
    try {
      await api.retryPlanStep(sessionId, stepId)
      waitingRetry.value = null
      executing.value = true
    } catch (e: any) {
      planError.value = e.message
    }
  }

  async function replan(sessionId: string, feedback: string) {
    try {
      await api.replan(sessionId, feedback)
      waitingRetry.value = null
      executing.value = true
    } catch (e: any) {
      planError.value = e.message
    }
  }

  function clearPlan() {
    eventCleanup?.()
    planDef.value = null
    steps.value = {}
    executing.value = false
    planError.value = null
    waitingRetry.value = null
    planDone.value = false
  }

  return {
    planDef, steps, executing, planError, waitingRetry, planDone,
    generatePlan, executePlan, cancelExecution, retryStep, replan, clearPlan,
  }
})
