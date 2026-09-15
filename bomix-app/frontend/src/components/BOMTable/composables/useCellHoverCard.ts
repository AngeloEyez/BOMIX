/**
 * @file useCellHoverCard.ts
 * @description BOM 表格儲存格互動式懸停卡片 (Hover Card) 狀態管理與滑鼠防抖過渡引擎 (Composable)
 * 
 * 本模組負責管理 Description、Location 與 Notes 欄位之互動懸停視窗：
 * 1. 提供 40ms 極致迅速顯示防抖，游標稍作停頓即刻彈出，直覺流暢。
 * 2. 提供 300ms 隱藏安全緩衝時間，讓使用者游標能平滑從單元格移入浮動懸停視窗內進行文字選取與複製。
 * 3. 游標移入視窗後自動清除隱藏定時器，維持視窗常駐；移出視窗時重新計時關閉。
 * 4. 針對 Notes 欄位提供多行編輯狀態管理 (isEditing, draftNotes, save/cancel)，編輯中鎖定不自動關閉。
 * 5. 支援視窗實質滾動 (>8px)、點擊外部與按下 Esc 鍵智慧關閉機制。
 * 
 * 導出函式：
 * - useCellHoverCard: 建立並管理懸停卡片狀態之 Composable
 * 
 * 依賴模組：Vue 3 響應式核心 (ref, onMounted, onUnmounted) 與 BOMDisplayRow 型別
 */

import { ref, onMounted, onUnmounted } from 'vue'
import type { BOMDisplayRow } from '../types'

/** 支援互動式懸停卡片之欄位類型 */
export type HoverCardField = 'description' | 'locations' | 'notes'

/** 儲存格螢幕座標矩形之純物件定義 (避免 DOMRect 原生 getter 在 Vue Proxy 上引發 Illegal invocation) */
export interface CellRect {
  top: number
  bottom: number
  left: number
  right: number
  width: number
  height: number
}

/**
 * 建立 BOM 表格儲存格互動式懸停卡片管理器
 * 
 * @returns {object} 懸停卡片響應式狀態與事件處理方法
 */
export function useCellHoverCard() {
  /** 懸停卡片是否顯示中 */
  const isCardVisible = ref(false)

  /** 當前觸發之欄位名稱 */
  const activeField = ref<HoverCardField | null>(null)

  /** 當前觸發列之資料物件 */
  const activeRow = ref<BOMDisplayRow | null>(null)

  /** 當前顯示之文字內容 */
  const activeContent = ref('')

  /** 觸發單元格之螢幕幾何邊界矩形 (用於精準定位) */
  const targetRect = ref<CellRect | null>(null)

  /** 當前單元格 DOM 元素參考 (用於比對游標是否仍在同一格) */
  let activeElement: HTMLElement | null = null

  /** 記錄開啟卡片當下的滾動位置 (用於過濾次像素抖動) */
  let initialScrollTop = 0
  let initialScrollLeft = 0

  /** Notes 欄位是否處於多行文字編輯模式 */
  const isEditing = ref(false)

  /** Notes 編輯中的草稿文字內容 */
  const draftNotes = ref('')

  /** 顯示延遲定時器 (40ms) */
  let showTimer: ReturnType<typeof setTimeout> | null = null

  /** 隱藏緩衝定時器 (300ms) */
  let hideTimer: ReturnType<typeof setTimeout> | null = null

  /**
   * 清除顯示延遲定時器
   */
  function clearShowTimer(): void {
    if (showTimer !== null) {
      clearTimeout(showTimer)
      showTimer = null
    }
  }

  /**
   * 清除隱藏緩衝定時器
   */
  function clearHideTimer(): void {
    if (hideTimer !== null) {
      clearTimeout(hideTimer)
      hideTimer = null
    }
  }

  /**
   * 滑鼠移入支援懸停之儲存格 (Description / Location / Notes)
   * 
   * @param {MouseEvent} event - 滑鼠事件物件
   * @param {HoverCardField} field - 觸發之欄位代碼
   * @param {BOMDisplayRow} row - 當前列資料物件
   */
  function handleCellMouseEnter(
    event: MouseEvent,
    field: HoverCardField,
    row: BOMDisplayRow
  ): void {
    const currentTarget = event.currentTarget as HTMLElement
    if (!currentTarget) return

    // 僅當儲存格文字超出寬度被截斷 (Ellipsis) 時才觸發顯示；若能完整顯示則不出現
    const isOverflow = currentTarget.scrollWidth > currentTarget.clientWidth + 1
    if (!isOverflow) {
      return
    }

    // 清除即將關閉的隱藏計時
    clearHideTimer()

    // 若正處於 Notes 編輯中，且滑鼠移到其它單元格，保護使用者輸入內容，不自動切換
    if (isEditing.value && activeElement !== currentTarget) {
      return
    }

    // 若已經在同一個單元格且卡片已顯示，維持顯示不重新計時
    if (isCardVisible.value && activeElement === currentTarget && activeField.value === field) {
      return
    }

    clearShowTimer()

    // 啟動 40ms 極致靈敏顯示
    showTimer = setTimeout(() => {
      // 二次檢查文字是否依然截斷 (避免視窗尺寸瞬間改變)
      if (currentTarget.scrollWidth <= currentTarget.clientWidth + 1) {
        return
      }

      activeElement = currentTarget
      activeField.value = field
      activeRow.value = row

      // 安全轉換為 Plain Object 邊界，避免 DOMRect 在 Vue 響應式 Proxy 存取 getter 異常
      const r = currentTarget.getBoundingClientRect()
      targetRect.value = {
        top: r.top,
        bottom: r.bottom,
        left: r.left,
        right: r.right,
        width: r.width,
        height: r.height
      }

      // 提取對應欄位內容
      let rawText = ''
      if (field === 'description') {
        rawText = row.description || ''
      } else if (field === 'locations') {
        rawText = row.locations || ''
      } else if (field === 'notes') {
        rawText = row.notes || ''
      }

      activeContent.value = rawText

      // 若未在編輯狀態，初始化 draftNotes
      if (!isEditing.value) {
        draftNotes.value = rawText
      }

      // 記錄觸發當下的捲動位置
      const scroller = document.querySelector('.p-datatable-table-container, [data-pc-name="virtualscroller"]')
      if (scroller) {
        initialScrollTop = scroller.scrollTop
        initialScrollLeft = scroller.scrollLeft
      }

      isCardVisible.value = true
    }, 40)
  }

  /**
   * 滑鼠移出儲存格
   * 啟動 300ms 緩衝延遲，允許使用者將游標移入懸停視窗
   */
  function handleCellMouseLeave(): void {
    clearShowTimer()

    // 若處於編輯狀態，嚴格鎖定不關閉
    if (isEditing.value) {
      return
    }

    clearHideTimer()
    hideTimer = setTimeout(() => {
      closeCard()
    }, 300)
  }

  /**
   * 滑鼠移入懸停卡片本體
   * 清除隱藏計時器，保持視窗常駐以供選取複製
   */
  function handleCardMouseEnter(): void {
    clearHideTimer()
  }

  /**
   * 滑鼠移出懸停卡片本體
   * 若非編輯中，啟動 300ms 延遲關閉
   */
  function handleCardMouseLeave(): void {
    if (isEditing.value) {
      return
    }

    clearHideTimer()
    hideTimer = setTimeout(() => {
      closeCard()
    }, 300)
  }

  /**
   * 關閉懸停卡片
   * 
   * @param {boolean} [force=false] - 是否強制關閉 (即使處於編輯狀態亦關閉)
   */
  function closeCard(force: boolean = false): void {
    if (isEditing.value && !force) {
      return
    }

    clearShowTimer()
    clearHideTimer()
    isCardVisible.value = false
    activeField.value = null
    activeRow.value = null
    activeElement = null
    targetRect.value = null
    isEditing.value = false
  }

  /**
   * 開啟 Notes 多行編輯模式
   */
  function startEditing(): void {
    if (activeField.value !== 'notes') return
    isEditing.value = true
    draftNotes.value = activeRow.value?.notes || ''
    clearHideTimer()
  }

  /**
   * 取消 Notes 編輯並還原草稿
   */
  function cancelEditing(): void {
    draftNotes.value = activeRow.value?.notes || ''
    isEditing.value = false
    closeCard(true)
  }

  /**
   * 儲存 Notes 修改內容
   * 
   * @param {(row: BOMDisplayRow, newNotes: string) => void} [onSaveCallback] - 外部儲存回呼函式
   */
  function saveNotes(onSaveCallback?: (row: BOMDisplayRow, newNotes: string) => void): void {
    if (!activeRow.value) return

    const newNotes = draftNotes.value.trim()
    activeRow.value.notes = newNotes
    activeContent.value = newNotes

    if (onSaveCallback) {
      onSaveCallback(activeRow.value, newNotes)
    }

    isEditing.value = false
    closeCard(true)
  }

  /**
   * 表格滾動時處理函式 (過濾次像素抖動，滾動超過 8px 時關閉卡片)
   */
  function onTableScroll(event?: Event): void {
    if (!isCardVisible.value || isEditing.value) {
      return
    }

    const target = (event?.target as HTMLElement) || document.querySelector('.p-datatable-table-container, [data-pc-name="virtualscroller"]')
    if (target) {
      const deltaY = Math.abs(target.scrollTop - initialScrollTop)
      const deltaX = Math.abs(target.scrollLeft - initialScrollLeft)
      // 若滾動距離小於 8px，視為次像素重繪抖動，不予關閉
      if (deltaY < 8 && deltaX < 8) {
        return
      }
    }

    closeCard()
  }

  /**
   * 全域鍵盤事件監聽 (按下 Esc 鍵關閉或取消編輯)
   * 
   * @param {KeyboardEvent} event - 鍵盤事件物件
   */
  function handleKeyDown(event: KeyboardEvent): void {
    if (event.key === 'Escape' && isCardVisible.value) {
      if (isEditing.value) {
        cancelEditing()
      } else {
        closeCard(true)
      }
    }
  }

  onMounted(() => {
    window.addEventListener('keydown', handleKeyDown)
  })

  onUnmounted(() => {
    clearShowTimer()
    clearHideTimer()
    window.removeEventListener('keydown', handleKeyDown)
  })

  return {
    isCardVisible,
    activeField,
    activeRow,
    activeContent,
    targetRect,
    isEditing,
    draftNotes,
    handleCellMouseEnter,
    handleCellMouseLeave,
    handleCardMouseEnter,
    handleCardMouseLeave,
    closeCard,
    startEditing,
    cancelEditing,
    saveNotes,
    onTableScroll
  }
}
