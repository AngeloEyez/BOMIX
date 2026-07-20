import { createApp } from 'vue'
import PrimeVue from 'primevue/config'
import { createRouter, createWebHashHistory } from 'vue-router'
import { createPinia } from 'pinia'
import App from './App.vue'
import './style.css'

// Import @primeuix/themes presets
import Aura from '@primeuix/themes/aura'

// Create router instance
const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', name: 'home', component: { template: '<div>Home</div>' } },
    { path: '/workspace', name: 'workspace', component: { template: '<div>Workspace</div>' } },
    { path: '/settings', name: 'settings', component: { template: '<div>Settings</div>' } },
  ],
})

// Create Pinia instance
const pinia = createPinia()

// Create and mount app
const app = createApp(App)

// Use PrimeVue with @primeuix/themes
// @primeuix/themes v3 provides theme presets like Aura, Lara, Material, etc.
app.use(PrimeVue, {
  theme: {
    preset: Aura,
    options: {
      darkModeSelector: false,
    },
  },
})

app.use(router)
app.use(pinia)

app.mount('#app')
