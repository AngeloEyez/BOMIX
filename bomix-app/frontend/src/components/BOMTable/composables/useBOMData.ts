/**
 * @file useBOMData.ts
 * @description BOM 資料載入、過濾、排序與平鋪資料流管理 (Composable)
 * 
 * 本模組為 BOMTable 的核心資料引擎，負責：
 * 1. 監聽 props.revisionIds 變化，透過 Wails API (GetBOMView) 獲取後端資料與機種中繼資訊。
 * 2. 處理搜尋關鍵字過濾 (同時比對主料與所屬 2nd 替代料)。
 * 3. 呼叫 `sortBOMPartGroups` 進行群組層級之自然數字排序。
 * 4. 將階層式 `ViewPartGroup[]` 動態平鋪展平為虛擬滾動專用的 `BOMDisplayRow[]`，
 *    並與 `useCollapseState` 聯動，將展開之替代料精準插入於對應主料正下方。
 * 5. 提供各 Model 欄位料號對映、用量統計等輔助函式。
 * 
 * 導出函式：
 * - useBOMData: 建立並管理 BOM 核心資料流之 Composable
 * 
 * 依賴模組：
 * - Vue 3 (ref, shallowRef, computed, watch, type Ref)
 * - services/api (GetBOMView, ViewPartGroup, ViewRevision)
 * - stores (useLogStore)
 * - ../types (BOMDisplayRow, BOMViewType, BOMModeType, ViewDropdownOption)
 * - ../utils/sort (sortBOMPartGroups)
 * - ./useCollapseState (useCollapseState)
 */

import { ref, shallowRef, computed, watch, type Ref } from 'vue'
import type { DataTableSortEvent } from 'primevue/datatable'
import { GetBOMView, SetMatrixSelection, SetMatrixModelSelection, type ViewPartGroup, type ViewRevision } from '../../../services/api'
import { useLogStore, useAppStore, useBOMTableStore } from '../../../stores'
import type { BOMDisplayRow, BOMModeType, RevisionColumnInfo, MatrixModelColumnInfo, ViewDropdownOption } from '../types'
import { sortBOMPartGroups } from '../utils/sort'
import { useCollapseState } from './useCollapseState'
import { measureTextWidth, measureProjectCodeWidth } from '../utils/textMeasure'
import { getViewFilterInfo, formatViewFilterLog } from '../utils/viewConditions'

/** 視圖下拉選單選項規格常數 */
export const VIEW_OPTIONS: ViewDropdownOption[] = [
  { label: 'All', value: 'all' },
  { label: 'SMD', value: 'smd' },
  { label: 'PTH', value: 'pth' },
  { label: 'Bottom', value: 'bottom' },
  { label: 'NI', value: 'ni' },
  { label: 'PROTO', value: 'proto' },
  { label: 'MP', value: 'mp' },
  { label: 'CCL', value: 'ccl' },
]

/** BOM 視圖模式選項常數 */
export const BOM_TYPE_OPTIONS: BOMModeType[] = ['EBOM', 'Matrix']

interface UseBOMDataOptions {
  /** 外部傳入之 BOM Revision ID 陣列響應式參照 */
  revisionIds: Ref<number[] | undefined>
}

/**
 * 建立 BOM 核心資料流控制器
 * 
 * @param {UseBOMDataOptions} options - 配置選項物件
 */
export function useBOMData(options: UseBOMDataOptions) {
  const { revisionIds } = options
  const logStore = useLogStore()
  const appStore = useAppStore()
  const bomTableStore = useBOMTableStore()

  // ── 視圖狀態 (優先自 Pinia 記憶體快取載入，避免切換分頁後遺失使用者設定) ────
  const selectedBomType = ref<BOMModeType>(bomTableStore.bomType)
  const selectedView = ref(bomTableStore.view)
  const searchQuery = ref(bomTableStore.searchQuery)
  // CCL Only 狀態 (自 Pinia 快取還原，切換頁面時可保持之前狀態)
  const cclOnly = ref<boolean>(bomTableStore.cclOnly)
  const sortField = ref(bomTableStore.sortField)
  const sortOrder = ref(bomTableStore.sortOrder)

  // 監聽模式、視圖、搜尋關鍵字與 CCL Only 變更，即時同步寫入快取
  watch(selectedBomType, (newType) => {
    bomTableStore.setBomType(newType)
  })
  watch(selectedView, (newView) => {
    bomTableStore.setView(newView)
  })
  watch(searchQuery, (newQuery) => {
    bomTableStore.setSearchQuery(newQuery)
  })
  watch(cclOnly, (newVal) => {
    bomTableStore.setCclOnly(newVal)
  })

  // ── 後端原始資料與中繼資料 ────
  const aggregatedParts = shallowRef<ViewPartGroup[]>([])
  const currentRevisionMetadata = shallowRef<ViewRevision | null>(null)
  const allRevisionMetadata = shallowRef<ViewRevision[]>([])

  // ── 替代料展開/收合狀態管理器 ────
  const collapseState = useCollapseState(aggregatedParts)

  /** 當前主顯示的 Revision ID (多選時以第 1 個為主) */
  const currentRevisionId = computed(() => {
    if (revisionIds.value && revisionIds.value.length > 0) {
      return revisionIds.value[0]
    }
    return 0
  })

  /** SMD 物料筆數統計 */
  const smdPartsCount = computed(() => {
    return aggregatedParts.value.filter(p => p.type === 'SMD').length
  })

  /** PTH 物料筆數統計 */
  const pthPartsCount = computed(() => {
    return aggregatedParts.value.filter(p => p.type === 'PTH').length
  })

  /** 當前版本所包含之機種 (Model) 名稱清單 */
  const currentRevisionModels = computed(() => {
    if (!currentRevisionMetadata.value || !currentRevisionMetadata.value.model_names) return []
    return currentRevisionMetadata.value.model_names
  })

  /**
   * 參與視圖之 Revision 欄位清單 (依 projectExportOrder 排序)
   */
  const revisionColumns = computed<RevisionColumnInfo[]>(() => {
    if (!allRevisionMetadata.value || allRevisionMetadata.value.length === 0) return []

    const cols: RevisionColumnInfo[] = allRevisionMetadata.value.map(r => {
      const pCodeWidth = measureProjectCodeWidth(r.project_code || '')
      const phaseVersionText = [r.phase, r.version].filter(Boolean).join(' ')
      const pPhaseWidth = measureTextWidth(phaseVersionText)
      // 依據專案代號與階段版號最大寬度 + 8px (左右 padding 4px + 4px 安全呼吸邊距)，保底 54px
      const colWidth = Math.max(54, Math.ceil(Math.max(pCodeWidth, pPhaseWidth) + 8))

      return {
        revisionId: r.id,
        projectCode: r.project_code || '',
        phase: r.phase || '',
        version: r.version || '',
        modelNames: r.model_names || [],
        modelQty: (r.model_qty as Record<string, number>) || {},
        modelQtyByOrder: (r.model_qty_by_order as Record<number, number>) || {},
        models: (r.models as any) || [],
        columnWidth: colWidth,
      }
    })

    const order = appStore.seriesInfo?.projectExportOrder || []
    if (order.length > 0) {
      cols.sort((a, b) => {
        const idxA = order.indexOf(a.projectCode)
        const idxB = order.indexOf(b.projectCode)
        const rankA = idxA === -1 ? 9999 : idxA
        const rankB = idxB === -1 ? 9999 : idxB
        if (rankA !== rankB) {
          return rankA - rankB
        }
        return a.revisionId - b.revisionId
      })
    }

    return cols
  })

  /**
   * 將 0-based 數字索引轉換為 Excel 欄位字母序號 (0 -> A, 1 -> B, 25 -> Z, 26 -> AA)
   * 
   * @param {number} index - 0-based 索引
   * @returns {string} 字母序號
   */
  function getModelOrderAlias(index: number): string {
    if (index < 0) return 'A'
    let name = ''
    let num = index
    while (num >= 0) {
      name = String.fromCharCode(65 + (num % 26)) + name
      num = Math.floor(num / 26) - 1
    }
    return name
  }

  /**
   * 依據 BigMatrix 原則解析第 sortOrder 個 Model 的打件數量 Qty
   * 
   * @param {RevisionColumnInfo} rev - Revision 欄位資訊
   * @param {number} sortOrder - Model 排序索引
   * @param {string} modelAlias - 純字母代號 (A, B, C...)
   * @returns {number} 打件數量 (若無設定則回傳 0)
   */
  function resolveRevisionModelQty(rev: RevisionColumnInfo, sortOrder: number, modelAlias: string): number {
    if (rev.models && rev.models.length > 0) {
      const item = rev.models.find(m => m.sort_order === sortOrder)
      if (item && item.qty > 0) return item.qty
    }
    if (rev.modelQtyByOrder && rev.modelQtyByOrder[sortOrder] !== undefined && rev.modelQtyByOrder[sortOrder] > 0) {
      return rev.modelQtyByOrder[sortOrder]
    }
    if (rev.modelQty) {
      if (rev.modelQty[modelAlias] !== undefined && rev.modelQty[modelAlias] > 0) {
        return rev.modelQty[modelAlias]
      }
      const nameWithModel = `Model ${modelAlias}`
      if (rev.modelQty[nameWithModel] !== undefined && rev.modelQty[nameWithModel] > 0) {
        return rev.modelQty[nameWithModel]
      }
      const nameWithNum = `Model ${sortOrder + 1}`
      if (rev.modelQty[nameWithNum] !== undefined && rev.modelQty[nameWithNum] > 0) {
        return rev.modelQty[nameWithNum]
      }
      if (rev.modelNames && sortOrder < rev.modelNames.length) {
        const customName = rev.modelNames[sortOrder]
        if (rev.modelQty[customName] !== undefined && rev.modelQty[customName] > 0) {
          return rev.modelQty[customName]
        }
      }
    }
    return 0
  }

  /**
   * 參與 Matrix 視圖之動態 Model 欄位清單 (依據 BigMatrix 匯出原則動態展開)
   * 
   * 展開原則：
   * 1. 遍歷 revisionColumns 中已排序的 Revisions。
   * 2. 計算每個 Revision 實際包含的 Model 數量：
   *    - rev.models 的長度或最大 sort_order + 1
   *    - rev.modelNames 的長度
   *    - rev.modelQtyByOrder 的長度或最大 key + 1
   *    - rev.modelQty 的數量
   *    - aggregatedParts 中該 Revision 之 selections 的最大 sort_order + 1
   *    - 系列專案儲存之自訂 Model 數量 (appStore.seriesInfo.projectModelCounts)
   *    - 保底至少 1 個 Model
   * 3. 依序產生 MatrixModelColumnInfo，包含純字母代號與打件數量。
   */
  const matrixModelColumns = computed<MatrixModelColumnInfo[]>(() => {
    if (!revisionColumns.value || revisionColumns.value.length === 0) return []

    const cols: MatrixModelColumnInfo[] = []

    for (let revIdx = 0; revIdx < revisionColumns.value.length; revIdx++) {
      const rev = revisionColumns.value[revIdx]
      // 判斷是否為整個 Matrix 表格的第一個 Revision
      const isFirstRevision = revIdx === 0

      // 1. 計算該 Revision 實際存在的 Model 數量
      let count = 0

      if (rev.models && rev.models.length > 0) {
        count = Math.max(count, rev.models.length)
        for (const m of rev.models) {
          if (m.sort_order + 1 > count) {
            count = m.sort_order + 1
          }
        }
      }

      if (rev.modelNames && rev.modelNames.length > 0) {
        count = Math.max(count, rev.modelNames.length)
      }

      if (rev.modelQtyByOrder) {
        const orderKeys = Object.keys(rev.modelQtyByOrder).map(Number)
        count = Math.max(count, orderKeys.length)
        for (const k of orderKeys) {
          if (k + 1 > count) {
            count = k + 1
          }
        }
      }

      if (rev.modelQty) {
        count = Math.max(count, Object.keys(rev.modelQty).length)
      }

      // 檢查物料在該 Revision 中的 Selection 最大 SortOrder
      if (aggregatedParts.value && aggregatedParts.value.length > 0) {
        for (const p of aggregatedParts.value) {
          if (p.selections) {
            for (const sel of p.selections) {
              if (sel.revision_id === rev.revisionId && (sel.selected_material_id || sel.selected_pn)) {
                if (sel.sort_order + 1 > count) {
                  count = sel.sort_order + 1
                }
              }
            }
          }
        }
      }

      // 檢查是否有儲存的自訂 Model 數量紀錄
      const savedCount = appStore.seriesInfo?.projectModelCounts?.[rev.projectCode]
      if (savedCount && savedCount > count) {
        count = savedCount
      }

      // 保底原則：若該 Revision 存在，預設至少提供 1 個 Model 欄位
      if (count <= 0) {
        count = 1
      }

      // 2. 智慧中心定位演算法與自適應欄寬計算
      // 演算法規則：
      // 1 個 model: 位置 1 (index 0)
      // 2 個 model: 位置 1 (index 0)
      // 3 個 model: 位置 2 (index 1，正中央)
      // 4 個 model: 位置 2 (index 1)
      // 5 個 model: 位置 3 (index 2，正中央)
      // 通用公式：Math.floor((count - 1) / 2)
      const centerTargetIndex = Math.floor((count - 1) / 2)

      // 首欄特殊補償：僅整個 Matrix 表格第 0 欄擁有 project-start-col (2px 粗左框線)，額外補償 +2px 精確抵銷，其餘欄位維持極限最小寬度
      const startBorderCompensation = isFirstRevision ? 2 : 0

      // 欄寬自適應計算 (極致緊湊：最小化兩側間距，最大化橫向可視空間)：
      let colWidth = 54
      if (count === 1) {
        // 單 Model 狀態：依專案代碼精準像素寬度 + 左右各 3px 最小視覺舒適間距 (合計 +6px)
        // 若為首欄，補償 project-start-col 之 2px 左邊框 (保底 56px，計算 +8px)
        const textWidth = measureProjectCodeWidth(rev.projectCode || '')
        colWidth = Math.max(54 + startBorderCompensation, Math.ceil(textWidth + 6 + startBorderCompensation))
      } else {
        // 多 Model 狀態：統一緊湊 54px，點擊區域與字母用量標籤比例極佳
        colWidth = 54
      }

      // 3. 展開各 Model 欄位
      for (let i = 0; i < count; i++) {
        const alias = getModelOrderAlias(i)
        const qty = resolveRevisionModelQty(rev, i, alias)

        let modelId = 0
        let modelName = `Model ${alias}`
        if (rev.models && rev.models.length > 0) {
          const item = rev.models.find(m => m.sort_order === i)
          if (item) {
            modelId = item.id || 0
            if (item.model_name) {
              modelName = item.model_name
            }
          }
        }

        const qtyStr = qty > 0 ? `(${qty})` : ''
        const headerTitle = [rev.projectCode, rev.phase, rev.version].filter(Boolean).join(' ')

        // 若為多 Model 狀態，且為整個表格之第 0 欄 (帶有 2px 左邊框)，亦補償 2px 保持內容區域對齊
        const actualColWidth = (count > 1 && isFirstRevision && i === 0) ? (colWidth + startBorderCompensation) : colWidth

        cols.push({
          key: `matrix-rev-${rev.revisionId}-model-${i}`,
          revisionId: rev.revisionId,
          projectCode: rev.projectCode,
          phase: rev.phase,
          version: rev.version,
          sortOrder: i,
          modelId: modelId,
          modelAlias: alias,
          modelName: modelName,
          qty: qty,
          headerTitle: headerTitle,
          isFirstInRevision: i === 0,
          isLastInRevision: i === count - 1,
          revisionModelCount: count,
          showProjectCode: i === centerTargetIndex,
          columnWidth: actualColWidth,
        })
      }
    }

    return cols
  })

  /**
   * 檢查物料群組在任何 Revision 的任何 Model 中是否具有選中記錄 (Matrix Selection)
   * 
   * 判定範圍包含：
   * 1. part.selections 陣列中存在 selected_material_id > 0 或 selected_pn / selected_material
   * 2. part.main_selections_by_order 中存在 true
   * 3. part.second_sources[].selections_by_order 中存在 true
   * 
   * @param {ViewPartGroup} part - 物料群組物件
   * @returns {boolean} 是否包含任何勾選選中項目
   */
  function hasAnyMatrixSelection(part: ViewPartGroup): boolean {
    // 1. 檢查 part.selections 陣列 (跨 revision x model 的選中紀錄)
    if (part.selections && part.selections.length > 0) {
      const hasSel = part.selections.some(s => 
        (s.selected_material_id !== undefined && s.selected_material_id > 0) || 
        Boolean(s.selected_pn) || 
        Boolean(s.selected_material)
      )
      if (hasSel) return true
    }

    // 2. 檢查 main_selections_by_order (主料在各 Model 的勾選狀態)
    if (part.main_selections_by_order) {
      const hasMainSel = Object.values(part.main_selections_by_order).some(Boolean)
      if (hasMainSel) return true
    }

    // 3. 檢查 second_sources 中是否有替代料被勾選
    if (part.second_sources && part.second_sources.length > 0) {
      for (const ss of part.second_sources) {
        if (ss.selections_by_order) {
          const hasSSSel = Object.values(ss.selections_by_order).some(Boolean)
          if (hasSSSel) return true
        }
      }
    }

    return false
  }

  /**
   * 依據搜尋關鍵字與 CCL Only 過濾並進行自然數字排序後之物料群組
   */
  const sortedAggregatedParts = computed<ViewPartGroup[]>(() => {
    if (!aggregatedParts.value || aggregatedParts.value.length === 0) return []

    let list = [...aggregatedParts.value]

    // 關鍵字搜尋過濾 (同時檢查主料與所有替代料)
    if (searchQuery.value) {
      const q = searchQuery.value.toLowerCase()
      list = list.filter((part: any) => {
        const mainMatch = 
          (part.hhpn && part.hhpn.toLowerCase().includes(q)) ||
          (part.description && part.description.toLowerCase().includes(q)) ||
          (part.main_supplier && part.main_supplier.toLowerCase().includes(q)) ||
          (part.main_supplier_pn && part.main_supplier_pn.toLowerCase().includes(q)) ||
          (part.locations && part.locations.toLowerCase().includes(q)) ||
          (part.remark && part.remark.toLowerCase().includes(q))

        if (mainMatch) return true

        // 檢查替代料 (任一替代料符合，保留該整組主料)
        if (part.second_sources && part.second_sources.length > 0) {
          return part.second_sources.some((ss: any) => 
            (ss.hhpn && ss.hhpn.toLowerCase().includes(q)) ||
            (ss.description && ss.description.toLowerCase().includes(q)) ||
            (ss.supplier && ss.supplier.toLowerCase().includes(q)) ||
            (ss.supplier_pn && ss.supplier_pn.toLowerCase().includes(q)) ||
            (ss.remark && ss.remark.toLowerCase().includes(q))
          )
        }

        return false
      })
    }

    // CCL Only 按鈕過濾 (以物料群組 Group 為單位進行過濾)
    if (cclOnly.value) {
      list = list.filter((part: ViewPartGroup) => {
        // 規則 2: 在 Matrix 模式下，只要該 group 有任何 revision 有任何 selection，就忽略 CCL only 按鈕，一律顯示
        if (selectedBomType.value === 'Matrix' && hasAnyMatrixSelection(part)) {
          return true
        }
        // 規則 1: 僅保留 CCL 為 true 的物料群組
        return Boolean(part.ccl)
      })
    }

    // 執行群組層級之自然數字排序
    return sortBOMPartGroups(list, sortField.value, sortOrder.value)
  })

  /** 過濾後之主要物料群組總筆數 */
  const filteredPartsCount = computed(() => sortedAggregatedParts.value.length)

  /**
   * 平鋪展平後之單列資料清單 (供 DataTable 虛擬滾動使用)
   */
  const displayRows = computed<BOMDisplayRow[]>(() => {
    const rows: BOMDisplayRow[] = []

    sortedAggregatedParts.value.forEach((part, groupIndex) => {
      const parentKey = collapseState.getPartKey(part)
      const hasSS = Boolean(part.second_sources && part.second_sources.length > 0)
      const ssCount = part.second_sources ? part.second_sources.length : 0

      // 建置機種料號選定對應表
      const selectionsMap: Record<string, string> = {}
      if (part.selections) {
        part.selections.forEach(sel => {
          if (sel.model_name && sel.selected_pn) {
            selectionsMap[sel.model_name] = sel.selected_pn
          }
        })
      }

      // 解析 QtyByRevision
      const qtyByRev: Record<number, number> = {}
      if (part.qty_by_revision) {
        Object.entries(part.qty_by_revision).forEach(([k, v]) => {
          qtyByRev[Number(k)] = Number(v) || 0
        })
      }

      // 建立 Model SortOrder 勾選映射
      const mainSelectionsByOrder: Record<number, boolean> = {}
      if (part.main_selections_by_order) {
        Object.entries(part.main_selections_by_order).forEach(([k, v]) => {
          mainSelectionsByOrder[Number(k)] = Boolean(v)
        })
      }

      // 當群組中的 location 在每個 revision 中 bom_status 都為 P 時，part.bom_status 為 'P'
      const isProto = (part.bom_status || '').toUpperCase() === 'P'

      // 1. 加入主料列 (Main Source Row)
      rows.push({
        rowId: `${parentKey}-main`,
        parentKey: parentKey,
        groupIndex: groupIndex,
        mainMaterialId: part.material_id || 0,
        materialId: part.material_id || 0,
        isSecondSource: false,
        hasSecondSources: hasSS,
        secondSourcesCount: ssCount,
        item: part.item || '',
        hhpn: part.hhpn || '',
        description: part.description || '',
        supplier: part.main_supplier || '',
        supplier_pn: part.main_supplier_pn || '',
        qty: part.qty ?? '',
        qtyByRevision: qtyByRev,
        locations: part.locations || '',
        ccl: Boolean(part.ccl),
        remark: part.remark || '',
        notes: part.notes || '',
        sourceRevisionIds: part.source_revision_ids || [],
        selectionsByOrder: {},
        mainSelectionsByOrder: mainSelectionsByOrder,
        selections: selectionsMap,
        bomStatus: part.bom_status || 'I',
        isProto: isProto,
      })

      // 2. 加入 2nd 替代料列 (若未被收合，緊排於主料正下方)
      if (hasSS && part.second_sources && !collapseState.isCollapsed(parentKey)) {
        part.second_sources.forEach((ss, idx) => {
          const ssQtyByRev: Record<number, number> = {}
          if (ss.qty_by_revision) {
            Object.entries(ss.qty_by_revision).forEach(([k, v]) => {
              ssQtyByRev[Number(k)] = Number(v) || 0
            })
          }
          const ssSelectionsByOrder: Record<number, boolean> = {}
          if (ss.selections_by_order) {
            Object.entries(ss.selections_by_order).forEach(([k, v]) => {
              ssSelectionsByOrder[Number(k)] = Boolean(v)
            })
          }

          rows.push({
            rowId: `${parentKey}-ss-${idx}-${ss.supplier_pn || idx}`,
            parentKey: parentKey,
            groupIndex: groupIndex,
            mainMaterialId: part.material_id || 0,
            materialId: ss.material_id || 0,
            isSecondSource: true,
            hasSecondSources: false,
            secondSourcesCount: 0,
            item: '', // 替代料無 Item 號碼
            hhpn: ss.hhpn || '',
            description: ss.description || '',
            supplier: ss.supplier || '',
            supplier_pn: ss.supplier_pn || '',
            qty: '', // 在 EBOM 模式下，動態 Qty 欄位取用 qtyByRevision[revId]
            qtyByRevision: ssQtyByRev,
            locations: '',
            ccl: false,
            remark: ss.remark || '',
            notes: ss.notes || '',
            sourceRevisionIds: ss.source_revision_ids || [],
            selectionsByOrder: ssSelectionsByOrder,
            mainSelectionsByOrder: {},
            selections: selectionsMap,
            bomStatus: part.bom_status || 'I',
            isProto: false,
          })
        })
      }
    })

    return rows
  })

  /**
   * 表頭排序事件處理
   * @param {DataTableSortEvent} event - DataTable 排序事件物件
   */
  function onSort(event: DataTableSortEvent): void {
    if (typeof event.sortField === 'string') {
      sortField.value = event.sortField
      sortOrder.value = event.sortOrder ?? 1
      bomTableStore.setSort(sortField.value, sortOrder.value)
    }
  }

  /**
   * 載入指定 BOM Revision 清單的物料群組視圖資料
   * @param {number[]} revIds - BOM Revision ID 陣列
   * @param {boolean} [force=false] - 是否強制重新自資料庫查詢 (預設 false，若條件未變則直接沿用記憶體快取)
   */
  async function loadBOMData(revIds: number[], force: boolean = false): Promise<void> {
    if (!revIds || revIds.length === 0) {
      aggregatedParts.value = []
      collapseState.resetCollapse()
      currentRevisionMetadata.value = null
      allRevisionMetadata.value = []
      bomTableStore.clearDataCache()
      return
    }

    const revKey = [...revIds].sort((a, b) => a - b).join(',')
    const viewType = (selectedView.value || 'all').toUpperCase()

    // 1. 若非強制重查，且記憶體快取中已有相同查詢條件 (revisions, view) 的資料，直接同步還原快取 (0ms 延遲、無資料庫負擔)
    if (!force && bomTableStore.hasDataCache(revKey, viewType)) {
      logStore.addLogEntry('DEBUG', `[BOM Cache Hit] 查詢條件未變動，直接沿用記憶體快取資料: RevisionKey="${revKey}", ViewType="${viewType}"`)
      // 採用新陣列賦值以確保觸發 Vue shallowRef 響應式依賴更新
      aggregatedParts.value = [...bomTableStore.cachedPartGroups]
      allRevisionMetadata.value = [...bomTableStore.cachedRevisions]
      currentRevisionMetadata.value = bomTableStore.cachedRevisions[0] || null
      // 確保替代料展開狀態正確初始化
      collapseState.expandAll()
      return
    }

    // 2. 快取未命中或指定強制重查時，向後端資料庫發送 GetBOMView 查詢
    try {
      const currentMode = currentRevisionMetadata.value?.phase?.toUpperCase().includes('MP') ? 'MP' : 'NPI'
      const filterInfo = getViewFilterInfo(selectedView.value, currentMode)
      logStore.addLogEntry('DEBUG', `[View System] 自資料庫查詢 View: RevisionIDs=[${revIds.join(', ')}], ViewType="${viewType}" | 過濾條件: ${filterInfo.summary}`)
      const result = await GetBOMView(revIds, viewType)
      
      if (result && result.part_groups) {
        aggregatedParts.value = result.part_groups
        // 預設全部 2nd 替代料展開直接顯示於主料正下方
        collapseState.expandAll()
      } else {
        aggregatedParts.value = []
        collapseState.resetCollapse()
      }

      if (result && result.revisions && result.revisions.length > 0) {
        allRevisionMetadata.value = result.revisions
        currentRevisionMetadata.value = result.revisions[0]
      } else {
        allRevisionMetadata.value = []
        currentRevisionMetadata.value = null
      }

      // 成功查詢後寫入快取 (僅在有資料或正常查詢完成時寫入)
      bomTableStore.setDataCache(
        revKey,
        viewType,
        aggregatedParts.value,
        allRevisionMetadata.value
      )
    } catch (error) {
      const msg = error instanceof Error ? error.message : String(error)
      logStore.addLogEntry('ERROR', `載入 BOM 資料失敗: ${msg}`)
    }
  }

  /**
   * 視圖類別 (All, SMD, PTH...) 變更處理
   * 當使用者選定一個 View 設定時，輸出該 View 的過濾條件 Debug Log 並強制自後端查詢
   */
  function onViewChange(): void {
    const currentMode = currentRevisionMetadata.value?.phase?.toUpperCase().includes('MP') ? 'MP' : 'NPI'
    const logMsg = formatViewFilterLog(selectedView.value, currentMode)
    logStore.addLogEntry('DEBUG', logMsg)
    console.debug(logMsg)

    collapseState.resetCollapse()
    bomTableStore.resetScroll()
    if (revisionIds.value && revisionIds.value.length > 0) {
      // 使用者手動切換 View 屬於明確查詢操作，強制重新向資料庫獲取資料
      loadBOMData(revisionIds.value, true)
    }
  }

  /**
   * 取得指定機種之總用量
   * @param {string} modelName - 機種名稱
   */
  function getModelQty(modelName: string): number {
    if (!currentRevisionMetadata.value || !currentRevisionMetadata.value.model_qty) return 0
    return currentRevisionMetadata.value.model_qty[modelName] || 0
  }

  /**
   * 取得指定列在該機種下被選取的料號
   * 僅當選擇的料號與此 Row 的 supplier_pn 相符時才回傳料號
   */
  function getModelSelectedPN(row: BOMDisplayRow, modelName: string): string {
    const selectedPN = row.selections[modelName]
    if (!selectedPN) return ''
    return row.supplier_pn === selectedPN ? selectedPN : ''
  }

  /**
   * 判斷此列料號是否在該機種下被選用
   */
  function isModelSelected(row: BOMDisplayRow, modelName: string): boolean {
    return getModelSelectedPN(row, modelName) !== ''
  }

  /**
   * 判斷指定列在該 Revision 中是否被選中 (for Matrix 模式)
   * @param {BOMDisplayRow} row - 資料列
   * @param {RevisionColumnInfo} revCol - Revision 欄位資訊
   */
  function isSelectedInRevision(row: BOMDisplayRow, revCol: RevisionColumnInfo): boolean {
    const part = aggregatedParts.value.find(p => p.material_id === row.mainMaterialId || (p.main_supplier === row.supplier && p.main_supplier_pn === row.supplier_pn))
    if (!part || !part.selections) return false
    
    const revSel = part.selections.find(s => s.revision_id === revCol.revisionId)
    if (!revSel) return false
    
    if (revSel.selected_material_id && row.materialId) {
      return revSel.selected_material_id === row.materialId
    }
    if (revSel.selected_pn) {
      return revSel.selected_pn === row.supplier_pn
    }
    return false
  }

  /**
   * 判斷指定列在該 Revision 中是否存在 (若不存在則 disabled)
   * @param {BOMDisplayRow} row - 資料列
   * @param {RevisionColumnInfo} revCol - Revision 欄位資訊
   */
  function isAvailableInRevision(row: BOMDisplayRow, revCol: RevisionColumnInfo): boolean {
    if (!row.sourceRevisionIds || row.sourceRevisionIds.length === 0) {
      return false
    }
    return row.sourceRevisionIds.includes(revCol.revisionId)
  }

  /**
   * 處理 Matrix 模式 Checkbox 勾選變更
   * 互斥勾選：若目前已勾選則取消勾選；若未勾選則勾選此物料（後端自動覆蓋同組舊有勾選）
   * @param {BOMDisplayRow} row - 資料列
   * @param {RevisionColumnInfo} revCol - Revision 欄位資訊
   */
  async function onMatrixSelectionChange(row: BOMDisplayRow, revCol: RevisionColumnInfo): Promise<void> {
    try {
      const isCurrentlySelected = isSelectedInRevision(row, revCol)
      const targetSelectedMaterialId = isCurrentlySelected ? 0 : row.materialId

      // 取得 modelID (若無則傳 0 由後端自動配對或建立)
      const part = aggregatedParts.value.find(p => p.material_id === row.mainMaterialId || (p.main_supplier === row.supplier && p.main_supplier_pn === row.supplier_pn))
      const revSel = part?.selections?.find(s => s.revision_id === revCol.revisionId)
      const modelId = revSel?.model_id || 0

      logStore.addLogEntry(
        'DEBUG',
        `[Matrix Selection] 變更選取: revisionID=${revCol.revisionId}, modelID=${modelId}, mainMatID=${row.mainMaterialId}, selectedMatID=${targetSelectedMaterialId}`
      )

      await SetMatrixSelection(revCol.revisionId, modelId, row.mainMaterialId, targetSelectedMaterialId)

      // 立即重新載入資料以反映最新的勾選狀態
      if (revisionIds.value && revisionIds.value.length > 0) {
        await loadBOMData(revisionIds.value)
      }
    } catch (error) {
      const msg = error instanceof Error ? error.message : String(error)
      logStore.addLogEntry('ERROR', `更新 Matrix 勾選狀態失敗: ${msg}`)
    }
  }

  /**
   * 判斷指定列在該 Revision 的特定 Model 是否被選中
   * 
   * 比對優先順序：
   * 1. 匹配 part.selections 中 (revision_id, sort_order) 的紀錄
   * 2. 比對 selected_material_id 或 selected_material / selected_pn
   * 
   * @param {BOMDisplayRow} row - 資料列
   * @param {MatrixModelColumnInfo} modelCol - Matrix Model 欄位資訊
   * @returns {boolean} 是否被勾選
   */
  function isModelSelectedInRevision(row: BOMDisplayRow, modelCol: MatrixModelColumnInfo): boolean {
    const part = aggregatedParts.value.find(p => 
      p.material_id === row.mainMaterialId || 
      (p.main_supplier === row.supplier && p.main_supplier_pn === row.supplier_pn)
    )
    if (!part || !part.selections) return false

    const sel = part.selections.find(s => 
      s.revision_id === modelCol.revisionId && 
      s.sort_order === modelCol.sortOrder
    )
    if (!sel) return false

    if (sel.selected_material_id && row.materialId) {
      return sel.selected_material_id === row.materialId
    }
    if (sel.selected_material) {
      const rowMatKey = `${row.supplier}|${row.supplier_pn}`
      return sel.selected_material.toLowerCase() === rowMatKey.toLowerCase()
    }
    if (sel.selected_pn) {
      return sel.selected_pn.toLowerCase() === row.supplier_pn.toLowerCase()
    }
    return false
  }

  /**
   * 判斷指定列在該 Revision 中是否存在
   * 依據使用者需求：「若該物料在該revision不存在，則不繪製checkbox」
   * 
   * @param {BOMDisplayRow} row - 資料列
   * @param {MatrixModelColumnInfo} modelCol - Matrix Model 欄位資訊
   * @returns {boolean} 是否存在於該 Revision
   */
  function isModelAvailableInRevision(row: BOMDisplayRow, modelCol: MatrixModelColumnInfo): boolean {
    if (!row.sourceRevisionIds || row.sourceRevisionIds.length === 0) {
      return false
    }
    return row.sourceRevisionIds.includes(modelCol.revisionId)
  }

  /**
   * 處理 Matrix 模式 Checkbox 勾選變更
   * 互斥勾選：若目前已勾選則取消勾選；若未勾選則勾選此物料（後端自動覆蓋同組舊有勾選）
   * 依據 modelCol.sortOrder 精準操作特定 Model，避免誤綁第 0 個 Model
   * 
   * @param {BOMDisplayRow} row - 資料列
   * @param {MatrixModelColumnInfo} modelCol - Matrix Model 欄位資訊
   */
  async function onMatrixModelSelectionChange(row: BOMDisplayRow, modelCol: MatrixModelColumnInfo): Promise<void> {
    try {
      const isCurrentlySelected = isModelSelectedInRevision(row, modelCol)
      const targetSelectedMaterialId = isCurrentlySelected ? 0 : row.materialId

      logStore.addLogEntry(
        'DEBUG',
        `[Matrix Selection] 變更選取: revisionID=${modelCol.revisionId}, sortOrder=${modelCol.sortOrder}, Model=${modelCol.modelAlias}, mainMatID=${row.mainMaterialId}, selectedMatID=${targetSelectedMaterialId}`
      )

      await SetMatrixModelSelection(
        modelCol.revisionId,
        modelCol.sortOrder,
        row.mainMaterialId,
        targetSelectedMaterialId
      )

      // 強制自後端重新載入資料以反映最新的勾選狀態與持久化結果
      if (revisionIds.value && revisionIds.value.length > 0) {
        await loadBOMData(revisionIds.value, true)
      }
    } catch (error) {
      const msg = error instanceof Error ? error.message : String(error)
      logStore.addLogEntry('ERROR', `更新 Matrix 勾選狀態失敗: ${msg}`)
    }
  }

  /**
   * 取得資料列 CSS Class (群組斑馬紋底色類別與主/替代料類型)
   * 群組之間以底色交替區分 (group-even / group-odd)，主料與替代料以文字色彩區分
   * @param {BOMDisplayRow} data - 單列 BOM 資料
   * @returns {string} 組合後的 CSS Class 名稱
   */
  function getRowClass(data: BOMDisplayRow): string {
    const zebraClass = data.groupIndex % 2 === 0 ? 'group-even' : 'group-odd'
    const sourceClass = data.isSecondSource ? 'second-source-row' : 'main-source-row'
    const protoClass = (!data.isSecondSource && data.isProto) ? 'proto-row' : ''
    return [zebraClass, sourceClass, protoClass].filter(Boolean).join(' ')
  }

  /**
   * 取得 CCL 樣式類別
   */
  function getCCLClass(ccl: boolean): string {
    return ccl ? 'ccl-critical' : 'ccl-normal'
  }

  /**
   * 更新特定物料在前端快取中的 Notes 註記內容
   * 
   * 同步更新 aggregatedParts 中所有符合該 materialId（或 supplier+supplier_pn）
   * 的主料與替代料的 notes 欄位，並重新賦值觸發 displayRows 計算屬性響應更新，
   * 確保同一個物料在整份 BOM 所有出現的位置皆為最新內容。
   * 
   * @param {number} materialId - 全域物料 ID
   * @param {string} notes - 新的 Notes 內容
   * @param {string} [supplier] - 物料供應商名稱 (選填，輔助精確比對)
   * @param {string} [supplierPn] - 供應商料號 (選填，輔助精確比對)
   */
  function updateMaterialNotesInCache(
    materialId: number,
    notes: string,
    supplier?: string,
    supplierPn?: string
  ): void {
    if (!aggregatedParts.value || aggregatedParts.value.length === 0) return

    let hasChange = false
    const updatedParts = aggregatedParts.value.map(part => {
      let partModified = false
      let newPart = part

      // 檢查主料是否匹配
      const isMainMatch = (materialId > 0 && part.material_id === materialId) ||
        Boolean(supplier && supplierPn && part.main_supplier === supplier && part.main_supplier_pn === supplierPn)

      if (isMainMatch && part.notes !== notes) {
        newPart = { ...newPart, notes }
        partModified = true
      }

      // 檢查替代料清單是否匹配
      if (newPart.second_sources && newPart.second_sources.length > 0) {
        let ssModified = false
        const updatedSS = newPart.second_sources.map(ss => {
          const isSSMatch = (materialId > 0 && ss.material_id === materialId) ||
            Boolean(supplier && supplierPn && ss.supplier === supplier && ss.supplier_pn === supplierPn)

          if (isSSMatch && ss.notes !== notes) {
            ssModified = true
            return { ...ss, notes }
          }
          return ss
        })

        if (ssModified) {
          newPart = { ...newPart, second_sources: updatedSS }
          partModified = true
        }
      }

      if (partModified) {
        hasChange = true
        return newPart
      }
      return part
    })

    if (hasChange) {
      aggregatedParts.value = updatedParts
      // 同步更新記憶體快取中的物料清單，避免切換頁面後丟失剛修改之 Notes
      bomTableStore.updateCachedParts(updatedParts)
    }
  }

  // 監聽外部 revisionIds 變化
  watch(
    () => revisionIds.value,
    (newIds) => {
      if (newIds && newIds.length > 0) {
        const revKey = [...newIds].sort((a, b) => a - b).join(',')
        const isDifferentRevision = Boolean(bomTableStore.revisionKey && bomTableStore.revisionKey !== revKey)
        // 若使用者在專案樹切換了不同的 BOM 版本，清除舊資料快取、重置卷軸位置並重置 CCL Only 為預設不選中
        if (isDifferentRevision) {
          bomTableStore.clearDataCache()
          bomTableStore.resetScroll()
          cclOnly.value = false
          bomTableStore.setCclOnly(false)
        }
        bomTableStore.setRevisionKey(revKey)
        // 切換不同版本時強制向後端查詢；若為同版本 (例如頁面切換往返) 則允許使用快取
        loadBOMData(newIds, isDifferentRevision)
      } else {
        aggregatedParts.value = []
        collapseState.resetCollapse()
        currentRevisionMetadata.value = null
        allRevisionMetadata.value = []
        bomTableStore.clearDataCache()
        cclOnly.value = false
        bomTableStore.setCclOnly(false)
        bomTableStore.setRevisionKey('')
      }
    },
    { deep: true, immediate: true }
  )

  return {
    // 狀態
    selectedBomType,
    selectedView,
    searchQuery,
    cclOnly,
    sortField,
    sortOrder,
    // 快取 Store
    bomTableStore,
    // 資料
    aggregatedParts,
    sortedAggregatedParts,
    filteredPartsCount,
    displayRows,
    currentRevisionMetadata,
    allRevisionMetadata,
    revisionColumns,
    matrixModelColumns,
    currentRevisionModels,
    currentRevisionId,
    smdPartsCount,
    pthPartsCount,
    // 展開收合委派
    collapseState,
    // 方法
    onSort,
    loadBOMData,
    onViewChange,
    getModelQty,
    getModelSelectedPN,
    isModelSelected,
    isSelectedInRevision,
    isAvailableInRevision,
    onMatrixSelectionChange,
    isModelSelectedInRevision,
    isModelAvailableInRevision,
    onMatrixModelSelectionChange,
    getRowClass,
    getCCLClass,
    updateMaterialNotesInCache,
    // 視圖條件定義與日誌工具
    getViewFilterInfo,
    formatViewFilterLog,
  }
}
