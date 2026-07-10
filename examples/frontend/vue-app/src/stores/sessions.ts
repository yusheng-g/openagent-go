import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { SessionInfo } from '@/types'
import * as api from '@/api'

export type AgentMode = 'single' | 'team' | 'plan'

export const useSessionsStore = defineStore('sessions', () => {
  const sessions = ref<SessionInfo[]>([])
  const currentSessionId = ref<string | null>(null)
  const loading = ref(false)

  const currentSession = computed(() =>
    sessions.value.find(s => s.id === currentSessionId.value) ?? null,
  )

  async function fetchSessions(mode: AgentMode) {
    loading.value = true
    try {
      switch (mode) {
        case 'single': sessions.value = await api.listSessions(); break
        case 'team': sessions.value = await api.listTeamSessions(); break
        case 'plan': sessions.value = await api.listPlanSessions(); break
      }
    } catch (e) {
      console.error('fetchSessions:', e)
      sessions.value = []
    } finally {
      loading.value = false
    }
  }

  async function createSession(mode: AgentMode, title?: string) {
    try {
      let info: SessionInfo
      switch (mode) {
        case 'single': info = await api.createSession({ title }); break
        case 'team': info = await api.createTeamSession({ title }); break
        case 'plan': info = await api.createPlanSession({ title }); break
      }
      sessions.value.unshift(info)
      currentSessionId.value = info.id
      return info
    } catch (e) {
      console.error('createSession:', e)
      throw e
    }
  }

  async function deleteSession(id: string, mode: AgentMode) {
    try {
      switch (mode) {
        case 'single': await api.deleteSession(id); break
        case 'team': await api.deleteTeamSession(id); break
        case 'plan': await api.deletePlanSession(id); break
      }
      sessions.value = sessions.value.filter(s => s.id !== id)
      if (currentSessionId.value === id) {
        currentSessionId.value = null
      }
    } catch (e) {
      console.error('deleteSession:', e)
    }
  }

  function selectSession(id: string) {
    currentSessionId.value = id
  }

  return {
    sessions, currentSessionId, currentSession, loading,
    fetchSessions, createSession, deleteSession, selectSession,
  }
})
