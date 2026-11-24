<template>
  <div class="matrix-checkbox-cell" :class="{ 'is-disabled': disabled }">
    <!-- 若該物料在該 Revision 存在才繪製 Checkbox；若不存在則不繪製 Checkbox -->
    <Checkbox
      v-if="isAvailable"
      :model-value="isSelected"
      :disabled="disabled"
      binary
      @change="!disabled && emit('change')"
    />
  </div>
</template>

<script setup lang="ts">
/**
 * @file BOMMatrixCheckboxCell.vue
 * @description Matrix 模式 Model 互斥選取 Checkbox 單元格元件
 * 
 * 負責渲染互斥單選/勾選框，並透過事件通知上層執行 Model 切換與用量重新統計。
 * 當 disabled 為 true 時（例如全隱藏保底顯示 Model A 唯讀資料），禁止使用者勾選修改。
 */

import Checkbox from 'primevue/checkbox'

interface Props {
  /** 物料是否在該版本存在/可用 */
  isAvailable?: boolean
  /** 物料是否已於該版本選中 */
  isSelected?: boolean
  /** 是否為唯讀禁用狀態（例如隱藏機種保底顯示時） */
  disabled?: boolean
}

withDefaults(defineProps<Props>(), {
  isAvailable: false,
  isSelected: false,
  disabled: false
})

const emit = defineEmits<{
  /** Checkbox 狀態變更事件 */
  (e: 'change'): void
}>()
</script>

<style scoped>
.matrix-checkbox-cell {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
}

.matrix-checkbox-cell.is-disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

:deep(.p-checkbox) {
  width: 16px;
  height: 16px;
}

:deep(.p-checkbox-box) {
  width: 16px;
  height: 16px;
  border-radius: 3px;
}
</style>
