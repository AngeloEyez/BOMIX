<template>
  <div
    class="two-line-header matrix-header-group cursor-pointer select-none rounded hover:bg-slate-200/60 dark:hover:bg-slate-700/50 transition-colors"
    :title="`${column.headerTitle} (點擊編輯機種設定)`"
    @click="$emit('click', column)"
  >
    <!-- 第一行：專案代碼 (比照 EBOM 採用純文字節點，精準 12px 鎖定，絕不撐開表頭) -->
    <div class="header-line1" :title="column.headerTitle">
      {{ column.showProjectCode ? column.projectCode : '' }}
    </div>
    <!-- 第二行：純字母 (粗體) 與數量 (精準 12px 鎖定) -->
    <div class="header-line2">
      <span class="model-alias-bold">{{ column.modelAlias }}</span>{{ column.qty > 0 ? ` (${column.qty})` : '' }}
    </div>
  </div>
</template>

<script setup lang="ts">
/**
 * @file BOMMatrixHeader.vue
 * @description Matrix 模式動態 Model 欄位雙行表頭元件
 * 
 * 第一行：依據專案 Model 數量智慧置中顯示專案代碼 (Project Code)；
 * 第二行：顯示 Model 別名字母 (粗體) 與關聯用量括號標記；
 * 支援點擊表頭以觸發機種設定 (MatrixModelEditDialog) 編輯視窗。
 * 採用與 EBOM 完全一致的 12px/24px 高度鎖定規範，確保模式切換時 0px 高度跳動。
 */

import type { MatrixModelColumnInfo } from '../types'

interface Props {
  /** Matrix Model 欄位中繼設定 */
  column: MatrixModelColumnInfo
}

interface Emits {
  (e: 'click', col: MatrixModelColumnInfo): void
}

defineProps<Props>()
defineEmits<Emits>()
</script>

<style scoped>
.two-line-header {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 24px !important;
  max-height: 24px !important;
  line-height: 12px !important;
  text-align: center;
  overflow: hidden;
  box-sizing: border-box !important;
}

.matrix-header-group {
  width: 100%;
  height: 24px !important;
  max-height: 24px !important;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.header-line1 {
  font-size: 10px;
  font-weight: 700;
  color: var(--text-color);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
  letter-spacing: -0.25px;
  height: 12px !important;
  line-height: 12px !important;
  max-height: 12px !important;
  display: block;
}

.header-line2 {
  font-size: 10px;
  font-weight: 500;
  color: var(--text-color-secondary);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
  height: 12px !important;
  line-height: 12px !important;
  max-height: 12px !important;
  display: block;
}

.model-alias-bold {
  font-weight: 700;
  color: var(--text-color);
  line-height: inherit;
}
</style>
