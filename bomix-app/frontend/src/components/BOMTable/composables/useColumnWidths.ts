/**
 * @file useColumnWidths.ts
 * @description BOM 表格最適欄寬動態計算與自適應分配 (Composable)
 * 
 * 本模組負責實現 VS Code 風格之高緊湊欄寬自適應演算法：
 * 1. 利用離屏 Canvas 測量文字長度，精準計算固定欄位 (Item, HHPN, Supplier, Qty, CCL 等) 之最適最小安全寬度。
 * 2. 取得 DataTable 可視工作區總寬度 (自動扣除捲軸)，並扣除固定欄位寬度總和。
 * 3. 剩餘空間由主要欄位 Description (規格描述，min 220px) 與 Location (位置標號，min 90px) 精確瓜分，
 *    使所有欄寬總和恰好完全貼合可視寬度，杜絕多餘的橫向捲軸。
 * 4. 透過 ResizeObserver 與視窗 resize 事件即時監聽父容器尺寸變化，自適應動態重算。
 * 
 * 導出函式：
 * - useColumnWidths: 建立並管理欄寬計算邏輯之 Composable
 * 
 * 依賴模組：
 * - Vue 3 (ref, onMounted, onUnmounted, nextTick)
 * - ../types (ColumnWidthConfig, BOMDisplayRow)
 * - ../utils/textMeasure (measureTextWidth)
 */

import { ref, onMounted, onUnmounted } from 'vue'
import type { ColumnWidthConfig, BOMDisplayRow } from '../types'
import { measureTextWidth } from '../utils/textMeasure'

/** 預設欄寬基礎配置 (像素，高緊湊優化) */
export const defaultColumnWidths: ColumnWidthConfig = {
  item: 46,
  hhpn: 130,
  description: 220,
  supplier: 110,
  supplier_pn: 140,
  qty: 45,
  locations: 180,
  ccl: 34,
  remark: 120,
  notes: 110,
  models: {}
}

/**
 * 基礎固定欄寬量測結果快取介面
 */
export interface BaseColumnWidths {
  item: number
  hhpn: number
  supplier: number
  supplier_pn: number
  qty: number
  ccl: number
  remark: number
  notes: number
  maxDesc: number
}

/**
 * 建立最適欄寬計算器與響應式監聽
 */
export function useColumnWidths() {
  /** 各欄位響應式寬度配置 */
  const columnWidths = ref<ColumnWidthConfig>({ ...defaultColumnWidths })

  /** 表格容器 template ref，用於精準讀取父層分配之可用寬度 */
  const tableWrapperRef = ref<HTMLElement | null>(null)

  /** 快取的基礎固定欄位寬度量測結果 (資料未變動前完全複用，切換模式或 CCL 篩選時 0ms 完成) */
  let cachedBaseWidths: BaseColumnWidths | null = null

  /** ResizeObserver 實例 */
  let resizeObserver: ResizeObserver | null = null

  /**
   * 清除基礎欄寬快取
   * 
   * 當切換 BOM Revision 版本組合或強制重新向後端查詢資料時呼叫，
   * 確保下次載入新資料時重新測量最適固定欄寬。
   */
  function invalidateBaseWidthsCache(): void {
    cachedBaseWidths = null
  }

  /**
   * 取得 DataTable 實際可用欄寬總量
   * 
   * 量測策略：
   * 1. 優先讀取 DataTable 內部 VirtualScroller 的 clientWidth (已扣除垂直捲軸)
   * 2. 回退至 tableWrapperRef.clientWidth 扣除 17px 捲軸估算值
   * 3. 最後回退至 window.innerWidth * 0.75
   * 
   * @returns {number} 可用總寬度 (像素)
   */
  function getWorkspaceVisibleWidth(): number {
    const wrapperEl = tableWrapperRef.value

    if (wrapperEl) {
      const vscroller = wrapperEl.querySelector('[data-pc-name="virtualscroller"]') as HTMLElement | null
      if (vscroller && vscroller.clientWidth > 0) {
        return Math.max(vscroller.clientWidth - 1, 400)
      }

      if (wrapperEl.clientWidth > 0) {
        return Math.max(wrapperEl.clientWidth - 17, 400)
      }
    }

    return Math.max(Math.floor((typeof window !== 'undefined' ? window.innerWidth : 1200) * 0.75), 400)
  }

  /**
   * 測量所有固定欄位之基礎最大內容寬度
   * 
   * 採用上限提早截斷 (Early Exit) 策略：
   * 各固定欄位均有設計上限 (如 HHPN 140px, Supplier 130px, Supplier PN 150px, Qty 55px, Remark 130px, Notes 150px)。
   * 當某一欄位在掃描過程中已達到上限，後續列即跳過該欄位之測量，大幅減少遍歷耗時。
   * 
   * @param {BOMDisplayRow[]} rows - 平鋪列資料清單
   * @returns {BaseColumnWidths} 各欄位計算後之基礎寬度
   */
  function measureBaseColumnWidths(rows: BOMDisplayRow[]): BaseColumnWidths {
    const itemWidth = 44
    const cclColWidth = 34

    // 預設表頭文字寬度量測 (極緊湊間距：標題文字 + 排序箭頭 + 邊距)
    let maxHhpn = measureTextWidth('HHPN', true) + 14
    let maxSupplier = measureTextWidth('Supplier', true) + 14
    let maxSupplierPn = measureTextWidth('Supplier PN', true) + 14
    let maxQty = measureTextWidth('Qty', true) + 14
    let maxRemark = measureTextWidth('Remark', true) + 10
    let maxNotes = measureTextWidth('Notes', true) + 10
    let maxDesc = measureTextWidth('Description', true) + 14

    // 各欄位寬度上限門檻 (達到後即可提早結束該欄位的量測)
    const MAX_HHPN_THRESHOLD = 140
    const MAX_SUPPLIER_THRESHOLD = 130
    const MAX_SUPPLIER_PN_THRESHOLD = 150
    const MAX_QTY_THRESHOLD = 55
    const MAX_REMARK_THRESHOLD = 130
    const MAX_NOTES_THRESHOLD = 150

    let needHhpn = true
    let needSupplier = true
    let needSupplierPn = true
    let needQty = true
    let needRemark = true
    let needNotes = true

    const len = rows.length
    for (let i = 0; i < len; i++) {
      const row = rows[i]

      if (needHhpn && row.hhpn) {
        const w = measureTextWidth(row.hhpn, false, true) + 8
        if (w > maxHhpn) {
          maxHhpn = w
          if (maxHhpn >= MAX_HHPN_THRESHOLD) needHhpn = false
        }
      }

      if (needSupplier && row.supplier) {
        const w = measureTextWidth(row.supplier) + 8
        if (w > maxSupplier) {
          maxSupplier = w
          if (maxSupplier >= MAX_SUPPLIER_THRESHOLD) needSupplier = false
        }
      }

      if (needSupplierPn && row.supplier_pn) {
        const w = measureTextWidth(row.supplier_pn, false, true) + 8
        if (w > maxSupplierPn) {
          maxSupplierPn = w
          if (maxSupplierPn >= MAX_SUPPLIER_PN_THRESHOLD) needSupplierPn = false
        }
      }

      if (needQty && row.qty !== '' && row.qty !== undefined && row.qty !== null) {
        const w = measureTextWidth(String(row.qty), false, true) + 8
        if (w > maxQty) {
          maxQty = w
          if (maxQty >= MAX_QTY_THRESHOLD) needQty = false
        }
      }

      if (needRemark && row.remark) {
        const w = measureTextWidth(row.remark) + 8
        if (w > maxRemark) {
          maxRemark = w
          if (maxRemark >= MAX_REMARK_THRESHOLD) needRemark = false
        }
      }

      if (needNotes && row.notes) {
        const w = measureTextWidth(row.notes) + 8
        if (w > maxNotes) {
          maxNotes = w
          if (maxNotes >= MAX_NOTES_THRESHOLD) needNotes = false
        }
      }

      if (row.description) {
        const w = measureTextWidth(row.description) + 8
        if (w > maxDesc) maxDesc = w
      }

      // 若所有固定欄位皆已達上限，僅需快速檢查剩餘列的 description
      if (!needHhpn && !needSupplier && !needSupplierPn && !needQty && !needRemark && !needNotes) {
        for (let j = i + 1; j < len; j++) {
          const d = rows[j].description
          if (d) {
            const w = measureTextWidth(d) + 8
            if (w > maxDesc) maxDesc = w
          }
        }
        break
      }
    }

    return {
      item: itemWidth,
      hhpn: Math.ceil(Math.min(140, Math.max(75, maxHhpn))),
      supplier: Math.ceil(Math.min(130, Math.max(60, maxSupplier))),
      supplier_pn: Math.ceil(Math.min(150, Math.max(80, maxSupplierPn))),
      qty: Math.ceil(Math.min(55, Math.max(36, maxQty))),
      ccl: cclColWidth,
      remark: Math.ceil(Math.min(130, Math.max(50, maxRemark))),
      notes: Math.ceil(Math.min(150, Math.max(100, maxNotes))),
      maxDesc: maxDesc,
    }
  }

  /**
   * 計算各欄位最適欄寬
   * 
   * 固定欄寬優先讀取快取；若為首次載入則計算一次並寫入快取。
   * 後續不論模式切換 (EBOM/Matrix)、CCL 篩選、折疊展開或視窗縮放，
   * 均以 O(1) 常數時間瓜分可視寬度，杜絕卡頓與延遲。
   * 
   * @param {BOMDisplayRow[]} rows - 當前顯示之平鋪列資料清單
   * @param {string} [bomType='EBOM'] - 視圖模式 (EBOM 或 Matrix)
   * @param {number} [revisionCount=0] - 參與顯示的 Revision 數量 (EBOM 模式使用)
   * @param {number} [matrixModelColumnCount=0] - 參與顯示的 Model 總欄位數 (Matrix 模式使用)
   * @param {number} [totalMatrixModelWidth=0] - 參與顯示的 Model 欄位總像素寬度 (Matrix 模式精確使用)
   * @param {number} [totalEBOMRevisionWidth=0] - 參與顯示的 Revision 欄位總像素寬度 (EBOM 模式精確使用)
   */
  function computeColumnWidths(
    rows: BOMDisplayRow[],
    bomType: string = 'EBOM',
    revisionCount: number = 0,
    matrixModelColumnCount: number = 0,
    totalMatrixModelWidth: number = 0,
    totalEBOMRevisionWidth: number = 0
  ): void {
    const MIN_DESC_WIDTH = 220
    const MIN_LOC_WIDTH = 90

    // 1. 若尚未快取基礎欄寬且存在資料列，執行初次測量並寫入快取
    if (!cachedBaseWidths && rows && rows.length > 0) {
      cachedBaseWidths = measureBaseColumnWidths(rows)
    }

    // 取得基礎欄寬 (若無資料列則使用預設配置備援)
    const base = cachedBaseWidths || {
      item: defaultColumnWidths.item,
      hhpn: defaultColumnWidths.hhpn,
      supplier: defaultColumnWidths.supplier,
      supplier_pn: defaultColumnWidths.supplier_pn,
      qty: defaultColumnWidths.qty,
      ccl: defaultColumnWidths.ccl,
      remark: defaultColumnWidths.remark,
      notes: defaultColumnWidths.notes,
      maxDesc: defaultColumnWidths.description,
    }

    // 2. 根據 EBOM / Matrix 模式決定固定欄位寬度總和
    let fixedTotal = base.item + base.hhpn + base.supplier + base.supplier_pn

    if (bomType === 'EBOM') {
      const totalQtyWidth = totalEBOMRevisionWidth > 0
        ? totalEBOMRevisionWidth
        : Math.max(revisionCount, 1) * 54
      fixedTotal += totalQtyWidth + base.ccl + base.remark
    } else {
      let totalRevWidth = 0
      if (totalMatrixModelWidth > 0) {
        totalRevWidth = totalMatrixModelWidth
      } else {
        const modelCount = matrixModelColumnCount > 0 ? matrixModelColumnCount : Math.max(revisionCount, 1)
        totalRevWidth = modelCount * 72
      }
      fixedTotal += base.qty + totalRevWidth + base.notes
    }

    const totalRequiredMinWidth = fixedTotal + MIN_LOC_WIDTH + MIN_DESC_WIDTH

    // 3. 取得可用可視寬度
    const visibleWidth = getWorkspaceVisibleWidth()

    let finalDescWidth = MIN_DESC_WIDTH
    let finalLocWidth = MIN_LOC_WIDTH

    if (visibleWidth <= totalRequiredMinWidth) {
      // 情況 1：可視寬度不足以容納所有欄位最小寬度，退守至各欄最小寬度
      finalDescWidth = MIN_DESC_WIDTH
      finalLocWidth = MIN_LOC_WIDTH
    } else {
      // 情況 2：可視寬度充裕，精密瓜分 Description 與 Location 空間，杜絕橫向捲軸
      const safeTotal = Math.floor(visibleWidth) - 1
      const rem = safeTotal - fixedTotal // 供 Description 與 Location 瓜分之空間

      // Location 初始目標寬度 (容納 20 個等寬字元，約 144px，不低於 90px)
      const twentyCharsWidth = measureTextWidth('0'.repeat(20), false, true)
      const locInit = Math.max(MIN_LOC_WIDTH, Math.ceil(twentyCharsWidth + 12))

      if (rem - locInit >= MIN_DESC_WIDTH) {
        let descW = rem - locInit
        let locW = locInit

        // 檢查 Description 是否已足夠完整顯示其所有資料
        const descNeeded = Math.max(MIN_DESC_WIDTH, Math.ceil(base.maxDesc + 14))
        if (descW > descNeeded) {
          const surplus = descW - descNeeded
          descW = descNeeded
          locW = locW + surplus
        }

        finalDescWidth = descW
        finalLocWidth = locW
      } else {
        finalDescWidth = MIN_DESC_WIDTH
        finalLocWidth = Math.max(MIN_LOC_WIDTH, rem - MIN_DESC_WIDTH)
      }
    }

    columnWidths.value = {
      item: base.item,
      hhpn: base.hhpn,
      description: finalDescWidth,
      supplier: base.supplier,
      supplier_pn: base.supplier_pn,
      qty: base.qty,
      locations: finalLocWidth,
      ccl: base.ccl,
      remark: base.remark,
      notes: base.notes,
      models: {}
    }
  }

  /**
   * 啟動容器尺寸監聽器 (ResizeObserver 與 window resize)
   * 
   * @param {() => void} onResizeCallback - 尺寸變更時觸發之回呼 (通常包裝 requestAnimationFrame 與 computeColumnWidths)
   */
  function setupResizeListener(onResizeCallback: () => void): void {
    const containerEl = tableWrapperRef.value
    if (containerEl && typeof ResizeObserver !== 'undefined') {
      resizeObserver = new ResizeObserver(() => {
        requestAnimationFrame(() => {
          onResizeCallback()
        })
      })
      resizeObserver.observe(containerEl)
    }

    const windowResizeHandler = () => {
      requestAnimationFrame(() => {
        onResizeCallback()
      })
    }

    window.addEventListener('resize', windowResizeHandler)

    // 保存引用以便在 unmount 時清理
    cleanups.push(() => {
      window.removeEventListener('resize', windowResizeHandler)
      if (resizeObserver) {
        resizeObserver.disconnect()
        resizeObserver = null
      }
    })
  }

  const cleanups: (() => void)[] = []

  onUnmounted(() => {
    cleanups.forEach(fn => fn())
    cleanups.length = 0
  })

  return {
    columnWidths,
    tableWrapperRef,
    computeColumnWidths,
    invalidateBaseWidthsCache,
    setupResizeListener,
  }
}

