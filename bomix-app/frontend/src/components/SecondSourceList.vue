<template>
  <div class="second-source-container">
    <div class="second-source-wrapper">
      <div class="section-header">
        <i class="pi pi-shield"></i>
        <span>替代料 (2nd Source) — 共 {{ secondSources.length }} 筆</span>
      </div>

      <div v-if="secondSources.length === 0" class="no-data">
        目前無替代料資料
      </div>

      <DataTable
        v-else
        :value="secondSources"
        table-style="min-width: 45rem"
        striped-rows
        size="small"
        class="second-source-table"
      >
        <Column field="hhpn" header="HHPN" style="width: 140px" />
        <Column field="supplier" header="Supplier" style="width: 140px" />
        <Column field="supplier_pn" header="Supplier PN" style="width: 160px" />
        <Column field="description" header="Description" style="min-width: 200px" />
      </DataTable>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import type { ViewSecondSource, ViewPartGroup } from '../services/api'

const props = defineProps<{
  part: ViewPartGroup
  revisionId: number
}>()

const secondSources = computed<ViewSecondSource[]>(() => {
  return props.part.second_sources || []
})
</script>

<style scoped>
.second-source-container {
  padding: 0.5rem 0.5rem 0.5rem 2.5rem;
  background: var(--surface-card, #f8fafc);
}

.second-source-wrapper {
  background: var(--surface-section, #ffffff);
  border-left: 3px solid var(--p-amber-500, #f59e0b);
  border-radius: 6px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  overflow: hidden;
}

.section-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  background: var(--surface-100, #f1f5f9);
  border-bottom: 1px solid var(--surface-border, #e2e8f0);
  font-weight: 600;
  font-size: 0.8125rem;
  color: var(--text-color, #334155);
}

.section-header i {
  color: var(--p-amber-500, #f59e0b);
}

.second-source-table {
  font-size: 0.8125rem;
}

.no-data {
  padding: 1rem;
  text-align: center;
  color: var(--text-color-secondary, #64748b);
  font-style: italic;
  font-size: 0.8125rem;
}
</style>
