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

// 註冊 PrimeVue Tooltip 指令，並加入文字溢出截斷 (Ellipsis) 偵測邏輯
// 僅當儲存格文字超出寬度被截斷時，才啟動 Tooltip 顯示完整內容
import Tooltip from 'primevue/tooltip'

const EllipsisTooltip = {
  ...Tooltip,
  beforeMount(el: any, binding: any, vnode: any, prevVnode: any) {
    if (typeof (Tooltip as any).beforeMount === 'function') {
      (Tooltip as any).beforeMount(el, binding, vnode, prevVnode)
    }
    el._checkEllipsis = () => {
      const targetEl = el.querySelector?.('.cell-text') || el
      // 容差 1px 避免瀏覽器次像素 (subpixel) 誤差
      const isOverflow = targetEl.scrollWidth > targetEl.clientWidth + 1
      el.$_ptooltipDisabled = !isOverflow
    }
    el.addEventListener('mouseenter', el._checkEllipsis, true)
  },
  updated(el: any, binding: any, vnode: any, prevVnode: any) {
    if (typeof (Tooltip as any).updated === 'function') {
      (Tooltip as any).updated(el, binding, vnode, prevVnode)
    }
  },
  unmounted(el: any, binding: any, vnode: any, prevVnode: any) {
    if (el._checkEllipsis) {
      el.removeEventListener('mouseenter', el._checkEllipsis, true)
      delete el._checkEllipsis
    }
    if (typeof (Tooltip as any).unmounted === 'function') {
      (Tooltip as any).unmounted(el, binding, vnode, prevVnode)
    }
  }
}

app.directive('tooltip', EllipsisTooltip)

app.mount('#app')
