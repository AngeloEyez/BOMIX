/**
 * @file index.ts
 * @description BOMIX 前端路由設定檔，集中管理頁面路由與視窗標題切換邏輯
 */

import { createRouter, createWebHashHistory } from 'vue-router'

const routes = [
  {
    path: '/',
    name: 'home',
    component: () => import('../views/WelcomePage.vue'),
    meta: { title: 'Welcome' },
  },
  {
    path: '/workspace',
    name: 'workspace',
    component: () => import('../views/WorkspacePage.vue'),
    meta: { title: 'Workspace' },
  },
  {
    path: '/settings',
    name: 'settings',
    component: () => import('../views/SettingsPage.vue'),
    meta: { title: 'Settings' },
  },
  {
    path: '/ai-chat',
    name: 'ai-chat',
    component: () => import('../views/AIChatPage.vue'),
    meta: { title: 'AI Assistant' },
  },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

import { useAIChatStore } from '../stores/aiChat'

// 依據路由 meta 設定頁面標題與存取權限
router.beforeEach((to, _from, next) => {
  document.title = `${to.meta.title || 'BOMIX'} - BOMIX`

  if (to.name === 'ai-chat' || to.path === '/ai-chat') {
    const aiChatStore = useAIChatStore()
    if (!aiChatStore.isEnabled) {
      next({ path: '/workspace' })
      return
    }
  }

  next()
})

export default router

