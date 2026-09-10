<template>
  <div class="bom-table-container">
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
      :virtual-scroller-options="{ itemSize: 28 }"
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
      <Column field="item" sortable style="width: 62px" class="item-col" header-class="item-header-col">
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
            <span class="item-header-label">Item</span>
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

      <!-- HHPN (2nd 替代料具縮排效果) -->
      <Column field="hhpn" header="HHPN" style="width: 150px" sortable>
        <template #body="slotProps">
          <div :class="{'ss-indented': slotProps.data.isSecondSource}">
            <template v-for="(part, idx) in getHighlightedParts(slotProps.data.hhpn, searchQuery)" :key="idx">
              <mark v-if="part.isMatch" class="highlight-text">{{ part.text }}</mark>
              <span v-else>{{ part.text }}</span>
            </template>
          </div>
        </template>
      </Column>

      <!-- Description -->
      <Column field="description" header="Description" style="width: 220px">
        <template #body="slotProps">
          <template v-for="(part, idx) in getHighlightedParts(slotProps.data.description, searchQuery)" :key="idx">
            <mark v-if="part.isMatch" class="highlight-text">{{ part.text }}</mark>
            <span v-else>{{ part.text }}</span>
          </template>
        </template>
      </Column>

      <!-- Supplier -->
      <Column field="supplier" header="Supplier" style="width: 130px" sortable>
        <template #body="slotProps">
          <template v-for="(part, idx) in getHighlightedParts(slotProps.data.supplier, searchQuery)" :key="idx">
            <mark v-if="part.isMatch" class="highlight-text">{{ part.text }}</mark>
            <span v-else>{{ part.text }}</span>
          </template>
        </template>
      </Column>

      <!-- Supplier PN -->
      <Column field="supplier_pn" header="Supplier PN" style="width: 160px" sortable>
        <template #body="slotProps">
          <template v-for="(part, idx) in getHighlightedParts(slotProps.data.supplier_pn, searchQuery)" :key="idx">
            <mark v-if="part.isMatch" class="highlight-text">{{ part.text }}</mark>
            <span v-else>{{ part.text }}</span>
          </template>
        </template>
      </Column>

      <!-- Qty -->
      <Column field="qty" header="Qty" style="width: 65px" sortable />

      <!-- Location -->
      <Column field="locations" header="Location" style="width: 140px">
        <template #body="slotProps">
          <template v-for="(part, idx) in getHighlightedParts(slotProps.data.locations, searchQuery)" :key="idx">
            <mark v-if="part.isMatch" class="highlight-text">{{ part.text }}</mark>
            <span v-else>{{ part.text }}</span>
          </template>
        </template>
      </Column>

      <!-- CCL -->
      <Column field="ccl" header="CCL" style="width: 60px" sortable>
        <template #body="slotProps">
          <span v-if="slotProps.data.ccl" :class="getCCLClass(slotProps.data.ccl)">
            Y
          </span>
        </template>
      </Column>

      <!-- Remark -->
      <Column field="remark" header="Remark" style="width: 140px">
        <template #body="slotProps">
          <template v-for="(part, idx) in getHighlightedParts(slotProps.data.remark, searchQuery)" :key="idx">
            <mark v-if="part.isMatch" class="highlight-text">{{ part.text }}</mark>
            <span v-else>{{ part.text }}</span>
          </template>
        </template>
      </Column>

      <!-- Dynamic Model Columns -->
      <Column
        v-for="modelName in currentRevisionModels"
        :key="modelName"
        :header="`${modelName} (Qty: ${getModelQty(modelName)})`"
        style="width: 130px"
      >
        <template #body="slotProps">
          <span :class="{'model-selected': isModelSelected(slotProps.data, modelName)}">
            {{ getModelSelectedPN(slotProps.data, modelName) || '-' }}
          </span>
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
import { ref, shallowRef, computed, watch, onMounted, onUnmounted } from 'vue'
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
const sortField = ref('item')
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

onMounted(() => {
  window.addEventListener('keydown', handleKeyDown)
  if (props.revisionIds && props.revisionIds.length > 0) {
    loadBOMData(props.revisionIds)
  }
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeyDown)
})
</script>

<style scoped>
.bom-table-container {
  height: 100%;
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
}

/* Table styling - VS Code 風格高緊湊表格，最大化可視範圍 */
:deep(.p-datatable) {
  font-size: 0.75rem;
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

:deep(.p-datatable-table-container),
:deep(.p-datatable-wrapper) {
  flex: 1;
  min-height: 0;
}

:deep(.p-datatable-header) {
  background: var(--surface-ground);
  border-bottom: 1px solid var(--surface-border);
}

:deep(.bom-table .p-datatable-thead > tr > th) {
  padding: 0.2rem 0.35rem !important;
  font-size: 0.75rem !important;
  font-weight: 600;
  height: 28px !important;
  white-space: nowrap;
  border-bottom: 1px solid var(--surface-border);
  background: var(--surface-section);
}

:deep(.bom-table .p-datatable-tbody > tr) {
  height: 28px !important;
  max-height: 28px !important;
}

:deep(.bom-table .p-datatable-tbody > tr > td) {
  padding: 0.1rem 0.35rem !important;
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
  gap: 0.1rem;
}

.item-header-content {
  display: inline-flex;
  align-items: center;
  gap: 0.15rem; /* 約半個字元寬度 (~2.5px) */
}

.item-header-label {
  cursor: pointer;
  user-select: none;
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
