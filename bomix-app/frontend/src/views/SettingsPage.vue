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
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import Select from 'primevue/select'
import SelectButton from 'primevue/selectbutton'
import Checkbox from 'primevue/checkbox'
import InputNumber from 'primevue/inputnumber'
import { GetSettings, UpdateSettings, type Settings } from '../services/api'
import { useAppStore } from '../stores/app'
import { useLogStore } from '../stores/log'

const appStore = useAppStore()
const logStore = useLogStore()

// Settings state
const settings = ref<Settings>({
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
})

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
    }
    
    // 初始化同步至 logStore
    logStore.globalLogLevel = settings.value.logger.level
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
</style>

