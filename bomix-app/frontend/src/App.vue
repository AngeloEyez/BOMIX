<template>
  <div
    class="app-container"
    data-file-drop-target="true"
  >
    <!-- 全域檔案拖曳視覺遮罩 (VS Code Drag Overlay) -->
    <div
      v-if="isDraggingOver"
      class="global-drag-overlay"
      @dragover.prevent
      @drop="handleDrop"
      @click="resetDragState"
    >
      <div class="drag-card" @click.stop>
        <i class="pi pi-file-excel drag-icon"></i>
        <span class="drag-title">釋放滑鼠以匯入 BOM 檔案</span>
        <span class="drag-sub">支援 EBOM, BigMatrix, Matrix Excel 檔案 (.xlsx, .xls)</span>
      </div>
    </div>

    <!-- Header / Title Bar -->
    <header class="header">
      <div class="header-left">
        <div class="logo">
          <i class="pi pi-box"></i>
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

        <!-- 3. Settings -->
        <Button
          icon="pi pi-cog"
          text
          :severity="isSettingsActive ? 'primary' : 'secondary'"
          :class="['title-bar-btn', { 'title-bar-btn-active': isSettingsActive }]"
          @click="handleSettingsClick"
          title="Settings"
        />
      </div>
    </header>

    <!-- Main Content with Splitter -->
    <Splitter
      class="main-splitter"
      @resize="onSplitterResize"
      @resizeend="onSplitterResize"
    >
      <!-- Sidebar Panel -->
      <SplitterPanel
        :size="sidebarWidth"
        :min-size="0"
        @dblclick="resetSidebarWidth"
      >
        <SidebarPanel />
      </SplitterPanel>

      <!-- Main Content Panel -->
      <SplitterPanel :size="100 - sidebarWidth">
        <router-view />
      </SplitterPanel>
    </Splitter>

    <!-- Bottom Log Panel：resize-handle 使用與 splitter gutter 相同的細線風格 -->
    <div class="bottom-panel" :style="{ height: `${bottomPanelHeight}px` }">
      <div
        class="resize-handle"
        @mousedown="startBottomResize"
        @dblclick="resetBottomHeight"
      ></div>
      <LogPanel />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Splitter from 'primevue/splitter'
import SplitterPanel from 'primevue/splitterpanel'
import SplitButton from 'primevue/splitbutton'
import Button from 'primevue/button'
import type { MenuItem } from 'primevue/menuitem'
import { useAppStore, useProjectStore, useLogStore, useTaskStore } from './stores'
import LogPanel from './components/LogPanel.vue'
import SidebarPanel from './components/SidebarPanel.vue'
import { GetSettings, ListenToEvents } from './services/api'

const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const projectStore = useProjectStore()
const logStore = useLogStore()
const taskStore = useTaskStore()

/**
 * 全域攔截右鍵選單，防止 WebView2 彈出瀏覽器預設網頁選單 (Reload, Back, Inspect 等)
 * 讓特定元件可以安全使用 PrimeVue 自定義桌面 ContextMenu
 */
function handleGlobalContextMenu(event: MouseEvent): void {
  event.preventDefault()
}

/**
 * 判斷 Main View (BOM) 按鈕之 Active 高亮狀態
 */
const isMainViewActive = computed(() => {
  if (route.path === '/settings') return false
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
  if (route.path !== '/workspace') {
    appStore.setWorkspaceView('table')
    router.push('/workspace')
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
    label: 'Import',
    icon: 'pi pi-upload',
    disabled: !appStore.isOpen,
    command: () => {
      handleImportClick()
    }
  },
  {
    label: 'Matrix',
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
  dragCounter++
  if (e.dataTransfer?.types?.includes('Files')) {
    isDraggingOver.value = true
  }
}

/**
 * 處理拖曳懸浮視窗事件
 * @param {DragEvent} e - 原生拖曳事件
 */
function handleDragOver(e: DragEvent): void {
  e.preventDefault()
  if (e.dataTransfer) {
    e.dataTransfer.dropEffect = 'copy'
  }
}

/**
 * 處理拖曳離開視窗事件
 * @param {DragEvent} e - 原生拖曳事件
 */
function handleDragLeave(e: DragEvent): void {
  e.preventDefault()
  dragCounter--
  if (dragCounter <= 0) {
    resetDragState()
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
 * 視窗失焦防護處理 (當視窗失焦或切換至外部程式時自動重置拖曳遮罩)
 */
function handleWindowBlur(): void {
  if (isDraggingOver.value) {
    resetDragState()
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
 * 處理接收到的拖放檔案清單 (Wails 原生視窗拖放事件)
 * @param {any} data - Wails 後端推送之拖放資料 (支援路徑陣列或物件格式)
 */
function onFilesReceived(data: any): void {
  // 原生拖放作業已完成，無論檔案過濾結果為何，均應立即重置拖曳提示狀態
  resetDragState()

  const paths = extractFileList(data)
  const validPaths = paths.filter(path => {
    const p = String(path).toLowerCase()
    return p.endsWith('.xlsx') || p.endsWith('.xls')
  })

  if (validPaths.length > 0) {
    appStore.handleDroppedFiles(validPaths)
  }
}

/**
 * 處理放開拖曳檔案事件 (HTML5 原生事件處理)
 * @param {DragEvent} e - 原生拖曳事件物件
 */
function handleDrop(e: DragEvent): void {
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
    appStore.handleDroppedFiles(validPaths)
  }
}

// Sidebar width management
const sidebarWidth = ref(20) // Default 20%

// Bottom panel height
const bottomPanelHeight = ref(window.innerHeight * 0.1) // Default 10%
let isResizingBottom = false
let startY = 0
let startHeight = 0

onMounted(async () => {
  // 註冊全域右鍵選單攔截與全域拖曳防護 (capture 模式優先監聽 drop 與 blur/keydown)
  window.addEventListener('contextmenu', handleGlobalContextMenu)
  window.addEventListener('dragenter', handleDragEnter)
  window.addEventListener('dragover', handleDragOver)
  window.addEventListener('dragleave', handleDragLeave)
  window.addEventListener('drop', handleDrop, true)
  window.addEventListener('keydown', handleGlobalKeyDown)
  window.addEventListener('blur', handleWindowBlur)

  // 監聽 Wails 原生與後端視窗拖放事件 (包含絕對路徑)
  ListenToEvents('files:dropped', onFilesReceived)

  // Start listening to events
  logStore.startListening()
  taskStore.startListening()

  // Load initial settings for theme and log level
  try {
    const s = await GetSettings()
    appStore.applyTheme(s.theme)
    useLogStore().globalLogLevel = s.logger?.level || 'info'
    if (s.import) {
      appStore.confirmOverwrite = s.import.confirmOverwrite ?? true
    }
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
})

onUnmounted(() => {
  window.removeEventListener('contextmenu', handleGlobalContextMenu)
  window.removeEventListener('dragenter', handleDragEnter)
  window.removeEventListener('dragover', handleDragOver)
  window.removeEventListener('dragleave', handleDragLeave)
  window.removeEventListener('drop', handleDrop, true)
  window.removeEventListener('keydown', handleGlobalKeyDown)
  window.removeEventListener('blur', handleWindowBlur)
})

// Watch for series open changes
watch(() => appStore.isOpen, (isOpen) => {
  if (isOpen) {
    loadProjects()
    router.push('/workspace')
  } else {
    projectStore.clearProjects()
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
 * 處理 Main Splitter (側邊欄與工作區) 拖曳分割條事件
 * 廣播 window resize 事件，確保 BOMTable 即時重新計算欄寬並消除不必要的橫向捲軸
 * @param {any} event - PrimeVue Splitter resize 事件物件
 */
function onSplitterResize(event: any): void {
  if (event && event.sizes && event.sizes.length > 0) {
    sidebarWidth.value = event.sizes[0]
  } else if (typeof event === 'number') {
    sidebarWidth.value = event
  }
  window.dispatchEvent(new Event('resize'))
}

function resetSidebarWidth(): void {
  sidebarWidth.value = 20
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
  // 最小高度 18px（單行 log 高度），最大 50% 視窗高度
  bottomPanelHeight.value = Math.max(18, Math.min(newHeight, window.innerHeight * 0.5))
}

function stopBottomResize(): void {
  isResizingBottom = false
  document.removeEventListener('mousemove', handleBottomResize)
  document.removeEventListener('mouseup', stopBottomResize)
}

function resetBottomHeight(): void {
  bottomPanelHeight.value = window.innerHeight * 0.1
}

async function handleCloseSeries(): Promise<void> {
  await appStore.closeSeries()
  projectStore.clearProjects()
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

.main-splitter {
  flex: 1;
  min-height: 0;
  width: 100%;
  overflow: hidden;
}

:deep(.main-splitter .p-splitterpanel) {
  overflow: hidden !important;
  min-width: 0 !important;
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

/* Header */
.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  height: 36px;
  padding: 0 0.5rem;
  background: var(--surface-card);
  border-bottom: 1px solid var(--surface-border);
  flex-shrink: 0;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.logo {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.logo i {
  font-size: 1.5rem;
  color: var(--primary-color);
}

.logo-text {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--text-color);
  letter-spacing: 0.05em;
}

.close-series-btn {
  margin-left: 0.25rem;
  opacity: 0.85;
  transition: opacity 0.15s ease, background-color 0.15s ease, color 0.15s ease;
}

.close-series-btn:hover {
  opacity: 1;
  color: var(--text-color) !important;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

.title-bar-btn {
  height: 28px !important;
  min-height: 28px !important;
  font-size: 0.82rem;
  padding: 0 0.5rem !important;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: background-color 0.15s ease, color 0.15s ease;
}

.title-bar-btn.title-bar-btn-active {
  color: var(--primary-color) !important;
  background-color: var(--surface-hover) !important;
}

/* SplitButton in Title Bar */
.title-bar-splitbtn {
  height: 28px !important;
  min-height: 28px !important;
  display: inline-flex;
  align-items: center;
  vertical-align: middle;
}

:deep(.title-bar-splitbtn .p-splitbutton-button) {
  height: 28px !important;
  min-height: 28px !important;
  font-size: 0.82rem;
  padding: 0 0.4rem 0 0.5rem !important;
  display: inline-flex;
  align-items: center;
  justify-content: center;
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

:deep(.title-bar-splitbtn-active .p-splitbutton-button),
:deep(.title-bar-splitbtn-active .p-splitbutton-dropdown) {
  color: var(--primary-color) !important;
  background-color: var(--surface-hover) !important;
}

:deep(.title-bar-splitbtn .p-splitbutton-button:hover),
:deep(.title-bar-splitbtn .p-splitbutton-dropdown:hover) {
  background-color: var(--surface-hover) !important;
}

/* Main Splitter */
.main-splitter {
  flex: 1;
  overflow: hidden;
  border: none;
}

:deep(.p-splitter-gutter) {
  background-color: var(--surface-border) !important;
  width: 4px !important;
  transition: background-color 0.2s;
  cursor: col-resize;
  border-left: 1px solid var(--surface-hover);
  border-right: 1px solid var(--surface-hover);
}

:deep(.p-splitter-gutter:hover) {
  background-color: var(--primary-color) !important;
}

/* Bottom Panel */
.bottom-panel {
  display: flex;
  flex-direction: column;
  background: var(--surface-card);
  flex-shrink: 0;
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
.global-drag-overlay {
  position: fixed;
  inset: 0;
  z-index: 99999;
  background-color: rgba(0, 0, 0, 0.45);
  backdrop-filter: blur(2px);
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: auto;
  cursor: pointer;
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
