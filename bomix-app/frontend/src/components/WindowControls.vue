<template>
  <div class="window-controls" aria-label="視窗控制" @mouseenter="syncNonClientRegions">
    <!-- 最小化按鈕 -->
    <button
      ref="minBtnRef"
      type="button"
      class="win-btn win-btn-minimize"
      title="最小化"
      @click="handleMinimize"
    >
      <svg width="10" height="10" viewBox="0 0 10 10">
        <path d="M 0,5 L 10,5" stroke="currentColor" stroke-width="1" />
      </svg>
    </button>

    <!-- 最大化 / 還原按鈕 (註冊 HTMAXBUTTON 以支援 Windows 11 Snap Layouts) -->
    <button
      ref="maxBtnRef"
      type="button"
      class="win-btn win-btn-maximize"
      :title="isMaximized ? '向下還原' : '最大化'"
      @click="handleToggleMaximize"
    >
      <!-- 還原狀態圖示 (雙框) -->
      <svg v-if="isMaximized" width="10" height="10" viewBox="0 0 10 10">
        <!-- 後方邊框 (右上) -->
        <path d="M 2.5,2.5 L 2.5,0.5 L 9.5,0.5 L 9.5,7.5 L 7.5,7.5" fill="none" stroke="currentColor" stroke-width="1" />
        <!-- 前方方框 (左下) -->
        <rect x="0.5" y="2.5" width="7" height="7" fill="none" stroke="currentColor" stroke-width="1" />
      </svg>

      <!-- 最大化狀態圖示 (單框) -->
      <svg v-else width="10" height="10" viewBox="0 0 10 10">
        <rect x="0.5" y="0.5" width="9" height="9" fill="none" stroke="currentColor" stroke-width="1" />
      </svg>
    </button>

    <!-- 關閉按鈕 -->
    <button
      ref="closeBtnRef"
      type="button"
      class="win-btn win-btn-close"
      title="關閉"
      @click="handleClose"
    >
      <svg width="10" height="10" viewBox="0 0 10 10">
        <path d="M 0.5,0.5 L 9.5,9.5 M 9.5,0.5 L 0.5,9.5" stroke="currentColor" stroke-width="1" />
      </svg>
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { Window } from '@wailsio/runtime'

const minBtnRef = ref<HTMLButtonElement | null>(null)
const maxBtnRef = ref<HTMLButtonElement | null>(null)
const closeBtnRef = ref<HTMLButtonElement | null>(null)

/** 視窗是否處於最大化狀態 */
const isMaximized = ref(false)

/**
 * 查詢並更新當前視窗最大化狀態
 */
async function checkWindowState(): Promise<void> {
  try {
    if (Window && typeof Window.IsMaximised === 'function') {
      isMaximized.value = await Window.IsMaximised()
    }
  } catch (err) {
    console.debug('[WindowControls] 無法取得視窗最大化狀態:', err)
  }
}

/**
 * 點擊最小化視窗
 */
function handleMinimize(): void {
  try {
    Window.Minimise()
  } catch (err) {
    console.error('[WindowControls] 最小化失敗:', err)
  }
}

/**
 * 點擊最大化或還原視窗
 */
async function handleToggleMaximize(): Promise<void> {
  try {
    await Window.ToggleMaximise()
    // 等待 DOM 與狀態更新
    setTimeout(checkWindowState, 50)
  } catch (err) {
    console.error('[WindowControls] 切換最大化失敗:', err)
  }
}

/**
 * 點擊關閉視窗 (退出應用)
 */
function handleClose(): void {
  try {
    Window.Close()
  } catch (err) {
    console.error('[WindowControls] 關閉視窗失敗:', err)
  }
}

/**
 * 向 Wails 後端底層發送訊息 (支援 WebView2 postMessage 與多平台適配)
 */
function postToWails(msg: string): void {
  try {
    if ((window as any).chrome?.webview?.postMessage) {
      (window as any).chrome.webview.postMessage(msg)
    } else if ((window as any).webkit?.messageHandlers?.external?.postMessage) {
      (window as any).webkit.messageHandlers.external.postMessage(msg)
    } else if ((window as any)._wails?.invoke) {
      (window as any)._wails.invoke(msg)
    }
  } catch (err) {
    console.debug('[WindowControls] postToWails 失敗:', err)
  }
}

/**
 * 向 Wails 後端同步非客戶區 (Non-Client Region) 命中測試矩形區域
 * 當滑鼠懸停在最大化按鈕時，讓 Windows 11 DWM 能識別 HTMAXBUTTON 並彈出原生的 Snap Layouts 選單
 */
function syncNonClientRegions(): void {
  if (!minBtnRef.value || !maxBtnRef.value || !closeBtnRef.value) return

  const minRect = minBtnRef.value.getBoundingClientRect()
  const maxRect = maxBtnRef.value.getBoundingClientRect()
  const closeRect = closeBtnRef.value.getBoundingClientRect()

  // 取得裝置像素比，轉換為實體螢幕像素座標
  const dpi = window.devicePixelRatio || 1

  const regions = [
    {
      kind: 'minimize',
      left: Math.round(minRect.left * dpi),
      top: Math.round(minRect.top * dpi),
      right: Math.round(minRect.right * dpi),
      bottom: Math.round(minRect.bottom * dpi)
    },
    {
      kind: 'maximize',
      left: Math.round(maxRect.left * dpi),
      top: Math.round(maxRect.top * dpi),
      right: Math.round(maxRect.right * dpi),
      bottom: Math.round(maxRect.bottom * dpi)
    },
    {
      kind: 'close',
      left: Math.round(closeRect.left * dpi),
      top: Math.round(closeRect.top * dpi),
      right: Math.round(closeRect.right * dpi),
      bottom: Math.round(closeRect.bottom * dpi)
    }
  ]

  const message =
    'wails:non-client-region:' +
    JSON.stringify({
      version: 1,
      regions
    })

  postToWails(message)
}

/**
 * 視窗尺寸改變時更新狀態與非客戶區座標
 */
function handleResize(): void {
  checkWindowState()
  nextTick(() => {
    syncNonClientRegions()
  })
}

onMounted(() => {
  checkWindowState()
  // 延遲初次同步，確保 DOM 完成渲染與佈局
  setTimeout(() => {
    syncNonClientRegions()
  }, 100)

  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped>
.window-controls {
  display: flex;
  align-items: center;
  height: 100%;
  margin-left: 0.25rem;
  /* 視窗按鈕不可拖曳，並確保層級高於背景拖曳區 */
  -webkit-app-region: no-drag;
  --wails-draggable: none;
  user-select: none;
  z-index: 100;
}

/* Windows 11 Fluent 視窗控制按鈕樣式 */
.win-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 46px;
  height: 100%;
  background: transparent;
  border: none;
  outline: none;
  padding: 0;
  margin: 0;
  color: var(--text-color, #a1a1aa);
  cursor: pointer;
  transition: background-color 0.12s ease, color 0.12s ease;
  -webkit-app-region: no-drag;
  --wails-draggable: none;
}

.win-btn svg {
  pointer-events: none;
  shape-rendering: geometricPrecision;
}

/* 最小化與最大化按鈕 Hover 與 Active 狀態 */
.win-btn-minimize:hover,
.win-btn-maximize:hover {
  background-color: var(--surface-hover, rgba(128, 128, 128, 0.15));
  color: var(--text-color, #ffffff);
}

.win-btn-minimize:active,
.win-btn-maximize:active {
  background-color: rgba(128, 128, 128, 0.25);
}

/* 關閉按鈕 Hover (微軟經典紅底白字) 與 Active */
.win-btn-close:hover {
  background-color: #e81123 !important;
  color: #ffffff !important;
}

.win-btn-close:active {
  background-color: #bf101f !important;
  color: #ffffff !important;
}
</style>
