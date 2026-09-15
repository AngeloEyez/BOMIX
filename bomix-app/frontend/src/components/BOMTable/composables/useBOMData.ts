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
import { GetBOMView, SetMatrixSelection, type ViewPartGroup, type ViewRevision } from '../../../services/api'
import { useLogStore, useAppStore } from '../../../stores'
import type { BOMDisplayRow, BOMModeType, RevisionColumnInfo, ViewDropdownOption } from '../types'
import { sortBOMPartGroups } from '../utils/sort'
import { useCollapseState } from './useCollapseState'

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

  // ── 視圖狀態 ────
  const selectedBomType = ref<BOMModeType>('EBOM')
  const selectedView = ref('all')
  const searchQuery = ref('')
  const sortField = ref('')
  const sortOrder = ref(1)

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

    const cols: RevisionColumnInfo[] = allRevisionMetadata.value.map(r => ({
      revisionId: r.id,
      projectCode: r.project_code || '',
      phase: r.phase || '',
      version: r.version || '',
      modelNames: r.model_names || [],
      modelQty: (r.model_qty as Record<string, number>) || {},
      modelQtyByOrder: (r.model_qty_by_order as Record<number, number>) || {},
    }))

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
   * 依據搜尋關鍵字過濾並進行自然數字排序後之物料群組
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

    // 執行群組層級之自然數字排序
    return sortBOMPartGroups(list, sortField.value, sortOrder.value)
  })

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
    }
  }

  /**
   * 載入指定 BOM Revision 清單的物料群組視圖資料
   * @param {number[]} revIds - BOM Revision ID 陣列
   */
  async function loadBOMData(revIds: number[]): Promise<void> {
    if (!revIds || revIds.length === 0) {
      aggregatedParts.value = []
      collapseState.resetCollapse()
      currentRevisionMetadata.value = null
      allRevisionMetadata.value = []
      return
    }

    try {
      const viewType = selectedView.value === 'all' ? '' : selectedView.value.toUpperCase()
      logStore.addLogEntry('DEBUG', `[View System] 準備建立 View: RevisionIDs=[${revIds.join(', ')}], ViewType="${viewType || 'ALL'}"`)
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
    } catch (error) {
      const msg = error instanceof Error ? error.message : String(error)
      logStore.addLogEntry('ERROR', `載入 BOM 資料失敗: ${msg}`)
    }
  }

  /**
   * 視圖類別 (All, SMD, PTH...) 變更處理
   */
  function onViewChange(): void {
    collapseState.resetCollapse()
    if (revisionIds.value && revisionIds.value.length > 0) {
      loadBOMData(revisionIds.value)
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
   * 取得資料列 CSS Class (群組斑馬紋底色類別與主/替代料類型)
   * 群組之間以底色交替區分 (group-even / group-odd)，主料與替代料以文字色彩區分
   * @param {BOMDisplayRow} data - 單列 BOM 資料
   * @returns {string} 組合後的 CSS Class 名稱
   */
  function getRowClass(data: BOMDisplayRow): string {
    const zebraClass = data.groupIndex % 2 === 0 ? 'group-even' : 'group-odd'
    const sourceClass = data.isSecondSource ? 'second-source-row' : 'main-source-row'
    return `${zebraClass} ${sourceClass}`
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
    }
  }

  // 監聽外部 revisionIds 變化
  watch(
    () => revisionIds.value,
    (newIds) => {
      if (newIds && newIds.length > 0) {
        loadBOMData(newIds)
      } else {
        aggregatedParts.value = []
        collapseState.resetCollapse()
        currentRevisionMetadata.value = null
        allRevisionMetadata.value = []
      }
    },
    { deep: true, immediate: true }
  )

  return {
    // 狀態
    selectedBomType,
    selectedView,
    searchQuery,
    sortField,
    sortOrder,
    // 資料
    aggregatedParts,
    sortedAggregatedParts,
    displayRows,
    currentRevisionMetadata,
    allRevisionMetadata,
    revisionColumns,
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
    getRowClass,
    getCCLClass,
    updateMaterialNotesInCache,
  }
}
