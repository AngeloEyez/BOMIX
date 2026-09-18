<template>
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
          v-model="confirmOverwrite"
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
          v-model="autoImportPreviousMatrix"
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
</template>

<script setup lang="ts">
/**
 * @file ImportSettings.vue
 * @description Excel 匯入與版本合併偏好設定子元件
 */
import ToggleSwitch from 'primevue/toggleswitch'

/**
 * 是否在相同版本號時主動跳出覆蓋確認
 */
const confirmOverwrite = defineModel<boolean>('confirmOverwrite', { default: true })

/**
 * 是否在匯入全新 EBOM 時自動沿用上一版本的 Matrix 勾選設定
 */
const autoImportPreviousMatrix = defineModel<boolean>('autoImportPreviousMatrix', { default: true })
</script>

<style scoped>
@import './styles/settings.css';
</style>
