<template>
  <div class="bom-table-container">
    <!-- View Filter Toolbar -->
    <div class="view-toolbar">
      <div class="view-filters">
        <span class="filter-label">View:</span>
        <Select
          v-model="selectedView"
          :options="viewOptions"
          option-label="label"
          option-value="value"
          placeholder="Select View"
          class="view-dropdown"
          @change="onViewChange"
        />
      </div>

      <div class="view-actions">
        <Button
          label="Expand All"
          icon="pi pi-angle-down"
          text
          @click="expandAll"
        />
        <Button
          label="Collapse All"
          icon="pi pi-angle-right"
          text
          @click="collapseAll"
        />
        <Tag v-if="bomMode" :value="bomMode" severity="info" class="mode-badge" />
      </div>
    </div>

    <!-- BOM Data Table (Flattened single table with shared columns) -->
    <DataTable
      :value="displayRows"
      dataKey="rowId"
      :scrollable="true"
      scroll-height="flex"
      scroll-direction="both"
      :row-hover="true"
      :row-class="getRowClass"
      striped-rows
      table-class="bom-table"
      v-model:sort-field="sortField"
      v-model:sort-order="sortOrder"
    >
      <!-- 收合 / 展開 控制欄 -->
      <Column style="width: 3.2rem" header="">
        <template #body="slotProps">
          <Button
            v-if="!slotProps.data.isSecondSource && slotProps.data.hasSecondSources"
            :icon="isCollapsed(slotProps.data.parentKey) ? 'pi pi-chevron-right' : 'pi pi-chevron-down'"
            text
            rounded
            size="small"
            class="toggle-ss-btn"
            :title="isCollapsed(slotProps.data.parentKey) ? '展開替代料' : '收合替代料'"
            @click.stop="toggleCollapse(slotProps.data.parentKey)"
          />
        </template>
      </Column>

      <!-- Item (2nd 替代料該欄位為空白) -->
      <Column field="item" header="Item" style="width: 80px" sortable />

      <!-- HHPN (2nd 替代料前附帶 ↳ 2nd 縮排標籤) -->
      <Column field="hhpn" header="HHPN" style="width: 170px" sortable>
        <template #body="slotProps">
          <div :class="{'ss-indented': slotProps.data.isSecondSource}">
            <Tag v-if="slotProps.data.isSecondSource" value="↳ 2nd" severity="warn" class="ss-inline-tag" />
            <span>{{ slotProps.data.hhpn }}</span>
          </div>
        </template>
      </Column>

      <!-- Description -->
      <Column field="description" header="Description" style="width: 220px" />

      <!-- Supplier -->
      <Column field="supplier" header="Supplier" style="width: 150px" sortable />

      <!-- Supplier PN -->
      <Column field="supplier_pn" header="Supplier PN" style="width: 180px" sortable />

      <!-- Qty -->
      <Column field="qty" header="Qty" style="width: 80px" sortable />

      <!-- Location -->
      <Column field="locations" header="Location" style="width: 150px" />

      <!-- CCL -->
      <Column field="ccl" header="CCL" style="width: 80px" sortable>
        <template #body="slotProps">
          <span v-if="slotProps.data.ccl" :class="getCCLClass(slotProps.data.ccl)">
            {{ slotProps.data.ccl }}
          </span>
        </template>
      </Column>

      <!-- Remark -->
      <Column field="remark" header="Remark" style="width: 150px" />

      <!-- Dynamic Model Columns -->
      <Column
        v-for="modelName in currentRevisionModels"
        :key="modelName"
        :header="`${modelName} (Qty: ${getModelQty(modelName)})`"
        style="width: 150px"
      >
        <template #body="slotProps">
          <span :class="{'model-selected': isModelSelected(slotProps.data, modelName)}">
            {{ getModelSelectedPN(slotProps.data, modelName) || '-' }}
          </span>
        </template>
      </Column>
    </DataTable>

    <!-- Summary Statistics -->
    <div class="table-summary">
      <span>Total Main Parts: {{ aggregatedParts.length }}</span>
      <span v-if="selectedView === 'smd'">| SMD Parts: {{ smdPartsCount }}</span>
      <span v-if="selectedView === 'pth'">| PTH Parts: {{ pthPartsCount }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Select from 'primevue/select'
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
  ccl: string
  remark: string
  selections: Record<string, string>
}

const props = defineProps<{
  revisionId?: number
}>()

const emit = defineEmits<{
  (e: 'part-selected', part: ViewPartGroup): void
}>()

const projectStore = useProjectStore()
const logStore = useLogStore()

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
const selectedView = ref('all')
const collapsedParents = ref<Set<string>>(new Set())
const sortField = ref('item')
const sortOrder = ref(1)

// Computed properties
const currentRevisionId = computed(() => props.revisionId || 0)

const aggregatedParts = ref<ViewPartGroup[]>([])
const currentRevisionMetadata = ref<ViewRevision | null>(null)
const bomMode = ref<string>('')

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

// Flattened display rows computed property
const displayRows = computed<BOMDisplayRow[]>(() => {
  const rows: BOMDisplayRow[] = []

  aggregatedParts.value.forEach((part) => {
    const parentKey = `${part.main_supplier}|${part.main_supplier_pn}`
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
      ccl: part.ccl || '',
      remark: part.remark || '',
      selections: selectionsMap,
    })

    // 2. Add 2nd Source Rows directly below Main Source (if not collapsed)
    if (hasSS && part.second_sources && !collapsedParents.value.has(parentKey)) {
      part.second_sources.forEach((ss, idx) => {
        rows.push({
          rowId: `${parentKey}-ss-${idx}`,
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
          ccl: '',
          remark: '',
          selections: selectionsMap,
        })
      })
    }
  })

  return rows
})

function getPartKey(part: ViewPartGroup): string {
  return `${part.main_supplier}|${part.main_supplier_pn}`
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

function expandAll(): void {
  collapsedParents.value = new Set()
}

function collapseAll(): void {
  const allKeys = new Set<string>()
  aggregatedParts.value.forEach(part => {
    if (part.second_sources && part.second_sources.length > 0) {
      allKeys.add(getPartKey(part))
    }
  })
  collapsedParents.value = allKeys
}

function getRowClass(data: BOMDisplayRow) {
  return data.isSecondSource ? 'second-source-row' : 'main-source-row'
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
  if (currentRevisionId.value) {
    loadBOMData(currentRevisionId.value)
  }
}

function getCCLClass(ccl: string): string {
  return ccl === 'Y' ? 'ccl-critical' : 'ccl-normal'
}

// Watch for revision changes
watch(() => props.revisionId, (newId) => {
  if (newId) {
    loadBOMData(newId)
  }
})

async function loadBOMData(revisionId: number): Promise<void> {
  try {
    const viewType = selectedView.value === 'all' ? '' : selectedView.value.toUpperCase()
    logStore.addLogEntry('DEBUG', `[View System] 準備建立 View: RevisionIDs=[${revisionId}], ViewType="${viewType || 'ALL'}", ModeOverride=""`)
    const result = await GetBOMView([revisionId], viewType, '')
    
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
      bomMode.value = result.revisions[0].mode || ''
    } else {
      currentRevisionMetadata.value = null
      bomMode.value = ''
    }
  } catch (error) {
    const msg = error instanceof Error ? error.message : String(error)
    logStore.addLogEntry('ERROR', `Failed to load BOM data: ${msg}`)
  }
}

onMounted(() => {
  if (props.revisionId) {
    loadBOMData(props.revisionId)
  }
})
</script>

<style scoped>
.bom-table-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.view-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem 1rem;
  background: var(--surface-ground);
  border-bottom: 1px solid var(--surface-border);
  margin-bottom: 0.5rem;
}

.view-filters {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.filter-label {
  font-weight: 600;
  font-size: 0.875rem;
  color: var(--text-color);
}

.view-dropdown {
  min-width: 150px;
}

.view-actions {
  display: flex;
  gap: 0.25rem;
  align-items: center;
}

.mode-badge {
  margin-left: 0.5rem;
  font-weight: 600;
}

.supplier-pn-cell {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.ss-badge {
  font-size: 0.7rem;
  padding: 0.15rem 0.4rem;
}

/* Flattened Table 2nd Source Row styling */
:deep(.second-source-row) {
  background-color: var(--surface-50, #f8fafc) !important;
  color: var(--text-color-secondary, #475569);
  font-size: 0.825rem;
}

:deep(.second-source-row:hover) {
  background-color: var(--surface-100, #f1f5f9) !important;
}

.ss-indented {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  padding-left: 0.25rem;
}

.ss-inline-tag {
  font-size: 0.65rem;
  padding: 0.1rem 0.35rem;
  line-height: 1;
}

.toggle-ss-btn {
  width: 1.75rem !important;
  height: 1.75rem !important;
  padding: 0 !important;
}

/* Table styling */
:deep(.p-datatable) {
  font-size: 0.875rem;
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

:deep(.p-datatable-tbody > tr) {
  cursor: pointer;
}

/* Summary */
.table-summary {
  display: flex;
  gap: 1rem;
  padding: 0.5rem 1rem;
  background: var(--surface-ground);
  border-top: 1px solid var(--surface-border);
  font-size: 0.75rem;
  color: var(--text-color-secondary);
}

.ccl-normal {
  color: var(--text-color-secondary);
}
</style>
