/**
 * @file useBOMNotesEditing.ts
 * @description BOM 表格 Notes 欄位行內編輯管理 Composable
 * 
 * 負責管理 Notes 欄位專用小編輯視窗之顯示狀態、定位矩形座標、資料雙向綁定、
 * 前端局部快取同步 (主料與替代料連動) 以及後端 API 資料庫持久化存取。
 */

import { ref } from 'vue'
import type { BOMDisplayRow } from '../types'
import type { CellRect } from './useCellHoverCard'
import { UpdateMaterialNote } from '../../../services/api'
import { useLogStore } from '../../../stores'

export interface UseBOMNotesEditingOptions {
  /** 懸停卡片關閉函式 */
  closeCard: (immediate?: boolean) => void
  /** 懸停卡片 Notes 儲存委託函式 */
  saveNotes: (callback: (row: BOMDisplayRow, notes: string) => Promise<void>) => void
  /** 前端物料快取 Notes 同步更新函式 */
  updateMaterialNotesInCache: (materialId: number, notes: string, supplier?: string, supplierPn?: string) => void
}

export function useBOMNotesEditing(options: UseBOMNotesEditingOptions) {
  const { closeCard, saveNotes, updateMaterialNotesInCache } = options

  const logStore = useLogStore()

  /** Notes 行內編輯視窗是否可見 */
  const isNotesEditorVisible = ref(false)
  /** 當前正處於編輯中的資料列 */
  const editingNotesRow = ref<BOMDisplayRow | null>(null)
  /** 觸發編輯單元格之螢幕定位矩形 */
  const notesEditorTargetRect = ref<CellRect | null>(null)
  /** 編輯初始 Notes 內容 */
  const initialNotesValue = ref('')

  /**
   * 判斷指定資料列是否正處於 Notes 編輯狀態
   * 
   * @param {BOMDisplayRow} row - 資料列物件
   * @returns {boolean} 是否為當前編輯列
   */
  function isEditingNotesCell(row: BOMDisplayRow): boolean {
    return isNotesEditorVisible.value && editingNotesRow.value?.rowId === row.rowId
  }

  /**
   * 點擊 Notes 儲存格開啟小編輯視窗
   * 
   * @param {MouseEvent} event - 點擊事件物件
   * @param {BOMDisplayRow} row - 當前儲存格所屬列資料
   */
  function handleNotesCellClick(event: MouseEvent, row: BOMDisplayRow): void {
    // 若已在編輯同一列，不重複處理
    if (isEditingNotesCell(row)) return

    // 關閉任何可能開啟中的懸停卡片
    closeCard(true)

    const currentTarget = event.currentTarget as HTMLElement
    if (!currentTarget) return

    const r = currentTarget.getBoundingClientRect()
    notesEditorTargetRect.value = {
      top: r.top,
      bottom: r.bottom,
      left: r.left,
      right: r.right,
      width: r.width,
      height: r.height
    }

    editingNotesRow.value = row
    initialNotesValue.value = row.notes || ''
    isNotesEditorVisible.value = true
  }

  /**
   * 處理 Notes 小編輯視窗儲存事件
   * 
   * 1. 更新前端當前列之 notes
   * 2. 透過 updateMaterialNotesInCache 即時局部更新前端快取 (確保同一物料在整份 BOM 任何位置皆同步為最新資料)
   * 3. 呼叫後端 API UpdateMaterialNote 將變更持久化寫入資料庫
   * 4. 關閉編輯視窗
   * 
   * @param {string} newNotes - 使用者編輯後之 Notes 內容
   */
  async function handleNotesEditorSave(newNotes: string): Promise<void> {
    if (!editingNotesRow.value) {
      isNotesEditorVisible.value = false
      return
    }

    const row = editingNotesRow.value
    const trimmed = newNotes.trim()
    const oldNotes = (row.notes || '').trim()

    // 立即關閉小編輯視窗
    isNotesEditorVisible.value = false
    editingNotesRow.value = null
    notesEditorTargetRect.value = null

    // 若內容未發生變動，無需執行後續更新
    if (trimmed === oldNotes) {
      return
    }

    // 1. 立即更新當前列
    row.notes = trimmed

    // 2. 即時局部更新前端快取 (主料與替代料全面同步最新資料)
    updateMaterialNotesInCache(row.materialId, trimmed, row.supplier, row.supplier_pn)

    // 3. 呼叫後端 API 持久化寫入資料庫
    if (row.materialId > 0) {
      try {
        await UpdateMaterialNote(row.materialId, trimmed)
        logStore.addLogEntry(
          'INFO',
          `[BOMTable] 成功更新物料 (ID=${row.materialId}, ${row.supplier} ${row.supplier_pn}) 的 Notes 註記`
        )
      } catch (error) {
        const msg = error instanceof Error ? error.message : String(error)
        logStore.addLogEntry('ERROR', `[BOMTable] 更新物料 Notes 至資料庫失敗: ${msg}`)
      }
    }
  }

  /**
   * 取消 Notes 編輯並關閉視窗
   */
  function handleNotesEditorCancel(): void {
    isNotesEditorVisible.value = false
    editingNotesRow.value = null
    notesEditorTargetRect.value = null
  }

  /**
   * 處理 HoverCard 上的 Notes 儲存操作
   * 同樣更新本機模型、快取全域同步並持久化至資料庫
   * 
   * @param {string} _newNotes - 新編輯的 Notes 內容
   */
  function handleSaveNotes(_newNotes: string): void {
    saveNotes(async (row, notes) => {
      const trimmed = notes.trim()
      // 同步前端快取
      updateMaterialNotesInCache(row.materialId, trimmed, row.supplier, row.supplier_pn)

      // 持久化至資料庫
      if (row.materialId > 0) {
        try {
          await UpdateMaterialNote(row.materialId, trimmed)
          logStore.addLogEntry(
            'INFO',
            `[BOMTable] HoverCard 成功更新物料 (ID=${row.materialId}) 的 Notes`
          )
        } catch (error) {
          const msg = error instanceof Error ? error.message : String(error)
          logStore.addLogEntry('ERROR', `[BOMTable] HoverCard 更新物料 Notes 失敗: ${msg}`)
        }
      }
    })
  }

  return {
    isNotesEditorVisible,
    editingNotesRow,
    notesEditorTargetRect,
    initialNotesValue,
    isEditingNotesCell,
    handleNotesCellClick,
    handleNotesEditorSave,
    handleNotesEditorCancel,
    handleSaveNotes,
  }
}
