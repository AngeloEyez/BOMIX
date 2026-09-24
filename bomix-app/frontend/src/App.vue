<template>
  <div
    class="app-container"
    data-file-drop-target="true"
  >
    <!-- 全域檔案拖曳視覺遮罩 (VS Code Drag Overlay，pointer-events: none 純視覺層) -->
    <div
      v-if="isDraggingOver"
      class="global-drag-overlay"
      data-file-drop-target="true"
    >
      <div class="drag-card" data-file-drop-target="true">
        <i class="pi pi-file-excel drag-icon"></i>
        <span class="drag-title">釋放滑鼠以匯入 BOM 檔案</span>
        <span class="drag-sub">支援 EBOM, BigMatrix, Matrix Excel 檔案 (.xlsx, .xls)</span>
      </div>
    </div>

    <!-- Header / Title Bar -->
    <header class="header" @dblclick="handleTitleBarDblClick">
      <div class="header-left">
        <div class="logo">
          <img src="/app-logo.png" alt="BOMIX" class="app-header-logo" />
          <span v-if="appStore.seriesInfo?.name" class="logo-text">
            {{ appStore.seriesInfo.name }}
          </span>
          <!-- Close Series 按鈕 (僅在系列開啟時顯示，緊接著系列名稱) -->
          <Button
            v-if="appStore.isOpen"
            icon="pi pi-sign-out"
            text
            severity="secondary"
            class="title-bar-btn close-series-btn"
            @click="handleCloseSeries"
            title="Close Series"
          />
        </div>
      </div>
      <div class="header-right">
        <!-- 1. Main View (BOM) with Dropdown (Import & Matrix) -->
        <SplitButton
          icon="pi pi-home"
          label="BOM"
          text
          size="small"
          dropdown-icon="pi pi-chevron-down"
          :severity="isMainViewActive ? 'primary' : 'secondary'"
          :class="['title-bar-splitbtn', { 'title-bar-splitbtn-active': isMainViewActive }]"
          :model="bomMenuItems"
          @click="handleMainViewClick"
          :button-props="{
            title: 'Main View (BOM Table)',
            class: 'title-bar-splitbtn-action'
          }"
          :menu-button-props="{
            title: 'BOM Actions (Import / Matrix)',
            class: 'title-bar-splitbtn-dropdown'
          }"
        />

        <!-- 2. Export (僅在系列開啟時顯示/可用) -->
        <Button
          v-if="appStore.isOpen"
          icon="pi pi-download"
          label="Export"
          text
          :severity="isExportActive ? 'primary' : 'secondary'"
          :class="['title-bar-btn', { 'title-bar-btn-active': isExportActive }]"
          @click="handleExportClick"
          title="Export BOM"
        />

        <!-- 3. AI (僅在系列開啟且啟用 AI Assistant 時顯示/可用，與 BOM/Export 同級) -->
        <Button
          v-if="appStore.isOpen && aiChatStore.isEnabled"
          icon="pi pi-sparkles"
          label="AI"
          text
          :severity="isAIChatActive ? 'primary' : 'secondary'"
          :class="['title-bar-btn', { 'title-bar-btn-active': isAIChatActive }]"
          @click="handleAIChatClick"
          title="AI Assistant"
        />

        <!-- 4. Settings -->
        <Button
          icon="pi pi-cog"
          text
          :severity="isSettingsActive ? 'primary' : 'secondary'"
          :class="['title-bar-btn', { 'title-bar-btn-active': isSettingsActive }]"
          @click="handleSettingsClick"
          title="Settings"
        />

        <!-- 4. Windows 11 原生風格視窗控制三鍵 (最小化、最大化/還原、關閉) -->
        <WindowControls />
      </div>
    </header>

    <!-- Main Content Area with Fixed-Pixel Resizable Sidebar -->
    <div class="main-workspace-container">
      <!-- Sidebar Panel (固定像素寬度，最小 5px，雙擊重置) -->
      <aside
        class="sidebar-container"
        :style="{ width: `${sidebarWidth}px` }"
      >
        <SidebarPanel />
      </aside>

      <!-- Splitter Gutter (細線分割拖曳條，雙擊重置為預設 140px) -->
      <div
        class="sidebar-gutter"
        @mousedown="startSidebarResize"
        @dblclick="resetSidebarWidth"
        title="拖曳以調整寬度，雙擊重置為 140px"
      ></div>

      <!-- Main Content Panel (自動吸收主視窗所有剩餘橫向寬度) -->
      <main class="main-content-container">
        <router-view />
      </main>
    </div>

    <!-- Bottom Log Panel：resize-handle 使用與 splitter gutter 相同的細線風格 -->
    <div class="bottom-panel" :style="{ height: `${bottomPanelHeight}px` }">
      <div
        class="resize-handle"
        @mousedown="startBottomResize"
        @dblclick="resetBottomHeight"
        title="拖曳以調整高度，雙擊重置為單行預設高度"
      ></div>
      <LogPanel />
    </div>

    <!-- 全域匯入對話框 (支援在任意介面獨立彈出並在背景執行) -->
    <ImportDialog
      v-model:visible="appStore.importDialogVisible"
      @importSuccess="onGlobalImportSuccess"
    />

    <!-- 全域匯入即時狀態與進度監控對話框 -->
    <ImportResultsDialog
      v-model:visible="appStore.importResultDialogVisible"
      :results="appStore.importResults"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Window } from '@wailsio/runtime'

import SplitButton from 'primevue/splitbutton'
import Button from 'primevue/button'
import type { MenuItem } from 'primevue/menuitem'
import { useAppStore, useProjectStore, useLogStore, useTaskStore, useAIChatStore, useSettingsStore, useBOMTableStore } from './stores'
import LogPanel from './components/LogPanel.vue'
import SidebarPanel from './components/SidebarPanel.vue'
import WindowControls from './components/WindowControls.vue'
import ImportDialog from './components/workspace/ImportDialog.vue'
import ImportResultsDialog from './components/workspace/ImportResultsDialog.vue'
import { ListenToEvents } from './services/api'

const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const projectStore = useProjectStore()
const logStore = useLogStore()
const taskStore = useTaskStore()
const aiChatStore = useAIChatStore()
const settingsStore = useSettingsStore()
const bomTableStore = useBOMTableStore()

/**
 * 全域攔截右鍵選單，防止 WebView2 彈出瀏覽器預設網頁選單 (Reload, Back, Inspect 等)
 * 讓特定元件可以安全使用 PrimeVue 自定義桌面 ContextMenu
 */
function handleGlobalContextMenu(event: MouseEvent): void {
  event.preventDefault()
}

/**
 * 雙擊自訂標題列空白處切換視窗最大化與向下還原
 */
function handleTitleBarDblClick(event: MouseEvent): void {
  const target = event.target as HTMLElement
  // 若點擊目標位於按鈕、下拉選單或視窗控制項內部，則不觸發視窗縮放
  if (
    target.closest('button') ||
    target.closest('.p-button') ||
    target.closest('.p-splitbutton') ||
    target.closest('.window-controls')
  ) {
    return
  }
  Window.ToggleMaximise()
}

/**
 * 判斷 Main View (BOM) 按鈕之 Active 高亮狀態
 */
const isMainViewActive = computed(() => {
  if (route.path === '/settings' || route.path === '/ai-chat') return false
  if (route.path === '/workspace') {
    return appStore.workspaceView === 'table'
  }
  return route.path === '/'
})

/**
 * 判斷 Export 匯出按鈕之 Active 高亮狀態
 */
const isExportActive = computed(() => {
  return route.path === '/workspace' && appStore.workspaceView === 'export'
})

/**
 * 判斷 AI 對話按鈕之 Active 高亮狀態
 */
const isAIChatActive = computed(() => {
  return route.path === '/ai-chat'
})

/**
 * 判斷 Settings 設定按鈕之 Active 高亮狀態
 */
const isSettingsActive = computed(() => {
  return route.path === '/settings'
})

/**
 * 處理 Main View (BOM Table) 按鈕點擊事件
 */
function handleMainViewClick(): void {
  if (appStore.isOpen) {
    appStore.setWorkspaceView('table')
    if (route.path !== '/workspace') {
      router.push('/workspace')
    }
  } else {
    if (route.path !== '/') {
      router.push('/')
    }
  }
}

/**
 * 處理下拉選單點擊 Import 事件
 */
function handleImportClick(): void {
  if (!appStore.isOpen) {
    logStore.addLogEntry('WARN', '請先建立或開啟系列專案，方可匯入 BOM 檔案')
    return
  }
  appStore.openImportDialog()
}

/**
 * 處理下拉選單點擊 Matrix 事件
 */
function handleCopyMatrixClick(): void {
  if (!appStore.isOpen) {
    logStore.addLogEntry('WARN', '請先建立或開啟系列專案，方可複製 Matrix')
    return
  }
  if (route.path !== '/workspace') {
    appStore.setWorkspaceView('table')
    router.push('/workspace')
  }
  appStore.openCopyMatrixDialog()
}

/**
 * Main View (BOM) SplitButton 下拉選單項目模型
 */
const bomMenuItems = computed<MenuItem[]>(() => [
  {
    label: 'Import BOM',
    icon: 'pi pi-upload',
    disabled: !appStore.isOpen,
    command: () => {
      handleImportClick()
    }
  },
  {
    label: 'Matrix Copy',
    icon: 'pi pi-copy',
    disabled: !appStore.isOpen,
    command: () => {
      handleCopyMatrixClick()
    }
  }
])

/**
 * 處理 Export 按鈕點擊事件
 */
function handleExportClick(): void {
  if (!appStore.isOpen) return
  appStore.setWorkspaceView('export')
  if (route.path !== '/workspace') {
    router.push('/workspace')
  }
}

/**
 * 處理 AI Assistant 按鈕點擊事件
 */
function handleAIChatClick(): void {
  if (!appStore.isOpen || !aiChatStore.isEnabled) return
  if (route.path !== '/ai-chat') {
    router.push('/ai-chat')
  }
}

/**
 * 處理 Settings 按鈕點擊事件
 */
function handleSettingsClick(): void {
  if (route.path !== '/settings') {
    router.push('/settings')
  }
}

// 全域拖曳檔案狀態管理
const isDraggingOver = ref(false)
let dragCounter = 0

/**
 * 重置全域拖曳狀態，關閉拖曳提示遮罩
 */
function resetDragState(): void {
  dragCounter = 0
  isDraggingOver.value = false
}

/**
 * 處理拖曳進入視窗事件
 * @param {DragEvent} e - 原生拖曳事件
 */
function handleDragEnter(e: DragEvent): void {
  e.preventDefault()
  if (e.dataTransfer?.types?.includes('Files')) {
    dragCounter++
    isDraggingOver.value = true
  }
}

/**
 * 處理拖曳懸浮視窗事件
 * @param {DragEvent} e - 原生拖曳事件
 */
function handleDragOver(e: DragEvent): void {
  e.preventDefault()
  if (e.dataTransfer && e.dataTransfer.types.includes('Files')) {
    e.dataTransfer.dropEffect = 'copy'
  }
}

/**
 * 處理拖曳離開視窗事件
 * @param {DragEvent} e - 原生拖曳事件
 */
function handleDragLeave(e: DragEvent): void {
  e.preventDefault()
  if (e.dataTransfer?.types?.includes('Files')) {
    dragCounter--
    if (dragCounter <= 0) {
      resetDragState()
    }
  }
}

/**
 * 全域鍵盤按鍵事件處理 (按下 ESC 鍵時安全重置拖曳遮罩)
 * @param {KeyboardEvent} e - 原生鍵盤事件
 */
function handleGlobalKeyDown(e: KeyboardEvent): void {
  if (e.key === 'Escape' && isDraggingOver.value) {
    resetDragState()
  }
}

/**
 * 視窗失焦防護處理 (當視窗失焦或切換至外部程式時自動重置拖曳遮罩、游標樣式與 Wails 縮放狀態)
 */
function handleWindowBlur(): void {
  if (isDraggingOver.value) {
    resetDragState()
  }
  // 防護：若游標處於 Wails 邊界縮放樣式，於失焦時重置，避免焦點返回時游標滯留
  if (document.body.style.cursor && document.body.style.cursor.includes('resize')) {
    document.body.style.cursor = 'auto'
  }
  // 強制重置 Wails drag.js 內部模組級別的殘留縮放狀態 (resizeEdge, canResize, resizing)
  if ((window as any)._wails?.setResizable) {
    ;(window as any)._wails.setResizable(false)
    ;(window as any)._wails.setResizable(true)
  }
}

/**
 * 全域滑鼠離開視窗防護處理
 * 當滑鼠由頂部或邊界快速滑出視窗外時，重置 body 縮放游標樣式，並徹底清理 Wails drag.js 內部狀態
 * @param {MouseEvent} e - 滑鼠事件物件
 */
function handleGlobalMouseLeave(e: MouseEvent): void {
  // 如果移動至視窗外 (relatedTarget 為 null)
  if (!e.relatedTarget) {
    if (document.body.style.cursor && document.body.style.cursor.includes('resize')) {
      document.body.style.cursor = 'auto'
    }
    // 強制重置 Wails drag.js 內部模組級別的殘留縮放狀態
    if ((window as any)._wails?.setResizable) {
      ;(window as any)._wails.setResizable(false)
      ;(window as any)._wails.setResizable(true)
    }
  }
}

/**
 * 判斷路徑是否為絕對磁碟路徑
 * @param {string} path - 檔案路徑字串
 * @returns {boolean} 是否為合法絕對路徑
 */
function isAbsolutePath(path: string): boolean {
  if (!path) return false
  return /^[a-zA-Z]:[\\/]/.test(path) || path.startsWith('\\\\') || path.startsWith('/')
}

/**
 * 安全解析各類事件格式中的檔案清單 (字串陣列、單一字串、物件包裝)
 * @param {any} data - 事件資料物件或字串
 * @returns {string[]} 解析出之檔案路徑陣列
 */
function extractFileList(data: any): string[] {
  if (!data) return []
  if (Array.isArray(data)) {
    if (data.length > 0 && Array.isArray(data[0])) {
      return data[0].map(String)
    }
    return data.map(String)
  }
  if (typeof data === 'string') {
    return [data]
  }
  if (typeof data === 'object') {
    if (Array.isArray(data.files)) {
      return data.files.map(String)
    }
    if (Array.isArray(data.data)) {
      return extractFileList(data.data)
    }
  }
  return []
}

/**
 * 全域匯入提交成功處理函式
 * 彈出進度監控對話框，清除 BOMTable 快取，若系列已開啟則更新專案資料
 * @param {any[]} results - 匯入結果項目清單
 */
function onGlobalImportSuccess(results: any[]): void {
  appStore.openImportResultsDialog(results)
  bomTableStore.triggerReload()
  if (appStore.isOpen) {
    loadProjects()
  }
}

/**
 * 統一處理傳入的檔案路徑清單 (啟動全域匯入流程)
 * @param {string[]} paths - 欲匯入之檔案絕對路徑清單
 */
function handleIncomingFiles(paths: string[]): void {
  if (!paths || paths.length === 0) return

  if (!appStore.isOpen) {
    logStore.addLogEntry('WARN', '請先建立或開啟系列專案，方可匯入 BOM 檔案')
    return
  }

  logStore.addLogEntry('DEBUG', `[DragDrop] 收到 ${paths.length} 個檔案拖曳請求: ${paths.join(', ')}`)
  // 在當前頁面直接開啟全域匯入對話框，不強行跳轉頁面，獨立背景執行
  appStore.handleDroppedFiles(paths)
}

/**
 * 處理接收到的拖放檔案清單 (Wails 原生視窗拖放事件)
 * @param {any} data - Wails 後端推送之拖放資料 (支援路徑陣列或物件格式)
 */
function onFilesReceived(data: any): void {
  resetDragState()

  const paths = extractFileList(data)
  logStore.addLogEntry('DEBUG', `[DragDrop] Wails 原生拖放事件觸發，解析到 ${paths.length} 個項目: ${JSON.stringify(paths)}`)

  const validPaths = paths.filter(path => {
    const p = String(path).toLowerCase()
    return p.endsWith('.xlsx') || p.endsWith('.xls')
  })

  if (validPaths.length > 0) {
    handleIncomingFiles(validPaths)
  } else if (paths.length > 0) {
    logStore.addLogEntry('WARN', `拖入的檔案並非支援的 Excel 格式 (.xlsx, .xls)`)
  }
}

/**
 * 處理放開拖曳檔案事件 (HTML5 原生事件處理)
 * @param {DragEvent} e - 原生拖曳事件物件
 */
function handleDrop(e: DragEvent): void {
  e.preventDefault()
  resetDragState()

  const files = e.dataTransfer?.files
  if (!files || files.length === 0) return

  const validPaths: string[] = []
  for (let i = 0; i < files.length; i++) {
    const file = files[i]
    const path = (file as any).path || ''
    if (path && isAbsolutePath(path)) {
      const lower = path.toLowerCase()
      if (lower.endsWith('.xlsx') || lower.endsWith('.xls')) {
        validPaths.push(path)
      }
    }
  }

  if (validPaths.length > 0) {
    handleIncomingFiles(validPaths)
  }
}

// 側邊欄寬度管理 (像素制)：預設為 140px，不隨視窗縮放變動
const sidebarWidth = ref(140)
let isResizingSidebar = false
let startSidebarX = 0
let startSidebarWidth = 0

// Bottom panel 預設高度：單行模式 (24px：含 4px 拖曳控制條與 20px 日誌面板)
const DEFAULT_BOTTOM_HEIGHT = 24
const bottomPanelHeight = ref(DEFAULT_BOTTOM_HEIGHT)
let isResizingBottom = false
let startY = 0
let startHeight = 0

onMounted(async () => {
  // 註冊全域右鍵選單攔截與全域拖曳防護 (冒泡模式監聽 drop，避免干擾底層 Wails 運行時)
  window.addEventListener('contextmenu', handleGlobalContextMenu)
  window.addEventListener('dragenter', handleDragEnter)
  window.addEventListener('dragover', handleDragOver)
  window.addEventListener('dragleave', handleDragLeave)
  window.addEventListener('drop', handleDrop, false)
  window.addEventListener('keydown', handleGlobalKeyDown)
  window.addEventListener('blur', handleWindowBlur)
  document.addEventListener('mouseleave', handleGlobalMouseLeave)

  // 確保根節點具備 Wails v3 所需之 data-file-drop-target 屬性
  document.documentElement.setAttribute('data-file-drop-target', 'true')

  // 監聽 Wails 原生與後端視窗拖放事件 (包含絕對路徑，相容 files:dropped 與 files-dropped)
  ListenToEvents('files:dropped', onFilesReceived)
  ListenToEvents('files-dropped', onFilesReceived)

  // Start listening to events
  logStore.startListening()
  taskStore.startListening()

  // Load initial settings via settingsStore (SSOT)
  try {
    await settingsStore.initSettings()
  } catch (e) {
    appStore.applyTheme('system')
  }

  // Listen to OS theme changes
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
    if (appStore.currentTheme === 'system') {
      appStore.applyTheme('system')
    }
  })

  // Load initial data if series is open
  if (appStore.isOpen) {
    loadProjects()
  }

  // Check for auto-open last file
  checkAutoOpen()

  // App 啟動時自動讀取設定中選中的模型並更新伺服器可用模型清單
  try {
    await aiChatStore.fetchCurrentModel()
    aiChatStore.fetchAvailableModels()
  } catch (err) {
    console.warn('App 啟動拉取 AI 模型清單失敗:', err)
  }
})

onUnmounted(() => {
  window.removeEventListener('contextmenu', handleGlobalContextMenu)
  window.removeEventListener('dragenter', handleDragEnter)
  window.removeEventListener('dragover', handleDragOver)
  window.removeEventListener('dragleave', handleDragLeave)
  window.removeEventListener('drop', handleDrop, false)
  window.removeEventListener('keydown', handleGlobalKeyDown)
  window.removeEventListener('blur', handleWindowBlur)
  document.removeEventListener('mouseleave', handleGlobalMouseLeave)
  document.removeEventListener('mousemove', handleSidebarResize)
  document.removeEventListener('mouseup', stopSidebarResize)
})

// Watch for series open changes
watch(() => appStore.isOpen, (isOpen) => {
  if (isOpen) {
    loadProjects()
    router.push('/workspace')
  } else {
    projectStore.clearProjects()
    aiChatStore.clearMessages()
    router.push('/')
  }
})

async function loadProjects(): Promise<void> {
  if (!appStore.seriesInfo?.id) return

  try {
    // 載入專案列表 (包含 Revisions)
    await projectStore.loadProjects(appStore.seriesInfo.id)
  } catch (error) {
    const msg = error instanceof Error ? error.message : String(error)
    useLogStore().addLogEntry('ERROR', `載入專案資料失敗：${msg}`)
    console.error('Failed to load projects:', error)
  }
}

/**
 * 開始拖曳調整側邊欄寬度
 * @param {MouseEvent} event - 原生滑鼠按下事件
 */
function startSidebarResize(event: MouseEvent): void {
  isResizingSidebar = true
  startSidebarX = event.clientX
  startSidebarWidth = sidebarWidth.value
  document.addEventListener('mousemove', handleSidebarResize)
  document.addEventListener('mouseup', stopSidebarResize)
}

/**
 * 處理側邊欄寬度拖曳過程
 * 最小保留 5px 方便邊界還能選取拖曳，最大限制保留右側主內容區至少 200px
 * @param {MouseEvent} event - 原生滑鼠移動事件
 */
function handleSidebarResize(event: MouseEvent): void {
  if (!isResizingSidebar) return
  const deltaX = event.clientX - startSidebarX
  const newWidth = startSidebarWidth + deltaX
  // 最小保留 5px，最大保留右側至少 200px 空間
  sidebarWidth.value = Math.max(5, Math.min(newWidth, window.innerWidth - 200))
  // 廣播 window resize 事件，通知 BOMTable 等元件即時重新計算欄位寬度
  window.dispatchEvent(new Event('resize'))
}

/**
 * 停止側邊欄拖曳
 */
function stopSidebarResize(): void {
  isResizingSidebar = false
  document.removeEventListener('mousemove', handleSidebarResize)
  document.removeEventListener('mouseup', stopSidebarResize)
}

/**
 * 雙擊分割條重置側邊欄寬度為預設 140px
 */
function resetSidebarWidth(): void {
  sidebarWidth.value = 140
  window.dispatchEvent(new Event('resize'))
}

function startBottomResize(event: MouseEvent): void {
  isResizingBottom = true
  startY = event.clientY
  startHeight = bottomPanelHeight.value
  document.addEventListener('mousemove', handleBottomResize)
  document.addEventListener('mouseup', stopBottomResize)
}

function handleBottomResize(event: MouseEvent): void {
  if (!isResizingBottom) return
  const deltaY = event.clientY - startY
  const newHeight = startHeight - deltaY
  // 最小高度為單行預設高度 (24px)，最大限制為 50% 視窗高度
  bottomPanelHeight.value = Math.max(DEFAULT_BOTTOM_HEIGHT, Math.min(newHeight, window.innerHeight * 0.5))
}

function stopBottomResize(): void {
  isResizingBottom = false
  document.removeEventListener('mousemove', handleBottomResize)
  document.removeEventListener('mouseup', stopBottomResize)
}

function resetBottomHeight(): void {
  bottomPanelHeight.value = DEFAULT_BOTTOM_HEIGHT
}

/**
 * 處理使用者點擊標題列關閉系列按鈕
 * 關閉系列連線、重置專案快取並清空 AI 對話紀錄
 * @returns {Promise<void>}
 */
async function handleCloseSeries(): Promise<void> {
  await appStore.closeSeries()
  projectStore.clearProjects()
  aiChatStore.clearMessages()
}

async function checkAutoOpen(): Promise<void> {
  // This will check the settings and auto-open the last file if enabled
}
</script>

<style>
/* Global styles */
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html,
body {
  height: 100%;
  overflow: hidden;
}

.main-workspace-container {
  display: flex;
  flex-direction: row;
  flex: 1;
  min-height: 0;
  width: 100%;
  overflow: hidden;
}

.main-content-container {
  flex: 1 1 0%;
  min-width: 0;
  min-height: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding-right: 5px;
  box-sizing: border-box;
}

/* 確保所有透過 router-view 渲染的主視圖元件強制填滿 Main Content 區域並隨 sidebar/視窗彈性調適 */
.main-content-container > * {
  flex: 1 1 0%;
  min-width: 0;
  min-height: 0;
  height: 100%;
  width: 100%;
}

#app {
  height: 100%;
}

/* PrimeVue overrides */
:root {
  --p-primary-500: #6366f1;
  --p-primary-600: #4f46e5;
}
</style>

<style scoped>
.app-container {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: var(--surface-ground);
}

/* Header / Custom Titlebar */
.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  height: 38px;
  padding-top: 2px;
  padding-left: 0.6rem;
  padding-right: 0; /* 右側貼齊視窗邊界，符合 Windows 原生視窗按鈕貼邊規範 */
  background: var(--surface-card);
  border-bottom: 1px solid var(--surface-border);
  flex-shrink: 0;
  user-select: none;
  -webkit-app-region: drag;
  --wails-draggable: drag;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  -webkit-app-region: no-drag;
  --wails-draggable: none;
}

.logo {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  -webkit-app-region: no-drag;
  --wails-draggable: none;
}

.app-header-logo {
  width: 20px;
  height: 20px;
  object-fit: contain;
  border-radius: 4px;
  user-select: none;
  -webkit-user-drag: none;
  pointer-events: none;
}

.logo-text {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--text-color);
  letter-spacing: 0.05em;
  -webkit-app-region: no-drag;
  --wails-draggable: none;
}

.close-series-btn {
  margin-left: 0.25rem;
  opacity: 0.85;
  transition: opacity 0.15s ease, background-color 0.15s ease, color 0.15s ease;
  -webkit-app-region: no-drag;
  --wails-draggable: none;
}

.close-series-btn:hover {
  opacity: 1;
  color: var(--text-color) !important;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  height: 100%;
  -webkit-app-region: no-drag;
  --wails-draggable: none;
}

.title-bar-btn {
  height: 28px !important;
  min-height: 28px !important;
  font-size: 0.82rem;
  padding: 0 0.5rem !important;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 3px !important;
  transition: background-color 0.15s ease, color 0.15s ease;
  -webkit-app-region: no-drag;
  --wails-draggable: none;
}

/* 未選中時 hover：淺灰底色 */
.title-bar-btn:not(.title-bar-btn-active):hover {
  background-color: var(--surface-hover) !important;
  color: var(--text-color) !important;
}

/* 選中狀態：綠底白字 */
.title-bar-btn.title-bar-btn-active {
  color: #ffffff !important;
  background-color: var(--primary-color) !important;
}

.title-bar-btn.title-bar-btn-active .p-button-icon {
  color: #ffffff !important;
}

/* 選中時 hover：接近 primary color 的微調綠色 */
.title-bar-btn.title-bar-btn-active:hover {
  background-color: color-mix(in srgb, var(--primary-color) 85%, black) !important;
  color: #ffffff !important;
}

/* SplitButton in Title Bar */
.title-bar-splitbtn {
  height: 28px !important;
  min-height: 28px !important;
  display: inline-flex;
  align-items: center;
  vertical-align: middle;
  border-radius: 3px !important;
  -webkit-app-region: no-drag;
  --wails-draggable: none;
}

:deep(.title-bar-splitbtn .p-splitbutton-button) {
  height: 28px !important;
  min-height: 28px !important;
  font-size: 0.82rem;
  padding: 0 0.4rem 0 0.5rem !important;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-top-left-radius: 3px !important;
  border-bottom-left-radius: 3px !important;
  border-top-right-radius: 0 !important;
  border-bottom-right-radius: 0 !important;
  border-right: none !important;
  transition: background-color 0.15s ease, color 0.15s ease;
}

:deep(.title-bar-splitbtn .p-splitbutton-dropdown) {
  height: 28px !important;
  min-height: 28px !important;
  width: 20px !important;
  min-width: 20px !important;
  max-width: 20px !important;
  padding: 0 !important;
  margin: 0 !important;
  gap: 0 !important;
  box-sizing: border-box !important;
  display: inline-flex !important;
  align-items: center !important;
  justify-content: center !important;
  border-top-left-radius: 0 !important;
  border-bottom-left-radius: 0 !important;
  border-top-right-radius: 3px !important;
  border-bottom-right-radius: 3px !important;
  border-left: 1px solid var(--surface-border) !important;
  transition: background-color 0.15s ease, color 0.15s ease;
}

/* 移除 PrimeVue 按鈕預設的 ::after 偽元素，防止 flex gap 造成圖示偏左 */
:deep(.title-bar-splitbtn .p-splitbutton-dropdown::after) {
  display: none !important;
  content: none !important;
  width: 0 !important;
  margin: 0 !important;
}

/* 確保下拉箭頭大小與主按鈕中的 icon (pi-home) 大小相同且精準置中 */
:deep(.title-bar-splitbtn .p-splitbutton-button .p-button-icon),
:deep(.title-bar-splitbtn .p-splitbutton-dropdown .p-button-icon),
:deep(.title-bar-splitbtn .p-splitbutton-dropdown svg) {
  font-size: 0.72rem !important;
  width: 0.72rem !important;
  height: 0.72rem !important;
  line-height: 1 !important;
}

:deep(.title-bar-splitbtn .p-splitbutton-dropdown .p-button-icon) {
  margin: 0 !important;
  padding: 0 !important;
  display: inline-flex !important;
  align-items: center !important;
  justify-content: center !important;
  text-align: center !important;
}

/* 未選中時 hover：淺灰底色 */
:deep(.title-bar-splitbtn:not(.title-bar-splitbtn-active) .p-splitbutton-button:hover),
:deep(.title-bar-splitbtn:not(.title-bar-splitbtn-active) .p-splitbutton-dropdown:hover) {
  background-color: var(--surface-hover) !important;
  color: var(--text-color) !important;
}

/* 選中狀態：綠底白字 */
:deep(.title-bar-splitbtn-active .p-splitbutton-button),
:deep(.title-bar-splitbtn-active .p-splitbutton-dropdown) {
  color: #ffffff !important;
  background-color: var(--primary-color) !important;
}

:deep(.title-bar-splitbtn-active .p-splitbutton-button .p-button-icon),
:deep(.title-bar-splitbtn-active .p-splitbutton-dropdown .p-button-icon),
:deep(.title-bar-splitbtn-active .p-splitbutton-dropdown svg) {
  color: #ffffff !important;
  fill: #ffffff !important;
}

:deep(.title-bar-splitbtn-active .p-splitbutton-dropdown) {
  border-left: 1px solid rgba(255, 255, 255, 0.25) !important;
}

/* 選中時 hover：接近 primary color 的微調綠色 */
:deep(.title-bar-splitbtn-active .p-splitbutton-button:hover),
:deep(.title-bar-splitbtn-active .p-splitbutton-dropdown:hover) {
  background-color: color-mix(in srgb, var(--primary-color) 85%, black) !important;
  color: #ffffff !important;
}

/* 側邊欄容器：固定像素寬度，不參與 flex 伸縮 */
.sidebar-container {
  flex-shrink: 0;
  flex-grow: 0;
  overflow: hidden;
  min-width: 5px;
}

/* 側邊欄細線拖曳條：與下方 resize-handle 保持一致的 4px 細線風格 */
.sidebar-gutter {
  width: 4px;
  background-color: var(--surface-border);
  border-left: 1px solid var(--surface-hover);
  border-right: 1px solid var(--surface-hover);
  cursor: col-resize;
  flex-shrink: 0;
  transition: background-color 0.2s;
  user-select: none;
}

.sidebar-gutter:hover {
  background-color: var(--primary-color) !important;
}

/* Bottom Panel */
.bottom-panel {
  display: flex;
  flex-direction: column;
  background: var(--surface-card);
  flex-shrink: 0;
  padding-right: 5px;
  box-sizing: border-box;
}

/*
  resize-handle：與 PrimeVue .p-splitter-gutter 保持相同的視覺風格。
  使用 4px 高的細線，hover 時高亮為 primary-color，雙擊重置高度。
*/
.resize-handle {
  height: 4px;
  background-color: var(--surface-border);
  border-top: 1px solid var(--surface-hover);
  border-bottom: 1px solid var(--surface-hover);
  cursor: ns-resize;
  flex-shrink: 0;
  transition: background-color 0.2s;
}

.resize-handle:hover {
  background-color: var(--primary-color);
}

/* 全域檔案拖曳視覺遮罩 (VS Code Drag Overlay) */
/* 全域檔案拖曳視覺遮罩 (VS Code Drag Overlay，pointer-events: none 保證不干擾滑鼠與拖曳) */
.global-drag-overlay {
  position: fixed;
  inset: 0;
  z-index: 99999;
  background-color: rgba(0, 0, 0, 0.45);
  backdrop-filter: blur(2px);
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: none !important;
  user-select: none;
}

.drag-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.6rem;
  padding: 2.5rem 3.5rem;
  background: var(--surface-card);
  border: 2px dashed var(--primary-color);
  border-radius: 6px;
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.35);
  animation: pulse-border 1.5s infinite ease-in-out;
  cursor: default;
}

.drag-icon {
  font-size: 3rem;
  color: #22c55e;
}

.drag-title {
  font-size: 1.15rem;
  font-weight: 600;
  color: var(--text-color);
}

.drag-sub {
  font-size: 0.85rem;
  color: var(--text-color-secondary);
}

@keyframes pulse-border {
  0%, 100% {
    border-color: var(--primary-color);
    transform: scale(1);
  }
  50% {
    border-color: #3b82f6;
    transform: scale(1.02);
  }
}
</style>
