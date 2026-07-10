import type { RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: '/single',
  },
  {
    path: '/single',
    name: 'single',
    component: () => import('@/views/SingleAgentView.vue'),
  },
  {
    path: '/single/:sessionId',
    name: 'single-session',
    component: () => import('@/views/SingleAgentView.vue'),
  },
  {
    path: '/team',
    name: 'team',
    component: () => import('@/views/TeamAgentView.vue'),
  },
  {
    path: '/team/:sessionId',
    name: 'team-session',
    component: () => import('@/views/TeamAgentView.vue'),
  },
  {
    path: '/plan',
    name: 'plan',
    component: () => import('@/views/PlanView.vue'),
  },
  {
    path: '/plan/:sessionId',
    name: 'plan-session',
    component: () => import('@/views/PlanView.vue'),
  },
]
