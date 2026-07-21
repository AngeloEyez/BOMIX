<template>
  <div class="settings-page">
    <div class="settings-container">
      <h1 class="page-title">Settings</h1>

      <div class="settings-section">
        <h2 class="section-title">Appearance</h2>
        <div class="setting-item">
          <label for="theme-select" class="setting-label">Theme</label>
          <Select
            id="theme-select"
            v-model="settings.theme"
            :options="themeOptions"
            option-label="label"
            option-value="value"
            placeholder="Select a theme"
            class="setting-input"
          />
        </div>
      </div>

      <div class="settings-section">
        <h2 class="section-title">Import Settings</h2>
        <div class="setting-item">
          <Checkbox
            id="confirm-overwrite"
            v-model="settings.import.confirmOverwrite"
            :binary="true"
          />
          <label for="confirm-overwrite" class="setting-label">
            Confirm before overwriting existing BOM
          </label>
        </div>
        <div class="setting-item">
          <Checkbox
            id="auto-import-matrix"
            v-model="settings.autoImportPreviousMatrix"
            :binary="true"
          />
          <label for="auto-import-matrix" class="setting-label">
            Automatically import previous Matrix when importing EBOM
          </label>
        </div>
      </div>

      <div class="settings-section">
        <h2 class="section-title">General</h2>
        <div class="setting-item">
          <Checkbox
            id="auto-open-last-file"
            v-model="settings.autoOpenLastFile"
            :binary="true"
          />
          <label for="auto-open-last-file" class="setting-label">
            Automatically open last file on startup
          </label>
        </div>
        <div v-if="settings.lastOpenedFile" class="setting-item">
          <span class="setting-label">Last opened file:</span>
          <span class="setting-value">{{ settings.lastOpenedFile }}</span>
        </div>
      </div>

      <div class="settings-section">
        <h2 class="section-title">Logger</h2>
        <div class="setting-item">
          <label for="log-level" class="setting-label">Log Level</label>
          <Select
            id="log-level"
            v-model="settings.logger.level"
            :options="logLevelOptions"
            option-label="label"
            option-value="value"
            placeholder="Select log level"
            class="setting-input"
          />
        </div>
        <div class="setting-item">
          <label for="max-entries" class="setting-label">Max Log Entries</label>
          <InputNumber
            id="max-entries"
            v-model="settings.logger.maxEntries"
            :min="100"
            :max="5000"
            class="setting-input"
          />
        </div>
      </div>

      <div class="settings-section">
        <h2 class="section-title">Recent Files</h2>
        <div class="setting-item">
          <label for="max-recent" class="setting-label">Max Recent Files</label>
          <InputNumber
            id="max-recent"
            v-model="settings.recentFiles.maxRecentFiles"
            :min="1"
            :max="50"
            class="setting-input"
          />
        </div>
      </div>

    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import Select from 'primevue/select'
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

async function loadSettings(): Promise<void> {
  try {
    const data = await GetSettings()
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
  } catch (error) {
    console.error('Failed to load settings:', error)
  }
}

// Auto-save logic
let saveTimeout: any
let isLoaded = false

watch(settings, (newVal) => {
  if (!isLoaded) return

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
  await loadSettings()
  // Allow time for initial reactive trigger to settle before enabling auto-save
  setTimeout(() => { isLoaded = true }, 100)
})
</script>

<style scoped>
.settings-page {
  display: flex;
  justify-content: center;
  padding: 1rem 2rem;
  background: var(--surface-ground);
  height: 100%;
  overflow-y: auto;
}

.settings-container {
  width: 100%;
  max-width: 800px;
}

.page-title {
  font-size: 1.5rem;
  font-weight: 600;
  color: var(--text-color);
  margin: 0 0 1rem;
}

.settings-section {
  padding: 0.75rem 0;
  margin-bottom: 0.5rem;
}

.section-title {
  font-size: 0.85rem;
  font-weight: 700;
  color: var(--text-color);
  margin: 0 0 0.5rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.setting-item {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 0.35rem 0;
}

.setting-label {
  flex: 1;
  font-size: 0.85rem;
  color: var(--text-color);
}

.setting-input {
  width: 250px;
}

.setting-value {
  font-size: 0.85rem;
  color: var(--text-color-secondary);
  word-break: break-all;
}
</style>
