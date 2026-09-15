<template>
  <div v-if="!row.isSecondSource" class="item-cell-content">
    <Button
      v-if="row.hasSecondSources"
      :icon="collapsed ? 'pi pi-chevron-right' : 'pi pi-chevron-down'"
      text
      rounded
      size="small"
      class="toggle-ss-btn"
      :title="collapsed ? '展開替代料' : '收合替代料'"
      @click.stop="onToggle"
    />
    <span v-else class="toggle-placeholder" />
    <span class="item-number">{{ row.item }}</span>
  </div>
  <!-- 2nd 替代料該欄位保持空白，留出收合圖標空間對齊 -->
  <div v-else class="item-cell-content">
    <span class="toggle-placeholder" />
  </div>
</template>

<script setup lang="ts">
/**
 * @file BOMItemCell.vue
 * @description Item 序號欄位儲存格元件
 * 
 * 負責渲染物料序號、主料替代料展開/收合圖示按鈕，以及替代料對齊佔位符號。
 */

import Button from 'primevue/button'
import type { BOMDisplayRow } from '../types'

interface Props {
  /** 當前資料列物件 */
  row: BOMDisplayRow
  /** 該主料群組目前是否處於收合狀態 */
  collapsed?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  collapsed: false
})

const emit = defineEmits<{
  /** 點擊展開/收合圖示觸發事件 */
  (e: 'toggle', parentKey: string): void
}>()

/** 處理點擊開合按鈕 */
function onToggle(): void {
  if (props.row.parentKey) {
    emit('toggle', props.row.parentKey)
  }
}
</script>

<style scoped>
.item-cell-content {
  display: flex;
  align-items: center;
  gap: 2px;
  width: 100%;
}

.toggle-ss-btn {
  width: 0.85rem !important;
  height: 1rem !important;
  min-width: unset !important;
  padding: 0 !important;
  margin: 0 !important;
  flex-shrink: 0;
}

:deep(.toggle-ss-btn .p-button-icon) {
  font-size: 10px !important;
}

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
  font-size: 11.5px;
  font-variant-numeric: tabular-nums;
  line-height: 1;
  font-family: "Cascadia Mono", "Cascadia Code", Consolas, "SF Mono", monospace !important;
  letter-spacing: -0.25px;
}
</style>
