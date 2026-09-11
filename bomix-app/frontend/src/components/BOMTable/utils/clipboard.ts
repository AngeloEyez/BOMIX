/**
 * @file clipboard.ts
 * @description 剪貼簿操作與 TSV 資料格式化工具
 * 
 * 本工具負責處理文字複製至系統剪貼簿之非同步操作，
 * 並提供將 BOM 列資料 (BOMDisplayRow) 轉換為 Tab 分隔字串 (TSV 格式) 的功能，
 * 方便使用者將資料直接 Ctrl+V 貼入 Microsoft Excel 等試算表軟體中。
 * 
 * 導出函式：
 * - copyText: 將任意純文字寫入系統剪貼簿
 * - copyRowTSV: 將 BOMDisplayRow 格式化為 TSV 並寫入剪貼簿
 * 
 * 依賴模組：../types (BOMDisplayRow 型別)
 */

import type { BOMDisplayRow } from '../types'

/**
 * 複製純文字至系統剪貼簿
 * 
 * @param {string} [text] - 要複製的字串內容
 * @returns {Promise<boolean>} 是否成功複製至剪貼簿
 */
export async function copyText(text?: string): Promise<boolean> {
  if (!text) return false
  if (typeof navigator === 'undefined' || !navigator.clipboard) {
    console.warn('[Clipboard] 瀏覽器環境不支援 navigator.clipboard API')
    return false
  }

  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch (err) {
    console.error('[Clipboard] 複製文字至剪貼簿失敗:', err)
    return false
  }
}

/**
 * 將指定 BOM 資料列轉為 Tab 分隔字串 (TSV 格式) 並複製至系統剪貼簿
 * 依序包含：Item, HHPN, Description, Supplier, Supplier PN, Qty, Locations, CCL, Remark
 * 
 * @param {BOMDisplayRow | null} [row] - 待複製的資料列物件
 * @returns {Promise<boolean>} 是否成功完成複製
 */
export async function copyRowTSV(row?: BOMDisplayRow | null): Promise<boolean> {
  if (!row) return false

  const cols = [
    row.item || '',
    row.hhpn || '',
    row.description || '',
    row.supplier || '',
    row.supplier_pn || '',
    row.qty ?? '',
    row.locations || '',
    row.ccl ? 'Y' : '',
    row.remark || ''
  ]

  return copyText(cols.join('\t'))
}
