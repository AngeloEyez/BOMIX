<template>
  <div class="bom-table-container" ref="tableWrapperRef" @mousedown="onTableMouseDown">
    <!-- View Filter & Mode Toolbar (PrimeVue Toolbar) -->
    <Toolbar class="bom-toolbar">
      <template #start>
        <div class="toolbar-start">
          <span class="filter-label">View:</span>
          <Select
            v-model="selectedView"
            :options="VIEW_OPTIONS"
            option-label="label"
            option-value="value"
            placeholder="Select View"
            size="small"
            class="view-dropdown"
            @change="onViewChange"
          />
          <div class="search-input-wrapper">
            <InputText
              v-model="searchQuery"
              placeholder="Filter"
              size="small"
              class="search-input"
            />
            <i
              v-if="searchQuery"
              class="pi pi-times clear-btn"
              title="Clear filter"
              @click="searchQuery = ''"
            />
          </div>
        </div>
      </template>

      <template #end>
        <div class="toolbar-end">
          <SelectButton
            v-model="selectedBomType"
            :options="BOM_TYPE_OPTIONS"
            :allow-empty="false"
            size="small"
            class="bom-type-toggle"
            @change="emit('update:bom-type', selectedBomType)"
          />
        </div>
      </template>
    </Toolbar>

    <!-- BOM Data Table (Flattened single table with shared columns) -->
    <DataTable
      ref="dataTableRef"
      :value="displayRows"
      dataKey="rowId"
      size="small"
      :scrollable="true"
      scroll-height="flex"
      :virtual-scroller-options="{ itemSize: 26 }"
      :total-records="displayRows.length"
      :row-hover="true"
      :row-class="getRowClass"
      striped-rows
      table-class="bom-table"
      :lazy="true"
      :sort-field="sortField"
      :sort-order="sortOrder"
      @sort="onSort"
      @row-click="onRowClick"
      contextMenu
      v-model:contextMenuSelection="selectedContextRow"
      @row-contextmenu="onRowContextMenu"
    >
      <!-- Item 欄位 (合併開合符號與 Item 號碼，緊湊間距，點擊符號切換收合，點擊標題依 Item 排序) -->
      <Column field="item" sortable :style="{ width: columnWidths.item + 'px' }" class="item-col" header-class="item-header-col">
        <template #header>
          <div class="item-header-content">
            <Button
              :icon="isAllCollapsed ? 'pi pi-chevron-right' : 'pi pi-chevron-down'"
              text
              rounded
              size="small"
              class="toggle-all-btn"
              :disabled="totalExpandableCount === 0"
              :title="isAllCollapsed ? '全部展開替代料 (Expand All)' : '全部收合替代料 (Collapse All)'"
              @click.stop="toggleAllCollapse"
            />
            <span class="p-datatable-column-title item-header-label" data-pc-section="columntitle">#</span>
          </div>
        </template>
        <template #body="slotProps">
          <div v-if="!slotProps.data.isSecondSource" class="item-cell-content">
            <Button
              v-if="slotProps.data.hasSecondSources"
              :icon="isCollapsed(slotProps.data.parentKey) ? 'pi pi-chevron-right' : 'pi pi-chevron-down'"
              text
              rounded
              size="small"
              class="toggle-ss-btn"
              :title="isCollapsed(slotProps.data.parentKey) ? '展開替代料' : '收合替代料'"
              @click.stop="toggleCollapse(slotProps.data.parentKey)"
            />
            <span v-else class="toggle-placeholder" />
            <span class="item-number">{{ slotProps.data.item }}</span>
          </div>
          <!-- 2nd 替代料該欄位保持空白，留出收合圖標空間對齊 -->
          <div v-else class="item-cell-content">
            <span class="toggle-placeholder" />
          </div>
        </template>
      </Column>

      <!-- HHPN -->
      <Column field="hhpn" header="HHPN" :style="{ width: columnWidths.hhpn + 'px' }" sortable>
        <template #body="slotProps">
          <div
            class="cell-text"
            :class="{ 'ss-indented': slotProps.data.isSecondSource }"
            v-tooltip.bottom="slotProps.data.hhpn"
          >
            <Tag v-if="slotProps.data.isSecondSource" value="2nd" severity="secondary" class="ss-badge" />
            <template v-for="(part, idx) in getHighlightedParts(slotProps.data.hhpn, searchQuery)" :key="idx">
              <mark v-if="part.isMatch" class="highlight-text">{{ part.text }}</mark>
              <span v-else>{{ part.text }}</span>
            </template>
          </div>
        </template>
      </Column>

      <!-- Description -->
      <Column field="description" header="Description" :style="{ width: columnWidths.description + 'px', minWidth: '250px', maxWidth: columnWidths.description + 'px' }" sortable>
        <template #body="slotProps">
          <div class="cell-text" v-tooltip.bottom="slotProps.data.description">
            <template v-for="(part, idx) in getHighlightedParts(slotProps.data.description, searchQuery)" :key="idx">
              <mark v-if="part.isMatch" class="highlight-text">{{ part.text }}</mark>
              <span v-else>{{ part.text }}</span>
            </template>
          </div>
        </template>
      </Column>

      <!-- Supplier -->
      <Column field="supplier" header="Supplier" :style="{ width: columnWidths.supplier + 'px' }" sortable>
        <template #body="slotProps">
          <div class="cell-text" v-tooltip.bottom="slotProps.data.supplier">
            <template v-for="(part, idx) in getHighlightedParts(slotProps.data.supplier, searchQuery)" :key="idx">
              <mark v-if="part.isMatch" class="highlight-text">{{ part.text }}</mark>
              <span v-else>{{ part.text }}</span>
            </template>
          </div>
        </template>
      </Column>

      <!-- Supplier PN -->
      <Column field="supplier_pn" header="Supplier PN" :style="{ width: columnWidths.supplier_pn + 'px' }" sortable>
        <template #body="slotProps">
          <div class="cell-text" v-tooltip.bottom="slotProps.data.supplier_pn">
            <template v-for="(part, idx) in getHighlightedParts(slotProps.data.supplier_pn, searchQuery)" :key="idx">
              <mark v-if="part.isMatch" class="highlight-text">{{ part.text }}</mark>
              <span v-else>{{ part.text }}</span>
            </template>
          </div>
        </template>
      </Column>

      <!-- Qty -->
      <Column field="qty" header="Qty" :style="{ width: columnWidths.qty + 'px' }" sortable>
        <template #body="slotProps">
          <div class="cell-text" v-tooltip.bottom="String(slotProps.data.qty ?? '')">
            {{ slotProps.data.qty }}
          </div>
        </template>
      </Column>

      <!-- Location -->
      <Column field="locations" header="Location" :style="{ width: columnWidths.locations + 'px', minWidth: '150px', maxWidth: columnWidths.locations + 'px' }">
        <template #body="slotProps">
          <div class="cell-text" v-tooltip.bottom="slotProps.data.locations">
            <template v-for="(part, idx) in getHighlightedParts(slotProps.data.locations, searchQuery)" :key="idx">
              <mark v-if="part.isMatch" class="highlight-text">{{ part.text }}</mark>
              <span v-else>{{ part.text }}</span>
            </template>
          </div>
        </template>
      </Column>

      <!-- CCL -->
      <Column field="ccl" header="CCL" :style="{ width: columnWidths.ccl + 'px' }" sortable>
        <template #body="slotProps">
          <span v-if="slotProps.data.ccl" :class="getCCLClass(slotProps.data.ccl)">
            Y
          </span>
        </template>
      </Column>

      <!-- Remark -->
      <Column field="remark" header="Remark" :style="{ width: columnWidths.remark + 'px' }">
        <template #body="slotProps">
          <div class="cell-text" v-tooltip.bottom="slotProps.data.remark">
            <template v-for="(part, idx) in getHighlightedParts(slotProps.data.remark, searchQuery)" :key="idx">
              <mark v-if="part.isMatch" class="highlight-text">{{ part.text }}</mark>
              <span v-else>{{ part.text }}</span>
            </template>
          </div>
        </template>
      </Column>

      <!-- Dynamic Model Columns -->
      <Column
        v-for="modelName in currentRevisionModels"
        :key="modelName"
        :header="`${modelName} (Qty: ${getModelQty(modelName)})`"
        :style="{ width: (columnWidths.models[modelName] || 110) + 'px' }"
      >
        <template #body="slotProps">
          <div
            class="cell-text"
            :class="{'model-selected': isModelSelected(slotProps.data, modelName)}"
            v-tooltip.bottom="getModelSelectedPN(slotProps.data, modelName)"
          >
            {{ getModelSelectedPN(slotProps.data, modelName) || '-' }}
          </div>
        </template>
      </Column>
    </DataTable>

    <!-- 右鍵選單元件 -->
    <ContextMenu ref="contextMenuRef" :model="contextMenuItems" />

    <!-- Summary Statistics -->
    <div class="table-summary">
      <span>Total Main Parts: {{ aggregatedParts.length }}</span>
      <span v-if="selectedView === 'smd'">| SMD Parts: {{ smdPartsCount }}</span>
      <span v-if="selectedView === 'pth'">| PTH Parts: {{ pthPartsCount }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
/**
 * @file BOMTable.vue
 * @description BOM 表格核心組裝視圖元件
 * 
 * 本元件為 BOMTable 模組之純粹視圖裝配層，將後端 API 資料流、狀態控制、自適應欄寬計算、
 * 拖曳滾動動畫與右鍵快顯選單等子模組裝配至 PrimeVue DataTable 虛擬滾動表格中。
 * 
 * 模組依賴關係：
 * - ./types: 匯入型別定義
 * - ./utils/textHighlight: 關鍵字比對切割高亮
 * - ./composables/useBOMData: 核心資料流、過濾、排序與平鋪
 * - ./composables/useColumnWidths: 動態欄寬分配與 ResizeObserver
 * - ./composables/useCellAutoScroll: 拖曳文字自動平移滾動動畫
 * - ./composables/useBOMContextMenu: 右鍵選單與 Ctrl+C 快捷鍵
 */

import { ref, toRef, watch, nextTick, onMounted } from 'vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Toolbar from 'primevue/toolbar'
import SelectButton from 'primevue/selectbutton'
import ContextMenu from 'primevue/contextmenu'
import Select from 'primevue/select'
import InputText from 'primevue/inputtext'
import Button from 'primevue/button'
import Tag from 'primevue/tag'

import type { ViewPartGroup } from '../../services/api'
import { getHighlightedParts } from './utils/textHighlight'
import { useBOMData, VIEW_OPTIONS, BOM_TYPE_OPTIONS } from './composables/useBOMData'
import { useColumnWidths } from './composables/useColumnWidths'
import { useCellAutoScroll } from './composables/useCellAutoScroll'
import { useBOMContextMenu } from './composables/useBOMContextMenu'

const props = withDefaults(
  defineProps<{
    revisionIds?: number[]
  }>(),
  {
    revisionIds: () => []
  }
)

const emit = defineEmits<{
  (e: 'part-selected', part: ViewPartGroup): void
  (e: 'update:bom-type', type: 'EBOM' | 'Matrix'): void
}>()

// ── 1. BOM 核心資料流 ────
const bomData = useBOMData({
  revisionIds: toRef(props, 'revisionIds')
})

const {
  selectedBomType,
  selectedView,
  searchQuery,
  sortField,
  sortOrder,
  aggregatedParts,
  displayRows,
  currentRevisionModels,
  smdPartsCount,
  pthPartsCount,
  collapseState,
  onSort,
  onViewChange,
  getModelQty,
  getModelSelectedPN,
  isModelSelected,
  getRowClass,
  getCCLClass,
} = bomData

const {
  isCollapsed,
  toggleCollapse,
  isAllCollapsed,
  totalExpandableCount,
  toggleAllCollapse,
} = collapseState

// ── 2. 最適欄寬計算與自適應分配 ────
const {
  columnWidths,
  tableWrapperRef,
  computeColumnWidths,
  setupResizeListener,
} = useColumnWidths()

/** 執行欄寬重算 */
function triggerColumnWidthsCompute(): void {
  computeColumnWidths(
    displayRows.value,
    currentRevisionModels.value,
    getModelQty,
    getModelSelectedPN
  )
}

// 監聽顯示列資料變化 (篩選、折疊、切換版本)，自動重新精確計算各欄最適欄寬
watch(
  () => displayRows.value,
  () => {
    nextTick(() => {
      requestAnimationFrame(() => {
        triggerColumnWidthsCompute()
      })
    })
  },
  { deep: false }
)

// ── 3. 儲存格拖曳文字自動平移滾動引擎 ────
const { onTableMouseDown } = useCellAutoScroll()

// ── 4. 右鍵快顯選單與鍵盤快捷鍵 ────
const {
  contextMenuRef,
  selectedContextRow,
  contextMenuItems,
  onRowClick,
  onRowContextMenu,
} = useBOMContextMenu({
  searchQuery,
  isCollapsed,
  toggleCollapse,
})

/** DataTable 模板引用 */
const dataTableRef = ref()

onMounted(() => {
  setupResizeListener(() => {
    triggerColumnWidthsCompute()
  })
})
</script>

<style scoped>
.bom-table-container {
  height: 100%;
  width: 100%;
  min-width: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* PrimeVue Toolbar - VS Code 風格高緊湊工具列 */
:deep(.bom-toolbar) {
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

/* Table styling - VS Code 風格高緊湊表格，最大化可視範圍 */
:deep(.p-datatable) {
  font-size: 0.75rem;
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  width: 100%;
  min-width: 0;
  overflow: hidden;
}

:deep(.p-datatable-table-container),
:deep(.p-datatable-wrapper) {
  flex: 1;
  min-height: 0;
  width: 100%;
  min-width: 0;
}

/* 強制表格遵守設定欄寬，徹底防止長文字欄位將單元格無限撐開 */
:deep(.bom-table table),
:deep(.bom-table .p-datatable-table) {
  table-layout: fixed !important;
}

:deep(.p-datatable-header) {
  background: var(--surface-ground);
  border-bottom: 1px solid var(--surface-border);
}

/* 標題列：背景微調為 surface-100，搭配清楚的底線 */
:deep(.bom-table .p-datatable-thead > tr > th) {
  padding: 0.12rem 0.25rem !important;
  font-size: 0.75rem !important;
  font-weight: 600 !important;
  white-space: nowrap !important;
  overflow: hidden !important;
  text-overflow: ellipsis !important;
  line-height: 1.2 !important;
  height: 26px !important;
  background-color: var(--surface-100, #f1f5f9) !important;
  border-bottom: 2px solid var(--surface-300, #cbd5e1) !important;
  border-top: none !important;
  color: var(--text-color, #1e293b) !important;
}

/* 表格單元格緊湊化與文字截斷 */
:deep(.bom-table .p-datatable-tbody > tr > td) {
  padding: 0.12rem 0.25rem !important;
  font-size: 0.75rem !important;
  line-height: 1.2 !important;
  height: 26px !important;
  box-sizing: border-box !important;
  overflow: hidden !important;
  white-space: nowrap !important;
  border-bottom: 1px solid var(--surface-border, #e2e8f0) !important;
}

/* 虛擬滾動行高度固定 26px */
:deep(.bom-table .p-virtualscroller .p-datatable-tbody > tr) {
  height: 26px !important;
}

/* 儲存格文字容器：預設溢出顯示省略號，選取時支援平滑滾動 */
.cell-text {
  display: block;
  overflow-x: scroll;
  overflow-y: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  line-height: 1.2;
  cursor: text;
  user-select: text;
  -webkit-user-select: text;
  scrollbar-width: none;
  -ms-overflow-style: none;
}

.cell-text.is-selecting,
.cell-text.has-selection {
  text-overflow: clip !important;
}

.cell-text::-webkit-scrollbar {
  display: none !important;
  width: 0 !important;
  height: 0 !important;
}

/* 合併欄位：Item + 開合按鈕 */
:deep(.item-col) {
  padding-left: 0.15rem !important;
  padding-right: 0.15rem !important;
}

:deep(.item-header-col) {
  padding-left: 0.15rem !important;
  padding-right: 0.15rem !important;
}

:deep(.item-header-col .p-datatable-column-header-content) {
  display: flex !important;
  align-items: center !important;
  gap: 0.25rem !important;
}

.item-header-content {
  display: inline-flex;
  align-items: center;
  gap: 0.15rem;
}

.item-header-label {
  cursor: pointer;
  user-select: none;
  font-size: 0.75rem !important;
  font-weight: 600 !important;
  color: inherit !important;
  line-height: 1;
}

.item-cell-content {
  display: flex;
  align-items: center;
  gap: 0.15rem;
  width: 100%;
}

.toggle-all-btn,
.toggle-ss-btn {
  width: 0.85rem !important;
  height: 1rem !important;
  min-width: unset !important;
  padding: 0 !important;
  margin: 0 !important;
  flex-shrink: 0;
}

:deep(.toggle-all-btn .p-button-icon),
:deep(.toggle-ss-btn .p-button-icon) {
  font-size: 0.65rem !important;
}

:deep(.toggle-all-btn),
:deep(.toggle-all-btn *),
:deep(.toggle-ss-btn),
:deep(.toggle-ss-btn *) {
  -webkit-user-select: none !important;
  user-select: none !important;
  cursor: pointer !important;
}

.toggle-placeholder {
  display: inline-block;
  width: 0.85rem;
  height: 1rem;
  flex-shrink: 0;
}

.item-number {
  font-size: 0.75rem;
  font-variant-numeric: tabular-nums;
  line-height: 1;
}

/* 2nd 替代料資料列 */
:deep(.second-source-row) {
  background-color: var(--surface-50, #f8fafc) !important;
  color: var(--text-color-secondary, #475569);
  font-size: 0.75rem !important;
}

:deep(.second-source-row:hover) {
  background-color: var(--surface-100, #f1f5f9) !important;
}

.ss-indented {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  padding-left: 0.5rem;
}

/* 底部統計摘要 */
.table-summary {
  display: flex;
  gap: 0.75rem;
  padding: 0.2rem 0.75rem;
  background: var(--surface-ground);
  border-top: 1px solid var(--surface-border);
  font-size: 0.7rem;
  color: var(--text-color-secondary);
  line-height: 1.2;
}

.ccl-normal {
  color: var(--text-color-secondary);
}

.ccl-critical {
  color: var(--p-red-500, #ef4444);
  font-weight: 600;
}

.model-selected {
  font-weight: 600;
  color: var(--p-primary-color, #3b82f6);
}

.highlight-text {
  background-color: #fef08a;
  color: #854d0e;
  font-weight: 700;
  padding: 0 1px;
  border-radius: 2px;
}
</style>
