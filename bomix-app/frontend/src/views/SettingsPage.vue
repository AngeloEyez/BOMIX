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
        <!-- ==================== 1. Appearance ==================== -->
        <section id="category-appearance" class="category-section">
          <div class="category-header">
            <div class="category-title">
              <i class="pi pi-palette category-icon"></i>
              <h2>Appearance</h2>
            </div>
            <p class="category-desc">自訂應用程式的視覺主題與介面外觀風格。</p>
          </div>

          <!-- Theme Mode -->
          <div class="vscode-setting-item">
            <div class="setting-title-line">
              <span class="setting-name">Theme Mode</span>
            </div>
            <p class="setting-desc">
              控制全域介面的色彩主題。可選擇淺色 (Light)、深色 (Dark) 或依照系統偏好設定 (System)。
            </p>
            <div class="setting-control">
              <SelectButton
                v-model="settings.theme"
                :options="themeOptions"
                option-label="label"
                option-value="value"
                :allow-empty="false"
                size="small"
                class="theme-select-toggle"
              />
            </div>
          </div>
        </section>

        <Divider class="section-divider" />

        <!-- ==================== 2. General ==================== -->
        <section id="category-general" class="category-section">
          <div class="category-header">
            <div class="category-title">
              <i class="pi pi-sliders-h category-icon"></i>
              <h2>General</h2>
            </div>
            <p class="category-desc">配置系統啟動行為與專案檔案歷程管理設定。</p>
          </div>

          <!-- Auto Open Last File -->
          <div class="vscode-setting-item">
            <div class="setting-title-line switch-title-line">
              <ToggleSwitch
                input-id="auto-open-last-file"
                v-model="settings.autoOpenLastFile"
              />
              <label for="auto-open-last-file" class="setting-name switch-label">
                Auto Open Last File
              </label>
            </div>
            <p class="setting-desc">
              啟動應用程式時，自動開啟上次最後使用的 BOM 系列專案檔 (.bomx)。
            </p>
            <div v-if="settings.lastOpenedFile" class="last-opened-info">
              <span class="text-xs text-[var(--text-color-secondary)]">上次開啟：</span>
              <Badge :value="settings.lastOpenedFile" severity="secondary" class="last-file-badge" />
            </div>
          </div>

          <!-- Max Recent Files Count -->
          <div class="vscode-setting-item">
            <div class="setting-title-line">
              <span class="setting-name">Max Recent Files Count</span>
            </div>
            <p class="setting-desc">
              控制歡迎頁面中「最近開啟檔案」歷史清單所保留的最大筆數。
            </p>
            <div class="setting-control">
              <InputNumber
                id="max-recent"
                v-model="settings.recentFiles.maxRecentFiles"
                :show-buttons="true"
                :min="1"
                :max="50"
                size="small"
                class="compact-input-number"
              />
            </div>
          </div>
        </section>

        <Divider class="section-divider" />

        <!-- ==================== 3. Import Settings ==================== -->
        <section id="category-import" class="category-section">
          <div class="category-header">
            <div class="category-title">
              <i class="pi pi-upload category-icon"></i>
              <h2>Import</h2>
            </div>
            <p class="category-desc">設定解析與合併 Excel BOM 版本時的處理規則與行為。</p>
          </div>

          <!-- Confirm Overwrite -->
          <div class="vscode-setting-item">
            <div class="setting-title-line switch-title-line">
              <ToggleSwitch
                input-id="confirm-overwrite"
                v-model="settings.import.confirmOverwrite"
              />
              <label for="confirm-overwrite" class="setting-name switch-label">
                Confirm Overwrite
              </label>
            </div>
            <p class="setting-desc">
              匯入 Excel BOM 時，若偵測到相同版本號則主動提示確認，防止意外覆蓋現有資料。
            </p>
          </div>

          <!-- Auto Import Previous Matrix -->
          <div class="vscode-setting-item">
            <div class="setting-title-line switch-title-line">
              <ToggleSwitch
                input-id="auto-import-matrix"
                v-model="settings.import.autoImportPreviousMatrix"
              />
              <label for="auto-import-matrix" class="setting-name switch-label">
                Auto Import Previous Matrix
              </label>
            </div>
            <p class="setting-desc">
              匯入全新 EBOM 版本時，自動搜尋同專案上一版本的 Matrix 勾選設定並帶入沿用。
            </p>
          </div>
        </section>

        <Divider class="section-divider" />

        <!-- ==================== 4. Logger ==================== -->
        <section id="category-logger" class="category-section">
          <div class="category-header">
            <div class="category-title">
              <i class="pi pi-list category-icon"></i>
              <h2>Logger</h2>
            </div>
            <p class="category-desc">管理執行時期日誌輸出的詳細程度與記憶體緩衝區容量。</p>
          </div>

          <!-- Log Level -->
          <div class="vscode-setting-item">
            <div class="setting-title-line">
              <span class="setting-name">Log Level</span>
            </div>
            <p class="setting-desc">
              設定底部日誌面板顯示的最低嚴重性層級門檻 (Debug / Info / Warning / Error)。
            </p>
            <div class="setting-control">
              <Select
                id="log-level"
                v-model="settings.logger.level"
                :options="logLevelOptions"
                option-label="label"
                option-value="value"
                placeholder="選擇日誌層級"
                size="small"
                class="compact-select"
              />
            </div>
          </div>

          <!-- Max Log Entries -->
          <div class="vscode-setting-item">
            <div class="setting-title-line">
              <span class="setting-name">Max Log Entries</span>
            </div>
            <p class="setting-desc">
              控制記憶體環形緩衝區（Ring Buffer）保留的日誌記錄最大筆數，超出上限將自動覆蓋最舊紀錄。
            </p>
            <div class="setting-control">
              <InputNumber
                id="max-entries"
                v-model="settings.logger.maxEntries"
                :show-buttons="true"
                :min="100"
                :max="5000"
                :step="100"
                size="small"
                class="compact-input-number"
              />
            </div>
          </div>
        </section>

        <Divider class="section-divider" />

        <!-- ==================== 5. AI Assistant ==================== -->
        <section id="category-ai" class="category-section">
          <div class="category-header">
            <div class="category-title">
              <i class="pi pi-sparkles category-icon text-primary-500"></i>
              <h2>AI Assistant</h2>
            </div>
            <p class="category-desc">配置大型語言模型（LLM）端點、金鑰憑證、推論參數與對話互動行為。</p>
          </div>

          <!-- Enable AI Assistant -->
          <div class="vscode-setting-item">
            <div class="setting-title-line switch-title-line">
              <ToggleSwitch
                input-id="ai-enabled"
                v-model="settings.ai.enabled"
              />
              <label for="ai-enabled" class="setting-name switch-label">
                Enable Assistant
              </label>
            </div>
            <p class="setting-desc">
              啟用 AI 智慧對話助手，支援自然語言問答、BOM 資料比對分析與多功能對話輔助。
            </p>
          </div>

          <!-- AI 細部設定群組 (當 Enable Assistant 關閉時整體淡化且禁用) -->
          <div :class="['ai-sub-settings-container', { 'is-disabled': !settings.ai.enabled }]">
            <!-- Response Language -->
            <div class="vscode-setting-item">
              <div class="setting-title-line">
                <span class="setting-name">Response Language</span>
              </div>
              <p class="setting-desc">
                設定 AI 助手在回答問題、產生分析報告時優先使用的自然語言。
              </p>
              <div class="setting-control">
                <Select
                  id="ai-language"
                  v-model="settings.ai.language"
                  :options="aiLanguageOptions"
                  option-label="label"
                  option-value="value"
                  :disabled="!settings.ai.enabled"
                  size="small"
                  class="compact-select-wide"
                />
              </div>
            </div>

            <!-- API Base URL -->
            <div class="vscode-setting-item">
              <div class="setting-title-line">
                <span class="setting-name">API Base URL</span>
              </div>
              <p class="setting-desc">
                相容於 OpenAI 規格的 API 端點基礎網址（如 <code>https://api.openai.com/v1</code> 或本地自架/反向代理伺服器）。
              </p>
              <div class="setting-control">
                <InputText
                  id="ai-base-url"
                  v-model="settings.ai.baseUrl"
                  placeholder="https://api.openai.com/v1"
                  :disabled="!settings.ai.enabled"
                  size="small"
                  class="compact-input-text-wide"
                  @blur="onBaseUrlBlur"
                />
              </div>
            </div>

            <!-- API Key -->
            <div class="vscode-setting-item">
              <div class="setting-title-line">
                <span class="setting-name">API Key</span>
              </div>
              <p class="setting-desc">
                用於向 API 伺服器進行身分驗證的金鑰憑證，安全儲存於本機工作站設定檔中。
              </p>
              <div class="setting-control">
                <Password
                  id="ai-api-key"
                  v-model="settings.ai.apiKey"
                  placeholder="sk-..."
                  :feedback="false"
                  :disabled="!settings.ai.enabled"
                  toggle-mask
                  size="small"
                  class="compact-password"
                  input-class="compact-password-input"
                />
              </div>
            </div>

            <!-- Connection Test Action -->
            <div class="vscode-setting-item test-connection-item">
              <div class="setting-title-line">
                <span class="setting-name">Connection Test</span>
              </div>
              <p class="setting-desc">
                即時測試網路連線狀態，並驗證目前的端點網址與 API 金鑰是否有效。
              </p>
              <div class="test-connection-action-row">
                <Button
                  label="測試連線"
                  icon="pi pi-check-circle"
                  size="small"
                  severity="secondary"
                  outlined
                  class="test-btn-neutral"
                  :loading="isTestingAI"
                  :disabled="!settings.ai.enabled || isTestingAI"
                  @click="testConnection"
                />
                <span v-if="testAIResult?.success" class="test-success-msg">
                  <i class="pi pi-check-circle"></i>
                  {{ testAIResult.message }}
                </span>
                <span v-else-if="testAIResult && !testAIResult.success" class="test-fail-msg">
                  <i class="pi pi-exclamation-circle"></i>
                  {{ testAIResult.message }}
                </span>
              </div>
            </div>

            <!-- Model Selection -->
            <div class="vscode-setting-item">
              <div class="setting-title-line flex items-center justify-between">
                <div>
                  <span class="setting-name">Model</span>
                </div>
                <button
                  type="button"
                  class="refresh-model-icon-btn"
                  title="重新整理伺服器可用模型清單"
                  :disabled="!settings.ai.enabled || aiChatStore.isLoadingModels || !settings.ai.baseUrl"
                  @click="onRefreshModels"
                >
                  <i :class="['pi', aiChatStore.isLoadingModels ? 'pi-spin pi-spinner' : 'pi-refresh']"></i>
                </button>
              </div>
              <p class="setting-desc">
                從伺服器拉取的模型清單中挑選，或直接鍵入任意自訂模型代碼（如 gpt-4o, deepseek-chat）。
              </p>
              <div class="setting-control">
                <Select
                  id="ai-model"
                  v-model="settings.ai.model"
                  :options="modelOptions"
                  option-label="label"
                  option-value="value"
                  editable
                  :disabled="!settings.ai.enabled"
                  placeholder="選擇或直接輸入模型代碼 (如 gpt-4o, deepseek-chat)..."
                  size="small"
                  class="compact-select-wide"
                  @change="onModelChange"
                  @blur="onModelChange"
                />
              </div>
            </div>

            <!-- Timeout -->
            <div class="vscode-setting-item">
              <div class="setting-title-line">
                <span class="setting-name">Timeout</span>
              </div>
              <p class="setting-desc">
                呼叫 AI 模型生成回應的逾時上限（秒），超過設定時間未回應將中止請求。
              </p>
              <div class="setting-control">
                <InputNumber
                  id="ai-timeout"
                  v-model="settings.ai.timeout"
                  :show-buttons="true"
                  :min="5"
                  :max="300"
                  :step="5"
                  :disabled="!settings.ai.enabled"
                  size="small"
                  class="compact-input-number"
                />
              </div>
            </div>

            <!-- Temperature -->
            <div class="vscode-setting-item">
              <div class="setting-title-line">
                <span class="setting-name">Temperature</span>
              </div>
              <p class="setting-desc">
                模型採樣隨機性係數（0.0 - 2.0）。數值越低回答越聚焦嚴謹；數值越高回答越具多樣性。
              </p>
              <div class="setting-control">
                <InputNumber
                  id="ai-temperature"
                  v-model="settings.ai.temperature"
                  :show-buttons="true"
                  :min="0"
                  :max="2"
                  :step="0.1"
                  :max-fraction-digits="2"
                  :disabled="!settings.ai.enabled"
                  size="small"
                  class="compact-input-number"
                />
              </div>
            </div>

            <!-- Max Tokens -->
            <div class="vscode-setting-item">
              <div class="setting-title-line">
                <span class="setting-name">Max Output Tokens</span>
              </div>
              <p class="setting-desc">
                單次呼叫模型生成回覆時允許輸出的最大 Token 長度上限。
              </p>
              <div class="setting-control">
                <InputNumber
                  id="ai-max-tokens"
                  v-model="settings.ai.maxTokens"
                  :show-buttons="true"
                  :min="256"
                  :max="32768"
                  :step="512"
                  :disabled="!settings.ai.enabled"
                  size="small"
                  class="compact-input-number"
                />
              </div>
            </div>

            <!-- Max Iterations (推論輪次上限) -->
            <div class="vscode-setting-item">
              <div class="setting-title-line">
                <span class="setting-name">Max Reasoning Iterations</span>
              </div>
              <p class="setting-desc">
                AI 助手推論思考與工具調用的最大迭代輪次上限（預設 8 輪，範圍 1 - 30 輪）。
              </p>
              <div class="setting-control">
                <InputNumber
                  id="ai-max-iterations"
                  v-model="settings.ai.maxIterations"
                  :show-buttons="true"
                  :min="1"
                  :max="30"
                  :step="1"
                  :disabled="!settings.ai.enabled"
                  size="small"
                  class="compact-input-number"
                />
              </div>
            </div>
          </div>
        </section>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import Tree from 'primevue/tree'
import Select from 'primevue/select'
import SelectButton from 'primevue/selectbutton'
import ToggleSwitch from 'primevue/toggleswitch'
import InputNumber from 'primevue/inputnumber'
import InputText from 'primevue/inputtext'
import Password from 'primevue/password'
import Button from 'primevue/button'
import Badge from 'primevue/badge'
import Divider from 'primevue/divider'
import { GetDefaultSettings, UpdateSettings, AIChatTestConnection, type Settings, type AISettings } from '../services/api'
import { useAppStore } from '../stores/app'
import { useLogStore } from '../stores/log'
import { useAIChatStore } from '../stores/aiChat'
import { useSettingsStore } from '../stores/settings'

const router = useRouter()
const route = useRoute()
const appStore = useAppStore()
const logStore = useLogStore()
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

// 左側樹狀導覽節點資料 (目前現有設定為一階大分類)
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

// 可用模型選項 (與 aiChatStore.availableModels 雙向同步)
const modelOptions = computed(() => {
  const list = aiChatStore.availableModels.map(m => ({ label: m, value: m }))
  const current = settings.value.ai?.model?.trim()
  if (current && !list.some(item => item.value === current)) {
    return [{ label: current, value: current }, ...list]
  }
  return list
})

/**
 * Base URL 失焦時自動拉取伺服器可用模型
 */
async function onBaseUrlBlur(): Promise<void> {
  const url = settings.value.ai?.baseUrl?.trim()
  if (url) {
    await aiChatStore.fetchAvailableModels(url, settings.value.ai?.apiKey)
  }
}

/**
 * 點擊手動重新整理模型
 */
async function onRefreshModels(): Promise<void> {
  const url = settings.value.ai?.baseUrl?.trim()
  if (url) {
    await aiChatStore.fetchAvailableModels(url, settings.value.ai?.apiKey)
  }
}

/**
 * 模型變更時同步目前選擇給 aiChatStore
 */
function onModelChange(): void {
  const model = settings.value.ai?.model?.trim()
  if (model) {
    aiChatStore.currentModel = model
  }
}

// Theme options
const themeOptions = [
  { label: 'Light', value: 'light' },
  { label: 'Dark', value: 'dark' },
  { label: 'System', value: 'system' },
]

// Log level options
const logLevelOptions = [
  { label: 'Debug', value: 'debug' },
  { label: 'Info', value: 'info' },
  { label: 'Warning', value: 'warn' },
  { label: 'Error', value: 'error' },
]

// AI 回應語言選項
const aiLanguageOptions = [
  { label: '繁體中文 (Traditional Chinese)', value: 'zh-TW' },
  { label: '簡體中文 (Simplified Chinese)', value: 'zh-CN' },
  { label: 'English', value: 'en' },
]

const isTestingAI = ref(false)
const testAIResult = ref<{ success: boolean; message: string } | null>(null)

/**
 * 測試與 AI 伺服器的連線
 */
async function testConnection(): Promise<void> {
  isTestingAI.value = true
  testAIResult.value = null
  try {
    await UpdateSettings(settings.value)
    await AIChatTestConnection()
    const models = await aiChatStore.fetchAvailableModels(settings.value.ai.baseUrl, settings.value.ai.apiKey)
    if (models.length > 0) {
      testAIResult.value = { success: true, message: `連線成功！已同步 ${models.length} 個可用伺服器模型。` }
    } else {
      testAIResult.value = { success: true, message: '連線成功！API 端點與金鑰有效。' }
    }
  } catch (err: any) {
    testAIResult.value = { success: false, message: err?.message || '連線測試失敗' }
  } finally {
    isTestingAI.value = false
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

/* 大分類區塊 */
.category-section {
  display: flex;
  flex-direction: column;
  gap: 1.4rem;
  scroll-margin-top: 1rem;
}

.category-header {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
  border-bottom: 1px solid var(--surface-border);
  padding-bottom: 0.6rem;
}

.category-title {
  display: flex;
  align-items: center;
  gap: 0.55rem;
}

.category-title h2 {
  font-size: 1.35rem;
  font-weight: 700;
  color: var(--text-color);
  margin: 0;
  line-height: 1.25;
}

.category-icon {
  font-size: 1.15rem;
  color: var(--primary-color);
}

.category-desc {
  font-size: 0.82rem;
  color: var(--text-color-secondary);
  margin: 0;
}

/* 分割線 */
.section-divider {
  margin: 0.5rem 0 !important;
  opacity: 0.5;
}

/* ==================== VSCode 設定項目條目格式 ==================== */
.vscode-setting-item {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.setting-title-line {
  display: flex;
  align-items: baseline;
  gap: 0.4rem;
  font-size: 0.85rem;
}

.setting-name {
  font-weight: 700;
  color: var(--text-color);
}

.setting-desc {
  font-size: 0.78rem;
  color: var(--text-color-secondary);
  line-height: 1.45;
  margin: 0;
  max-width: 720px;
}

.setting-desc code {
  background: var(--surface-hover);
  padding: 0.1rem 0.3rem;
  border-radius: 3px;
  font-size: 0.75rem;
}

.setting-control {
  margin-top: 0.2rem;
}

/* ToggleSwitch 形式設定排版 (放置於標題左側) */
.switch-title-line {
  display: flex;
  align-items: center;
  gap: 0.65rem;
}

:deep(.switch-title-line .p-toggleswitch) {
  flex-shrink: 0;
}

.switch-label {
  cursor: pointer;
  user-select: none;
}

.last-opened-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-top: 0.35rem;
  padding-left: 1.6rem;
}

.last-file-badge {
  max-width: 500px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: monospace;
  font-size: 0.72rem;
}

/* Theme 切換按鈕：仿照 BOMTable 中的 EBOM/Matrix 切換按鈕樣式 (Primary Color 激活) */
:deep(.theme-select-toggle.p-selectbutton),
:deep(.theme-select-toggle) {
  background: transparent !important;
  border: 1px solid var(--surface-border) !important;
  box-shadow: none !important;
  border-radius: 4px !important;
  padding: 1.5px !important;
  gap: 2px !important;
  display: inline-flex !important;
  align-items: center !important;
  height: 26px !important;
  box-sizing: border-box !important;
}

/* 移除 PrimeVue 4 ToggleButton 內層 .p-togglebutton-content 預設的白底與陰影滑塊 */
:deep(.theme-select-toggle .p-togglebutton-content),
:deep(.theme-select-toggle .p-togglebutton.p-togglebutton-checked .p-togglebutton-content),
:deep(.theme-select-toggle .p-togglebutton[data-p-checked="true"] .p-togglebutton-content) {
  background: transparent !important;
  box-shadow: none !important;
  padding: 0 !important;
}

:deep(.theme-select-toggle .p-togglebutton) {
  padding: 0 0.65rem !important;
  font-size: 0.76rem !important;
  font-weight: 600;
  height: 21px !important;
  border-radius: 2.5px !important;
  border: none !important;
  background-color: transparent !important;
  color: var(--text-color-secondary) !important;
  transition: background-color 0.15s ease, color 0.15s ease;
  display: inline-flex !important;
  align-items: center !important;
  justify-content: center !important;
}

/* 未選中時 hover：淺灰底色 */
:deep(.theme-select-toggle .p-togglebutton:not(.p-togglebutton-checked):not([data-p-checked="true"]):hover) {
  background-color: var(--surface-hover) !important;
  color: var(--text-color) !important;
}

/* 切換 Theme 的 SelectButton 選中狀態：Primary color 底白字 */
:deep(.theme-select-toggle .p-togglebutton.p-togglebutton-checked),
:deep(.theme-select-toggle .p-togglebutton[data-p-checked="true"]) {
  color: #ffffff !important;
  background-color: var(--primary-color) !important;
  font-weight: 600 !important;
}

/* 選中時 hover：微深的主色 */
:deep(.theme-select-toggle .p-togglebutton.p-togglebutton-checked:hover),
:deep(.theme-select-toggle .p-togglebutton[data-p-checked="true"]:hover) {
  background-color: color-mix(in srgb, var(--primary-color) 85%, black) !important;
  color: #ffffff !important;
}

:deep(.theme-select-toggle .p-togglebutton .p-togglebutton-label) {
  color: inherit !important;
}

:deep(.theme-select-toggle .p-togglebutton.p-togglebutton-checked .p-togglebutton-label),
:deep(.theme-select-toggle .p-togglebutton[data-p-checked="true"] .p-togglebutton-label) {
  color: #ffffff !important;
}

/* 其他控制元件尺寸規格 */
.compact-select {
  width: 160px !important;
  height: 28px !important;
  font-size: 0.8rem;
}

:deep(.compact-select .p-select-label) {
  padding: 0.2rem 0.55rem !important;
  font-size: 0.8rem !important;
}

.compact-select-wide {
  width: 280px !important;
  height: 28px !important;
  font-size: 0.8rem;
}

:deep(.compact-select-wide .p-select-label) {
  padding: 0.2rem 0.55rem !important;
  font-size: 0.8rem !important;
}

.compact-input-number {
  width: 140px !important;
  height: 28px !important;
}

:deep(.compact-input-number.p-inputnumber) {
  position: relative !important;
  display: inline-flex !important;
  width: 140px !important;
  height: 28px !important;
  box-sizing: border-box !important;
}

:deep(.compact-input-number .p-inputnumber-input) {
  width: 100% !important;
  height: 28px !important;
  padding: 0.15rem 1.6rem 0.15rem 0.55rem !important;
  font-size: 0.8rem !important;
  box-sizing: border-box !important;
}

:deep(.compact-input-number .p-inputnumber-button-group) {
  position: absolute !important;
  top: 1px !important;
  right: 1px !important;
  bottom: 1px !important;
  width: 18px !important;
  height: calc(100% - 2px) !important;
  display: flex !important;
  flex-direction: column !important;
  z-index: 2 !important;
}

:deep(.compact-input-number .p-inputnumber-button) {
  width: 100% !important;
  height: 50% !important;
  padding: 0 !important;
  display: flex !important;
  align-items: center !important;
  justify-content: center !important;
}

:deep(.compact-input-number .p-inputnumber-button .p-icon) {
  font-size: 0.55rem !important;
  width: 0.55rem !important;
  height: 0.55rem !important;
}

.compact-input-text-wide {
  width: 380px !important;
  max-width: 100% !important;
  height: 28px !important;
  padding: 0.15rem 0.55rem !important;
  font-size: 0.8rem !important;
}

.compact-password {
  width: 380px !important;
  max-width: 100% !important;
  height: 28px !important;
}

:deep(.compact-password .p-password-input) {
  width: 100% !important;
  height: 28px !important;
  padding: 0.15rem 2rem 0.15rem 0.55rem !important;
  font-size: 0.8rem !important;
}

.refresh-model-icon-btn {
  font-size: 0.85rem;
  color: var(--text-color-secondary);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  cursor: pointer;
  background: transparent;
  border: none;
  border-radius: 4px;
  transition: color 0.15s ease, background-color 0.15s ease;
}

.refresh-model-icon-btn:hover:not(:disabled) {
  color: var(--text-color);
  background: var(--surface-hover);
}

.refresh-model-icon-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

.test-connection-action-row {
  display: flex;
  align-items: center;
  gap: 1.5rem;
  margin-top: 0.35rem;
}

.test-btn-neutral {
  height: 28px !important;
  background: var(--surface-card) !important;
  border: 1px solid var(--surface-border) !important;
  color: var(--text-color) !important;
  font-size: 0.8rem !important;
  padding: 0 0.75rem !important;
  border-radius: 4px !important;
  transition: all 0.15s ease !important;
}

.test-btn-neutral:hover:not(:disabled) {
  background: var(--surface-hover) !important;
  border-color: var(--surface-border-hover, #94a3b8) !important;
  color: var(--text-color) !important;
}

:deep(.test-btn-neutral .p-button-icon) {
  color: var(--text-color-secondary) !important;
  font-size: 0.82rem !important;
}

:deep(.test-btn-neutral:hover:not(:disabled) .p-button-icon) {
  color: var(--text-color) !important;
}

.test-success-msg {
  color: #10b981;
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.8rem;
  font-weight: 500;
}

.test-fail-msg {
  color: #f43f5e;
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.8rem;
  font-weight: 500;
}

/* AI Assistant 子設定項淡化與禁用樣式 */
.ai-sub-settings-container {
  display: flex;
  flex-direction: column;
  transition: opacity 0.2s ease, filter 0.2s ease;
}

.ai-sub-settings-container.is-disabled {
  opacity: 0.45;
  pointer-events: none;
  filter: grayscale(0.5);
  user-select: none;
}
</style>
