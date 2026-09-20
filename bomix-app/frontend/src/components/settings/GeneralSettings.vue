<!--
  @file GeneralSettings.vue
  @description 一般系統設定元件 (General Settings)
  
  職責說明：
  1. 系統啟動行為：配置啟動應用程式時是否自動開啟上次最後使用的專案檔 (.bomx)。
  2. 專案歷程管理：設定歡迎頁面「最近開啟檔案」歷史清單所保留的最大筆數上限 (1 ~ 50 筆)。
  3. 狀態資訊展示：唯讀呈現上次開啟之專案完整檔案路徑 (Badge 徽章展示)。
  4. 雙向資料綁定：透過 defineModel 雙向連動父層表單物件，觸發父層集中防抖存檔。
-->

<template>
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
          v-model="autoOpenLastFile"
        />
        <label for="auto-open-last-file" class="setting-name switch-label">
          Auto Open Last File
        </label>
      </div>
      <p class="setting-desc">
        啟動應用程式時，自動開啟上次最後使用的 BOM 系列專案檔 (.bomx)。
      </p>
      <div v-if="lastOpenedFile" class="last-opened-info">
        <span class="text-xs text-[var(--text-color-secondary)]">上次開啟：</span>
        <Badge :value="lastOpenedFile" severity="secondary" class="last-file-badge" />
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
          v-model="maxRecentFiles"
          :show-buttons="true"
          :min="1"
          :max="50"
          size="small"
          class="compact-input-number"
        />
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
/**
 * @file GeneralSettings.vue
 * @description 一般系統啟動與專案歷程設定子元件
 */
import ToggleSwitch from 'primevue/toggleswitch'
import InputNumber from 'primevue/inputnumber'
import Badge from 'primevue/badge'

/**
 * 是否在啟動時自動開啟上次專案檔
 */
const autoOpenLastFile = defineModel<boolean>('autoOpenLastFile', { default: false })

/**
 * 歷史清單保留之最大檔案筆數
 */
const maxRecentFiles = defineModel<number>('maxRecentFiles', { default: 10 })

/**
 * 上次開啟的檔案路徑 (唯讀展示)
 */
defineProps<{
  lastOpenedFile?: string
}>()
</script>

<style scoped>
@import './styles/settings.css';

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
</style>
