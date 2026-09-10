<template>
  <div class="bom-table-container" ref="tableWrapperRef">
    <!-- View Filter & Mode Toolbar (PrimeVue Toolbar) -->
    <Toolbar class="bom-toolbar">
      <template #start>
        <div class="toolbar-start">
          <span class="filter-label">View:</span>
          <Select
            v-model="selectedView"
            :options="viewOptions"
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
            :options="bomTypeOptions"
            :allow-empty="false"
            size="small"
            class="bom-type-toggle"
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
        </template>
      </Column>

      <!-- HHPN -->
      <Column field="hhpn" header="HHPN" :style="{ width: columnWidths.hhpn + 'px' }" sortable>
        <template #body="slotProps">
          <div class="cell-text" v-tooltip.bottom="slotProps.data.hhpn">
            <template v-for="(part, idx) in getHighlightedParts(slotProps.data.hhpn, searchQuery)" :key="idx">
              <mark v-if="part.isMatch" class="highlight-text">{{ part.text }}</mark>
              <span v-else>{{ part.text }}</span>
            </template>
          </div>
        </template>
      </Column>

      <!-- Description -->
      <Column field="description" header="Description" :style="{ width: columnWidths.description + 'px', minWidth: '250px' }" sortable>
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
      <Column field="locations" header="Location" :style="{ width: columnWidths.locations + 'px', minWidth: '150px' }">
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
import { ref, shallowRef, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import DataTable, { type DataTableSortEvent } from 'primevue/datatable'
import Column from 'primevue/column'
import Toolbar from 'primevue/toolbar'
import SelectButton from 'primevue/selectbutton'
import ContextMenu from 'primevue/contextmenu'
import Select from 'primevue/select'
import InputText from 'primevue/inputtext'
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import { useProjectStore, useLogStore } from '../stores'
import { GetBOMView, type ViewPartGroup, type ViewRevision } from '../services/api'

// ── 欄寬計算與狀態管理 ────
interface ColumnWidthConfig {
  item: number
  hhpn: number
  description: number
  supplier: number
  supplier_pn: number
  qty: number
  locations: number
  ccl: number
  remark: number
  models: Record<string, number>
}

const defaultColumnWidths: ColumnWidthConfig = {
  item: 54,
  hhpn: 140,
  description: 240,
  supplier: 120,
  supplier_pn: 150,
  qty: 55,
  locations: 220,
  ccl: 50,
  remark: 130,
  models: {}
}

const columnWidths = ref<ColumnWidthConfig>({ ...defaultColumnWidths })
/** 表格容器 template ref，用於精準讀取可用寬度 */
const tableWrapperRef = ref<HTMLElement | null>(null)
let measureCanvas: HTMLCanvasElement | null = null
let resizeObserver: ResizeObserver | null = null

/**
 * 測量文字在指定字體下的像素寬度 (使用離屏 Canvas API 避免觸發 DOM 重新排版)
 * @param {string} text - 待測量的文字字串
 * @param {boolean} [isHeader=false] - 是否為表頭 (表頭使用加粗 600 字重)
 * @returns {number} 文字像素寬度
 */
function measureTextWidth(text: string, isHeader = false): number {
  if (!text) return 0
  if (!measureCanvas) {
    measureCanvas = document.createElement('canvas')
  }
  const ctx = measureCanvas.getContext('2d')
  if (!ctx) {
    return text.length * 7.5
  }
  ctx.font = isHeader
    ? '600 12px -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif'
    : '12px -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif'
  return ctx.measureText(text).width
}

/**
 * 取得 DataTable 實際可用欄寬總量（最精準量測策略）
 *
 * 量測優先順序：
 * 1. 讀取 DataTable 內部 VirtualScroller 的 clientWidth
 *    - VirtualScroller 是 DataTable body 的直接捲動容器（帶有垂直 scrollbar）
 *    - clientWidth 已自動扣除垂直 scrollbar 佔用的寬度，無需手動估算
 * 2. 回退到 tableWrapperRef.clientWidth 並扣除 scrollbar 估算值
 * 3. 最後回退使用 window.innerWidth 粗估
 *
 * @returns {number} DataTable 可用欄寬總量 (像素)
 */
function getWorkspaceVisibleWidth(): number {
  const wrapperEl = tableWrapperRef.value

  // 優先讀取 DataTable 內部 VirtualScroller 的 clientWidth
  // VirtualScroller 的 clientWidth 已自動扣除垂直 scrollbar，是最精準的量測基準
  if (wrapperEl) {
    const vscroller = wrapperEl.querySelector('[data-pc-name="virtualscroller"]') as HTMLElement | null
    if (vscroller && vscroller.clientWidth > 0) {
      // 減去 1px 避免瀏覽器次像素浮點數捨入引發 1px 溢出
      return Math.max(vscroller.clientWidth - 1, 400)
    }

    // 回退：直接讀取容器寬度，並扣除垂直 scrollbar 估算值
    // 在 Chromium / WebView2 上 Windows 預設 scrollbar 寬度約 15-17px，取 17 較保守
    if (wrapperEl.clientWidth > 0) {
      return Math.max(wrapperEl.clientWidth - 17, 400)
    }
  }

  // 最後回退：以 window.innerWidth 粗估（不建議，僅防止計算出 NaN 或 0）
  return Math.max(window.innerWidth / 2, 400)
}

/**
 * 計算各欄位最適欄寬 (包含最小寬度、Location 初始 30 字元、以及 Description 剩餘空間分配)
 * 依據規則：
 * 1. Description 欄位 min-width 為 250px，Location 欄位 min-width 為 150px。
 * 2. 總寬度以 app 整體分配給 Workspace 的可視寬度計算。
 * 3. 若可視寬度能滿足每個欄位的 min-width，欄位總寬度精確配合可視寬度，不出現橫向捲軸。
 * 4. 僅在可視寬度不足以滿足各欄 min-width 時，各欄退守 min-width 並以最緊湊寬度呈現。
 * 5. 拖曳 main content splitter、切換 revision/view、或視窗縮放時皆即時重新計算。
 */
function computeColumnWidths(): void {
  const rows = displayRows.value
  const models = currentRevisionModels.value

  const MIN_DESC_WIDTH = 250
  const MIN_LOC_WIDTH = 150

  // 1. 固定欄位最小寬度計算
  const itemWidth = 46

  // 各欄位表頭標題測量 (加上 padding 8px 與排序箭頭 16px)
  let maxHhpn = measureTextWidth('HHPN', true) + 20
  let maxSupplier = measureTextWidth('Supplier', true) + 20
  let maxSupplierPn = measureTextWidth('Supplier PN', true) + 20
  let maxQty = measureTextWidth('Qty', true) + 20
  let maxRemark = measureTextWidth('Remark', true) + 14
  let maxDesc = measureTextWidth('Description', true) + 20

  // 動態 Model 欄位文字寬度需求
  const modelMaxMap: Record<string, number> = {}
  for (const m of models) {
    const headerTitle = `${m} (Qty: ${getModelQty(m)})`
    modelMaxMap[m] = measureTextWidth(headerTitle, true) + 20
  }

  // 遍歷當前所有顯示列資料以取得實際長度
  for (let i = 0; i < rows.length; i++) {
    const row = rows[i]
    if (row.hhpn) {
      const w = measureTextWidth(row.hhpn) + 12
      if (w > maxHhpn) maxHhpn = w
    }
    if (row.supplier) {
      const w = measureTextWidth(row.supplier) + 12
      if (w > maxSupplier) maxSupplier = w
    }
    if (row.supplier_pn) {
      const w = measureTextWidth(row.supplier_pn) + 12
      if (w > maxSupplierPn) maxSupplierPn = w
    }
    if (row.qty !== '' && row.qty !== undefined && row.qty !== null) {
      const w = measureTextWidth(String(row.qty)) + 12
      if (w > maxQty) maxQty = w
    }
    if (row.remark) {
      const w = measureTextWidth(row.remark) + 12
      if (w > maxRemark) maxRemark = w
    }
    if (row.description) {
      const w = measureTextWidth(row.description) + 12
      if (w > maxDesc) maxDesc = w
    }
    for (const m of models) {
      const pn = getModelSelectedPN(row, m)
      if (pn) {
        const w = measureTextWidth(pn) + 12
        if (w > modelMaxMap[m]) modelMaxMap[m] = w
      }
    }
  }

  // 限制各固定欄位緊湊最小安全寬度 (根據內容長度緊貼，不浪費多餘像素)
  const hhpnColWidth = Math.ceil(Math.min(150, Math.max(80, maxHhpn)))
  const supplierColWidth = Math.ceil(Math.min(140, Math.max(65, maxSupplier)))
  const supplierPnColWidth = Math.ceil(Math.min(160, Math.max(85, maxSupplierPn)))
  const qtyColWidth = Math.ceil(Math.min(60, Math.max(40, maxQty)))
  const cclColWidth = 38
  const remarkColWidth = Math.ceil(Math.min(140, Math.max(55, maxRemark)))

  const finalModelWidths: Record<string, number> = {}
  let totalModelsWidth = 0
  for (const m of models) {
    const w = Math.ceil(Math.min(130, Math.max(85, modelMaxMap[m] || 85)))
    finalModelWidths[m] = w
    totalModelsWidth += w
  }

  // 固定欄位最小寬度總和
  const fixedTotal = itemWidth + hhpnColWidth + supplierColWidth + supplierPnColWidth + qtyColWidth + cclColWidth + remarkColWidth + totalModelsWidth
  // 所有欄位 min-width 需求總和
  const totalRequiredMinWidth = fixedTotal + MIN_LOC_WIDTH + MIN_DESC_WIDTH

  // 2. 取得 App 真正分配給 Workspace 的可視寬度 (不受內部 Table 寬度污染)
  const visibleWidth = getWorkspaceVisibleWidth()

  let finalDescWidth = MIN_DESC_WIDTH
  let finalLocWidth = MIN_LOC_WIDTH

  if (visibleWidth <= totalRequiredMinWidth) {
    // 情況 1：可視寬度不足以容納所有欄位的 min-width，退守至各自 min-width，以緊湊寬度呈現
    finalDescWidth = MIN_DESC_WIDTH
    finalLocWidth = MIN_LOC_WIDTH
  } else {
    // 情況 2：可視寬度足以滿足所有欄位的 min-width：所有欄位寬度總和完全貼合可視寬度，不出現橫向捲軸！
    // 預留 1px 避免瀏覽器次像素浮點數微幅溢出
    const safeTotal = Math.floor(visibleWidth) - 1
    const rem = safeTotal - fixedTotal // 供 Description 與 Location 瓜分之總空間 (必定 >= 400px)

    // Location 初始理想寬度 (容納 30 個字元，約 210px，但不低於 MIN_LOC_WIDTH 150px)
    const thirtyCharsWidth = measureTextWidth('0'.repeat(30))
    const locInit = Math.max(MIN_LOC_WIDTH, Math.ceil(thirtyCharsWidth + 12))

    if (rem - locInit >= MIN_DESC_WIDTH) {
      // 扣除 30 字元 Location 後，Description 至少能拿到 250px
      let descW = rem - locInit
      let locW = locInit

      // 檢查 Description 是否已足夠完整顯示其所有資料
      const descNeeded = Math.max(MIN_DESC_WIDTH, Math.ceil(maxDesc + 14))
      if (descW > descNeeded) {
        // Description 已足夠完整顯示，多餘寬度再全額分配給 Location
        const surplus = descW - descNeeded
        descW = descNeeded
        locW = locW + surplus
      }

      finalDescWidth = descW
      finalLocWidth = locW
    } else {
      // 空間大於 400px 但扣除 30 字元 Location 後 Description 不足 250px，
      // 優先保障 Description 的 250px min-width，其餘全部分配給 Location
      finalDescWidth = MIN_DESC_WIDTH
      finalLocWidth = rem - MIN_DESC_WIDTH
    }
  }

  columnWidths.value = {
    item: itemWidth,
    hhpn: hhpnColWidth,
    description: finalDescWidth,
    supplier: supplierColWidth,
    supplier_pn: supplierPnColWidth,
    qty: qtyColWidth,
    locations: finalLocWidth,
    ccl: cclColWidth,
    remark: remarkColWidth,
    models: finalModelWidths
  }

  console.debug('[BOMTable 欄寬計算]', {
    visibleWidth,
    fixedTotal,
    finalDescWidth,
    finalLocWidth,
    totalAssigned: fixedTotal + finalDescWidth + finalLocWidth
  })
}

// Display row interface for flattened table rendering
export interface BOMDisplayRow {
  rowId: string
  parentKey: string
  isSecondSource: boolean
  hasSecondSources: boolean
  secondSourcesCount: number
  item: string
  hhpn: string
  description: string
  supplier: string
  supplier_pn: string
  qty: string | number
  locations: string
  ccl: boolean
  remark: string
  selections: Record<string, string>
}

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

const projectStore = useProjectStore()
const logStore = useLogStore()

// BOM 視圖模式選項 (EBOM / Matrix 互斥選項，資料切換後續實現)
const bomTypeOptions = ['EBOM', 'Matrix']
const selectedBomType = ref<'EBOM' | 'Matrix'>('EBOM')

// View options
const viewOptions = [
  { label: 'All', value: 'all' },
  { label: 'SMD', value: 'smd' },
  { label: 'PTH', value: 'pth' },
  { label: 'Bottom', value: 'bottom' },
  { label: 'NI', value: 'ni' },
  { label: 'PROTO', value: 'proto' },
  { label: 'MP', value: 'mp' },
  { label: 'CCL', value: 'ccl' },
]

// State
const dataTableRef = ref()
const selectedView = ref('all')
const collapsedParents = ref<Set<string>>(new Set())
const sortField = ref('')
const sortOrder = ref(1)
const searchQuery = ref('')

// Computed properties
/**
 * 取得主顯示的 Revision ID（多選時目前先以第 1 個 revisionId 為主）
 */
const currentRevisionId = computed(() => {
  if (props.revisionIds && props.revisionIds.length > 0) {
    return props.revisionIds[0]
  }
  return 0
})

const aggregatedParts = shallowRef<ViewPartGroup[]>([])
const currentRevisionMetadata = shallowRef<ViewRevision | null>(null)

const smdPartsCount = computed(() => {
  return aggregatedParts.value.filter(p => p.type === 'SMD').length
})

const pthPartsCount = computed(() => {
  return aggregatedParts.value.filter(p => p.type === 'PTH').length
})

const currentRevisionModels = computed(() => {
  if (!currentRevisionMetadata.value || !currentRevisionMetadata.value.model_names) return []
  return currentRevisionMetadata.value.model_names
})

/**
 * 依據當前選定的排序欄位 (sortField) 與排序方向 (sortOrder)，
 * 對物料群組 (ViewPartGroup) 進行 Group 層級的排序。
 * 確保排序作用於主料，而非打散平鋪後的個別 row。
 */
const sortedAggregatedParts = computed<ViewPartGroup[]>(() => {
  if (!aggregatedParts.value || aggregatedParts.value.length === 0) return []

  let list = [...aggregatedParts.value]

  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    list = list.filter((part: any) => {
      // 檢查主料
      const mainMatch = 
        (part.hhpn && part.hhpn.toLowerCase().includes(q)) ||
        (part.description && part.description.toLowerCase().includes(q)) ||
        (part.main_supplier && part.main_supplier.toLowerCase().includes(q)) ||
        (part.main_supplier_pn && part.main_supplier_pn.toLowerCase().includes(q)) ||
        (part.locations && part.locations.toLowerCase().includes(q)) ||
        (part.remark && part.remark.toLowerCase().includes(q))

      if (mainMatch) return true

      // 檢查替代料 (只要任一替代料符合，整個 Group 都會顯示)
      if (part.second_sources && part.second_sources.length > 0) {
        return part.second_sources.some((ss: any) => 
          (ss.hhpn && ss.hhpn.toLowerCase().includes(q)) ||
          (ss.description && ss.description.toLowerCase().includes(q)) ||
          (ss.supplier && ss.supplier.toLowerCase().includes(q)) ||
          (ss.supplier_pn && ss.supplier_pn.toLowerCase().includes(q)) ||
          (ss.remark && ss.remark.toLowerCase().includes(q))
        )
      }

      return false
    })
  }

  const field = sortField.value
  const order = sortOrder.value

  if (!field) {
    // 預設依 item 自然數字序排序，保持穩定次序
    list.sort((a, b) => {
      const itemA = parseFloat(a.item)
      const itemB = parseFloat(b.item)
      if (!isNaN(itemA) && !isNaN(itemB)) {
        return itemA - itemB
      }
      return String(a.item || '').localeCompare(String(b.item || ''), undefined, { numeric: true, sensitivity: 'base' })
    })
    return list
  }

  list.sort((a, b) => {
    let valA: unknown = ''
    let valB: unknown = ''

    if (field === 'item') {
      valA = a.item || ''
      valB = b.item || ''
    } else if (field === 'hhpn') {
      valA = a.hhpn || ''
      valB = b.hhpn || ''
    } else if (field === 'supplier') {
      valA = a.main_supplier || ''
      valB = b.main_supplier || ''
    } else if (field === 'supplier_pn') {
      valA = a.main_supplier_pn || ''
      valB = b.main_supplier_pn || ''
    } else if (field === 'qty') {
      valA = a.qty ?? 0
      valB = b.qty ?? 0
    } else if (field === 'ccl') {
      valA = a.ccl ? 1 : 0
      valB = b.ccl ? 1 : 0
    } else if (field === 'description') {
      valA = a.description || ''
      valB = b.description || ''
    } else if (field === 'remark') {
      valA = a.remark || ''
      valB = b.remark || ''
    } else {
      valA = (a as unknown as Record<string, unknown>)[field] || ''
      valB = (b as unknown as Record<string, unknown>)[field] || ''
    }

    let compareRes = 0
    if (field === 'item' || field === 'qty') {
      const numA = parseFloat(String(valA))
      const numB = parseFloat(String(valB))
      if (!isNaN(numA) && !isNaN(numB)) {
        compareRes = numA - numB
      } else {
        compareRes = String(valA).localeCompare(String(valB), undefined, { numeric: true, sensitivity: 'base' })
      }
    } else if (field === 'ccl') {
      compareRes = (Number(valA) - Number(valB))
    } else {
      compareRes = String(valA).localeCompare(String(valB), undefined, { numeric: true, sensitivity: 'base' })
    }

    if (compareRes !== 0) {
      return compareRes * order
    }

    // Tie-breaker: 依 item 自然數字序進行次要排序
    const itemA = parseFloat(a.item)
    const itemB = parseFloat(b.item)
    if (!isNaN(itemA) && !isNaN(itemB)) {
      return itemA - itemB
    }
    return String(a.item).localeCompare(String(b.item), undefined, { numeric: true, sensitivity: 'base' })
  })

  return list
})

// Flattened display rows computed property
const displayRows = computed<BOMDisplayRow[]>(() => {
  const rows: BOMDisplayRow[] = []

  sortedAggregatedParts.value.forEach((part) => {
    const parentKey = getPartKey(part)
    const hasSS = Boolean(part.second_sources && part.second_sources.length > 0)
    const ssCount = part.second_sources ? part.second_sources.length : 0

    // Build Model selections map: model_name -> selected_pn
    const selectionsMap: Record<string, string> = {}
    if (part.selections) {
      part.selections.forEach(sel => {
        if (sel.model_name && sel.selected_pn) {
          selectionsMap[sel.model_name] = sel.selected_pn
        }
      })
    }

    // 1. Add Main Source Row
    rows.push({
      rowId: `${parentKey}-main`,
      parentKey: parentKey,
      isSecondSource: false,
      hasSecondSources: hasSS,
      secondSourcesCount: ssCount,
      item: part.item || '',
      hhpn: part.hhpn || '',
      description: part.description || '',
      supplier: part.main_supplier || '',
      supplier_pn: part.main_supplier_pn || '',
      qty: part.qty ?? '',
      locations: part.locations || '',
      ccl: Boolean(part.ccl),
      remark: part.remark || '',
      selections: selectionsMap,
    })

    // 2. Add 2nd Source Rows directly below Main Source (if not collapsed)
    if (hasSS && part.second_sources && !collapsedParents.value.has(parentKey)) {
      part.second_sources.forEach((ss, idx) => {
        rows.push({
          rowId: `${parentKey}-ss-${idx}-${ss.supplier_pn || idx}`,
          parentKey: parentKey,
          isSecondSource: true,
          hasSecondSources: false,
          secondSourcesCount: 0,
          item: '', // 2nd 替代料沒有 Item number，該欄位為空白！
          hhpn: ss.hhpn || '',
          description: ss.description || '',
          supplier: ss.supplier || '',
          supplier_pn: ss.supplier_pn || '',
          qty: '',
          locations: '',
          ccl: false,
          remark: ss.remark || '',
          selections: selectionsMap,
        })
      })
    }
  })

  return rows
})

/**
 * DataTable 排序事件處理常式
 * 當使用者點擊表頭進行排序時，更新 sortField 與 sortOrder，
 * 以觸發 Group 層級的重新排序，確保 2nd 替代料始終緊跟對應主料正下方。
 * 
 * @param event DataTable 的 sort 事件物件
 */
function onSort(event: DataTableSortEvent): void {
  if (typeof event.sortField === 'string') {
    sortField.value = event.sortField
    sortOrder.value = event.sortOrder ?? 1
  }
}

/**
 * 取得物料群組唯一識別鍵
 * 結合 item、type、main_supplier 與 main_supplier_pn，確保每個主料群組（包含同料不同上件類型）皆具備獨立唯一的鍵值
 * 
 * @param part 物料群組資料
 * @returns 唯一識別字串
 */
function getPartKey(part: ViewPartGroup): string {
  const item = part.item || ''
  const pType = part.type || ''
  const supplier = part.main_supplier || ''
  const pn = part.main_supplier_pn || ''
  return `${item}|${pType}|${supplier}|${pn}`
}

function isCollapsed(parentKey: string): boolean {
  return collapsedParents.value.has(parentKey)
}

function toggleCollapse(parentKey: string): void {
  const newSet = new Set(collapsedParents.value)
  if (newSet.has(parentKey)) {
    newSet.delete(parentKey)
  } else {
    newSet.add(parentKey)
  }
  collapsedParents.value = newSet
}

/**
 * 所有具有替代料的主料群組唯一識別鍵集合
 */
const allExpandableKeys = computed<Set<string>>(() => {
  const keys = new Set<string>()
  aggregatedParts.value.forEach(part => {
    if (part.second_sources && part.second_sources.length > 0) {
      keys.add(getPartKey(part))
    }
  })
  return keys
})

/**
 * 具有替代料的主料群組數量 (依據唯一識別鍵統計)
 */
const totalExpandableCount = computed(() => {
  return allExpandableKeys.value.size
})

/**
 * 判斷是否所有替代料群組皆處於收合狀態
 */
const isAllCollapsed = computed(() => {
  if (allExpandableKeys.value.size === 0) return false
  if (collapsedParents.value.size < allExpandableKeys.value.size) return false
  for (const key of allExpandableKeys.value) {
    if (!collapsedParents.value.has(key)) {
      return false
    }
  }
  return true
})

/**
 * 全部展開所有替代料
 */
function expandAll(): void {
  collapsedParents.value = new Set()
}

/**
 * 全部收合所有替代料
 */
function collapseAll(): void {
  collapsedParents.value = new Set(allExpandableKeys.value)
}

/**
 * 切換所有替代料群組之展開 / 收合狀態 (供表頭第一欄圖標點擊使用)
 */
function toggleAllCollapse(): void {
  if (isAllCollapsed.value) {
    expandAll()
  } else {
    collapseAll()
  }
}

function getRowClass(data: BOMDisplayRow) {
  return data.isSecondSource ? 'second-source-row' : 'main-source-row'
}

/**
 * 將文字依據關鍵字切割為符合與不符合的片段，用於高亮顯示關鍵字
 */
function getHighlightedParts(text: string | number | null | undefined, query: string): { text: string; isMatch: boolean }[] {
  const str = String(text ?? '')
  if (!query || !query.trim() || !str) {
    return [{ text: str, isMatch: false }]
  }

  const parts: { text: string; isMatch: boolean }[] = []
  const q = query.trim()
  const lowerStr = str.toLowerCase()
  const lowerQ = q.toLowerCase()

  let startIndex = 0
  let matchIndex = lowerStr.indexOf(lowerQ, startIndex)

  while (matchIndex !== -1) {
    if (matchIndex > startIndex) {
      parts.push({
        text: str.substring(startIndex, matchIndex),
        isMatch: false,
      })
    }
    parts.push({
      text: str.substring(matchIndex, matchIndex + q.length),
      isMatch: true,
    })
    startIndex = matchIndex + q.length
    matchIndex = lowerStr.indexOf(lowerQ, startIndex)
  }

  if (startIndex < str.length) {
    parts.push({
      text: str.substring(startIndex),
      isMatch: false,
    })
  }

  return parts
}

function getModelQty(modelName: string): number {
  if (!currentRevisionMetadata.value || !currentRevisionMetadata.value.model_qty) return 0
  return currentRevisionMetadata.value.model_qty[modelName] || 0
}

function getModelSelectedPN(row: BOMDisplayRow, modelName: string): string {
  const selectedPN = row.selections[modelName]
  if (!selectedPN) return ''
  // 僅當選擇的料號正是此 Row（主料或 2nd）的 supplier_pn 時才顯示
  return row.supplier_pn === selectedPN ? selectedPN : ''
}

function isModelSelected(row: BOMDisplayRow, modelName: string): boolean {
  return getModelSelectedPN(row, modelName) !== ''
}

function onViewChange(): void {
  collapsedParents.value = new Set()
  if (props.revisionIds && props.revisionIds.length > 0) {
    loadBOMData(props.revisionIds)
  }
}

function getCCLClass(ccl: boolean): string {
  return ccl ? 'ccl-critical' : 'ccl-normal'
}

// Watch for revisionIds changes
watch(
  () => props.revisionIds,
  (newIds) => {
    if (newIds && newIds.length > 0) {
      loadBOMData(newIds)
    } else {
      aggregatedParts.value = []
      collapsedParents.value = new Set()
      currentRevisionMetadata.value = null
    }
  },
  { deep: true }
)

/**
 * 載入指定 BOM Revision 清單的物料群組視圖資料
 * 目前架構接收 revisionIds 陣列，並以第一個 revisionId 載入顯示內容，為後續多版本聚合預留擴充接口
 * @param {number[]} revisionIds - BOM Revision ID 陣列
 */
async function loadBOMData(revisionIds: number[]): Promise<void> {
  if (!revisionIds || revisionIds.length === 0) {
    aggregatedParts.value = []
    collapsedParents.value = new Set()
    currentRevisionMetadata.value = null
    return
  }

  try {
    const primaryId = revisionIds[0]
    const viewType = selectedView.value === 'all' ? '' : selectedView.value.toUpperCase()
    logStore.addLogEntry('DEBUG', `[View System] 準備建立 View: RevisionIDs=[${revisionIds.join(', ')}] (Primary ID: ${primaryId}), ViewType="${viewType || 'ALL'}"`)
    const result = await GetBOMView([primaryId], viewType)
    
    if (result && result.part_groups) {
      aggregatedParts.value = result.part_groups
      // 預設全部 2nd 替代料展開直接顯示於主料正下方
      expandAll()
    } else {
      aggregatedParts.value = []
      collapsedParents.value = new Set()
    }

    if (result && result.revisions && result.revisions.length > 0) {
      currentRevisionMetadata.value = result.revisions[0]
    } else {
      currentRevisionMetadata.value = null
    }
  } catch (error) {
    const msg = error instanceof Error ? error.message : String(error)
    logStore.addLogEntry('ERROR', `載入 BOM 資料失敗: ${msg}`)
  }
}

// ── 右鍵選單與快捷鍵支援 (Custom Context Menu & Shortcuts) ────
const contextMenuRef = ref()
const selectedContextRow = ref<BOMDisplayRow | null>(null)
/** 目前左鍵點選啟用的資料列（用於 Ctrl+C 快捷鍵整列複製） */
const activeSelectedRow = ref<BOMDisplayRow | null>(null)

/**
 * 處理表格列左鍵點選事件
 * @param event - PrimeVue DataTable row-click 事件
 */
function onRowClick(event: any): void {
  activeSelectedRow.value = event.data
}

/**
 * 複製純文字至剪貼簿
 * @param text - 要複製的字串內容
 */
async function copyText(text?: string): Promise<void> {
  if (!text) return
  try {
    await navigator.clipboard.writeText(text)
  } catch (err) {
    console.error('Failed to copy text to clipboard:', err)
  }
}

/**
 * 複製整列資料為 Tab 分隔字串 (TSV 格式，方便貼入 Excel)
 * @param row - 表格列資料
 */
function copyRowTSV(row?: BOMDisplayRow | null): void {
  if (!row) return
  const cols = [
    row.item || '',
    row.hhpn || '',
    row.description || '',
    row.supplier || '',
    row.supplier_pn || '',
    row.qty ?? '',
    row.locations || '',
    row.ccl ? 'Y' : '',
    row.remark || ''
  ]
  copyText(cols.join('\t'))
}

/**
 * 鍵盤 Ctrl+C (或 Cmd+C) 事件監聽常式
 * 若使用者有框選文字，讓瀏覽器原生複製生效；
 * 若無框選文字但有選中表格列，則自動複製該列完整 TSV 資料並在 Log 中提示。
 */
function handleKeyDown(event: KeyboardEvent): void {
  if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'c') {
    const selectedText = window.getSelection()?.toString()
    if (selectedText && selectedText.trim().length > 0) {
      // 已有手動框選文字，讓原生複製行為生效
      return
    }

    const targetRow = selectedContextRow.value || activeSelectedRow.value
    if (targetRow) {
      event.preventDefault()
      copyRowTSV(targetRow)
      logStore.addLogEntry('INFO', `已將料號 ${targetRow.hhpn || targetRow.item || '-'} 整列資料複製至剪貼簿`)
    }
  }
}

/**
 * 處理表格列右鍵事件
 * @param event - PrimeVue DataTable 拋出的 row-contextmenu 事件
 */
function onRowContextMenu(event: any): void {
  selectedContextRow.value = event.data
  activeSelectedRow.value = event.data
  contextMenuRef.value?.show(event.originalEvent)
}

/**
 * 計算表格右鍵選單項目
 */
const contextMenuItems = computed(() => {
  const row = selectedContextRow.value
  if (!row) return []

  const items: any[] = [
    {
      label: `複製料號 (${row.hhpn || '-'})`,
      icon: 'pi pi-copy',
      disabled: !row.hhpn,
      command: () => copyText(row.hhpn)
    },
    {
      label: '複製規格描述 (Copy Description)',
      icon: 'pi pi-align-left',
      disabled: !row.description,
      command: () => copyText(row.description)
    },
    {
      label: `複製供應商料號 (${row.supplier_pn || '-'})`,
      icon: 'pi pi-tag',
      disabled: !row.supplier_pn,
      command: () => copyText(row.supplier_pn)
    },
    {
      label: '複製整列資料 (TSV)',
      icon: 'pi pi-table',
      command: () => copyRowTSV(row)
    },
    { separator: true },
    {
      label: '依此料號篩選 (Filter by PN)',
      icon: 'pi pi-filter',
      disabled: !row.hhpn,
      command: () => {
        searchQuery.value = row.hhpn
      }
    }
  ]

  // 若為有替代料的主料，提供展開/收合選項
  if (!row.isSecondSource && row.hasSecondSources) {
    const collapsed = isCollapsed(row.parentKey)
    items.push({
      label: collapsed ? '展開替代料 (Expand 2nd Source)' : '收合替代料 (Collapse 2nd Source)',
      icon: collapsed ? 'pi pi-chevron-down' : 'pi pi-chevron-right',
      disabled: false,
      command: () => toggleCollapse(row.parentKey)
    })
  }

  return items
})

// 監聽 displayRows 變化 (切換 revision、篩選、展開收合替代料等)，重新精確計算各欄最適欄寬
watch(
  () => displayRows.value,
  () => {
    // nextTick 確保 Vue 完成響應式更新，requestAnimationFrame 確保瀏覽器完成一次 layout pass
    // 這樣 DataTable 的 VirtualScroller 才會被完整渲染，clientWidth 才能讀到正確值
    nextTick(() => {
      requestAnimationFrame(() => {
        computeColumnWidths()
      })
    })
  },
  { deep: false }
)

/**
 * 視窗大小變更時重新計算欄寬
 * 使用 requestAnimationFrame 確保瀏覽器 layout 穩定後再量測
 */
function onWindowResize(): void {
  requestAnimationFrame(() => {
    computeColumnWidths()
  })
}

onMounted(() => {
  window.addEventListener('keydown', handleKeyDown)
  window.addEventListener('resize', onWindowResize)
  if (props.revisionIds && props.revisionIds.length > 0) {
    loadBOMData(props.revisionIds)
  }

  // 監聽表格容器尺寸變化 (視窗縮放或側邊欄拖曳)，動態重新計算分配欄寬
  // 使用 templateRef 直接綁定，避免 querySelector 的跨元件選擇器衝突風險
  const containerEl = tableWrapperRef.value
  if (containerEl && typeof ResizeObserver !== 'undefined') {
    resizeObserver = new ResizeObserver(() => {
      // requestAnimationFrame 確保 layout 穩定後再量測，避免讀到過渡中的錯誤值
      requestAnimationFrame(() => {
        computeColumnWidths()
      })
    })
    resizeObserver.observe(containerEl)
  }
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeyDown)
  window.removeEventListener('resize', onWindowResize)
  if (resizeObserver) {
    resizeObserver.disconnect()
    resizeObserver = null
  }
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

/* 切換 EBOM / Matrix 的 SelectButton 選中狀態高亮 (參照 App.vue Title Bar 按鈕風格) */
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

/* table-layout: fixed 已移除——與 PrimeVue VirtualScroller 的行高計算機制衝突，無法正確使用 */

:deep(.p-datatable-header) {
  background: var(--surface-ground);
  border-bottom: 1px solid var(--surface-border);
}

/* 標題列：背景微調為 surface-100 (暗黑模式為 surface-800)，搭配清楚的底線，避免與斑馬紋重複 */
:deep(.bom-table .p-datatable-thead > tr > th) {
  padding: 0.12rem 0.25rem !important;
  font-size: 0.75rem !important;
  font-weight: 600;
  height: 26px !important;
  white-space: nowrap;
  border-bottom: 2px solid var(--surface-300, #cbd5e1) !important;
  background: var(--surface-100, #f1f5f9) !important;
  position: relative !important;
}

:global(.app-dark) :deep(.bom-table .p-datatable-thead > tr > th) {
  background: var(--surface-800, #1e293b) !important;
  border-bottom: 2px solid var(--surface-700, #334155) !important;
}

/* 調整欄寬時的拖曳控柄 (Resizer) 樣式 */
:deep(.bom-table .p-datatable-column-resizer) {
  width: 8px !important;
  right: 0 !important;
  top: 0 !important;
  bottom: 0 !important;
  cursor: col-resize !important;
  position: absolute !important;
  z-index: 10 !important;
}

:deep(.bom-table .p-datatable-column-resizer:hover) {
  background-color: var(--primary-color) !important;
  opacity: 0.4;
}

/* 標題列排序符號微調為 10px 高緊湊風格 */
:deep(.bom-table .p-datatable-sort-icon),
:deep(.bom-table .p-datatable-sort-icon svg) {
  width: 10px !important;
  height: 10px !important;
  min-width: 10px !important;
  min-height: 10px !important;
  font-size: 10px !important;
  transition: color 0.15s ease;
}

/* 排序符號生效時，採用與整體 UI 按鈕生效同款綠色 */
:deep(.bom-table th.p-datatable-column-sorted .p-datatable-sort-icon),
:deep(.bom-table th[data-p-sorted="true"] .p-datatable-sort-icon),
:deep(.bom-table th[aria-sort="ascending"] .p-datatable-sort-icon),
:deep(.bom-table th[aria-sort="descending"] .p-datatable-sort-icon),
:deep(.bom-table th.p-datatable-column-sorted .p-datatable-sort-icon svg),
:deep(.bom-table th[data-p-sorted="true"] .p-datatable-sort-icon svg),
:deep(.bom-table th[aria-sort="ascending"] .p-datatable-sort-icon svg),
:deep(.bom-table th[aria-sort="descending"] .p-datatable-sort-icon svg) {
  color: var(--primary-color) !important;
  fill: var(--primary-color) !important;
}

/* 資料列高度緊湊化為 26px，左右內距縮小至 0.25rem，最大化可視範圍 */
:deep(.bom-table .p-datatable-tbody > tr) {
  height: 26px !important;
  max-height: 26px !important;
}

:deep(.bom-table .p-datatable-tbody > tr > td) {
  padding: 0.05rem 0.25rem !important;
  font-size: 0.75rem !important;
  line-height: 1.25 !important;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  -webkit-user-select: text !important;
  user-select: text !important;
  cursor: text;
}

:deep(.bom-table .p-datatable-tbody > tr > td *) {
  -webkit-user-select: text !important;
  user-select: text !important;
}

/* 儲存格文字容器：單行 CSS 截斷、支援鼠標完整選取拖拽 */
.cell-text {
  display: block;
  width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  user-select: text !important;
  -webkit-user-select: text !important;
  cursor: text;
}

/* 合併欄位：Item + 開合按鈕 (緊湊半字元間距) */
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
  gap: 0.25rem !important; /* 與一般欄位保持一致的圖示間距 */
}

.item-header-content {
  display: inline-flex;
  align-items: center;
  gap: 0.15rem; /* 約半個字元寬度 (~2.5px) */
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
  gap: 0.15rem; /* 約半個字元寬度 (~2.5px) */
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
