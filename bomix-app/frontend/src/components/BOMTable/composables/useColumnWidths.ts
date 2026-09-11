/**
 * @file useColumnWidths.ts
 * @description BOM 表格最適欄寬動態計算與自適應分配 (Composable)
 * 
 * 本模組負責實現 VS Code 風格之高緊湊欄寬自適應演算法：
 * 1. 利用離屏 Canvas 測量文字長度，精準計算固定欄位 (Item, HHPN, Supplier, Qty, CCL 等) 之最適最小安全寬度。
 * 2. 取得 DataTable 可視工作區總寬度 (自動扣除捲軸)，並扣除固定欄位寬度總和。
 * 3. 剩餘空間由主要欄位 Description (規格描述，min 250px) 與 Location (位置標號，min 150px) 精確瓜分，
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

/** 預設欄寬基礎配置 (像素) */
export const defaultColumnWidths: ColumnWidthConfig = {
  item: 54,
  hhpn: 140,
  description: 240,
  supplier: 120,
  supplier_pn: 150,
  qty: 55,
  locations: 220,
  ccl: 50,
  remark: 130,
  models: {}
}

/**
 * 建立最適欄寬計算器與響應式監聽
 */
export function useColumnWidths() {
  /** 各欄位響應式寬度配置 */
  const columnWidths = ref<ColumnWidthConfig>({ ...defaultColumnWidths })

  /** 表格容器 template ref，用於精準讀取父層分配之可用寬度 */
  const tableWrapperRef = ref<HTMLElement | null>(null)

  /** ResizeObserver 實例 */
  let resizeObserver: ResizeObserver | null = null

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
   * 計算各欄位最適欄寬
   * 
   * @param {BOMDisplayRow[]} rows - 當前顯示之平鋪列資料清單
   * @param {string[]} models - 當前版本的機種名稱清單
   * @param {(modelName: string) => number} getModelQty - 取得指定機種總用量之回呼函式
   * @param {(row: BOMDisplayRow, modelName: string) => string} getModelSelectedPN - 取得指定列之選定料號之回呼函式
   */
  function computeColumnWidths(
    rows: BOMDisplayRow[],
    models: string[],
    getModelQty: (modelName: string) => number,
    getModelSelectedPN: (row: BOMDisplayRow, modelName: string) => string
  ): void {
    const MIN_DESC_WIDTH = 250
    const MIN_LOC_WIDTH = 150

    // 1. 固定欄位最小寬度計算
    const itemWidth = 46

    // 表頭文字寬度量測
    let maxHhpn = measureTextWidth('HHPN', true) + 20
    let maxSupplier = measureTextWidth('Supplier', true) + 20
    let maxSupplierPn = measureTextWidth('Supplier PN', true) + 20
    let maxQty = measureTextWidth('Qty', true) + 20
    let maxRemark = measureTextWidth('Remark', true) + 14
    let maxDesc = measureTextWidth('Description', true) + 20

    // 動態 Model 欄位標題寬度
    const modelMaxMap: Record<string, number> = {}
    for (const m of models) {
      const headerTitle = `${m} (Qty: ${getModelQty(m)})`
      modelMaxMap[m] = measureTextWidth(headerTitle, true) + 20
    }

    // 遍歷當前所有顯示列以測量實際內容寬度
    for (let i = 0; i < rows.length; i++) {
      const row = rows[i]
      if (row.hhpn) {
        const w = measureTextWidth(row.hhpn) + 12
        if (w > maxHhpn) maxHhpn = w
      }
      if (row.supplier) {
        const w = measureTextWidth(row.supplier) + 12
        if (w > maxSupplier) maxSupplier = w
      }
      if (row.supplier_pn) {
        const w = measureTextWidth(row.supplier_pn) + 12
        if (w > maxSupplierPn) maxSupplierPn = w
      }
      if (row.qty !== '' && row.qty !== undefined && row.qty !== null) {
        const w = measureTextWidth(String(row.qty)) + 12
        if (w > maxQty) maxQty = w
      }
      if (row.remark) {
        const w = measureTextWidth(row.remark) + 12
        if (w > maxRemark) maxRemark = w
      }
      if (row.description) {
        const w = measureTextWidth(row.description) + 12
        if (w > maxDesc) maxDesc = w
      }
      for (const m of models) {
        const pn = getModelSelectedPN(row, m)
        if (pn) {
          const w = measureTextWidth(pn) + 12
          if (w > (modelMaxMap[m] || 0)) modelMaxMap[m] = w
        }
      }
    }

    // 限制各固定欄位之緊湊安全寬度
    const hhpnColWidth = Math.ceil(Math.min(150, Math.max(80, maxHhpn)))
    const supplierColWidth = Math.ceil(Math.min(140, Math.max(65, maxSupplier)))
    const supplierPnColWidth = Math.ceil(Math.min(160, Math.max(85, maxSupplierPn)))
    const qtyColWidth = Math.ceil(Math.min(60, Math.max(40, maxQty)))
    const cclColWidth = 38
    const remarkColWidth = Math.ceil(Math.min(140, Math.max(55, maxRemark)))

    const finalModelWidths: Record<string, number> = {}
    let totalModelsWidth = 0
    for (const m of models) {
      const w = Math.ceil(Math.min(130, Math.max(85, modelMaxMap[m] || 85)))
      finalModelWidths[m] = w
      totalModelsWidth += w
    }

    // 固定欄位最小寬度總和
    const fixedTotal = itemWidth + hhpnColWidth + supplierColWidth + supplierPnColWidth + qtyColWidth + cclColWidth + remarkColWidth + totalModelsWidth
    const totalRequiredMinWidth = fixedTotal + MIN_LOC_WIDTH + MIN_DESC_WIDTH

    // 2. 取得可用可視寬度
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

      // Location 初始目標寬度 (容納 30 個字元，約 210px，不低於 150px)
      const thirtyCharsWidth = measureTextWidth('0'.repeat(30))
      const locInit = Math.max(MIN_LOC_WIDTH, Math.ceil(thirtyCharsWidth + 12))

      if (rem - locInit >= MIN_DESC_WIDTH) {
        let descW = rem - locInit
        let locW = locInit

        // 檢查 Description 是否已足夠完整顯示其所有資料
        const descNeeded = Math.max(MIN_DESC_WIDTH, Math.ceil(maxDesc + 14))
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
      item: itemWidth,
      hhpn: hhpnColWidth,
      description: finalDescWidth,
      supplier: supplierColWidth,
      supplier_pn: supplierPnColWidth,
      qty: qtyColWidth,
      locations: finalLocWidth,
      ccl: cclColWidth,
      remark: remarkColWidth,
      models: finalModelWidths
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
    setupResizeListener,
  }
}
