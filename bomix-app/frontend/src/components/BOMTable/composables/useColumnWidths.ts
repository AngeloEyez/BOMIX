/**
 * @file useColumnWidths.ts
 * @description BOM 表格最適欄寬動態計算與自適應分配 (Composable)
 * 
 * 本模組負責實現 VS Code 風格之高緊湊欄寬自適應演算法：
 * 1. 利用離屏 Canvas 測量文字長度，精準計算固定欄位 (Item, HHPN, Supplier, Qty, CCL 等) 之最適最小安全寬度。
 * 2. 取得 DataTable 可視工作區總寬度 (自動扣除捲軸)，並扣除固定欄位寬度總和。
 * 3. 剩餘空間採三階段流水瀑布式分配：優先滿足 Description 最長內容，次分配給 Notes (若存在，依內容自適應上限最多 350px)，
 *    最後所有剩餘空間由 Location 全數吸收，使所有欄寬總和恰好完全貼合可視寬度，杜絕多餘的橫向捲軸。
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

/**
 * 各欄位寬度規格與上下限常數配置 (SSOT: 單一真實來源)
 * 
 * 集中管理各欄位之最小寬度 (min)、最大上限 (max) 或固定寬度 (fixed)，
 * 避免調整任何單一數值時需要同步更動多處程式碼。
 */
export const COLUMN_WIDTH_LIMITS = {
  /** 項目編號 (Item) 固定欄寬 (px) */
  item: { fixed: 44 },
  /** 鴻海料號 (HHPN) 內容適應寬度區間 (px) */
  hhpn: { min: 75, max: 140, default: 130 },
  /** 規格描述 (Description) 主要彈性欄位 (px) */
  description: { min: 220, default: 220 },
  /** 主要供應商 (Supplier) 內容適應寬度區間 (px) */
  supplier: { min: 60, max: 130, default: 110 },
  /** 供應商料號 (Supplier PN) 內容適應寬度區間 (px) */
  supplier_pn: { min: 80, max: 150, default: 140 },
  /** 用量 (Qty) 內容適應寬度區間 (px) */
  qty: { min: 36, max: 55, default: 45 },
  /** 位置標號 (Location) 次要彈性欄位，初始目標容納 20 個字元 (px) */
  locations: { min: 90, initChars: 20, default: 180 },
  /** 關鍵物料標記 (CCL) 固定欄寬 (px) */
  ccl: { fixed: 34 },
  /** 備註 (Remark) 內容適應寬度區間 (px) */
  remark: { min: 50, max: 130, default: 120 },
  /** 註記 (Notes) 彈性自適應欄位 (px) */
  notes: { min: 150, max: 350, default: 110 },
  /** EBOM 模式下單一 Revision Qty 預設欄寬 (px) */
  ebomRevisionDefault: 54,
  /** Matrix 模式下單一 Model 預設欄寬 (px) */
  matrixModelDefault: 72,
} as const

/** 預設欄寬基礎配置 (像素，高緊湊優化) */
export const defaultColumnWidths: ColumnWidthConfig = {
  item: COLUMN_WIDTH_LIMITS.item.fixed,
  hhpn: COLUMN_WIDTH_LIMITS.hhpn.default,
  description: COLUMN_WIDTH_LIMITS.description.default,
  supplier: COLUMN_WIDTH_LIMITS.supplier.default,
  supplier_pn: COLUMN_WIDTH_LIMITS.supplier_pn.default,
  qty: COLUMN_WIDTH_LIMITS.qty.default,
  locations: COLUMN_WIDTH_LIMITS.locations.default,
  ccl: COLUMN_WIDTH_LIMITS.ccl.fixed,
  remark: COLUMN_WIDTH_LIMITS.remark.default,
  notes: COLUMN_WIDTH_LIMITS.notes.default,
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
  maxNotes: number
}

/**
 * 建立最適欄寬計算器與響應式監聽
 */
export function useColumnWidths() {
  /** 各欄位響應式寬度配置 */
  const columnWidths = ref<ColumnWidthConfig>({ ...defaultColumnWidths })

  /** 表格容器 template ref，用於精準讀取父層分配之可用寬度 */
  const tableWrapperRef = ref<HTMLElement | null>(null)

  /** 所有欄位最小保底寬度之總和 (用於提供 DataTable tableStyle 之 min-width，確保多 Model 狀態不被擠壓) */
  const totalTableMinWidth = ref(0)

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
   * 各固定欄位均由 COLUMN_WIDTH_LIMITS 集中定義上限 (如 HHPN, Supplier, Supplier PN, Qty, Remark, Notes)。
   * 當某一欄位在掃描過程中已達到上限，後續列即跳過該欄位之測量，大幅減少遍歷耗時。
   * 
   * @param {BOMDisplayRow[]} rows - 平鋪列資料清單
   * @returns {BaseColumnWidths} 各欄位計算後之基礎寬度
   */
  function measureBaseColumnWidths(rows: BOMDisplayRow[]): BaseColumnWidths {
    const itemWidth = COLUMN_WIDTH_LIMITS.item.fixed
    const cclColWidth = COLUMN_WIDTH_LIMITS.ccl.fixed

    // 預設表頭文字寬度量測 (極緊湊間距：標題文字 + 排序箭頭 + 邊距)
    let maxHhpn = measureTextWidth('HHPN', true) + 14
    let maxSupplier = measureTextWidth('Supplier', true) + 14
    let maxSupplierPn = measureTextWidth('Supplier PN', true) + 14
    let maxQty = measureTextWidth('Qty', true) + 14
    let maxRemark = measureTextWidth('Remark', true) + 10
    let maxNotes = measureTextWidth('Notes', true) + 10
    let maxDesc = measureTextWidth('Description', true) + 14

    // 各欄位寬度上限門檻 (達到後即可提早結束該欄位的量測)
    const MAX_HHPN_THRESHOLD = COLUMN_WIDTH_LIMITS.hhpn.max
    const MAX_SUPPLIER_THRESHOLD = COLUMN_WIDTH_LIMITS.supplier.max
    const MAX_SUPPLIER_PN_THRESHOLD = COLUMN_WIDTH_LIMITS.supplier_pn.max
    const MAX_QTY_THRESHOLD = COLUMN_WIDTH_LIMITS.qty.max
    const MAX_REMARK_THRESHOLD = COLUMN_WIDTH_LIMITS.remark.max
    const MAX_NOTES_THRESHOLD = COLUMN_WIDTH_LIMITS.notes.max

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

      // 若所有固定與限制門檻欄位皆已達上限，僅需快速檢查剩餘列的 description
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
      hhpn: Math.ceil(Math.min(COLUMN_WIDTH_LIMITS.hhpn.max, Math.max(COLUMN_WIDTH_LIMITS.hhpn.min, maxHhpn))),
      supplier: Math.ceil(Math.min(COLUMN_WIDTH_LIMITS.supplier.max, Math.max(COLUMN_WIDTH_LIMITS.supplier.min, maxSupplier))),
      supplier_pn: Math.ceil(Math.min(COLUMN_WIDTH_LIMITS.supplier_pn.max, Math.max(COLUMN_WIDTH_LIMITS.supplier_pn.min, maxSupplierPn))),
      qty: Math.ceil(Math.min(COLUMN_WIDTH_LIMITS.qty.max, Math.max(COLUMN_WIDTH_LIMITS.qty.min, maxQty))),
      ccl: cclColWidth,
      remark: Math.ceil(Math.min(COLUMN_WIDTH_LIMITS.remark.max, Math.max(COLUMN_WIDTH_LIMITS.remark.min, maxRemark))),
      notes: Math.ceil(Math.min(COLUMN_WIDTH_LIMITS.notes.max, Math.max(COLUMN_WIDTH_LIMITS.notes.min, maxNotes))),
      maxDesc: maxDesc,
      maxNotes: maxNotes,
    }
  }

  /**
   * 計算各欄位最適欄寬
   * 
   * 固定欄寬優先讀取快取；若為首次載入則計算一次並寫入快取。
   * 後續不論模式切換 (EBOM/Matrix)、CCL 篩選、折疊展開或視窗縮放，
   * 均以 O(1) 常數時間瓜分可視寬度，杜絕卡頓與延遲。
   * 
   * 分配邏輯：
   * 1. 扣除固定欄位總寬度 (Item, HHPN, Supplier, Supplier PN, Revision/Model, CCL, Remark 等)。
   * 2. 可用空間採三階段流水式分配：
   *    - 第 1 階段：優先分配給 Description，直到滿足其最長內容所需上限。
   *    - 第 2 階段：若存在 Notes 欄位 (Matrix 模式)，分配給 Notes 直到上限 (由 COLUMN_WIDTH_LIMITS.notes.max 約束)；若無則略過。
   *    - 第 3 階段：剩餘所有空間全數由 Location 吸收。
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
    const MIN_DESC_WIDTH = COLUMN_WIDTH_LIMITS.description.min
    const MIN_LOC_WIDTH = COLUMN_WIDTH_LIMITS.locations.min
    const MIN_NOTES_WIDTH = COLUMN_WIDTH_LIMITS.notes.min
    const MAX_NOTES_LIMIT = COLUMN_WIDTH_LIMITS.notes.max

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
      maxNotes: defaultColumnWidths.notes,
    }

    // 2. 判斷當前視圖模式是否包含 Notes 欄位 (目前僅 Matrix 模式具有 Notes 欄位)
    const hasNotes = bomType === 'Matrix'

    // 3. 根據 EBOM / Matrix 模式決定固定欄位寬度總和
    // 注意：Notes 已轉為自適應動態分配欄位，不再計入 fixedTotal
    let fixedTotal = base.item + base.hhpn + base.supplier + base.supplier_pn

    if (bomType === 'EBOM') {
      const totalQtyWidth = totalEBOMRevisionWidth > 0
        ? totalEBOMRevisionWidth
        : Math.max(revisionCount, 1) * COLUMN_WIDTH_LIMITS.ebomRevisionDefault
      fixedTotal += totalQtyWidth + base.ccl + base.remark
    } else {
      let totalRevWidth = 0
      if (totalMatrixModelWidth > 0) {
        totalRevWidth = totalMatrixModelWidth
      } else {
        const modelCount = matrixModelColumnCount > 0 ? matrixModelColumnCount : Math.max(revisionCount, 1)
        totalRevWidth = modelCount * COLUMN_WIDTH_LIMITS.matrixModelDefault
      }
      fixedTotal += base.qty + totalRevWidth
    }

    const minNotes = hasNotes ? MIN_NOTES_WIDTH : 0
    const totalRequiredMinWidth = fixedTotal + MIN_LOC_WIDTH + MIN_DESC_WIDTH + minNotes

    // 同步更新表格最小總寬度，供外部 DataTable tableStyle 動態撐開橫向捲軸
    totalTableMinWidth.value = totalRequiredMinWidth

    // 4. 取得可用可視寬度
    const visibleWidth = getWorkspaceVisibleWidth()

    let finalDescWidth: number = MIN_DESC_WIDTH
    let finalLocWidth: number = MIN_LOC_WIDTH
    let finalNotesWidth: number = minNotes

    if (visibleWidth <= totalRequiredMinWidth) {
      // 情況 1：可視寬度不足以容納所有欄位最小寬度，退守至各欄最小寬度
      finalDescWidth = MIN_DESC_WIDTH
      finalLocWidth = MIN_LOC_WIDTH
      finalNotesWidth = minNotes
    } else {
      // 情況 2：可視寬度充裕，執行三階段流水瀑布式 (Waterfall) 空間分配
      const safeTotal = Math.floor(visibleWidth) - 1
      const rem = safeTotal - fixedTotal // 供 Description、Notes (若有) 與 Location 瓜分之空間

      // Location 初始目標寬度 (容納指定等寬字元數，不低於最小寬度)
      const targetCharsWidth = measureTextWidth('0'.repeat(COLUMN_WIDTH_LIMITS.locations.initChars), false, true)
      const locInit = Math.max(MIN_LOC_WIDTH, Math.ceil(targetCharsWidth + 12))
      const descInit = MIN_DESC_WIDTH
      const notesInit = minNotes
      const reserved = descInit + locInit + notesInit

      if (rem < reserved) {
        // 可用空間介於 totalRequiredMinWidth 與 reserved 之間時，依底線優先分配
        finalDescWidth = MIN_DESC_WIDTH
        finalNotesWidth = notesInit
        finalLocWidth = Math.max(MIN_LOC_WIDTH, rem - finalDescWidth - finalNotesWidth)
      } else {
        let extra = rem - reserved

        // ── 第 1 階段：優先分配給 Description 直到滿足最長內容所需上限 ────
        const descNeeded = Math.max(MIN_DESC_WIDTH, Math.ceil(base.maxDesc + 14))
        const descCapacity = Math.max(0, descNeeded - descInit)
        const descAlloc = Math.min(extra, descCapacity)
        finalDescWidth = descInit + descAlloc
        extra -= descAlloc

        // ── 第 2 階段：若存在 Notes 欄位且有剩餘空間，分配給 Notes 直到滿足上限 ────
        if (hasNotes && extra > 0) {
          const notesNeeded = Math.min(MAX_NOTES_LIMIT, Math.max(MIN_NOTES_WIDTH, Math.ceil(base.maxNotes + 14)))
          const notesCapacity = Math.max(0, notesNeeded - notesInit)
          const notesAlloc = Math.min(extra, notesCapacity)
          finalNotesWidth = notesInit + notesAlloc
          extra -= notesAlloc
        } else {
          finalNotesWidth = notesInit
        }

        // ── 第 3 階段：最後所有剩餘空間全數由 Location 吸收 ────
        finalLocWidth = locInit + extra
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
      notes: hasNotes ? finalNotesWidth : defaultColumnWidths.notes,
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
    totalTableMinWidth,
    computeColumnWidths,
    invalidateBaseWidthsCache,
    setupResizeListener,
  }
}

