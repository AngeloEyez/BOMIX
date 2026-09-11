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
import { GetBOMView, type ViewPartGroup, type ViewRevision } from '../../../services/api'
import { useLogStore } from '../../../stores'
import type { BOMDisplayRow, BOMModeType, ViewDropdownOption } from '../types'
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

  // ── 視圖狀態 ────
  const selectedBomType = ref<BOMModeType>('EBOM')
  const selectedView = ref('all')
  const searchQuery = ref('')
  const sortField = ref('')
  const sortOrder = ref(1)

  // ── 後端原始資料與中繼資料 ────
  const aggregatedParts = shallowRef<ViewPartGroup[]>([])
  const currentRevisionMetadata = shallowRef<ViewRevision | null>(null)

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

    sortedAggregatedParts.value.forEach((part) => {
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

      // 1. 加入主料列 (Main Source Row)
      rows.push({
        rowId: `${parentKey}-main`,
        parentKey: parentKey,
        isSecondSource: false,
        hasSecondSources: hasSS,
        secondSourcesCount: ssCount,
        item: part.item || '',
        hhpn: part.hhpn || '',
        description: part.description || '',
        supplier: part.main_supplier || '',
        supplier_pn: part.main_supplier_pn || '',
        qty: part.qty ?? '',
        locations: part.locations || '',
        ccl: Boolean(part.ccl),
        remark: part.remark || '',
        selections: selectionsMap,
      })

      // 2. 加入 2nd 替代料列 (若未被收合，緊排於主料正下方)
      if (hasSS && part.second_sources && !collapseState.isCollapsed(parentKey)) {
        part.second_sources.forEach((ss, idx) => {
          rows.push({
            rowId: `${parentKey}-ss-${idx}-${ss.supplier_pn || idx}`,
            parentKey: parentKey,
            isSecondSource: true,
            hasSecondSources: false,
            secondSourcesCount: 0,
            item: '', // 替代料無 Item 號碼
            hhpn: ss.hhpn || '',
            description: ss.description || '',
            supplier: ss.supplier || '',
            supplier_pn: ss.supplier_pn || '',
            qty: '',
            locations: '',
            ccl: false,
            remark: ss.remark || '',
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
      return
    }

    try {
      const primaryId = revIds[0]
      const viewType = selectedView.value === 'all' ? '' : selectedView.value.toUpperCase()
      logStore.addLogEntry('DEBUG', `[View System] 準備建立 View: RevisionIDs=[${revIds.join(', ')}] (Primary ID: ${primaryId}), ViewType="${viewType || 'ALL'}"`)
      const result = await GetBOMView([primaryId], viewType)
      
      if (result && result.part_groups) {
        aggregatedParts.value = result.part_groups
        // 預設全部 2nd 替代料展開直接顯示於主料正下方
        collapseState.expandAll()
      } else {
        aggregatedParts.value = []
        collapseState.resetCollapse()
      }

      if (result && result.revisions && result.revisions.length > 0) {
        currentRevisionMetadata.value = result.revisions[0]
      } else {
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
   * 取得資料列 CSS Class (主料或替代料)
   */
  function getRowClass(data: BOMDisplayRow): string {
    return data.isSecondSource ? 'second-source-row' : 'main-source-row'
  }

  /**
   * 取得 CCL 樣式類別
   */
  function getCCLClass(ccl: boolean): string {
    return ccl ? 'ccl-critical' : 'ccl-normal'
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
    getRowClass,
    getCCLClass,
  }
}
