<template>
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
          v-model="level"
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
          v-model="maxEntries"
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
</template>

<script setup lang="ts">
/**
 * @file LoggerSettings.vue
 * @description 日誌層級與記憶體緩衝區容量設定子元件
 */
import Select from 'primevue/select'
import InputNumber from 'primevue/inputnumber'

/**
 * 日誌輸出層級門檻 (debug, info, warn, error)
 */
const level = defineModel<string>('level', { default: 'info' })

/**
 * 記憶體環形緩衝區保留的最大日誌筆數
 */
const maxEntries = defineModel<number>('maxEntries', { default: 500 })

/**
 * 日誌層級可選項目
 */
const logLevelOptions = [
  { label: 'Debug', value: 'debug' },
  { label: 'Info', value: 'info' },
  { label: 'Warning', value: 'warn' },
  { label: 'Error', value: 'error' },
]
</script>

<style scoped>
@import './styles/settings.css';
</style>
