<template>
  <div class="second-source-list">
    <div class="section-header">
      <i class="pi pi-layer-group"></i>
      <span>Second Sources ({{ secondSources.length }})</span>
    </div>

    <div v-if="secondSources.length === 0" class="no-data">
      No second sources available
    </div>

    <DataTable
      v-else
      :value="secondSources"
      table-style="min-width: 50rem"
      striped-rows
      :scrollable="true"
      scroll-height="flex"
    >
      <Column field="supplier" header="Supplier" style="width: 150px" />
      <Column field="supplier_pn" header="Supplier PN" style="width: 150px" />
      <Column field="description" header="Description" style="width: 200px" />
      <Column field="cost" header="Cost" style="width: 100px" sortable>
        <template #body="slotProps">
          ${slotProps.data.cost.toFixed(2)}
        </template>
      </Column>
      <Column field="lead_time" header="Lead Time (days)" style="width: 120px" sortable />
      <Column field="is_active" header="Status" style="width: 100px">
        <template #body="slotProps">
          <Badge :value="slotProps.data.is_active ? 'Active' : 'Inactive'"
                 :severity="slotProps.data.is_active ? 'success' : 'secondary'" />
        </template>
      </Column>
    </DataTable>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Badge from 'primevue/badge'

interface SecondSource {
  id: number
  supplier: string
  supplier_pn: string
  description: string
  cost: number
  lead_time: number
  is_active: boolean
}

interface Part {
  second_sources: SecondSource[]
}

const props = defineProps<{
  part: Part
  revisionId: number
}>()

const secondSources = computed(() => {
  return props.part.second_sources || []
})
</script>

<style scoped>
.second-source-list {
  background: var(--surface-ground);
  border-radius: 4px;
  overflow: hidden;
}

.section-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1rem;
  background: var(--surface-100);
  border-bottom: 1px solid var(--surface-border);
  font-weight: 600;
  font-size: 0.875rem;
  color: var(--text-color);
}

.section-header i {
  color: var(--primary-color);
}

.no-data {
  padding: 2rem;
  text-align: center;
  color: var(--text-color-secondary);
  font-style: italic;
}
</style>
