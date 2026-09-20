<!--
  @file AppearanceSettings.vue
  @description 外觀視覺與主題模式設定元件 (Appearance Settings)
  
  職責說明：
  1. 主題模式切換：提供淺色 (Light)、深色 (Dark) 或依照系統 (System) 三種主題切換。
  2. 雙向資料綁定：透過 defineModel 與父層 settings.theme 雙向連動，變更時觸發全域主題切換與存檔。
  3. 擬真按鈕樣式封裝：封裝專屬 PrimeVue SelectButton 客製化樣式，提供擬真 EBOM/Matrix 按鈕操作質感。
-->

<template>
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
          v-model="theme"
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
</template>

<script setup lang="ts">
/**
 * @file AppearanceSettings.vue
 * @description 外觀主題模式設定子元件
 */
import SelectButton from 'primevue/selectbutton'

/**
 * 綁定全域色彩主題代碼 (light, dark, system)
 */
const theme = defineModel<string>({ required: true })

/**
 * 主題切換選項清單
 */
const themeOptions = [
  { label: 'Light', value: 'light' },
  { label: 'Dark', value: 'dark' },
  { label: 'System', value: 'system' },
]
</script>

<style scoped>
@import './styles/settings.css';

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
</style>
