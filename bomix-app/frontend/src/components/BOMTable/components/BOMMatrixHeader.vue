<template>
  <div class="two-line-header matrix-header-group" :title="column.headerTitle">
    <!-- 第一行：專案名稱依智慧中心定位演算法顯示於中心 Model 欄位，其餘欄位留白 -->
    <div class="header-line1 project-code-line">
      <span
        v-if="column.showProjectCode"
        class="project-code-label"
        :title="column.headerTitle"
      >
        {{ column.projectCode }}
      </span>
      <!-- 佔位符確保第二行在各 Model 欄位垂直對齊 -->
      <span v-else class="project-code-spacer">&nbsp;</span>
    </div>
    <!-- 第二行：純字母 (粗體) 與數量，中間以空格分隔，例如 A (102)、B (147) -->
    <div class="header-line2 model-alias-line">
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
 * 第二行：顯示 Model 別名字母 (粗體) 與關聯用量括號標記。
 */

import type { MatrixModelColumnInfo } from '../types'

interface Props {
  /** Matrix Model 欄位中繼設定 */
  column: MatrixModelColumnInfo
}

defineProps<Props>()
</script>

<style scoped>
.two-line-header {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  width: 100%;
  line-height: 1.15;
  text-align: center;
  overflow: hidden;
}

.matrix-header-group {
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.header-line1 {
  font-size: 11px;
  font-weight: 700;
  color: var(--text-color);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}

.project-code-line {
  min-height: 14px;
  line-height: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  overflow: hidden;
}

.project-code-label {
  display: inline-block;
  font-size: 10px;
  font-weight: 700;
  color: var(--text-color);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
  letter-spacing: -0.25px;
  text-align: center;
}

.project-code-spacer {
  display: inline-block;
  visibility: hidden;
  height: 14px;
}

.header-line2 {
  font-size: 10px;
  font-weight: 500;
  color: var(--text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}

.model-alias-line {
  font-size: 10px;
  font-weight: 500;
  color: var(--text-color-secondary);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
  line-height: 13px;
  margin-top: 1px;
}

.model-alias-bold {
  font-weight: 700;
  color: var(--text-color);
}
</style>
