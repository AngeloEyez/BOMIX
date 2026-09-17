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

/** 文字長度量測記憶體快取 (Key: `${isHeader}:${isMono}:${text}`, Value: width) */
const measureCache = new Map<string, number>()

/** 專案代碼文字量測記憶體快取 (Key: text, Value: width) */
const projectCodeCache = new Map<string, number>()

/**
 * 清空文字量測記憶體快取
 */
export function clearMeasureCache(): void {
  measureCache.clear()
  projectCodeCache.clear()
}

/**
 * 測量文字在指定字體下的像素寬度
 * 
 * @param {string} text - 待測量的文字字串
 * @param {boolean} [isHeader=false] - 是否為表頭 (表頭使用加粗 600 字重，一般文字使用 400 字重)
 * @param {boolean} [isMono=false] - 是否為等寬字型 (料號、數據等採用 Cascadia Mono 11.5px)
 * @returns {number} 文字像素寬度 (px)，若無輸入字串則回傳 0
 */
export function measureTextWidth(text: string, isHeader = false, isMono = false): number {
  if (!text) return 0

  const cacheKey = `${isHeader ? 1 : 0}:${isMono ? 1 : 0}:${text}`
  const cached = measureCache.get(cacheKey)
  if (cached !== undefined) {
    return cached
  }

  if (!measureCanvas && typeof document !== 'undefined') {
    measureCanvas = document.createElement('canvas')
  }

  if (!measureCanvas) {
    // 伺服器端渲染 (SSR) 或無 DOM 環境下的字元粗估備援
    const width = text.length * (isMono ? 7.2 : 7)
    measureCache.set(cacheKey, width)
    return width
  }

  const ctx = measureCanvas.getContext('2d')
  if (!ctx) {
    const width = text.length * (isMono ? 7.2 : 7)
    measureCache.set(cacheKey, width)
    return width
  }

  if (isMono) {
    ctx.font = '11.5px "Cascadia Mono", "Cascadia Code", Consolas, monospace'
  } else {
    ctx.font = isHeader
      ? '600 12px -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif'
      : '12px -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif'
  }

  const width = ctx.measureText(text).width
  measureCache.set(cacheKey, width)
  return width
}

/**
 * 精準量測專案代碼在 10px 粗體下的實際像素寬度
 * 
 * @param {string} text - 待測量的專案代碼字串
 * @returns {number} 文字精準寬度 (px)
 */
export function measureProjectCodeWidth(text: string): number {
  if (!text) return 0

  const cached = projectCodeCache.get(text)
  if (cached !== undefined) {
    return cached
  }

  if (!measureCanvas && typeof document !== 'undefined') {
    measureCanvas = document.createElement('canvas')
  }

  if (!measureCanvas) {
    const width = text.length * 6.2
    projectCodeCache.set(text, width)
    return width
  }

  const ctx = measureCanvas.getContext('2d')
  if (!ctx) {
    const width = text.length * 6.2
    projectCodeCache.set(text, width)
    return width
  }

  ctx.font = '700 10px -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif'
  const width = ctx.measureText(text).width
  projectCodeCache.set(text, width)
  return width
}

