/**
 * @file useBOMContextMenu.ts
 * @description BOM 表格右鍵快顯選單與鍵盤快捷鍵管理 (Composable)
 * 
 * 本模組負責管理使用者於表格列上的滑鼠右鍵點擊快顯選單 (ContextMenu) 以及全域鍵盤快捷鍵：
 * 1. 提供料號、規格描述、供應商料號、以及整列 TSV (可直接貼入 Excel) 之複製操作。
 * 2. 提供「依此料號篩選」與替代料展開/收合之快捷操作。
 * 3. 攔截 Ctrl+C (或 Cmd+C)：當使用者未反白選取文字時，自動將當前點選的整列物料資料以 TSV 複製至剪貼簿。
 * 
 * 導出函式：
 * - useBOMContextMenu: 建立並管理右鍵快顯選單之 Composable
 * 
 * 依賴模組：
 * - Vue 3 (ref, computed, onMounted, onUnmounted, type Ref)
 * - stores (useLogStore)
 * - ../types (BOMDisplayRow)
 * - ../utils/clipboard (copyText, copyRowTSV)
 */

import { ref, computed, onMounted, onUnmounted, type Ref } from 'vue'
import { useLogStore } from '../../../stores'
import type { BOMDisplayRow } from '../types'
import { copyText, copyRowTSV } from '../utils/clipboard'

interface ContextMenuOptions {
  /** 搜尋關鍵字響應式參照 (用於「依此料號篩選」功能) */
  searchQuery: Ref<string>
  /** 檢查指定主料群組是否處於收合狀態 */
  isCollapsed: (parentKey: string) => boolean
  /** 切換指定主料群組之展開/收合狀態 */
  toggleCollapse: (parentKey: string) => void
}

/**
 * 建立 BOM 表格右鍵快顯選單與快捷鍵控制器
 * 
 * @param {ContextMenuOptions} options - 配置選項物件
 */
export function useBOMContextMenu(options: ContextMenuOptions) {
  const { searchQuery, isCollapsed, toggleCollapse } = options
  const logStore = useLogStore()

  /** PrimeVue ContextMenu 元件 template ref */
  const contextMenuRef = ref()
  /** 目前透過右鍵點選所選定的資料列 */
  const selectedContextRow = ref<BOMDisplayRow | null>(null)
  /** 目前左鍵點選啟用的資料列 (用於 Ctrl+C 快捷鍵整列複製) */
  const activeSelectedRow = ref<BOMDisplayRow | null>(null)

  /**
   * 處理表格列左鍵點選事件
   * @param {any} event - PrimeVue DataTable row-click 事件物件
   */
  function onRowClick(event: any): void {
    activeSelectedRow.value = event.data
  }

  /**
   * 處理表格列滑鼠右鍵點擊事件
   * @param {any} event - PrimeVue DataTable row-contextmenu 事件物件
   */
  function onRowContextMenu(event: any): void {
    selectedContextRow.value = event.data
    activeSelectedRow.value = event.data
    contextMenuRef.value?.show(event.originalEvent)
  }

  /**
   * 鍵盤 Ctrl+C (或 Cmd+C) 事件監聽常式
   * 若使用者有手動框選文字，讓瀏覽器原生複製生效；
   * 若無框選文字但有選中表格列，則自動複製該列完整 TSV 資料並在 Log 中提示。
   * 
   * @param {KeyboardEvent} event - 鍵盤事件物件
   */
  function handleKeyDown(event: KeyboardEvent): void {
    if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'c') {
      const selectedText = typeof window !== 'undefined' ? window.getSelection()?.toString() : ''
      if (selectedText && selectedText.trim().length > 0) {
        // 已有手動框選文字，讓瀏覽器原生複製行為生效
        return
      }

      const targetRow = selectedContextRow.value || activeSelectedRow.value
      if (targetRow) {
        event.preventDefault()
        copyRowTSV(targetRow).then(success => {
          if (success) {
            logStore.addLogEntry('INFO', `已將料號 ${targetRow.hhpn || targetRow.item || '-'} 整列資料複製至剪貼簿`)
          }
        })
      }
    }
  }

  /**
   * 計算動態右鍵快顯選單項目
   */
  const contextMenuItems = computed(() => {
    const row = selectedContextRow.value
    if (!row) return []

    const items: any[] = [
      {
        label: `複製料號 (${row.hhpn || '-'})`,
        icon: 'pi pi-copy',
        disabled: !row.hhpn,
        command: () => copyText(row.hhpn)
      },
      {
        label: '複製規格描述 (Copy Description)',
        icon: 'pi pi-align-left',
        disabled: !row.description,
        command: () => copyText(row.description)
      },
      {
        label: `複製供應商料號 (${row.supplier_pn || '-'})`,
        icon: 'pi pi-tag',
        disabled: !row.supplier_pn,
        command: () => copyText(row.supplier_pn)
      },
      {
        label: '複製整列資料 (TSV)',
        icon: 'pi pi-table',
        command: () => copyRowTSV(row)
      },
      { separator: true },
      {
        label: '依此料號篩選 (Filter by PN)',
        icon: 'pi pi-filter',
        disabled: !row.hhpn,
        command: () => {
          searchQuery.value = row.hhpn
        }
      }
    ]

    // 若為具有替代料的主料，提供展開/收合切換選項
    if (!row.isSecondSource && row.hasSecondSources) {
      const collapsed = isCollapsed(row.parentKey)
      items.push({
        label: collapsed ? '展開替代料 (Expand 2nd Source)' : '收合替代料 (Collapse 2nd Source)',
        icon: collapsed ? 'pi pi-chevron-down' : 'pi pi-chevron-right',
        disabled: false,
        command: () => toggleCollapse(row.parentKey)
      })
    }

    return items
  })

  onMounted(() => {
    window.addEventListener('keydown', handleKeyDown)
  })

  onUnmounted(() => {
    window.removeEventListener('keydown', handleKeyDown)
  })

  return {
    contextMenuRef,
    selectedContextRow,
    activeSelectedRow,
    contextMenuItems,
    onRowClick,
    onRowContextMenu,
  }
}
