/**
 * @file useCollapseState.ts
 * @description 替代料 (2nd Source) 展開與收合狀態管理 (Composable)
 * 
 * 本模組負責管理物料群組中 2nd 替代料的折疊與展開狀態。
 * 維護 `collapsedParents` (Set<string>) 集合，並提供全部展開 (Expand All)、
 * 全部收合 (Collapse All) 及個別切換等響應式方法。
 * 
 * 導出函式：
 * - useCollapseState: 建立並管理替代料展開與收合狀態之 Composable
 * 
 * 依賴模組：
 * - Vue 3 (ref, computed, Ref)
 * - services/api (ViewPartGroup 型別)
 */

import { ref, computed, type Ref } from 'vue'
import type { ViewPartGroup } from '../../../services/api'

/**
 * 建立替代料展開/收合狀態管理器
 * 
 * @param {Ref<ViewPartGroup[]>} aggregatedParts - 原始物料群組清單之響應式參照
 */
export function useCollapseState(aggregatedParts: Ref<ViewPartGroup[]>) {
  /** 已被使用者手動收合之主料群組識別鍵集合 (Key 存在代表目前處於收合狀態) */
  const collapsedParents = ref<Set<string>>(new Set())

  /**
   * 取得物料群組唯一識別鍵
   * 結合 item、type、main_supplier 與 main_supplier_pn，確保每個主料群組具備獨立唯一的鍵值
   * 
   * @param {ViewPartGroup} part - 物料群組資料物件
   * @returns {string} 唯一識別字串
   */
  function getPartKey(part: ViewPartGroup): string {
    const item = part.item || ''
    const pType = part.type || ''
    const supplier = part.main_supplier || ''
    const pn = part.main_supplier_pn || ''
    return `${item}|${pType}|${supplier}|${pn}`
  }

  /**
   * 判斷指定主料群組是否處於收合狀態
   * 
   * @param {string} parentKey - 主料群組唯一識別鍵
   * @returns {boolean} true: 收合中 (隱藏 2nd 替代料)，false: 展開中
   */
  function isCollapsed(parentKey: string): boolean {
    return collapsedParents.value.has(parentKey)
  }

  /**
   * 切換單一主料群組之展開 / 收合狀態
   * 
   * @param {string} parentKey - 主料群組唯一識別鍵
   */
  function toggleCollapse(parentKey: string): void {
    const newSet = new Set(collapsedParents.value)
    if (newSet.has(parentKey)) {
      newSet.delete(parentKey)
    } else {
      newSet.add(parentKey)
    }
    collapsedParents.value = newSet
  }

  /**
   * 所有具有替代料的主料群組唯一識別鍵集合
   */
  const allExpandableKeys = computed<Set<string>>(() => {
    const keys = new Set<string>()
    aggregatedParts.value.forEach(part => {
      if (part.second_sources && part.second_sources.length > 0) {
        keys.add(getPartKey(part))
      }
    })
    return keys
  })

  /**
   * 具有替代料的主料群組數量 (依據唯一識別鍵統計)
   */
  const totalExpandableCount = computed(() => {
    return allExpandableKeys.value.size
  })

  /**
   * 判斷是否所有替代料群組皆處於收合狀態
   */
  const isAllCollapsed = computed(() => {
    if (allExpandableKeys.value.size === 0) return false
    if (collapsedParents.value.size < allExpandableKeys.value.size) return false
    for (const key of allExpandableKeys.value) {
      if (!collapsedParents.value.has(key)) {
        return false
      }
    }
    return true
  })

  /**
   * 全部展開所有替代料
   */
  function expandAll(): void {
    collapsedParents.value = new Set()
  }

  /**
   * 全部收合所有替代料
   */
  function collapseAll(): void {
    collapsedParents.value = new Set(allExpandableKeys.value)
  }

  /**
   * 切換所有替代料群組之展開 / 收合狀態 (供表頭第一欄 `#` 圖標點擊使用)
   */
  function toggleAllCollapse(): void {
    if (isAllCollapsed.value) {
      expandAll()
    } else {
      collapseAll()
    }
  }

  /**
   * 重設收合狀態為預設值 (清空收合紀錄)
   */
  function resetCollapse(): void {
    collapsedParents.value = new Set()
  }

  return {
    collapsedParents,
    getPartKey,
    isCollapsed,
    toggleCollapse,
    allExpandableKeys,
    totalExpandableCount,
    isAllCollapsed,
    expandAll,
    collapseAll,
    toggleAllCollapse,
    resetCollapse,
  }
}
