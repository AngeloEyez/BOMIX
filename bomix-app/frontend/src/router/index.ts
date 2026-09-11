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
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

// 依據路由 meta 設定頁面標題
router.beforeEach((to, _from, next) => {
  document.title = `${to.meta.title || 'BOMIX'} - BOMIX`
  next()
})

export default router

