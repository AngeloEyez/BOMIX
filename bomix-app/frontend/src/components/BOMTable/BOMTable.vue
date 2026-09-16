<template>
  <div class="bom-table-container" ref="tableWrapperRef" @mousedown="onTableMouseDown">
    <!-- View Filter & Mode Toolbar (獨立頂部工具列元件) -->
    <BOMToolbar
      v-model:view="selectedView"
      v-model:search="searchQuery"
      v-model:bom-type="selectedBomType"
      @view-change="onViewChange"
      @update:bom-type="emit('update:bom-type', $event)"
    />

    <!-- BOM Data Table (PrimeVue 虛擬滾動平鋪表格) -->
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
      class="bom-table"
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
      <!-- Item 欄位 (開合按鈕與序號) -->
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
          </div>
        </template>
        <template #body="slotProps">
          <BOMItemCell
            :row="slotProps.data"
            :collapsed="isCollapsed(slotProps.data.parentKey)"
            @toggle="toggleCollapse"
          />
        </template>
      </Column>

      <!-- HHPN -->
      <Column field="hhpn" header="HHPN" :style="{ width: columnWidths.hhpn + 'px' }" sortable>
        <template #body="slotProps">
          <div class="cell-text cell-mono" v-tooltip.bottom="slotProps.data.hhpn">
            <BOMHighlightText :text="slotProps.data.hhpn" :query="searchQuery" />
          </div>
        </template>
      </Column>

      <!-- Description -->
      <Column field="description" header="Description" :style="{ width: columnWidths.description + 'px', minWidth: '220px', maxWidth: columnWidths.description + 'px' }" sortable>
        <template #body="slotProps">
          <div
            class="cell-text"
            @mouseenter="handleCellMouseEnter($event, 'description', slotProps.data)"
            @mouseleave="handleCellMouseLeave"
          >
            <BOMHighlightText :text="slotProps.data.description" :query="searchQuery" />
          </div>
        </template>
      </Column>

      <!-- Supplier -->
      <Column field="supplier" header="Supplier" :style="{ width: columnWidths.supplier + 'px' }" sortable>
        <template #body="slotProps">
          <div class="cell-text" v-tooltip.bottom="slotProps.data.supplier">
            <BOMHighlightText :text="slotProps.data.supplier" :query="searchQuery" />
          </div>
        </template>
      </Column>

      <!-- Supplier PN -->
      <Column field="supplier_pn" header="Supplier PN" :style="{ width: columnWidths.supplier_pn + 'px' }" sortable>
        <template #body="slotProps">
          <div class="cell-text cell-mono" v-tooltip.bottom="slotProps.data.supplier_pn">
            <BOMHighlightText :text="slotProps.data.supplier_pn" :query="searchQuery" />
          </div>
        </template>
      </Column>

      <!-- Location -->
      <Column field="locations" header="Location" :style="{ width: columnWidths.locations + 'px', minWidth: '90px', maxWidth: columnWidths.locations + 'px' }">
        <template #body="slotProps">
          <div
            class="cell-text cell-mono"
            @mouseenter="handleCellMouseEnter($event, 'locations', slotProps.data)"
            @mouseleave="handleCellMouseLeave"
          >
            <BOMHighlightText :text="slotProps.data.locations" :query="searchQuery" />
          </div>
        </template>
      </Column>

      <!-- EBOM 模式欄位：動態 Qty 欄位 + CCL + Remark -->
      <template v-if="selectedBomType === 'EBOM'">
        <Column
          v-for="revCol in revisionColumns"
          :key="'ebom-qty-' + revCol.revisionId"
          :style="{
            width: (revCol.columnWidth || 54) + 'px',
            minWidth: (revCol.columnWidth || 54) + 'px',
            maxWidth: ((revCol.columnWidth || 54) + 8) + 'px'
          }"
          header-class="two-line-header-col"
        >
          <template #header>
            <div class="two-line-header" :title="`${revCol.projectCode} ${revCol.phase} ${revCol.version}`">
              <div class="header-line1">{{ revCol.projectCode }}</div>
              <div class="header-line2">{{ revCol.phase }} {{ revCol.version }}</div>
            </div>
          </template>
          <template #body="slotProps">
            <div class="cell-text qty-cell">
              {{ getRevisionQty(slotProps.data, revCol.revisionId) }}
            </div>
          </template>
        </Column>

        <!-- CCL (僅 EBOM 模式顯示，已移除排序功能以確保緊湊寬度正確顯示) -->
        <Column field="ccl" header="CCL" :style="{ width: columnWidths.ccl + 'px' }">
          <template #body="slotProps">
            <span v-if="slotProps.data.ccl" :class="getCCLClass(slotProps.data.ccl)">
              Y
            </span>
          </template>
        </Column>

        <!-- Remark (僅 EBOM 模式顯示) -->
        <Column field="remark" header="Remark" :style="{ width: columnWidths.remark + 'px' }">
          <template #body="slotProps">
            <div class="cell-text" v-tooltip.bottom="slotProps.data.remark">
              <BOMHighlightText :text="slotProps.data.remark" :query="searchQuery" />
            </div>
          </template>
        </Column>
      </template>

      <!-- Matrix 模式欄位：Qty (聚合總用量) + [Revisions] 互斥 Checkbox 欄位 + Notes -->
      <template v-else-if="selectedBomType === 'Matrix'">
        <!-- Qty (聚合總用量) -->
        <Column field="qty" header="Qty" :style="{ width: columnWidths.qty + 'px' }" sortable>
          <template #body="slotProps">
            <div class="cell-text qty-cell" v-tooltip.bottom="String(slotProps.data.qty ?? '')">
              {{ slotProps.data.qty }}
            </div>
          </template>
        </Column>

        <!-- [Matrix Models] 欄位 (自適應欄寬與貫穿分隔線) -->
        <Column
          v-for="(modelCol, colIdx) in matrixModelColumns"
          :key="modelCol.key"
          :style="{
            width: modelCol.columnWidth + 'px',
            minWidth: modelCol.columnWidth + 'px',
            maxWidth: (modelCol.columnWidth + 10) + 'px'
          }"
          :header-class="[
            'two-line-header-col',
            'matrix-model-col',
            modelCol.isLastInRevision ? 'project-divider-col' : 'project-inner-col',
            colIdx === 0 ? 'project-start-col' : ''
          ]"
          :class="[
            'matrix-checkbox-col',
            'matrix-model-col',
            modelCol.isLastInRevision ? 'project-divider-col' : 'project-inner-col',
            colIdx === 0 ? 'project-start-col' : ''
          ]"
        >
          <template #header>
            <BOMMatrixHeader :column="modelCol" @click="handleModelHeaderClick(modelCol)" />
          </template>
          <template #body="slotProps">
            <BOMMatrixCheckboxCell
              :is-available="isModelAvailableInRevision(slotProps.data, modelCol)"
              :is-selected="isModelSelectedInRevision(slotProps.data, modelCol)"
              @change="onMatrixModelSelectionChange(slotProps.data, modelCol)"
            />
          </template>
        </Column>

        <!-- Notes (僅 Matrix 模式顯示，下限 100px 且超過部分截斷顯示) -->
        <Column field="notes" header="Notes" :style="{ width: columnWidths.notes + 'px', minWidth: '100px', maxWidth: columnWidths.notes + 'px' }">
          <template #body="slotProps">
            <div
              class="bom-notes-cell cell-text cursor-pointer transition-colors rounded px-1 -mx-1 w-full min-h-[20px]"
              :class="[
                isEditingNotesCell(slotProps.data)
                  ? 'ring-1 ring-primary ring-inset bg-primary/10'
                  : 'hover:bg-slate-100 dark:hover:bg-[#2d2d2d]'
              ]"
              @click="handleNotesCellClick($event, slotProps.data)"
              @mouseenter="!isNotesEditorVisible && handleCellMouseEnter($event, 'notes', slotProps.data)"
              @mouseleave="handleCellMouseLeave"
            >
              <BOMHighlightText :text="slotProps.data.notes" :query="searchQuery" />
            </div>
          </template>
        </Column>
      </template>
    </DataTable>

    <!-- 右鍵選單元件 -->
    <ContextMenu ref="contextMenuRef" :model="contextMenuItems" />

    <!-- 儲存格互動式懸停卡片 (支援文字選取、複製與 Notes 多行編輯) -->
    <BOMCellHoverCard
      :visible="isCardVisible && !isNotesEditorVisible"
      :field="activeField"
      :row="activeRow"
      :content="activeContent"
      :target-rect="targetRect"
      :is-editing="isEditing"
      v-model:draft-notes="draftNotes"
      @card-mouse-enter="handleCardMouseEnter"
      @card-mouse-leave="handleCardMouseLeave"
      @close="closeCard(true)"
      @start-editing="startEditing"
      @cancel-editing="cancelEditing"
      @save-notes="handleSaveNotes"
    />

    <!-- Notes 欄位專用小編輯視窗 (支援多行 Shift+Enter 換行，Enter 或點擊外部即時儲存) -->
    <BOMNotesEditor
      :visible="isNotesEditorVisible"
      :row="editingNotesRow"
      :target-rect="notesEditorTargetRect"
      :initial-notes="initialNotesValue"
      @save="handleNotesEditorSave"
      @cancel="handleNotesEditorCancel"
      @close="handleNotesEditorCancel"
    />

    <!-- Summary Statistics 底部統計列 -->
    <BOMTableSummary
      :total-main-parts="aggregatedParts.length"
      :selected-view="selectedView"
      :smd-parts-count="smdPartsCount"
      :pth-parts-count="pthPartsCount"
    />

    <!-- Matrix Mode 機種設定編輯視窗 (Model 數量與 Model Qty 卡片式編輯) -->
    <MatrixModelEditDialog
      v-model:visible="isModelEditDialogVisible"
      :revision-id="modelEditRevisionId"
      :project-code="modelEditProjectCode"
      :phase="modelEditPhase"
      :version="modelEditVersion"
      :initial-models="modelEditInitialModels"
      @saved="handleModelEditSaved"
    />
  </div>
</template>

<script setup lang="ts">
/**
 * @file BOMTable.vue
 * @description BOM 表格核心裝配層元件
 * 
 * 本元件為 BOMTable 模組的純粹裝配層，將資料流 (useBOMData)、自適應欄寬 (useColumnWidths)、
 * 儲存格平滑拖曳 (useCellAutoScroll)、快顯選單 (useBOMContextMenu)、懸停卡片 (useCellHoverCard)
 * 以及 Notes 編輯引擎 (useBOMNotesEditing) 裝配至 PrimeVue DataTable 虛擬滾動容器中。
 */

import { ref, toRef, watch, nextTick, onMounted, onUnmounted } from 'vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import ContextMenu from 'primevue/contextmenu'
import Button from 'primevue/button'

import type { ViewPartGroup } from '../../services/api'
import type { BOMDisplayRow, MatrixModelColumnInfo } from './types'

// 子組件
import BOMToolbar from './components/BOMToolbar.vue'
import BOMTableSummary from './components/BOMTableSummary.vue'
import BOMHighlightText from './components/BOMHighlightText.vue'
import BOMItemCell from './components/BOMItemCell.vue'
import BOMMatrixHeader from './components/BOMMatrixHeader.vue'
import BOMMatrixCheckboxCell from './components/BOMMatrixCheckboxCell.vue'
import BOMCellHoverCard from './components/BOMCellHoverCard.vue'
import BOMNotesEditor from './components/BOMNotesEditor.vue'
import MatrixModelEditDialog from './components/MatrixModelEditDialog.vue'

// Composables
import { useBOMData } from './composables/useBOMData'
import { useColumnWidths } from './composables/useColumnWidths'
import { useCellAutoScroll } from './composables/useCellAutoScroll'
import { useBOMContextMenu } from './composables/useBOMContextMenu'
import { useCellHoverCard } from './composables/useCellHoverCard'
import { useBOMNotesEditing } from './composables/useBOMNotesEditing'

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
  revisionColumns,
  matrixModelColumns,
  smdPartsCount,
  pthPartsCount,
  collapseState,
  onSort,
  loadBOMData,
  onViewChange,
  isModelSelectedInRevision,
  isModelAvailableInRevision,
  onMatrixModelSelectionChange,
  getRowClass,
  getCCLClass,
  updateMaterialNotesInCache,
} = bomData

const {
  isCollapsed,
  toggleCollapse,
  isAllCollapsed,
  totalExpandableCount,
  toggleAllCollapse,
} = collapseState

/**
 * 取得指定資料列在特定 Revision 的用量顯示
 * 
 * @param {BOMDisplayRow} row - 資料列
 * @param {number} revisionId - BOM Revision ID
 * @returns {string | number} 用量數字或空字串
 */
function getRevisionQty(row: BOMDisplayRow, revisionId: number): string | number {
  if (!row.sourceRevisionIds || !row.sourceRevisionIds.includes(revisionId)) {
    return ''
  }
  const q = row.qtyByRevision?.[revisionId]
  return q !== undefined ? q : ''
}

// ── 2. 最適欄寬計算與自適應分配 ────
const {
  columnWidths,
  tableWrapperRef,
  computeColumnWidths,
  setupResizeListener,
} = useColumnWidths()

/** 執行欄寬重算 */
function triggerColumnWidthsCompute(): void {
  const totalMatrixModelWidth = matrixModelColumns.value.reduce((sum, c) => sum + (c.columnWidth || 72), 0)
  const totalEBOMRevisionWidth = revisionColumns.value.reduce((sum, c) => sum + (c.columnWidth || 54), 0)
  computeColumnWidths(
    displayRows.value,
    selectedBomType.value,
    revisionColumns.value.length,
    matrixModelColumns.value.length,
    totalMatrixModelWidth,
    totalEBOMRevisionWidth
  )
}

// 監聽顯示列資料、模式、版本欄位或 Model 欄位變化，自動重新精確計算各欄最適欄寬
watch(
  [() => displayRows.value, () => selectedBomType.value, () => revisionColumns.value.length, () => matrixModelColumns.value.length],
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

// ── 5. 儲存格互動式懸停卡片 ────
const {
  isCardVisible,
  activeField,
  activeRow,
  activeContent,
  targetRect,
  isEditing,
  draftNotes,
  handleCellMouseEnter,
  handleCellMouseLeave,
  handleCardMouseEnter,
  handleCardMouseLeave,
  closeCard,
  startEditing,
  cancelEditing,
  saveNotes,
  onTableScroll,
} = useCellHoverCard()

// ── 6. Notes 編輯狀態管理與持久化 ────
const {
  isNotesEditorVisible,
  editingNotesRow,
  notesEditorTargetRect,
  initialNotesValue,
  isEditingNotesCell,
  handleNotesCellClick,
  handleNotesEditorSave,
  handleNotesEditorCancel,
  handleSaveNotes,
} = useBOMNotesEditing({
  closeCard,
  saveNotes,
  updateMaterialNotesInCache,
})

/** 虛擬滾動容器元素引用 */
let scrollerEl: HTMLElement | null = null

/** 表格滾動處理監聽函式 */
function handleTableScrollerScroll(e: Event): void {
  onTableScroll(e)
  // 若滾動發生且編輯視窗開啟中，自動取消並關閉避免視窗飄移
  if (isNotesEditorVisible.value) {
    handleNotesEditorCancel()
  }
}

// ── 7. Matrix Model 機種設定對話框 (點擊表頭彈出) ────
const isModelEditDialogVisible = ref(false)
const modelEditRevisionId = ref(0)
const modelEditProjectCode = ref('')
const modelEditPhase = ref('')
const modelEditVersion = ref('')
const modelEditInitialModels = ref<MatrixModelColumnInfo[]>([])

/**
 * 點擊 Matrix Mode 表頭之 Model 雙行區域時開啟機種設定編輯對話框
 * @param {MatrixModelColumnInfo} col - 點擊之 Model 欄位資訊
 */
function handleModelHeaderClick(col: MatrixModelColumnInfo): void {
  modelEditRevisionId.value = col.revisionId
  modelEditProjectCode.value = col.projectCode
  modelEditPhase.value = col.phase
  modelEditVersion.value = col.version

  // 篩選屬於同一個 Revision 的所有 Model 欄位
  modelEditInitialModels.value = matrixModelColumns.value.filter(
    (c) => c.revisionId === col.revisionId
  )
  isModelEditDialogVisible.value = true
}

/**
 * 機種設定儲存完成後，即時刷新 BOM 資料以更新表頭與欄位
 */
async function handleModelEditSaved(): Promise<void> {
  if (props.revisionIds && props.revisionIds.length > 0) {
    await loadBOMData(props.revisionIds)
  }
}

onMounted(() => {
  setupResizeListener(() => {
    triggerColumnWidthsCompute()
  })

  // 監聽 DataTable 內部虛擬滾動容器之 scroll 事件，滾動時自動關閉懸停卡片避免漂移
  if (tableWrapperRef.value) {
    scrollerEl = tableWrapperRef.value.querySelector('.p-datatable-table-container, [data-pc-name="virtualscroller"]')
    if (scrollerEl) {
      scrollerEl.addEventListener('scroll', handleTableScrollerScroll, { passive: true })
    }
  }
})

onUnmounted(() => {
  if (scrollerEl) {
    scrollerEl.removeEventListener('scroll', handleTableScrollerScroll)
    scrollerEl = null
  }
})
</script>

<style scoped>
@import './styles/bomTable.css';
</style>
