/**
 * @file textHighlight.ts
 * @description 搜尋關鍵字文字片段高亮切割工具
 * 
 * 本工具負責將儲存格文字依據使用者輸入的搜尋關鍵字，進行不分大小寫的片段比對與切割，
 * 回傳由「匹配文字片段」與「一般文字片段」組成的結構化陣列，供 Vue 模板渲染 `<mark>` 高亮標籤。
 * 
 * 導出函式：
 * - getHighlightedParts: 文字片段切割與匹配分析
 * 
 * 依賴模組：無外部依賴 (純字串處理)
 */

export interface HighlightPart {
  /** 片段文字內容 */
  text: string
  /** 是否為關鍵字匹配片段 (true 需套用高亮 mark 樣式) */
  isMatch: boolean
}

/**
 * 將原始文字依據搜尋關鍵字切割為匹配與未匹配之片段清單
 * 
 * @param {string | number | null | undefined} text - 原始儲存格字串或數值
 * @param {string} query - 使用者輸入之搜尋關鍵字
 * @returns {HighlightPart[]} 切割後之片段陣列
 */
export function getHighlightedParts(
  text: string | number | null | undefined,
  query: string
): HighlightPart[] {
  const str = String(text ?? '')
  if (!query || !query.trim() || !str) {
    return [{ text: str, isMatch: false }]
  }

  const parts: HighlightPart[] = []
  const q = query.trim()
  const lowerStr = str.toLowerCase()
  const lowerQ = q.toLowerCase()

  let startIndex = 0
  let matchIndex = lowerStr.indexOf(lowerQ, startIndex)

  while (matchIndex !== -1) {
    // 匹配點前的一般非匹配文字
    if (matchIndex > startIndex) {
      parts.push({
        text: str.substring(startIndex, matchIndex),
        isMatch: false,
      })
    }
    // 關鍵字匹配片段 (保留原始文字之大小寫形態)
    parts.push({
      text: str.substring(matchIndex, matchIndex + q.length),
      isMatch: true,
    })
    startIndex = matchIndex + q.length
    matchIndex = lowerStr.indexOf(lowerQ, startIndex)
  }

  // 結尾剩餘的非匹配文字
  if (startIndex < str.length) {
    parts.push({
      text: str.substring(startIndex),
      isMatch: false,
    })
  }

  return parts
}
