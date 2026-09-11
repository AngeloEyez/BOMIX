/**
 * @file useCellAutoScroll.ts
 * @description 表格儲存格拖曳選取文字自動水平滾動動畫引擎 (Composable)
 * 
 * 當使用者按住滑鼠左鍵框選單元格 (特別是 Description、Location 等長文字欄位) 並向左右兩端拖曳時，
 * 本模組提供高達 60fps~120fps 的平滑自動平移滾動動畫，
 * 同時動態利用 Caret API (caretRangeFromPoint / caretPositionFromPoint) 延伸選取光棒，
 * 確保使用者能完整選取並看見被 text-overflow 省略的文字。
 * 
 * 導出函式：
 * - useCellAutoScroll: 回傳 onTableMouseDown 事件委派處理函式，並自動管理全域事件監聽與動畫幀清理
 * 
 * 依賴模組：Vue 3 核心生命週期 (onMounted, onUnmounted)
 */

import { onMounted, onUnmounted } from 'vue'

/**
 * 建立並啟用表格儲存格拖曳選取自動滾動引擎
 */
export function useCellAutoScroll() {
  /** 當前正在進行拖曳選取或具有選取狀態的儲存格元素 */
  let activeScrollCell: HTMLElement | null = null
  /** 是否正在進行滑鼠左鍵按住拖曳選取 */
  let isDraggingSelection = false
  /** requestAnimationFrame 動畫幀 ID */
  let autoScrollRafId: number | null = null
  /** 當前水平滾動速度 (px/frame，正數向右展開內容，負數向左復位) */
  let scrollVelocity = 0
  /** 最後記錄的滑鼠螢幕座標 */
  let lastMouseX = 0
  let lastMouseY = 0

  /**
   * 遞迴尋找指定 DOM 節點內部最末端的純文字節點 (Text Node)
   * @param {Node} node - 欲搜尋的 DOM 節點
   * @returns {Node | null} 最末端的 Text Node，若不存在則回傳 null
   */
  function getLastTextNode(node: Node): Node | null {
    if (node.nodeType === Node.TEXT_NODE) {
      return node
    }
    for (let i = node.childNodes.length - 1; i >= 0; i--) {
      const child = node.childNodes[i]
      if (!child) continue
      const found = getLastTextNode(child)
      if (found) return found
    }
    return null
  }

  /**
   * 將當前瀏覽器文字選取範圍 (Selection) 延伸至該儲存格文字的最末端
   * @param {HTMLElement} cell - 儲存格容器元素
   */
  function extendSelectionToEnd(cell: HTMLElement): void {
    const sel = window.getSelection()
    if (!sel || sel.rangeCount === 0) return

    const lastText = getLastTextNode(cell)
    if (lastText && sel.extend) {
      try {
        const len = lastText.textContent ? lastText.textContent.length : 0
        sel.extend(lastText, len)
      } catch {
        // 忽略部分瀏覽器在跨節點選取邊界時的無效操作例外
      }
    }
  }

  /**
   * 在滾動期間動態將選取範圍焦點 (Focus) 延伸至可視區域邊界的字元
   * @param {HTMLElement} cell - 儲存格容器元素
   */
  function updateSelectionDuringScroll(cell: HTMLElement): void {
    const sel = window.getSelection()
    if (!sel || sel.rangeCount === 0) return

    const rect = cell.getBoundingClientRect()
    // 取樣點限制在儲存格可視範圍邊界內部 (留 3px 緩衝確保命中文字節點)
    const sampleX = scrollVelocity > 0
      ? Math.min(rect.right - 3, lastMouseX)
      : Math.max(rect.left + 3, lastMouseX)
    const sampleY = Math.max(rect.top + 2, Math.min(rect.bottom - 2, lastMouseY))

    let targetNode: Node | null = null
    let targetOffset = 0

    if (document.caretRangeFromPoint) {
      const range = document.caretRangeFromPoint(sampleX, sampleY)
      if (range) {
        targetNode = range.startContainer
        targetOffset = range.startOffset
      }
    } else if ((document as unknown as { caretPositionFromPoint?: (x: number, y: number) => { offsetNode: Node; offset: number } | null }).caretPositionFromPoint) {
      const pos = (document as unknown as { caretPositionFromPoint: (x: number, y: number) => { offsetNode: Node; offset: number } | null }).caretPositionFromPoint(sampleX, sampleY)
      if (pos) {
        targetNode = pos.offsetNode
        targetOffset = pos.offset
      }
    }

    // 若滑鼠已超越右邊界且已接近捲動底端，直接鎖定到末尾最後一個文字節點
    const maxScroll = cell.scrollWidth - cell.clientWidth
    if (cell.scrollLeft >= maxScroll - 2 && lastMouseX >= rect.right - 3) {
      const lastText = getLastTextNode(cell)
      if (lastText) {
        targetNode = lastText
        targetOffset = lastText.textContent ? lastText.textContent.length : 0
      }
    }

    if (targetNode && cell.contains(targetNode) && sel.extend) {
      try {
        sel.extend(targetNode, targetOffset)
      } catch {
        // 忽略延伸邊界例外
      }
    }
  }

  /**
   * 拖曳選取自動滾動動畫幀步進循環 (60fps ~ 120fps)
   */
  function stepAutoScroll(): void {
    if (!isDraggingSelection || !activeScrollCell || scrollVelocity === 0) {
      autoScrollRafId = null
      return
    }

    const cell = activeScrollCell
    const maxScroll = cell.scrollWidth - cell.clientWidth

    // 到達最右邊界或最左邊界時平滑煞車
    if (scrollVelocity > 0 && cell.scrollLeft >= maxScroll) {
      cell.scrollLeft = maxScroll
      extendSelectionToEnd(cell)
      autoScrollRafId = null
      return
    } else if (scrollVelocity < 0 && cell.scrollLeft <= 0) {
      cell.scrollLeft = 0
      autoScrollRafId = null
      return
    }

    // 步進增加滾動位置
    cell.scrollLeft = Math.max(0, Math.min(maxScroll, cell.scrollLeft + scrollVelocity))

    // 同步延伸文字選取光棒
    updateSelectionDuringScroll(cell)

    // 繼續下一幀動畫
    autoScrollRafId = requestAnimationFrame(stepAutoScroll)
  }

  /**
   * 表格滑鼠按下事件處理 (委派偵測 cell-text 拖曳選取起點)
   * 需綁定於表格最外層容器的 @mousedown 事件
   * @param {MouseEvent} e - 滑鼠事件物件
   */
  function onTableMouseDown(e: MouseEvent): void {
    // 只響應滑鼠左鍵 (button === 0)
    if (e.button !== 0) return

    const target = e.target as HTMLElement | null
    if (!target) return

    // 略過按鈕、輸入框、下拉選單等互動元件，避免干擾正常操作
    if (target.closest('button, .p-button, input, select, .p-select, .toggle-all-btn, .toggle-ss-btn')) {
      return
    }

    const cell = target.closest('.cell-text') as HTMLElement | null
    if (!cell) {
      // 點擊非儲存格文字區域時，若前一個有選取的儲存格已取消選取，則將其復位
      if (activeScrollCell && !isDraggingSelection) {
        const sel = window.getSelection()
        if (!sel || sel.isCollapsed) {
          activeScrollCell.classList.remove('is-selecting', 'has-selection')
          activeScrollCell.scrollLeft = 0
          activeScrollCell = null
        }
      }
      return
    }

    // 若切換到不同儲存格，復位前一個儲存格
    if (activeScrollCell && activeScrollCell !== cell) {
      activeScrollCell.classList.remove('is-selecting', 'has-selection')
      activeScrollCell.scrollLeft = 0
    }

    activeScrollCell = cell
    isDraggingSelection = true
    lastMouseX = e.clientX
    lastMouseY = e.clientY
    scrollVelocity = 0

    // 立即套用 is-selecting class，將省略號切換為 clip，保證拖曳起點至終點所有文字完全可見
    cell.classList.add('is-selecting')
  }

  /**
   * 全域滑鼠移動事件處理 (偵測向邊界拖曳速度)
   * @param {MouseEvent} e - 滑鼠事件物件
   */
  function onGlobalMouseMove(e: MouseEvent): void {
    if (!isDraggingSelection || !activeScrollCell) return

    lastMouseX = e.clientX
    lastMouseY = e.clientY

    const rect = activeScrollCell.getBoundingClientRect()
    const edgeZone = 24 // 邊界感應區大小 (px)

    if (e.clientX > rect.right - edgeZone) {
      // 游標靠近或超出右側邊界：向右平滑滾動以展示末端隱藏文字
      const overflow = Math.max(0, e.clientX - rect.right)
      scrollVelocity = Math.min(20, 5 + Math.floor(overflow / 6) * 3)
    } else if (e.clientX < rect.left - edgeZone) {
      // 游標靠近或超出左側邊界：向左反向滾動復位
      const overflow = Math.max(0, rect.left - e.clientX)
      scrollVelocity = -Math.min(20, 5 + Math.floor(overflow / 6) * 3)
    } else {
      // 游標在儲存格中央區域，不進行自動滾動
      scrollVelocity = 0
    }

    if (scrollVelocity !== 0 && autoScrollRafId === null) {
      autoScrollRafId = requestAnimationFrame(stepAutoScroll)
    }
  }

  /**
   * 全域滑鼠釋放事件處理 (結束拖曳但維持選取狀態)
   */
  function onGlobalMouseUp(): void {
    if (!isDraggingSelection) return
    isDraggingSelection = false
    scrollVelocity = 0

    if (autoScrollRafId !== null) {
      cancelAnimationFrame(autoScrollRafId)
      autoScrollRafId = null
    }

    if (activeScrollCell) {
      activeScrollCell.classList.remove('is-selecting')
      const sel = window.getSelection()
      if (sel && !sel.isCollapsed && sel.toString().length > 0) {
        // 保持選取狀態標記，維持 text-overflow: clip，讓使用者可直接清楚看到完整末端文字並按 Ctrl+C 複製
        activeScrollCell.classList.add('has-selection')
      } else {
        // 若無任何選取文字，平滑將 scrollLeft 歸零並清空引用
        activeScrollCell.classList.remove('has-selection')
        activeScrollCell.scrollLeft = 0
        activeScrollCell = null
      }
    }
  }

  /**
   * 全域文字選取變更監聽 (處理雙擊/連按三下全選、以及選取取消之自動復位)
   */
  function onGlobalSelectionChange(): void {
    const sel = window.getSelection()

    // 1. 選取範圍被取消或折疊 (點擊別處、按 Esc、空白點擊)
    if (!sel || sel.isCollapsed || sel.toString().length === 0) {
      if (!isDraggingSelection && activeScrollCell) {
        activeScrollCell.classList.remove('is-selecting', 'has-selection')
        activeScrollCell.scrollLeft = 0
        activeScrollCell = null
      }
      // 確保清掃頁面上所有殘留的選取標記
      const lingering = document.querySelectorAll('.cell-text.has-selection')
      lingering.forEach(el => {
        el.classList.remove('has-selection')
        if (el instanceof HTMLElement && !isDraggingSelection) {
          el.scrollLeft = 0
        }
      })
      return
    }

    // 2. 有文字被選取 (例如連按兩下雙擊選字、連按三下全選等)
    const anchorNode = sel.anchorNode
    const focusNode = sel.focusNode

    const cell = (anchorNode instanceof HTMLElement ? anchorNode : anchorNode?.parentElement)?.closest('.cell-text') as HTMLElement | null ||
                 (focusNode instanceof HTMLElement ? focusNode : focusNode?.parentElement)?.closest('.cell-text') as HTMLElement | null

    if (cell) {
      cell.classList.add('has-selection')
      if (activeScrollCell && activeScrollCell !== cell && !isDraggingSelection) {
        activeScrollCell.classList.remove('has-selection', 'is-selecting')
        activeScrollCell.scrollLeft = 0
      }
      activeScrollCell = cell
    }
  }

  onMounted(() => {
    window.addEventListener('mousemove', onGlobalMouseMove)
    window.addEventListener('mouseup', onGlobalMouseUp)
    document.addEventListener('selectionchange', onGlobalSelectionChange)
  })

  onUnmounted(() => {
    window.removeEventListener('mousemove', onGlobalMouseMove)
    window.removeEventListener('mouseup', onGlobalMouseUp)
    document.removeEventListener('selectionchange', onGlobalSelectionChange)

    if (autoScrollRafId !== null) {
      cancelAnimationFrame(autoScrollRafId)
      autoScrollRafId = null
    }
  })

  return {
    onTableMouseDown,
  }
}
