/**
 * @file sort.ts
 * @description BOM 物料群組自然數字排序演算法工具
 * 
 * 本工具負責對後端回傳的階層式物料群組 (ViewPartGroup) 進行群組層級 (Group-level) 的排序。
 * 支援自然數字排序 (Natural Sorting，例如 "10" 排序在 "2" 之後)，
 * 並保證具備穩定 Tie-breaker (當前排序欄位值相同時，次要依據 Item 號碼排序)，
 * 確保展開的 2nd 替代料不會因為排序而與其主料分離。
 * 
 * 導出函式：
 * - sortBOMPartGroups: 對物料群組陣列進行指定欄位與方向之排序
 * 
 * 依賴模組：services/api (ViewPartGroup 型別)
 */

import type { ViewPartGroup } from '../../../services/api'

/**
 * 比較兩物料項目的 Item 欄位，採用自然數字排序
 * 
 * @param {string} itemA - 項目 A 的編號
 * @param {string} itemB - 項目 B 的編號
 * @returns {number} 比較結果 (負數、0 或正數)
 */
function compareItemNatural(itemA: string | undefined | null, itemB: string | undefined | null): number {
  const strA = itemA || ''
  const strB = itemB || ''
  const numA = parseFloat(strA)
  const numB = parseFloat(strB)

  if (!isNaN(numA) && !isNaN(numB)) {
    return numA - numB
  }

  return strA.localeCompare(strB, undefined, { numeric: true, sensitivity: 'base' })
}

/**
 * 對物料群組清單依據指定欄位與排序方向進行排序
 * 
 * @param {ViewPartGroup[]} parts - 待排序之物料群組陣列
 * @param {string} field - 排序欄位名稱 (例如 'item', 'hhpn', 'qty', 'supplier', 'ccl' 等)
 * @param {number} order - 排序方向 (1: 升冪 Ascending, -1: 降冪 Descending)
 * @returns {ViewPartGroup[]} 排序完成後的新陣列 (不修改原陣列)
 */
export function sortBOMPartGroups(parts: ViewPartGroup[], field: string, order: number): ViewPartGroup[] {
  if (!parts || parts.length === 0) return []

  const list = [...parts]

  // 若未指定欄位，預設依據 Item 欄位進行自然數字升冪排序
  if (!field) {
    list.sort((a, b) => compareItemNatural(a.item, b.item))
    return list
  }

  list.sort((a, b) => {
    let valA: unknown = ''
    let valB: unknown = ''

    if (field === 'item') {
      valA = a.item || ''
      valB = b.item || ''
    } else if (field === 'hhpn') {
      valA = a.hhpn || ''
      valB = b.hhpn || ''
    } else if (field === 'supplier') {
      valA = a.main_supplier || ''
      valB = b.main_supplier || ''
    } else if (field === 'supplier_pn') {
      valA = a.main_supplier_pn || ''
      valB = b.main_supplier_pn || ''
    } else if (field === 'qty') {
      valA = a.qty ?? 0
      valB = b.qty ?? 0
    } else if (field === 'ccl') {
      valA = a.ccl ? 1 : 0
      valB = b.ccl ? 1 : 0
    } else if (field === 'description') {
      valA = a.description || ''
      valB = b.description || ''
    } else if (field === 'remark') {
      valA = a.remark || ''
      valB = b.remark || ''
    } else {
      valA = (a as unknown as Record<string, unknown>)[field] || ''
      valB = (b as unknown as Record<string, unknown>)[field] || ''
    }

    let compareRes = 0
    if (field === 'item' || field === 'qty') {
      const numA = parseFloat(String(valA))
      const numB = parseFloat(String(valB))
      if (!isNaN(numA) && !isNaN(numB)) {
        compareRes = numA - numB
      } else {
        compareRes = String(valA).localeCompare(String(valB), undefined, { numeric: true, sensitivity: 'base' })
      }
    } else if (field === 'ccl') {
      compareRes = Number(valA) - Number(valB)
    } else {
      compareRes = String(valA).localeCompare(String(valB), undefined, { numeric: true, sensitivity: 'base' })
    }

    if (compareRes !== 0) {
      return compareRes * order
    }

    // Tie-breaker: 數值相同時，依據 Item 自然數字序進行次要排序以保證穩定性
    return compareItemNatural(a.item, b.item)
  })

  return list
}
