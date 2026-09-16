/**
 * @file bomTable.ts
 * @description BOMTable 顯示狀態記憶體快取 Store (Pinia)
 * 
 * 本模組負責管理 BOMTable 元件的記憶體快取狀態，包含：
 * 1. EBOM / Matrix 模式切換狀態 (`bomType`)
 * 2. 視圖分類篩選 (`view`)
 * 3. 關鍵字過濾搜尋字串 (`searchQuery`)
 * 4. 垂直滾動位置 (`scrollTop`) 與水平滾動位置 (`scrollLeft`)
 * 5. 排序欄位 (`sortField`) 與排序方向 (`sortOrder`)
 * 6. 當前快取所屬的 Revision 組合標記 (`revisionKey`)
 * 7. 具備高擴充性的自訂擴充字典 (`customStates`)
 * 
 * 解決使用者在切換頁面 (如進入 Settings 設定頁) 後返回時，畫面重置回預設值的問題。
 */

import { defineStore } from 'pinia'
import { ref, shallowRef } from 'vue'
import type { BOMModeType } from '../components/BOMTable/types'
import type { ViewPartGroup, ViewRevision } from '../services/api'

/**
 * BOMTable 記憶體快取狀態資料介面
 */
export interface BOMTableState {
  /** BOM 模式 ('EBOM' | 'Matrix') */
  bomType: BOMModeType
  /** 視圖類別 (如 'all', 'smd', 'pth', 'bottom', 'ni', 'proto', 'mp', 'ccl') */
  view: string
  /** 關鍵字過濾搜尋字串 */
  searchQuery: string
  /** 是否僅顯示 CCL 關鍵零件群組 */
  cclOnly: boolean
  /** 垂直滾動卷軸位置 (像素) */
  scrollTop: number
  /** 水平滾動卷軸位置 (像素) */
  scrollLeft: number
  /** 排序欄位名稱 */
  sortField: string
  /** 排序順序 (1: 升冪, -1: 降冪) */
  sortOrder: number
  /** 當前快取所屬之 Revision IDs 組合鍵 (如 "1,2") */
  revisionKey: string
  /** 自訂擴充狀態字典 (提供未來任意功能之狀態快取) */
  customStates: Record<string, any>
}

export const useBOMTableStore = defineStore('bomTable', () => {
  // ── 響應式狀態 (State) ────
  const bomType = ref<BOMModeType>('EBOM')
  const view = ref<string>('all')
  const searchQuery = ref<string>('')
  const cclOnly = ref<boolean>(false)
  const scrollTop = ref<number>(0)
  const scrollLeft = ref<number>(0)
  const sortField = ref<string>('')
  const sortOrder = ref<number>(1)
  const revisionKey = ref<string>('')
  const customStates = ref<Record<string, any>>({})

  // ── 物料資料快取 (避免相同 Revision 與 View 條件下重複向資料庫查詢) ────
  const cachedPartGroups = shallowRef<ViewPartGroup[]>([])
  const cachedRevisions = shallowRef<ViewRevision[]>([])
  const cachedQueryKey = ref<string>('')

  // ── 狀態變更方法 (Actions) ────

  /**
   * 設定是否僅顯示 CCL 關鍵零件群組
   * 
   * @param {boolean} only - 是否僅顯示 CCL
   */
  function setCclOnly(only: boolean): void {
    cclOnly.value = only
  }

  /**
   * 設定 BOM 視圖模式 ('EBOM' | 'Matrix')
   * 
   * @param {BOMModeType} type - BOM 模式
   */
  function setBomType(type: BOMModeType): void {
    bomType.value = type
  }

  /**
   * 設定當前選取的視圖類別代碼
   * 
   * @param {string} viewCode - 視圖代碼 (如 'all', 'smd', 'pth' 等)
   */
  function setView(viewCode: string): void {
    view.value = viewCode
  }

  /**
   * 設定過濾搜尋關鍵字
   * 
   * @param {string} query - 關鍵字字串
   */
  function setSearchQuery(query: string): void {
    searchQuery.value = query
  }

  /**
   * 設定垂直滾動位置
   * 
   * @param {number} top - 垂直滾動像素位置
   */
  function setScrollTop(top: number): void {
    scrollTop.value = Math.max(0, top)
  }

  /**
   * 設定水平滾動位置
   * 
   * @param {number} left - 水平滾動像素位置
   */
  function setScrollLeft(left: number): void {
    scrollLeft.value = Math.max(0, left)
  }

  /**
   * 同步設定垂直與水平滾動卷軸位置
   * 
   * @param {number} top - 垂直滾動像素位置
   * @param {number} [left=0] - 水平滾動像素位置
   */
  function setScrollPosition(top: number, left: number = 0): void {
    scrollTop.value = Math.max(0, top)
    scrollLeft.value = Math.max(0, left)
  }

  /**
   * 設定表格排序欄位與方向
   * 
   * @param {string} field - 排序欄位名稱
   * @param {number} order - 排序方向 (1: 升冪, -1: 降冪)
   */
  function setSort(field: string, order: number): void {
    sortField.value = field
    sortOrder.value = order
  }

  /**
   * 設定當前快取對應之 Revision 集合識別標記
   * 
   * @param {string} key - Revision 鍵值 (例如 "1,2")
   */
  function setRevisionKey(key: string): void {
    revisionKey.value = key
  }

  /**
   * 設定自訂擴充狀態屬性
   * 
   * @template T
   * @param {string} key - 自訂鍵名
   * @param {T} value - 狀態值
   */
  function setCustomState<T>(key: string, value: T): void {
    customStates.value[key] = value
  }

  /**
   * 取得自訂擴充狀態屬性值
   * 
   * @template T
   * @param {string} key - 自訂鍵名
   * @param {T} [defaultValue] - 預設回傳值
   * @returns {T | undefined} 狀態值或預設值
   */
  function getCustomState<T>(key: string, defaultValue?: T): T | undefined {
    return key in customStates.value ? (customStates.value[key] as T) : defaultValue
  }

  /**
   * 清除特定自訂擴充狀態屬性
   * 
   * @param {string} key - 自訂鍵名
   */
  function clearCustomState(key: string): void {
    delete customStates.value[key]
  }

  /**
   * 重置垂直與水平滾動卷軸至頂部與最左側 (0, 0)
   */
  function resetScroll(): void {
    scrollTop.value = 0
    scrollLeft.value = 0
  }

  /**
   * 檢查當前記憶體快取中是否已有相同查詢條件 (Revision 與 View) 之有效物料資料
   * 
   * @param {string} revKey - Revision IDs 組合鍵 (例如 "1,2")
   * @param {string} viewType - 視圖代碼 (例如 "ALL", "SMD")
   * @returns {boolean} 是否命中快取
   */
  function hasDataCache(revKey: string, viewType: string): boolean {
    const key = `${revKey}:${viewType.toUpperCase()}`
    return cachedQueryKey.value === key && cachedPartGroups.value.length > 0
  }

  /**
   * 設定查詢結果快取資料
   * 
   * @param {string} revKey - Revision IDs 組合鍵
   * @param {string} viewType - 視圖代碼
   * @param {ViewPartGroup[]} partGroups - 物料群組清單
   * @param {ViewRevision[]} revisions - Revision 中繼資料清單
   */
  function setDataCache(
    revKey: string,
    viewType: string,
    partGroups: ViewPartGroup[],
    revisions: ViewRevision[]
  ): void {
    cachedQueryKey.value = `${revKey}:${viewType.toUpperCase()}`
    cachedPartGroups.value = [...partGroups]
    cachedRevisions.value = [...revisions]
  }

  /**
   * 同步更新快取中的物料群組 (例如本機編輯 Notes 等無重查需求之異動)
   * 
   * @param {ViewPartGroup[]} partGroups - 更新後的物料群組清單
   */
  function updateCachedParts(partGroups: ViewPartGroup[]): void {
    cachedPartGroups.value = partGroups
  }

  /**
   * 清空物料資料快取 (當專案資料異動、系列關閉或強制重查時呼叫)
   */
  function clearDataCache(): void {
    cachedQueryKey.value = ''
    cachedPartGroups.value = []
    cachedRevisions.value = []
  }

  /**
   * 批次更新快取狀態 (支援部分更新)
   * 
   * @param {Partial<BOMTableState>} partial - 部分狀態更新物件
   */
  function updateState(partial: Partial<BOMTableState>): void {
    if (partial.bomType !== undefined) bomType.value = partial.bomType
    if (partial.view !== undefined) view.value = partial.view
    if (partial.searchQuery !== undefined) searchQuery.value = partial.searchQuery
    if (partial.cclOnly !== undefined) cclOnly.value = partial.cclOnly
    if (partial.scrollTop !== undefined) scrollTop.value = Math.max(0, partial.scrollTop)
    if (partial.scrollLeft !== undefined) scrollLeft.value = Math.max(0, partial.scrollLeft)
    if (partial.sortField !== undefined) sortField.value = partial.sortField
    if (partial.sortOrder !== undefined) sortOrder.value = partial.sortOrder
    if (partial.revisionKey !== undefined) revisionKey.value = partial.revisionKey
    if (partial.customStates !== undefined) {
      customStates.value = { ...customStates.value, ...partial.customStates }
    }
  }

  /**
   * 完全重置快取狀態至系統初始值 (專案系列關閉時呼叫)
   */
  function resetState(): void {
    bomType.value = 'EBOM'
    view.value = 'all'
    searchQuery.value = ''
    cclOnly.value = false
    scrollTop.value = 0
    scrollLeft.value = 0
    sortField.value = ''
    sortOrder.value = 1
    revisionKey.value = ''
    customStates.value = {}
    clearDataCache()
  }

  return {
    // 狀態
    bomType,
    view,
    searchQuery,
    cclOnly,
    scrollTop,
    scrollLeft,
    sortField,
    sortOrder,
    revisionKey,
    customStates,
    cachedPartGroups,
    cachedRevisions,
    cachedQueryKey,
    // 操作方法
    setBomType,
    setView,
    setSearchQuery,
    setCclOnly,
    setScrollTop,
    setScrollLeft,
    setScrollPosition,
    setSort,
    setRevisionKey,
    setCustomState,
    getCustomState,
    clearCustomState,
    resetScroll,
    hasDataCache,
    setDataCache,
    updateCachedParts,
    clearDataCache,
    updateState,
    resetState,
  }
})
