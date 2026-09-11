/**
 * @file index.ts
 * @description BOMTable 模組統一導出入口
 * 
 * 本檔案將 BOMTable 特性目錄中的核心視圖元件與所有型別/工具對外導出，
 * 外部頁面或元件只需透過 `import BOMTable from '@/components/BOMTable'` 即可完成引用。
 */

import BOMTable from './BOMTable.vue'

export default BOMTable
export { BOMTable }
export * from './types'
export { useBOMData } from './composables/useBOMData'
export { useColumnWidths } from './composables/useColumnWidths'
export { useCollapseState } from './composables/useCollapseState'
export { useBOMContextMenu } from './composables/useBOMContextMenu'
export { useCellAutoScroll } from './composables/useCellAutoScroll'
