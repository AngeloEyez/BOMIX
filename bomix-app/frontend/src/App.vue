<template>
  <div class="app-container">
    <!-- Header / Title Bar -->
    <header class="header">
      <div class="header-left">
        <div class="logo">
          <i class="pi pi-box"></i>
          <span v-if="appStore.seriesInfo?.name" class="logo-text">
            {{ appStore.seriesInfo.name }}
          </span>
        </div>
      </div>
      <div class="header-right">
        <!-- 1. Main View (BOM) -->
        <Button
          icon="pi pi-home"
          label="BOM"
          text
          :severity="isMainViewActive ? 'primary' : 'secondary'"
          :class="['title-bar-btn', { 'title-bar-btn-active': isMainViewActive }]"
          @click="handleMainViewClick"
          title="Main View (BOM Table)"
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

        <!-- 3. Close Series (僅在系列開啟時顯示) -->
        <Button
          v-if="appStore.isOpen"
          icon="pi pi-sign-out"
          text
          severity="secondary"
          class="title-bar-btn"
          @click="handleCloseSeries"
          title="Close Series"
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
      </div>
    </header>

    <!-- Main Content with Splitter -->
    <Splitter class="main-splitter">
      <!-- Sidebar Panel -->
      <SplitterPanel
        :size="sidebarWidth"
        :min-size="0"
        @resize="onSidebarResize"
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
import Button from 'primevue/button'
import { useAppStore, useProjectStore, useLogStore, useTaskStore } from './stores'
import LogPanel from './components/LogPanel.vue'
import SidebarPanel from './components/SidebarPanel.vue'
import { GetSettings } from './services/api'

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

// Sidebar width management
const sidebarWidth = ref(20) // Default 20%

// Bottom panel height
const bottomPanelHeight = ref(window.innerHeight * 0.1) // Default 10%
let isResizingBottom = false
let startY = 0
let startHeight = 0

onMounted(async () => {
  // 註冊全域右鍵選單攔截
  window.addEventListener('contextmenu', handleGlobalContextMenu)

  // Start listening to events
  logStore.startListening()
  taskStore.startListening()

  // Load initial settings for theme and log level
  try {
    const s = await GetSettings()
    appStore.applyTheme(s.theme)
    useLogStore().globalLogLevel = s.logger?.level || 'info'
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

function onSidebarResize(event: any): void {
  sidebarWidth.value = event.size || event
}

function resetSidebarWidth(): void {
  sidebarWidth.value = 20
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
</style>
