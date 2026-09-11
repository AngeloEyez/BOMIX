/**
 * @file textMeasure.ts
 * @description 離屏 Canvas 文字像素寬度測量工具
 * 
 * 本工具使用離屏 (Offscreen) HTML5 Canvas 2D Context 量測字串在指定字體與字重下的精確像素寬度，
 * 徹底避免透過插入 DOM 節點量測所引發的昂貴重排 (Reflow) 與重繪 (Repaint)。
 * 
 * 導出函式：
 * - measureTextWidth: 測量純文字或表頭標題的像素寬度
 * 
 * 依賴模組：瀏覽器原生 HTMLCanvasElement (DOM API)
 */

/** 離屏 Canvas 單例快取，避免重複建立 Canvas 元素 */
let measureCanvas: HTMLCanvasElement | null = null

/**
 * 測量文字在指定字體下的像素寬度
 * 
 * @param {string} text - 待測量的文字字串
 * @param {boolean} [isHeader=false] - 是否為表頭 (表頭使用加粗 600 字重，一般文字使用 400 字重)
 * @returns {number} 文字像素寬度 (px)，若無輸入字串則回傳 0
 */
export function measureTextWidth(text: string, isHeader = false): number {
  if (!text) return 0

  if (!measureCanvas && typeof document !== 'undefined') {
    measureCanvas = document.createElement('canvas')
  }

  if (!measureCanvas) {
    // 伺服器端渲染 (SSR) 或無 DOM 環境下的字元粗估備援
    return text.length * 7.5
  }

  const ctx = measureCanvas.getContext('2d')
  if (!ctx) {
    return text.length * 7.5
  }

  ctx.font = isHeader
    ? '600 12px -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif'
    : '12px -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif'

  return ctx.measureText(text).width
}
