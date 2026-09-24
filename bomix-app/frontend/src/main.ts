/**
 * 全域視窗拖曳與縮放安全防護守衛 (Window Drag & Resize Guard)
 * 徹底解決滑鼠快速進出視窗 (特別是左上角) 時，Wails v3 drag.js 因狀態殘留或按鍵計數推算錯誤，
 * 在無按下滑鼠左鍵時送出 wails:resize / wails:drag 觸發 Windows 原生 WM_NCLBUTTONDOWN，
 * 導致視窗「非自願跟著鼠標移動/調整大小」的 Windows 桌面應用嚴重 Bug。
 */
let realMouseButtons = 0
let lastMouseX = 0
let lastMouseY = 0

if (typeof window !== 'undefined') {
  ;(window as any)._wails = (window as any)._wails || {}
  ;(window as any)._wails.flags = (window as any)._wails.flags || {}
  // 將頂部/上下邊界感應高度設為 3px，大幅降低滑過標題列時的誤觸率
  ;(window as any)._wails.flags['system.resizeHandleHeight'] = 3
  // 保持左右邊界為標準 5px，配合全域右側 5px 安全感應帶
  ;(window as any)._wails.flags['system.resizeHandleWidth'] = 5
  // 角落額外感應縮減為 3px，避免吃掉標題列上方按鈕區
  ;(window as any)._wails.flags['resizeCornerExtra'] = 3

  // 1. 在捕獲階段 (Capture Phase) 最早追蹤真實硬體滑鼠按鍵狀態與坐標
  const updatePointerState = (e: MouseEvent) => {
    realMouseButtons = e.buttons
    lastMouseX = e.clientX
    lastMouseY = e.clientY
  }
  window.addEventListener('mousedown', updatePointerState, { capture: true })
  window.addEventListener('mouseup', updatePointerState, { capture: true })
  window.addEventListener('mousemove', updatePointerState, { capture: true })
  window.addEventListener('mouseleave', () => { realMouseButtons = 0 }, { capture: true })
  window.addEventListener('blur', () => { realMouseButtons = 0 })

  // 2. 攔截底層 WebView2 postMessage，堅決阻斷無滑鼠左鍵時的幽靈縮放/拖曳指令
  const guardWebview = (webview: any) => {
    if (!webview || webview.__bomixGuarded) return
    const originalPostMessage = webview.postMessage.bind(webview)
    webview.postMessage = function (message: any) {
      if (typeof message === 'string') {
        if (message.startsWith('wails:resize:') || message === 'wails:drag') {
          // 防線 1：滑鼠左鍵未按下 (buttons & 1 === 0)，絕對禁止向 Windows 發送 WM_NCLBUTTONDOWN
          if ((realMouseButtons & 1) === 0) {
            return
          }
          // 防線 2：左上角標題列敏感區 (Logo 區域 x < 30 且 y < 30) 禁止觸發 nw-resize
          if (message === 'wails:resize:nw-resize' && lastMouseX < 30 && lastMouseY < 30) {
            return
          }
        }
      }
      return originalPostMessage(message)
    }
    webview.__bomixGuarded = true
  }

  // 嘗試立即掛載
  if ((window as any).chrome?.webview) {
    guardWebview((window as any).chrome.webview)
  }

  // 監聽 window.chrome 動態注入
  let chromeObj = (window as any).chrome
  Object.defineProperty(window, 'chrome', {
    configurable: true,
    enumerable: true,
    get() {
      return chromeObj
    },
    set(val) {
      chromeObj = val
      if (val?.webview) {
        guardWebview(val.webview)
      }
    },
  })

  // 雙重保險定時器 (確保 WebView2 初始化注入被攔截)
  const pollTimer = setInterval(() => {
    if ((window as any).chrome?.webview?.__bomixGuarded) {
      clearInterval(pollTimer)
    } else if ((window as any).chrome?.webview) {
      guardWebview((window as any).chrome.webview)
    }
  }, 50)
}


import { createApp } from 'vue'
import PrimeVue from 'primevue/config'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import './style.css'
import 'primeicons/primeicons.css'

// Import @primeuix/themes presets
import Aura from '@primeuix/themes/aura'


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
