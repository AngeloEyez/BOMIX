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
 * 封裝 View 視圖切換 (All, SMD, PTH...)、搜尋關鍵字過濾框、快速清除按鈕，
 * 以及 EBOM / Matrix 顯示模式切換按鈕。
 */

import Toolbar from 'primevue/toolbar'
import Select from 'primevue/select'
import InputText from 'primevue/inputtext'
import SelectButton from 'primevue/selectbutton'
import { VIEW_OPTIONS, BOM_TYPE_OPTIONS } from '../composables/useBOMData'

interface Props {
  /** 當前選取的視圖類型 (如 'all', 'smd', 'pth') */
  view: string
  /** 當前搜尋過濾關鍵字 */
  search: string
  /** 當前 BOM 模式 ('EBOM' | 'Matrix') */
  bomType: 'EBOM' | 'Matrix'
}

const props = defineProps<Props>()

const emit = defineEmits<{
  /** 更新視圖類型 */
  (e: 'update:view', value: string): void
  /** 更新搜尋關鍵字 */
  (e: 'update:search', value: string): void
  /** 更新 BOM 模式 */
  (e: 'update:bomType', value: 'EBOM' | 'Matrix'): void
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

:deep(.bom-type-toggle .p-togglebutton) {
  padding: 0.15rem 0.5rem !important;
  font-size: 0.75rem !important;
  font-weight: 600;
  height: 26px !important;
  transition: background-color 0.15s ease, color 0.15s ease;
}

:deep(.bom-type-toggle .p-togglebutton:not(.p-togglebutton-checked):not([data-p-checked="true"])) {
  color: var(--text-color-secondary) !important;
}

/* 切換 EBOM / Matrix 的 SelectButton 選中狀態高亮 */
:deep(.bom-type-toggle .p-togglebutton.p-togglebutton-checked),
:deep(.bom-type-toggle .p-togglebutton[data-p-checked="true"]) {
  color: var(--primary-color) !important;
  background-color: var(--surface-hover) !important;
  font-weight: 700 !important;
}

:deep(.bom-type-toggle .p-togglebutton.p-togglebutton-checked .p-togglebutton-label),
:deep(.bom-type-toggle .p-togglebutton[data-p-checked="true"] .p-togglebutton-label) {
  color: var(--primary-color) !important;
}

:deep(.bom-type-toggle .p-togglebutton:hover) {
  background-color: var(--surface-hover) !important;
}
</style>
