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
      </div>
    </div>

    <!-- BOM Data Table -->
    <DataTable
      :value="aggregatedParts"
      :expanded-keys="expandedKeys"
      :scrollable="true"
      scroll-height="flex"
      scroll-direction="both"
      :virtualScrollerOptions="{ itemSize: 46 }"
      :row-hover="true"
      striped-rows
      table-class="bom-table"
      v-model:sort-field="sortField"
      v-model:sort-order="sortOrder"
    >
      <!-- Expandable Row -->
      <template #expansion="slotProps">
        <div class="row-expansion">
          <SecondSourceList
            :part="slotProps.data"
            :revision-id="currentRevisionId"
          />
        </div>
      </template>

      <!-- Row Columns -->
      <Column expander style="width: 5rem" />
      <Column field="item" header="Item" style="width: 80px" sortable />
      <Column field="type" header="Type" style="width: 100px" sortable>
        <template #body="slotProps">
          <Tag :value="slotProps.data.type" :severity="getTypeSeverity(slotProps.data.type)" />
        </template>
      </Column>
      <Column field="hhpn" header="HHPN" style="width: 150px" sortable />
      <Column field="description" header="Description" style="width: 200px" />
      <Column field="main_supplier" header="Supplier" style="width: 150px" sortable />
      <Column field="main_supplier_pn" header="Supplier PN" style="width: 150px" sortable />
      <Column field="qty" header="Qty" style="width: 80px" sortable />
      <Column field="locations" header="Location" style="width: 150px" />
      <Column field="bom_status" header="BOM Status" style="width: 100px" sortable>
        <template #body="slotProps">
          <span :class="getBOMStatusClass(slotProps.data.bom_status)">
            {{ slotProps.data.bom_status }}
          </span>
        </template>
      </Column>
      <Column field="ccl" header="CCL" style="width: 80px" sortable>
        <template #body="slotProps">
          <span :class="getCCLClass(slotProps.data.ccl)">
            {{ slotProps.data.ccl }}
          </span>
        </template>
      </Column>
      <Column field="remark" header="Remark" style="width: 150px" />
      
      <!-- Dynamic Model Columns -->
      <Column
        v-for="modelName in currentRevisionModels"
        :key="modelName"
        :header="`${modelName} (Qty: ${getModelQty(modelName)})`"
        style="width: 150px"
      >
        <template #body="slotProps">
          <span :class="{'model-selected': getModelSelectedPN(slotProps.data, modelName)}">
            {{ getModelSelectedPN(slotProps.data, modelName) || '-' }}
          </span>
        </template>
      </Column>
    </DataTable>

    <!-- Summary Statistics -->
    <div class="table-summary">
      <span>Total Parts: {{ aggregatedParts.length }}</span>
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
import SecondSourceList from './SecondSourceList.vue'
import { GetBOMView, type ViewPartGroup, type ViewRevision } from '../services/api'

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
const expandedKeys = ref<Record<string, boolean>>({})
const sortField = ref('item')
const sortOrder = ref(1)

// Computed properties
const currentRevisionId = computed(() => props.revisionId || 0)

const aggregatedParts = ref<ViewPartGroup[]>([])
const currentRevisionMetadata = ref<ViewRevision | null>(null)

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

function getModelQty(modelName: string): number {
  if (!currentRevisionMetadata.value || !currentRevisionMetadata.value.model_qty) return 0
  return currentRevisionMetadata.value.model_qty[modelName] || 0
}

function getModelSelectedPN(part: ViewPartGroup, modelName: string): string {
  if (!part.selections) return ''
  const sel = part.selections.find(s => s.model_name === modelName)
  return sel ? sel.selected_pn : ''
}

// Methods
function onViewChange(): void {
  // Filter parts based on selected view
  expandedKeys.value = {}
  if (currentRevisionId.value) {
    loadBOMData(currentRevisionId.value)
  }
}

function expandAll(): void {
  const allExpanded: Record<string, boolean> = {}
  aggregatedParts.value.forEach((part, index) => {
    allExpanded[index] = true
  })
  expandedKeys.value = allExpanded
}

function collapseAll(): void {
  expandedKeys.value = {}
}

function getTypeSeverity(type: string): string {
  switch (type) {
    case 'SMD':
      return 'info'
    case 'PTH':
      return 'success'
    case 'BOTTOM':
      return 'warning'
    default:
      return 'secondary'
  }
}

function getBOMStatusClass(status: string): string {
  switch (status) {
    case 'I':
      return 'bom-status-install'
    case 'X':
      return 'bom-status-exclude'
    case 'P':
      return 'bom-status-proto'
    case 'M':
      return 'bom-status-mp'
    default:
      return ''
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
    } else {
      aggregatedParts.value = []
    }

    if (result && result.revisions && result.revisions.length > 0) {
      currentRevisionMetadata.value = result.revisions[0]
    } else {
      currentRevisionMetadata.value = null
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
}

/* Row expansion */
.row-expansion {
  background: var(--surface-ground);
  padding: 1rem;
  border-top: 1px solid var(--surface-border);
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

:deep(.p-datatable-tbody > tr:hover) {
  background: var(--highlight-background);
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

/* Status classes */
.bom-status-install {
  color: var(--success-color);
  font-weight: 600;
}

.bom-status-exclude {
  color: var(--danger-color);
  font-weight: 600;
}

.bom-status-proto {
  color: var(--warning-color);
  font-weight: 600;
}

.bom-status-mp {
  color: var(--primary-color);
  font-weight: 600;
}

.ccl-critical {
  color: var(--danger-color);
  font-weight: 700;
}

.ccl-normal {
  color: var(--text-color-secondary);
}
</style>
