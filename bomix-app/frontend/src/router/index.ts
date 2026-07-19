import { createRouter, createWebHashHistory } from 'vue-router'

const routes = [
  {
    path: '/',
    name: 'welcome',
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

// Set page title
router.beforeEach((to, _from, next) => {
  document.title = `${to.meta.title || 'BOMIX'} - BOMIX`
  next()
})

export default router
