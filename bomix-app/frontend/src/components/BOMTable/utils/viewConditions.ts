/**
 * @file viewConditions.ts
 * @description BOM 視圖過濾條件定義與描述工具模組
 * 
 * 本模組依據產品規格 (product-spec 6.4.2) 與後端過濾器 (backend/view/filter.go)，
 * 定義各視圖 (ALL, SMD, PTH, BOTTOM, NI, PROTO, MP, CCL) 的過濾條件規則，
 * 並提供格式化 Debug Log 輸出的輔助函式。
 */

/**
 * 視圖過濾條件資訊介面
 */
export interface ViewFilterInfo {
  /** 視圖識別代碼 (全小寫，例如 'all', 'smd') */
  key: string
  /** 視圖顯示名稱 (例如 'All', 'SMD') */
  label: string
  /** 視圖全大寫代碼 (例如 'ALL', 'SMD') */
  name: string
  /** 視圖中文繁體說明 */
  description: string
  /** 具體過濾條件清單 */
  conditions: string[]
  /** 單行條件摘要字串 */
  summary: string
}

/**
 * 各視圖靜態條件定義映射表
 */
const VIEW_CONDITION_DEFINITIONS: Record<string, {
  label: string
  name: string
  description: string
  conditions: string[]
  summary: string
}> = {
  all: {
    label: 'All',
    name: 'ALL',
    description: '全部有效上件物料',
    conditions: [
      "bom_status in ('I', 'P', 'M')",
      "bom_status != 'X' (排除不上件物料)",
    ],
    summary: "bom_status in ('I', 'P', 'M') (排除 bom_status = 'X')",
  },
  smd: {
    label: 'SMD',
    name: 'SMD',
    description: '表面黏著零件 (Surface Mount)',
    conditions: [
      "type = 'SMD'",
      "bom_status != 'X'",
    ],
    summary: "type = 'SMD' AND bom_status != 'X'",
  },
  pth: {
    label: 'PTH',
    name: 'PTH',
    description: '通孔插裝零件 (Pin-Through-Hole)',
    conditions: [
      "type = 'PTH'",
      "bom_status != 'X'",
    ],
    summary: "type = 'PTH' AND bom_status != 'X'",
  },
  bottom: {
    label: 'Bottom',
    name: 'BOTTOM',
    description: '背面零件 (Bottom Side)',
    conditions: [
      "type = 'BOTTOM'",
      "bom_status != 'X'",
    ],
    summary: "type = 'BOTTOM' AND bom_status != 'X'",
  },
  ni: {
    label: 'NI',
    name: 'NI',
    description: '不上件物料 (Not Install)',
    conditions: [
      "bom_status = 'X'",
    ],
    summary: "bom_status = 'X'",
  },
  proto: {
    label: 'PROTO',
    name: 'PROTO',
    description: '試產專用物料 (Proto Phase)',
    conditions: [
      "bom_status = 'P'",
    ],
    summary: "bom_status = 'P'",
  },
  mp: {
    label: 'MP',
    name: 'MP',
    description: '量產專用物料 (Mass Production)',
    conditions: [
      "bom_status = 'M'",
    ],
    summary: "bom_status = 'M'",
  },
  ccl: {
    label: 'CCL',
    name: 'CCL',
    description: '關鍵零件清單 (Critical Component List)',
    conditions: [
      "ccl = true",
      "bom_status != 'X'",
    ],
    summary: "ccl = true AND bom_status != 'X'",
  },
}

/**
 * 取得指定視圖的過濾條件詳細資訊
 * 
 * @param {string} viewKey - 視圖代碼 (如 'all', 'smd', 'ni'，大小寫不拘)
 * @param {string} [mode] - 可選的 BOM 模式 (如 'NPI' 或 'MP')
 * @returns {ViewFilterInfo} 該視圖的過濾條件物件
 */
export function getViewFilterInfo(viewKey: string, mode?: string): ViewFilterInfo {
  const normalizedKey = (viewKey || 'all').trim().toLowerCase()
  const def = VIEW_CONDITION_DEFINITIONS[normalizedKey]

  if (!def) {
    return {
      key: normalizedKey,
      label: viewKey,
      name: viewKey.toUpperCase(),
      description: '自訂或未知視圖',
      conditions: ['不過濾 (顯示全部)'],
      summary: '不過濾',
    }
  }

  // 若提供模式 (NPI / MP)，針對 ALL 視圖與 CCL 視圖細化有效 bom_status 說明
  const upperMode = (mode || '').trim().toUpperCase()
  if (normalizedKey === 'all') {
    if (upperMode === 'MP') {
      return {
        key: normalizedKey,
        label: def.label,
        name: def.name,
        description: `${def.description} (MP 量產模式)`,
        conditions: [
          "bom_status in ('I', 'M')",
          "bom_status != 'X' (排除不上件)",
          "bom_status != 'P' (排除試產專用)",
        ],
        summary: "bom_status in ('I', 'M') (排除 'X' 不上件與 'P' 試產)",
      }
    } else if (upperMode === 'NPI') {
      return {
        key: normalizedKey,
        label: def.label,
        name: def.name,
        description: `${def.description} (NPI 試產模式)`,
        conditions: [
          "bom_status in ('I', 'P')",
          "bom_status != 'X' (排除不上件)",
          "bom_status != 'M' (排除量產專用)",
        ],
        summary: "bom_status in ('I', 'P') (排除 'X' 不上件與 'M' 量產)",
      }
    }
  } else if (normalizedKey === 'ccl') {
    if (upperMode === 'MP') {
      return {
        key: normalizedKey,
        label: def.label,
        name: def.name,
        description: `${def.description} (MP 量產模式)`,
        conditions: [
          "ccl = true",
          "bom_status in ('I', 'M')",
        ],
        summary: "ccl = true AND bom_status in ('I', 'M')",
      }
    } else if (upperMode === 'NPI') {
      return {
        key: normalizedKey,
        label: def.label,
        name: def.name,
        description: `${def.description} (NPI 試產模式)`,
        conditions: [
          "ccl = true",
          "bom_status in ('I', 'P')",
        ],
        summary: "ccl = true AND bom_status in ('I', 'P')",
      }
    }
  }

  return {
    key: normalizedKey,
    label: def.label,
    name: def.name,
    description: def.description,
    conditions: [...def.conditions],
    summary: def.summary,
  }
}

/**
 * 格式化視圖切換與條件的 Debug 日誌訊息
 * 
 * @param {string} viewKey - 視圖代碼 (如 'all', 'smd', 'ccl')
 * @param {string} [mode] - 可選的 BOM 模式 ('NPI' | 'MP')
 * @returns {string} 格式化後的 Debug 日誌字串
 */
export function formatViewFilterLog(viewKey: string, mode?: string): string {
  const info = getViewFilterInfo(viewKey, mode)
  return `[BOMTable View] 選定視圖: ${info.name} (${info.description}) | 過濾條件: ${info.summary}`
}
