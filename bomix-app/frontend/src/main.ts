import { createApp } from 'vue'
import PrimeVue from 'primevue/config'
import { createRouter, createWebHashHistory } from 'vue-router'
import { createPinia } from 'pinia'
import App from './App.vue'
import './style.css'

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

// Use PrimeVue with Aura theme
app.use(PrimeVue, {
  theme: {
    preset: 'aura',
    options: {
      darkModeSelector: false,
    },
  },
})

app.use(router)
app.use(pinia)

app.mount('#app')
