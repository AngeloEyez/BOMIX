<template>
  <div class="table-summary">
    <span>Total Main Parts: {{ totalMainParts }}</span>
    <span v-if="selectedView === 'smd'">| SMD Parts: {{ smdPartsCount }}</span>
    <span v-if="selectedView === 'pth'">| PTH Parts: {{ pthPartsCount }}</span>
  </div>
</template>

<script setup lang="ts">
/**
 * @file BOMTableSummary.vue
 * @description BOM 表格底部統計摘要列元件
 * 
 * 顯示目前載入的主要物料總筆數，以及依 SMD / PTH 視圖過濾時的特定零件計數。
 */

interface Props {
  /** 主料總筆數 */
  totalMainParts: number
  /** 當前選取的視圖代碼 (如 'all', 'smd', 'pth') */
  selectedView: string
  /** SMD 零件筆數 */
  smdPartsCount: number
  /** PTH 零件筆數 */
  pthPartsCount: number
}

defineProps<Props>()
</script>

<style scoped>
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
</style>
