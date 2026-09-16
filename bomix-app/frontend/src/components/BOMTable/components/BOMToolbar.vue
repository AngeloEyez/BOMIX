<template>
  <Toolbar class="bom-toolbar">
    <template #start>
      <div class="toolbar-start">
        <span class="filter-label">View:</span>
        <Select
          :model-value="view"
          :options="VIEW_OPTIONS"
          option-label="label"
          option-value="value"
          placeholder="Select View"
          size="small"
          class="view-dropdown"
          @update:model-value="onViewSelect"
        />
        <div class="search-input-wrapper">
          <InputText
            :model-value="search"
            placeholder="Filter"
            size="small"
            class="search-input"
            @input="onSearchInput"
          />
          <i
            v-if="search"
            class="pi pi-times clear-btn"
            title="Clear filter"
            @click="onClearSearch"
          />
        </div>
      </div>
    </template>

    <template #end>
      <div class="toolbar-end">
        <div class="ccl-toggle-wrapper">
          <ToggleButton
            :model-value="cclOnly"
            on-label="CCL Only"
            off-label="CCL Only"
            size="small"
            class="ccl-only-toggle"
            @update:model-value="onCclOnlyChange"
          />
        </div>
        <SelectButton
          :model-value="bomType"
          :options="BOM_TYPE_OPTIONS"
          :allow-empty="false"
          size="small"
          class="bom-type-toggle"
          @update:model-value="onBomTypeChange"
        />
      </div>
    </template>
  </Toolbar>
</template>

<script setup lang="ts">
/**
 * @file BOMToolbar.vue
 * @description BOM 表格頂部工具列元件
 * 
 * 封裝 View 視圖切換 (All, SMD, PTH...)、搜尋關鍵字過濾框、快速清除按鈕、
 * CCL Only 關鍵零件切換按鈕，以及 EBOM / Matrix 顯示模式切換按鈕。
 */

import Toolbar from 'primevue/toolbar'
import Select from 'primevue/select'
import InputText from 'primevue/inputtext'
import SelectButton from 'primevue/selectbutton'
import ToggleButton from 'primevue/togglebutton'
import { VIEW_OPTIONS, BOM_TYPE_OPTIONS } from '../composables/useBOMData'

interface Props {
  /** 當前選取的視圖類型 (如 'all', 'smd', 'pth') */
  view: string
  /** 當前搜尋過濾關鍵字 */
  search: string
  /** 當前 BOM 模式 ('EBOM' | 'Matrix') */
  bomType: 'EBOM' | 'Matrix'
  /** 是否僅顯示 CCL 關鍵零件物料群組 */
  cclOnly?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  cclOnly: false,
})

const emit = defineEmits<{
  /** 更新視圖類型 */
  (e: 'update:view', value: string): void
  /** 更新搜尋關鍵字 */
  (e: 'update:search', value: string): void
  /** 更新 BOM 模式 */
  (e: 'update:bomType', value: 'EBOM' | 'Matrix'): void
  /** 更新 CCL Only 篩選狀態 */
  (e: 'update:cclOnly', value: boolean): void
  /** 視圖變更觸發事件 */
  (e: 'viewChange'): void
}>()

/**
 * 處理視圖切換選擇
 * 
 * @param {string} value - 新選取的視圖代碼
 */
function onViewSelect(value: string): void {
  emit('update:view', value)
  emit('viewChange')
}

/**
 * 處理搜尋輸入變更
 * 
 * @param {Event} e - 原生輸入事件
 */
function onSearchInput(e: Event): void {
  const target = e.target as HTMLInputElement
  emit('update:search', target?.value ?? '')
}

/**
 * 清除搜尋關鍵字
 */
function onClearSearch(): void {
  emit('update:search', '')
}

/**
 * 處理 CCL Only 開關切換
 * 
 * @param {boolean} value - 新的開關狀態
 */
function onCclOnlyChange(value: boolean): void {
  emit('update:cclOnly', value)
}

/**
 * 處理 BOM 模式切換
 * 
 * @param {'EBOM' | 'Matrix'} value - 新的 BOM 模式
 */
function onBomTypeChange(value: 'EBOM' | 'Matrix'): void {
  if (value) {
    emit('update:bomType', value)
  }
}
</script>

<style scoped>
/* PrimeVue Toolbar - VS Code 風格高緊湊工具列 */
:deep(.bom-toolbar),
.bom-toolbar {
  padding: 0.25rem 0.5rem;
  border-radius: 0;
  border-width: 0 0 1px 0;
  border-color: var(--surface-border);
  background: var(--surface-ground);
  min-height: unset;
}

.toolbar-start,
.toolbar-end {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.filter-label {
  font-weight: 600;
  font-size: 0.75rem;
  color: var(--text-color);
}

.view-dropdown {
  min-width: 100px;
  font-size: 0.75rem;
}

:deep(.view-dropdown .p-select-label) {
  padding: 0.15rem 0.4rem;
  font-size: 0.75rem;
}

.search-input-wrapper {
  position: relative;
  display: inline-flex;
  align-items: center;
}

.search-input {
  width: 180px;
  padding: 0.15rem 1.6rem 0.15rem 0.4rem !important;
  font-size: 0.75rem !important;
  height: 26px !important;
}

.clear-btn {
  position: absolute;
  right: 0.4rem;
  cursor: pointer;
  color: var(--text-color-secondary);
  font-size: 0.75rem;
  transition: color 0.2s;
}

.clear-btn:hover {
  color: var(--text-color);
}

/* 移除 PrimeVue 4 ToggleButton 內層 .p-togglebutton-content 預設的白底與陰影滑塊 */
:deep(.ccl-only-toggle .p-togglebutton-content),
:deep(.ccl-only-toggle.p-togglebutton-checked .p-togglebutton-content),
:deep(.ccl-only-toggle[data-p-checked="true"] .p-togglebutton-content),
:deep(.bom-type-toggle .p-togglebutton-content),
:deep(.bom-type-toggle .p-togglebutton.p-togglebutton-checked .p-togglebutton-content),
:deep(.bom-type-toggle .p-togglebutton[data-p-checked="true"] .p-togglebutton-content) {
  background: transparent !important;
  box-shadow: none !important;
  padding: 0 !important;
}

/* CCL Only 外層容器樣式：與 EBOM/Matrix 相同的淡淡灰色外框與內縮 */
.ccl-toggle-wrapper {
  background: transparent;
  border: 1px solid var(--surface-border);
  box-shadow: none;
  border-radius: 4px;
  padding: 1.5px;
  display: inline-flex;
  align-items: center;
  height: 26px;
  box-sizing: border-box;
}

:deep(.ccl-only-toggle) {
  padding: 0 0.45rem !important;
  font-size: 0.72rem !important;
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

/* CCL Only 未選中時 hover：淺灰底色 */
:deep(.ccl-only-toggle:not(.p-togglebutton-checked):not([data-p-checked="true"]):hover) {
  background-color: var(--surface-hover) !important;
  color: var(--text-color) !important;
}

/* CCL Only 選中狀態：綠底白字 */
:deep(.ccl-only-toggle.p-togglebutton-checked),
:deep(.ccl-only-toggle[data-p-checked="true"]) {
  color: #ffffff !important;
  background-color: var(--primary-color) !important;
  font-weight: 600 !important;
}

/* CCL Only 選中時 hover：接近 primary color 的微調綠色 */
:deep(.ccl-only-toggle.p-togglebutton-checked:hover),
:deep(.ccl-only-toggle[data-p-checked="true"]:hover) {
  background-color: color-mix(in srgb, var(--primary-color) 85%, black) !important;
  color: #ffffff !important;
}

:deep(.ccl-only-toggle .p-togglebutton-label) {
  color: inherit !important;
}

:deep(.ccl-only-toggle.p-togglebutton-checked .p-togglebutton-label),
:deep(.ccl-only-toggle[data-p-checked="true"] .p-togglebutton-label) {
  color: #ffffff !important;
}

/* EBOM / Matrix 切換按鈕群組容器樣式：淡淡的灰色外框，表示是一組 */
:deep(.bom-type-toggle.p-selectbutton),
:deep(.bom-type-toggle) {
  background: transparent !important;
  border: 1px solid var(--surface-border) !important;
  box-shadow: none !important;
  border-radius: 4px !important;
  padding: 1.5px !important; /* 內縮 1.5px，使按鈕底色與外框拉開距離、互有區別 */
  gap: 2px !important;
  display: inline-flex !important;
  align-items: center !important;
  height: 26px !important;
  box-sizing: border-box !important;
}

:deep(.bom-type-toggle .p-togglebutton) {
  padding: 0 0.45rem !important;
  font-size: 0.72rem !important;
  font-weight: 600;
  height: 21px !important; /* 稍微縮小按鈕底色高度 */
  border-radius: 2.5px !important;
  border: none !important;
  background-color: transparent !important;
  color: var(--text-color-secondary) !important;
  transition: background-color 0.15s ease, color 0.15s ease;
  display: inline-flex !important;
  align-items: center !important;
  justify-content: center !important;
}

/* EBOM / Matrix 未選中時 hover：淺灰底色 */
:deep(.bom-type-toggle .p-togglebutton:not(.p-togglebutton-checked):not([data-p-checked="true"]):hover) {
  background-color: var(--surface-hover) !important;
  color: var(--text-color) !important;
}

/* 切換 EBOM / Matrix 的 SelectButton 選中狀態：綠底白字 */
:deep(.bom-type-toggle .p-togglebutton.p-togglebutton-checked),
:deep(.bom-type-toggle .p-togglebutton[data-p-checked="true"]) {
  color: #ffffff !important;
  background-color: var(--primary-color) !important;
  font-weight: 600 !important;
}

/* EBOM / Matrix 選中時 hover：接近 primary color 的微調綠色 */
:deep(.bom-type-toggle .p-togglebutton.p-togglebutton-checked:hover),
:deep(.bom-type-toggle .p-togglebutton[data-p-checked="true"]:hover) {
  background-color: color-mix(in srgb, var(--primary-color) 85%, black) !important;
  color: #ffffff !important;
}

:deep(.bom-type-toggle .p-togglebutton .p-togglebutton-label) {
  color: inherit !important;
}

:deep(.bom-type-toggle .p-togglebutton.p-togglebutton-checked .p-togglebutton-label),
:deep(.bom-type-toggle .p-togglebutton[data-p-checked="true"] .p-togglebutton-label) {
  color: #ffffff !important;
}
</style>
