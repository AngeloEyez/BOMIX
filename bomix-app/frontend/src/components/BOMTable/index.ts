/**
 * @file index.ts
 * @description BOMTable 模組統一導出入口
 * 
 * 本檔案將 BOMTable 特性目錄中的核心視圖元件、子元件與所有型別/工具對外導出，
 * 外部頁面或元件只需透過 `import BOMTable from '@/components/BOMTable'` 即可完成引用。
 */

import BOMTable from './BOMTable.vue'

export default BOMTable
export { BOMTable }
export * from './types'

// Composables
export { useBOMData } from './composables/useBOMData'
export { useColumnWidths } from './composables/useColumnWidths'
export { useCollapseState } from './composables/useCollapseState'
export { useBOMContextMenu } from './composables/useBOMContextMenu'
export { useCellAutoScroll } from './composables/useCellAutoScroll'
export { useCellHoverCard } from './composables/useCellHoverCard'
export { useBOMNotesEditing } from './composables/useBOMNotesEditing'

// Components
export { default as BOMToolbar } from './components/BOMToolbar.vue'
export { default as BOMTableSummary } from './components/BOMTableSummary.vue'
export { default as BOMHighlightText } from './components/BOMHighlightText.vue'
export { default as BOMItemCell } from './components/BOMItemCell.vue'
export { default as BOMMatrixHeader } from './components/BOMMatrixHeader.vue'
export { default as BOMMatrixCheckboxCell } from './components/BOMMatrixCheckboxCell.vue'
export { default as BOMCellHoverCard } from './components/BOMCellHoverCard.vue'
export { default as BOMNotesEditor } from './components/BOMNotesEditor.vue'
