<template>
  <div class="settings-page">
    <!-- 左側樹狀導覽 Panel (VSCode Style Navigation) -->
    <aside class="settings-sidebar">
      <div class="sidebar-top-title">
        <i class="pi pi-cog text-primary-500"></i>
        <span>Settings</span>
      </div>

      <div class="tree-nav-container">
        <Tree
          :value="navTreeNodes"
          v-model:selection-keys="selectedTreeKeys"
          selection-mode="single"
          class="vscode-tree"
          @node-select="onNodeSelect"
        >
          <template #nodeicon="slotProps">
            <i
              :class="[
                slotProps.node.icon,
                'node-icon',
                { 'node-icon-active': selectedTreeKeys[slotProps.node.key] }
              ]"
            ></i>
          </template>
          <template #default="slotProps">
            <span
              :class="[
                'node-label',
                { 'node-label-active': selectedTreeKeys[slotProps.node.key] }
              ]"
            >
              {{ slotProps.node.label }}
            </span>
          </template>
        </Tree>
      </div>
    </aside>

    <!-- 右側垂直捲動設定主要區域 -->
    <main
      ref="scrollContainerRef"
      class="settings-main-scroll"
      @scroll="handleRightScroll"
    >
      <div class="settings-content-wrapper">
        <!-- 1. Appearance -->
        <AppearanceSettings v-model="settings.theme" />

        <Divider class="section-divider" />

        <!-- 2. General -->
        <GeneralSettings
          v-model:auto-open-last-file="settings.autoOpenLastFile"
          v-model:max-recent-files="settings.recentFiles.maxRecentFiles"
          :last-opened-file="settings.lastOpenedFile"
        />

        <Divider class="section-divider" />

        <!-- 3. Import -->
        <ImportSettings
          v-model:confirm-overwrite="settings.import.confirmOverwrite"
          v-model:auto-import-previous-matrix="settings.import.autoImportPreviousMatrix"
        />

        <Divider class="section-divider" />

        <!-- 4. Logger -->
        <LoggerSettings
          v-model:level="settings.logger.level"
          v-model:max-entries="settings.logger.maxEntries"
        />

        <Divider class="section-divider" />

        <!-- 5. AI Assistant -->
        <AISettingsPanel
          v-model="settings.ai"
          @save-settings="handleImmediateSave"
        />
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
/**
 * @file SettingsPage.vue
 * @description 系統設定主頁面骨架（Shell）
 * 負責左側 VSCode 風格樹狀導覽、右側雙向滾動監聽（ScrollSpy）以及集中式全域防抖自動存檔。
 */
import { ref, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import Tree from 'primevue/tree'
import Divider from 'primevue/divider'
import { type Settings, type AISettings, UpdateSettings } from '../services/api'
import { useAIChatStore } from '../stores/aiChat'
import { useSettingsStore } from '../stores/settings'

// 子設定區塊元件
import AppearanceSettings from '../components/settings/AppearanceSettings.vue'
import GeneralSettings from '../components/settings/GeneralSettings.vue'
import ImportSettings from '../components/settings/ImportSettings.vue'
import LoggerSettings from '../components/settings/LoggerSettings.vue'
import AISettingsPanel from '../components/settings/AISettings.vue'

const router = useRouter()
const route = useRoute()
const aiChatStore = useAIChatStore()
const settingsStore = useSettingsStore()

/**
 * 內部表單設定介面，確保 AI 欄位完整存在
 */
interface SettingsForm extends Settings {
  ai: AISettings
}

/**
 * 導航樹節點定義 (支援未來最多擴充至三階)
 */
interface NavTreeNode {
  key: string
  label: string
  icon?: string
  children?: NavTreeNode[]
}

// 左側樹狀導覽節點資料
const navTreeNodes = ref<NavTreeNode[]>([
  {
    key: 'appearance',
    label: 'Appearance',
    icon: 'pi pi-palette',
  },
  {
    key: 'general',
    label: 'General',
    icon: 'pi pi-sliders-h',
  },
  {
    key: 'import',
    label: 'Import',
    icon: 'pi pi-upload',
  },
  {
    key: 'logger',
    label: 'Logger',
    icon: 'pi pi-list',
  },
  {
    key: 'ai',
    label: 'AI Assistant',
    icon: 'pi pi-sparkles',
  },
])

// 樹狀選取狀態，預設選中 appearance
const selectedTreeKeys = ref<Record<string, boolean>>({
  appearance: true,
})

// 右側可捲動容器參考
const scrollContainerRef = ref<HTMLElement | null>(null)
let isProgrammaticScroll = false
let programmaticScrollTimer: any = null

/**
 * 點擊樹狀節點，右側平滑捲動到對應一階分類，並即刻更新導航欄選中狀態
 * @param {any} node - 選取的樹節點
 */
function onNodeSelect(node: any): void {
  if (!node?.key) return
  const key = node.key

  // 1. 立即更新左側導航欄的選中狀態，確保 highlight 即時切換
  selectedTreeKeys.value = { [key]: true }

  // 2. 平滑滾動至右側對應章節
  const elementId = `category-${key}`
  const targetElement = document.getElementById(elementId)
  if (targetElement && scrollContainerRef.value) {
    isProgrammaticScroll = true
    targetElement.scrollIntoView({ behavior: 'smooth', block: 'start' })

    // 給予平滑滾動足夠的緩衝時間，結束後恢復 Scroll Spy
    if (programmaticScrollTimer) clearTimeout(programmaticScrollTimer)
    programmaticScrollTimer = setTimeout(() => {
      isProgrammaticScroll = false
    }, 750)
  }
}

/**
 * 右側滾動時雙向更新左側樹狀高亮 (Scroll Spy)
 */
function handleRightScroll(): void {
  if (isProgrammaticScroll || !scrollContainerRef.value) return

  const container = scrollContainerRef.value
  const containerRect = container.getBoundingClientRect()

  const categories = ['appearance', 'general', 'import', 'logger', 'ai']
  let currentKey = categories[0]

  for (const key of categories) {
    const el = document.getElementById(`category-${key}`)
    if (el) {
      const elRect = el.getBoundingClientRect()
      // 計算相對於滾動容器可視頂部的距離
      const relativeTop = elRect.top - containerRect.top
      // 當章節頂部接近或已經進入頂部 100px 以內時判定為目前閱讀分類
      if (relativeTop <= 100) {
        currentKey = key
      }
    }
  }

  // 若當前高亮與當前計算出的分類不同則更新
  if (!selectedTreeKeys.value[currentKey]) {
    selectedTreeKeys.value = { [currentKey]: true }
  }
}

// 建立表單初始結構 (優先自 settingsStore 複製，杜絕寫死業務預設值)
function getInitialFormData(): SettingsForm {
  const s = settingsStore.currentSettings || settingsStore.defaults
  if (s) {
    return JSON.parse(JSON.stringify(s)) as SettingsForm
  }
  return {
    theme: 'light',
    import: { confirmOverwrite: true, autoImportPreviousMatrix: true },
    logger: { level: 'info', maxEntries: 500 },
    recentFiles: { maxRecentFiles: 10, recentFiles: [] },
    autoOpenLastFile: false,
    lastOpenedFile: '',
    autoImportPreviousMatrix: true,
    ai: {
      enabled: false,
      baseUrl: 'https://api.openai.com/v1',
      apiKey: '',
      model: '',
      temperature: 0.1,
      maxTokens: 4096,
      maxIterations: 8,
      timeout: 60,
      language: 'zh-TW',
    },
  }
}

// Settings state
const settings = ref<SettingsForm>(getInitialFormData())

/**
 * 供子元件觸發的立即存檔（如 AI 連線測試前即時同步設定至後端）
 */
async function handleImmediateSave(): Promise<void> {
  if (saveTimeout) clearTimeout(saveTimeout)
  try {
    await UpdateSettings(settings.value)
    await settingsStore.saveSettings(settings.value)
  } catch (error) {
    console.error('Failed to immediate save settings:', error)
  }
}

/**
 * 從後端載入使用者設定 (SSOT 單一真實來源)
 */
async function loadSettings(): Promise<boolean> {
  try {
    const data = await settingsStore.initSettings()
    if (!data) return false

    settings.value = JSON.parse(JSON.stringify(data)) as SettingsForm

    if (settings.value.ai.model) {
      aiChatStore.currentModel = settings.value.ai.model
    }
    if (aiChatStore.availableModels.length === 0 && settings.value.ai.baseUrl) {
      aiChatStore.fetchAvailableModels(settings.value.ai.baseUrl, settings.value.ai.apiKey)
    }

    return true
  } catch (error) {
    console.error('Failed to load settings:', error)
    return false
  }
}

// Auto-save logic
let saveTimeout: any
let isLoaded = false

watch(settings, (newVal) => {
  if (!isLoaded || !settingsStore.isLoaded) return

  if (!newVal.theme || !newVal.logger?.level || !newVal.logger?.maxEntries) {
    console.warn('Ignore auto-save: invalid or incomplete settings payload', newVal)
    return
  }

  if (newVal.import) {
    newVal.autoImportPreviousMatrix = newVal.import.autoImportPreviousMatrix
  }

  if (newVal.ai) {
    if (!newVal.ai.enabled && route.path === '/ai-chat') {
      router.push('/workspace')
    }
  }

  if (saveTimeout) clearTimeout(saveTimeout)
  saveTimeout = setTimeout(async () => {
    try {
      await settingsStore.saveSettings(newVal)
    } catch (error) {
      console.error('Failed to auto-save settings:', error)
    }
  }, 500)
}, { deep: true })

onMounted(async () => {
  const success = await loadSettings()
  if (success) {
    setTimeout(() => { isLoaded = true }, 200)
  }
})
</script>

<style scoped>
/* 頁面外層左右兩欄結構 */
.settings-page {
  display: flex;
  flex-direction: row;
  height: 100%;
  width: 100%;
  overflow: hidden;
  background: var(--surface-ground);
}

/* ==================== 左側導航 Panel ==================== */
.settings-sidebar {
  width: 220px;
  min-width: 220px;
  max-width: 220px;
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--surface-section, var(--surface-card));
  border-right: 1px solid var(--surface-border);
  flex-shrink: 0;
  user-select: none;
}

.sidebar-top-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.85rem 1rem 0.65rem 1rem;
  font-size: 0.82rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--text-color);
  border-bottom: 1px solid var(--surface-border);
}

.tree-nav-container {
  flex: 1;
  overflow-y: auto;
  padding: 0.4rem 0.25rem;
}

/* PrimeVue Tree 客製化為 VSCode 風格 */
:deep(.vscode-tree) {
  background: transparent !important;
  border: none !important;
  padding: 0 !important;
}

:deep(.vscode-tree .p-tree-node-content) {
  padding: 0.35rem 0.6rem !important;
  border-radius: 4px !important;
  transition: background-color 0.15s ease;
  display: flex !important;
  align-items: center !important;
  gap: 0.45rem !important;
}

:deep(.vscode-tree .p-tree-node-content:hover) {
  background-color: var(--surface-hover) !important;
}

:deep(.vscode-tree .p-tree-node-content.p-tree-node-selected),
:deep(.vscode-tree .p-tree-node-content[data-p-selected="true"]),
:deep(.vscode-tree .p-tree-node[data-p-selected="true"] > .p-tree-node-content),
:deep(.vscode-tree .p-tree-node[aria-selected="true"] > .p-tree-node-content) {
  background-color: var(--primary-50, rgba(99, 102, 241, 0.12)) !important;
  color: var(--primary-color) !important;
  font-weight: 600 !important;
  border-left: 2.5px solid var(--primary-color) !important;
}

.node-icon {
  font-size: 0.85rem;
  opacity: 0.85;
}

.node-icon-active {
  color: var(--primary-color) !important;
  opacity: 1 !important;
}

.node-label {
  font-size: 0.82rem;
  letter-spacing: 0.01em;
}

.node-label-active {
  color: var(--primary-color) !important;
  font-weight: 600 !important;
}

/* ==================== 右側設定主要捲動區 ==================== */
.settings-main-scroll {
  flex: 1;
  min-width: 0;
  height: 100%;
  overflow-y: auto;
  scroll-behavior: smooth;
  padding: 1.5rem 2rem 4rem 2rem;
}

/* 設定內容寬度限制 (避免主 UI 太寬造成不易閱讀) */
.settings-content-wrapper {
  max-width: 820px;
  margin-left: 0;
  display: flex;
  flex-direction: column;
  gap: 2.2rem;
}

/* 分割線 */
.section-divider {
  margin: 0.5rem 0 !important;
  opacity: 0.5;
}
</style>
