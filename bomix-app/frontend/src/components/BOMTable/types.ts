/**
 * @file types.ts
 * @description BOMTable 模組專用型別與介面定義檔
 * 
 * 本模組集中管理 BOMTable 元件及其子 Composables 所需之所有資料結構，
 * 包括平鋪資料列結構 (BOMDisplayRow)、欄寬配置 (ColumnWidthConfig) 及視圖選項列舉。
 * 
 * 依賴模組：無外部依賴 (純型別定義)
 */

/**
 * BOM 表格各欄位寬度配置物件規格 (單位：像素 px)
 */
export interface ColumnWidthConfig {
  /** Item 項目編號與開合符號欄位寬度 */
  item: number
  /** 鴻海料號 HHPN 欄位寬度 */
  hhpn: number
  /** 規格描述 Description 欄位寬度 (自適應主要欄位，保證 >= 250px) */
  description: number
  /** 主要供應商 Supplier 欄位寬度 */
  supplier: number
  /** 供應商料號 Supplier PN 欄位寬度 */
  supplier_pn: number
  /** 用量 Qty 欄位寬度 */
  qty: number
  /** 位置標號 Location 欄位寬度 (自適應次要欄位，保證 >= 150px) */
  locations: number
  /** CCL 關鍵物料標記欄位寬度 */
  ccl: number
  /** 備註 Remark 欄位寬度 */
  remark: number
  /** 各動態 Model 欄位之寬度映射表 (key: model_name, value: 像素寬度) */
  models: Record<string, number>
}

/**
 * 平鋪後供 DataTable 虛擬滾動 (VirtualScroller) 渲染之單列資料結構
 * 將主料與其所屬之 2nd 替代料皆展平為此物件，2nd 替代料緊隨主料下方。
 */
export interface BOMDisplayRow {
  /** 列唯一鍵值 (例如：`${parentKey}-main` 或 `${parentKey}-ss-0-PN123`) */
  rowId: string
  /** 所屬主料群組唯一識別鍵 (用於關聯主料與替代料群組) */
  parentKey: string
  /** 是否為 2nd 替代料列 (true: 替代料，false: 主料) */
  isSecondSource: boolean
  /** 是否擁有 2nd 替代料 (僅主料可能為 true) */
  hasSecondSources: boolean
  /** 替代料筆數 */
  secondSourcesCount: number
  /** 項目編號 (替代料此欄位必定為空字串) */
  item: string
  /** 鴻海料號 */
  hhpn: string
  /** 零件規格描述 */
  description: string
  /** 供應商名稱 */
  supplier: string
  /** 供應商料號 */
  supplier_pn: string
  /** 用量 (替代料此欄位為空字串) */
  qty: string | number
  /** 位置標號 (替代料此欄位為空字串) */
  locations: string
  /** CCL 關鍵物料旗標 */
  ccl: boolean
  /** 備註說明 */
  remark: string
  /** 各機種料號選定對應表 (key: model_name, value: selected_pn) */
  selections: Record<string, string>
}

/**
 * BOM 上件面/製程視圖篩選選項值
 */
export type BOMViewType = 'all' | 'smd' | 'pth' | 'bottom' | 'ni' | 'proto' | 'mp' | 'ccl'

/**
 * BOM 視圖模式選項 (EBOM 單版模式 / Matrix 跨機種矩陣模式)
 */
export type BOMModeType = 'EBOM' | 'Matrix'

/**
 * 視圖下拉選單選項規格
 */
export interface ViewDropdownOption {
  label: string
  value: BOMViewType
}
