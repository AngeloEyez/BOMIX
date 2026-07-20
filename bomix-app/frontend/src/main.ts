import { createApp } from 'vue'
import PrimeVue from 'primevue/config'
import { createRouter, createWebHashHistory } from 'vue-router'
import { createPinia } from 'pinia'
import App from './App.vue'
import './style.css'
import 'primeicons/primeicons.css'

// Import @primeuix/themes presets
import Aura from '@primeuix/themes/aura'

// Import views
import WelcomePage from './views/WelcomePage.vue'
import WorkspacePage from './views/WorkspacePage.vue'
import SettingsPage from './views/SettingsPage.vue'

// Create router instance
const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', name: 'home', component: WelcomePage },
    { path: '/workspace', name: 'workspace', component: WorkspacePage },
    { path: '/settings', name: 'settings', component: SettingsPage },
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
      darkModeSelector: '.app-dark',
      cssLayer: false,
    },
  },
})

app.use(router)
app.use(pinia)

app.mount('#app')
