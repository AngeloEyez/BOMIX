<template>
  <div class="settings-page">
    <div class="settings-header">
      <div class="settings-title">
        <i class="pi pi-cog"></i>
        <span>Settings</span>
      </div>
      <!-- <span class="settings-subtitle">設定即時自動儲存 (Auto-saved)</span> -->
    </div>

    <div class="settings-grid">
      <!-- 1. Appearance -->
      <div class="settings-card">
        <div class="card-header">
          <i class="pi pi-palette card-icon"></i>
          <span class="card-title">Appearance</span>
        </div>
        <div class="setting-row">
          <label class="setting-label">Theme Mode</label>
          <SelectButton
            v-model="settings.theme"
            :options="themeOptions"
            option-label="label"
            option-value="value"
            :allow-empty="false"
            size="small"
            class="compact-select-btn"
          />
        </div>
      </div>

      <!-- 2. Import Settings -->
      <div class="settings-card">
        <div class="card-header">
          <i class="pi pi-upload card-icon"></i>
          <span class="card-title">Import Settings</span>
        </div>
        <div class="checkbox-row" @click="settings.import.confirmOverwrite = !settings.import.confirmOverwrite">
          <Checkbox
            id="confirm-overwrite"
            v-model="settings.import.confirmOverwrite"
            :binary="true"
            @click.stop
          />
          <label for="confirm-overwrite" class="checkbox-label" @click.stop="settings.import.confirmOverwrite = !settings.import.confirmOverwrite">
            Confirm before overwriting existing BOM
          </label>
        </div>
        <div class="checkbox-row" @click="settings.autoImportPreviousMatrix = !settings.autoImportPreviousMatrix">
          <Checkbox
            id="auto-import-matrix"
            v-model="settings.autoImportPreviousMatrix"
            :binary="true"
            @click.stop
          />
          <label for="auto-import-matrix" class="checkbox-label" @click.stop="settings.autoImportPreviousMatrix = !settings.autoImportPreviousMatrix">
            Automatically import previous Matrix when importing EBOM
          </label>
        </div>
      </div>

      <!-- 3. General Settings -->
      <div class="settings-card">
        <div class="card-header">
          <i class="pi pi-sliders-h card-icon"></i>
          <span class="card-title">General</span>
        </div>
        <div class="checkbox-row" @click="settings.autoOpenLastFile = !settings.autoOpenLastFile">
          <Checkbox
            id="auto-open-last-file"
            v-model="settings.autoOpenLastFile"
            :binary="true"
            @click.stop
          />
          <label for="auto-open-last-file" class="checkbox-label" @click.stop="settings.autoOpenLastFile = !settings.autoOpenLastFile">
            Automatically open last file on startup
          </label>
        </div>
        <div v-if="settings.lastOpenedFile" class="setting-row path-row">
          <span class="setting-label">Last Opened File:</span>
          <span class="setting-path-badge" :title="settings.lastOpenedFile">
            {{ settings.lastOpenedFile }}
          </span>
        </div>
      </div>

      <!-- 4. Logger Settings -->
      <div class="settings-card">
        <div class="card-header">
          <i class="pi pi-list card-icon"></i>
          <span class="card-title">Logger</span>
        </div>
        <div class="setting-row">
          <label for="log-level" class="setting-label">Log Level</label>
          <Select
            id="log-level"
            v-model="settings.logger.level"
            :options="logLevelOptions"
            option-label="label"
            option-value="value"
            placeholder="Select log level"
            size="small"
            class="compact-select"
          />
        </div>
        <div class="setting-row">
          <label for="max-entries" class="setting-label">Max Log Entries</label>
          <InputNumber
            id="max-entries"
            v-model="settings.logger.maxEntries"
            :showButtons="true"
            :min="100"
            :max="5000"
            :step="100"
            size="small"
            class="compact-input-number"
          />
        </div>
      </div>

      <!-- 5. Recent Files Settings -->
      <div class="settings-card">
        <div class="card-header">
          <i class="pi pi-history card-icon"></i>
          <span class="card-title">Recent Files</span>
        </div>
        <div class="setting-row">
          <label for="max-recent" class="setting-label">Max Recent Files Count</label>
          <InputNumber
            id="max-recent"
            v-model="settings.recentFiles.maxRecentFiles"
            :showButtons="true"
            :min="1"
            :max="50"
            size="small"
            class="compact-input-number"
          />
        </div>
      </div>

      <!-- 6. AI Assistant Settings -->
      <div class="settings-card col-span-full">
        <div class="card-header">
          <i class="pi pi-sparkles card-icon text-primary-500"></i>
          <span class="card-title">AI Assistant (智慧助手)</span>
        </div>

        <!-- 啟用開關 -->
        <div class="checkbox-row" @click="settings.ai.enabled = !settings.ai.enabled">
          <Checkbox
            id="ai-enabled"
            v-model="settings.ai.enabled"
            :binary="true"
            @click.stop
          />
          <label for="ai-enabled" class="checkbox-label" @click.stop="settings.ai.enabled = !settings.ai.enabled">
            啟用 AI 對話助手 (Enable AI Assistant)
          </label>
        </div>

        <!-- 回應語言偏好 -->
        <div class="setting-row">
          <label for="ai-language" class="setting-label">回應語言 (Response Language)</label>
          <Select
            id="ai-language"
            v-model="settings.ai.language"
            :options="aiLanguageOptions"
            option-label="label"
            option-value="value"
            size="small"
            class="compact-select-wide"
          />
        </div>

        <!-- API Base URL -->
        <div class="setting-row">
          <label for="ai-base-url" class="setting-label">API Base URL</label>
          <InputText
            id="ai-base-url"
            v-model="settings.ai.baseUrl"
            placeholder="https://api.openai.com/v1"
            size="small"
            class="compact-input-text"
            @blur="onBaseUrlBlur"
          />
        </div>

        <!-- API Key -->
        <div class="setting-row">
          <label for="ai-api-key" class="setting-label">API Key</label>
          <Password
            id="ai-api-key"
            v-model="settings.ai.apiKey"
            placeholder="sk-..."
            :feedback="false"
            toggle-mask
            size="small"
            class="compact-password"
            input-class="compact-password-input"
          />
        </div>

        <!-- Model (與 AIChatPage 同步，支援選擇伺服器可用模型或手動自訂) -->
        <div class="setting-row">
          <div class="flex items-center justify-between w-full">
            <label for="ai-model" class="setting-label">Model (模型)</label>
            <button
              type="button"
              class="text-[11px] text-primary hover:underline flex items-center gap-1 cursor-pointer bg-transparent border-none p-0"
              title="向伺服器重新拉取可用模型清單"
              :disabled="aiChatStore.isLoadingModels || !settings.ai.baseUrl"
              @click="onRefreshModels"
            >
              <i :class="['pi text-[10px]', aiChatStore.isLoadingModels ? 'pi-spin pi-spinner' : 'pi-refresh']"></i>
              <span>{{ aiChatStore.isLoadingModels ? '更新中...' : '重新整理模型' }}</span>
            </button>
          </div>
          <Select
            id="ai-model"
            v-model="settings.ai.model"
            :options="modelOptions"
            option-label="label"
            option-value="value"
            placeholder="請選擇伺服器提供的模型"
            size="small"
            class="compact-select-wide"
            @change="onModelChange"
          />
        </div>

        <!-- 自訂模型代碼 (手動設定) -->
        <div class="setting-row custom-model-input-row">
          <label for="ai-custom-model" class="setting-label">自訂模型代碼 (Custom Model)</label>
          <div class="flex items-center gap-2 w-full">
            <InputText
              id="ai-custom-model"
              v-model="customModelInput"
              placeholder="若伺服器未列出，請直接在此輸入自訂模型代碼 (如 deepseek-chat)..."
              size="small"
              class="compact-input-text flex-1"
              @keydown.enter.stop="applyCustomModel"
            />
            <Button
              label="套用"
              icon="pi pi-check"
              size="small"
              outlined
              :disabled="!customModelInput.trim()"
              @click="applyCustomModel"
            />
          </div>
          <span class="text-[11px] opacity-60">
            在此輸入的模型代碼將套用至設定檔，並自動同步至 AI Chat 對話頁面。
          </span>
        </div>

        <!-- Timeout (秒) -->
        <div class="setting-row">
          <label for="ai-timeout" class="setting-label">Timeout (秒 / Seconds)</label>
          <InputNumber
            id="ai-timeout"
            v-model="settings.ai.timeout"
            :showButtons="true"
            :min="5"
            :max="300"
            :step="5"
            size="small"
            class="compact-input-number"
          />
        </div>

        <!-- Temperature -->
        <div class="setting-row">
          <label for="ai-temperature" class="setting-label">Temperature (0.0 - 2.0)</label>
          <InputNumber
            id="ai-temperature"
            v-model="settings.ai.temperature"
            :showButtons="true"
            :min="0"
            :max="2"
            :step="0.1"
            :maxFractionDigits="2"
            size="small"
            class="compact-input-number"
          />
        </div>

        <!-- 連線測試按鈕與狀態反饋 -->
        <div class="setting-row test-connection-row">
          <div class="test-status-text">
            <span v-if="testAIResult?.success" class="text-emerald-600 dark:text-emerald-400 flex items-center gap-1 font-medium text-xs">
              <i class="pi pi-check-circle"></i>
              {{ testAIResult.message }}
            </span>
            <span v-else-if="testAIResult && !testAIResult.success" class="text-rose-500 dark:text-rose-400 flex items-center gap-1 font-medium text-xs">
              <i class="pi pi-exclamation-circle"></i>
              {{ testAIResult.message }}
            </span>
          </div>
          <Button
            label="測試連線"
            icon="pi pi-check-circle"
            size="small"
            outlined
            :loading="isTestingAI"
            @click="testConnection"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import Select from 'primevue/select'
import SelectButton from 'primevue/selectbutton'
import Checkbox from 'primevue/checkbox'
import InputNumber from 'primevue/inputnumber'
import InputText from 'primevue/inputtext'
import Password from 'primevue/password'
import Button from 'primevue/button'
import { GetSettings, UpdateSettings, AIChatTestConnection, type Settings, type AISettings } from '../services/api'
import { useAppStore } from '../stores/app'
import { useLogStore } from '../stores/log'
import { useAIChatStore } from '../stores/aiChat'

const appStore = useAppStore()
const logStore = useLogStore()
const aiChatStore = useAIChatStore()

// 內部表單設定介面，確保 AI 欄位完整存在
interface SettingsForm extends Settings {
  ai: AISettings
}

// Settings state
const settings = ref<SettingsForm>({
  theme: 'light',
  import: {
    confirmOverwrite: true,
    autoImportPreviousMatrix: false,
  },
  logger: {
    level: 'info',
    maxEntries: 500,
  },
  recentFiles: {
    maxRecentFiles: 10,
    recentFiles: [],
  },
  autoOpenLastFile: false,
  lastOpenedFile: '',
  autoImportPreviousMatrix: false,
  ai: {
    enabled: false,
    baseUrl: 'https://api.openai.com/v1',
    apiKey: '',
    model: '',
    temperature: 0.1,
    maxTokens: 4096,
    timeout: 60,
    language: 'zh-TW',
  },
})

// 可用模型選項 (與 aiChatStore.availableModels 雙向同步)
const modelOptions = computed(() => {
  const list = aiChatStore.availableModels.map(m => ({ label: m, value: m }))
  const current = settings.value.ai?.model?.trim()
  if (current && !list.some(item => item.value === current)) {
    return [{ label: current, value: current }, ...list]
  }
  return list
})

// Base URL 失焦時自動拉取伺服器可用模型
async function onBaseUrlBlur(): Promise<void> {
  const url = settings.value.ai?.baseUrl?.trim()
  if (url) {
    await aiChatStore.fetchAvailableModels(url, settings.value.ai?.apiKey)
  }
}

// 點擊手動重新整理模型
async function onRefreshModels(): Promise<void> {
  const url = settings.value.ai?.baseUrl?.trim()
  if (url) {
    await aiChatStore.fetchAvailableModels(url, settings.value.ai?.apiKey)
  }
}

// 自訂模型代碼輸入
const customModelInput = ref('')

// 套用自訂模型
function applyCustomModel(): void {
  const trimmed = customModelInput.value.trim()
  if (trimmed) {
    settings.value.ai.model = trimmed
    aiChatStore.currentModel = trimmed
    aiChatStore.switchModel(trimmed)
    customModelInput.value = ''
  }
}

// 模型變更時同步目前選擇給 aiChatStore
function onModelChange(): void {
  if (settings.value.ai?.model) {
    aiChatStore.currentModel = settings.value.ai.model
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

async function testConnection(): Promise<void> {
  isTestingAI.value = true
  testAIResult.value = null
  try {
    // 先儲存目前填入的設定，確保後端拿到最新資料
    await UpdateSettings(settings.value)
    await AIChatTestConnection()
    // 同步拉取模型清單
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

async function loadSettings(): Promise<boolean> {
  try {
    const data = await GetSettings()
    if (!data) return false

    settings.value = {
      ...data,
      import: {
        confirmOverwrite: data.import?.confirmOverwrite ?? true,
        autoImportPreviousMatrix: data.import?.autoImportPreviousMatrix ?? false,
      },
      logger: {
        level: data.logger?.level ?? 'info',
        maxEntries: data.logger?.maxEntries ?? 500,
      },
      recentFiles: {
        maxRecentFiles: data.recentFiles?.maxRecentFiles ?? 10,
        recentFiles: data.recentFiles?.recentFiles ?? [],
      },
      ai: {
        enabled: data.ai?.enabled ?? false,
        baseUrl: data.ai?.baseUrl || 'https://api.openai.com/v1',
        apiKey: data.ai?.apiKey || '',
        model: data.ai?.model || '',
        temperature: data.ai?.temperature ?? 0.1,
        maxTokens: data.ai?.maxTokens ?? 4096,
        timeout: data.ai?.timeout ?? 60,
        language: data.ai?.language || 'zh-TW',
      },
    }
    
    // 初始化同步至 logStore、appStore 與 aiChatStore
    logStore.globalLogLevel = settings.value.logger.level
    appStore.confirmOverwrite = settings.value.import.confirmOverwrite
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
  if (!isLoaded) return

  // 前護邏輯：過濾不完整或零值的 payload
  if (!newVal.theme || !newVal.logger?.level || !newVal.logger?.maxEntries) {
    console.warn('Ignore auto-save: invalid or incomplete settings payload', newVal)
    return
  }

  // Apply theme immediately on change
  appStore.applyTheme(newVal.theme)
  
  // Apply log level instantly
  logStore.globalLogLevel = newVal.logger.level

  // Apply confirmOverwrite instantly
  if (newVal.import) {
    appStore.confirmOverwrite = newVal.import.confirmOverwrite ?? true
  }

  if (saveTimeout) clearTimeout(saveTimeout)
  saveTimeout = setTimeout(async () => {
    try {
      await UpdateSettings(newVal)
    } catch (error) {
      console.error('Failed to auto-save settings:', error)
    }
  }, 500) // 500ms debounce
}, { deep: true })

onMounted(async () => {
  const success = await loadSettings()
  if (success) {
    // Allow time for initial reactive trigger to settle before enabling auto-save
    setTimeout(() => { isLoaded = true }, 200)
  }
})
</script>

<style scoped>
.settings-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow-y: auto;
  padding: 0.65rem 0.85rem;
  background: var(--surface-ground);
  gap: 0.5rem;
}

/* 頂部標題列 (VSCode Style) */
.settings-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.2rem 0.25rem 0.4rem 0.25rem;
  border-bottom: 1px solid var(--surface-border);
  flex-shrink: 0;
}

.settings-title {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  font-size: 0.85rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--text-color);
}

.settings-title i {
  color: var(--primary-color);
  font-size: 0.95rem;
}

.settings-subtitle {
  font-size: 0.72rem;
  color: var(--text-color-secondary);
}

/* 緊湊自適應網格 (最大化可視面積) */
.settings-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 0.55rem;
  width: 100%;
}

/* 卡片區塊 */
.settings-card {
  background: var(--surface-card);
  border: 1px solid var(--surface-border);
  border-radius: 4px;
  padding: 0.55rem 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
  transition: border-color 0.15s ease;
}

.settings-card:hover {
  border-color: var(--surface-border-hover, var(--primary-color));
}

.card-header {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  border-bottom: 1px solid var(--surface-border);
  padding-bottom: 0.3rem;
  margin-bottom: 0.15rem;
}

.card-icon {
  font-size: 0.8rem;
  color: var(--primary-color);
}

.card-title {
  font-size: 0.72rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-color-secondary);
}

/* 單行設定項目 */
.setting-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  box-sizing: border-box;
  padding: 0.25rem 0.35rem;
  border-radius: 3px;
  min-height: 32px;
  gap: 0.5rem;
  overflow: hidden;
  transition: background-color 0.12s ease;
}

.setting-row:hover {
  background: var(--surface-hover, rgba(255, 255, 255, 0.03));
}

.setting-label {
  font-size: 0.8rem;
  font-weight: 500;
  color: var(--text-color);
  min-width: 0;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.checkbox-row {
  display: flex;
  align-items: center;
  gap: 0.55rem;
  padding: 0.3rem 0.35rem;
  border-radius: 3px;
  cursor: pointer;
  min-height: 30px;
  transition: background-color 0.12s ease;
}

.checkbox-row:hover {
  background: var(--surface-hover, rgba(255, 255, 255, 0.03));
}

.checkbox-label {
  font-size: 0.8rem;
  color: var(--text-color);
  cursor: pointer;
  user-select: none;
}

/* 控制項尺寸與樣式 */
.compact-select-btn {
  height: 26px;
  flex-shrink: 0;
}

:deep(.compact-select-btn .p-button) {
  padding: 0.15rem 0.55rem !important;
  font-size: 0.78rem !important;
}

.compact-select {
  width: 110px !important;
  min-width: 110px !important;
  max-width: 110px !important;
  height: 26px !important;
  font-size: 0.8rem;
  flex-shrink: 0;
}

:deep(.compact-select .p-select-label) {
  padding: 0.15rem 0.45rem !important;
  font-size: 0.8rem !important;
}

.compact-input-number {
  width: 110px !important;
  min-width: 110px !important;
  max-width: 110px !important;
  height: 26px !important;
  flex-shrink: 0;
}

:deep(.compact-input-number.p-inputnumber) {
  width: 110px !important;
  min-width: 110px !important;
  max-width: 110px !important;
  height: 26px !important;
  display: inline-flex !important;
  flex-shrink: 0 !important;
}

:deep(.compact-input-number .p-inputnumber-input) {
  width: 100% !important;
  height: 24px !important;
  padding: 0.1rem 1.4rem 0.1rem 0.45rem !important;
  font-size: 0.78rem !important;
  text-align: left !important;
}

:deep(.compact-input-number .p-inputnumber-button) {
  width: 18px !important;
  padding: 0 !important;
  height: 50% !important;
}

:deep(.compact-input-number .p-inputnumber-button .p-icon) {
  font-size: 0.55rem !important;
  width: 0.55rem !important;
  height: 0.55rem !important;
}

.setting-path-badge {
  font-size: 0.75rem;
  color: var(--text-color-secondary);
  background: var(--surface-ground);
  padding: 0.1rem 0.4rem;
  border-radius: 3px;
  border: 1px solid var(--surface-border);
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compact-select-wide {
  width: 220px !important;
  min-width: 220px !important;
  height: 26px !important;
  font-size: 0.8rem;
  flex-shrink: 0;
}

:deep(.compact-select-wide .p-select-label) {
  padding: 0.15rem 0.45rem !important;
  font-size: 0.8rem !important;
}

.compact-input-text {
  width: 220px !important;
  max-width: 220px !important;
  height: 26px !important;
  padding: 0.1rem 0.45rem !important;
  font-size: 0.78rem !important;
}

.compact-password {
  width: 220px !important;
  max-width: 220px !important;
  height: 26px !important;
}

:deep(.compact-password .p-password-input) {
  width: 100% !important;
  height: 26px !important;
  padding: 0.1rem 1.8rem 0.1rem 0.45rem !important;
  font-size: 0.78rem !important;
}

.test-connection-row {
  margin-top: 0.5rem;
  padding-top: 0.5rem;
  border-top: 1px solid var(--surface-border, rgba(125, 125, 125, 0.15));
}

.custom-model-input-row {
  align-items: flex-start !important;
  flex-direction: column !important;
  gap: 0.35rem !important;
  background-color: var(--surface-hover, rgba(125, 125, 125, 0.05));
  padding: 0.45rem 0.6rem !important;
  border-radius: 4px;
  border: 1px dashed var(--surface-border, rgba(125, 125, 125, 0.2));
  margin-top: 0.25rem;
}
</style>

