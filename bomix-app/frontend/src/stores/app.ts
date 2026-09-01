import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { GetSeriesInfo, CloseSeries, OpenSeries, CreateSeries } from '../services/api'
import { useLogStore } from './log'

export interface SeriesInfo {
  id: number
  name: string
  description: string
  path: string
  lastExportPath?: string
  projectExportOrder?: string[]
  projectModelCounts?: Record<string, number>
}

export const useAppStore = defineStore('app', () => {
  // State
  const isOpen = ref(false)
  const seriesInfo = ref<SeriesInfo | null>(null)
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  const logStore = useLogStore()

  // Getters
  const isSeriesOpen = computed(() => isOpen.value)

  // Actions
  async function openSeries(path: string): Promise<void> {
    isLoading.value = true
    error.value = null
    try {
      // 1. Call backend to open database
      await OpenSeries(path)
      // 2. Fetch series info
      const info = await GetSeriesInfo()
      if (!info) {
        throw new Error('無法取得系列資訊')
      }
      seriesInfo.value = {
        id: info.id,
        name: info.name,
        description: info.description,
        path,
        lastExportPath: info.lastExportPath || '',
        projectExportOrder: info.projectExportOrder || [],
        projectModelCounts: info.projectModelCounts || {},
      }
      isOpen.value = true
    } catch (err) {
      const errMsg = err instanceof Error ? err.message : '開啟系列失敗'
      error.value = errMsg
      isOpen.value = false
      logStore.addLogEntry('ERROR', `開啟系列失敗：${errMsg}`)
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function createSeries(path: string, name: string, description: string): Promise<void> {
    isLoading.value = true
    error.value = null
    try {
      await CreateSeries(path, name, description)
      const info = await GetSeriesInfo()
      seriesInfo.value = {
        id: info?.id || 1,
        name: info?.name || name,
        description: info?.description || description,
        path,
        lastExportPath: info?.lastExportPath || '',
        projectExportOrder: info?.projectExportOrder || [],
        projectModelCounts: info?.projectModelCounts || {},
      }
      isOpen.value = true
    } catch (err) {
      const errMsg = err instanceof Error ? err.message : '建立系列失敗'
      error.value = errMsg
      isOpen.value = false
      logStore.addLogEntry('ERROR', `建立系列失敗：${errMsg}`)
      throw err
    } finally {
      isLoading.value = false
    }
  }

  const workspaceView = ref<'table' | 'export'>('table')

  /**
   * 設定工作區主畫面顯示之視圖模式 ('table' 表格 或 'export' 匯出)
   * @param {'table' | 'export'} view - 視圖模式
   */
  function setWorkspaceView(view: 'table' | 'export'): void {
    workspaceView.value = view
  }

  async function closeSeries(): Promise<void> {
    try {
      await CloseSeries()
    } catch (err) {
      const errMsg = err instanceof Error ? err.message : '關閉系列失敗'
      logStore.addLogEntry('ERROR', `關閉系列發生錯誤：${errMsg}`)
      console.error('Failed to close series:', err)
    } finally {
      isOpen.value = false
      seriesInfo.value = null
      workspaceView.value = 'table'
    }
  }

  function clearError(): void {
    error.value = null
  }

  const currentTheme = ref('system')

  function applyTheme(theme: string) {
    currentTheme.value = theme
    const isDark = theme === 'dark' || (theme === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches)
    if (isDark) {
      document.documentElement.classList.add('app-dark')
    } else {
      document.documentElement.classList.remove('app-dark')
    }
  }

  // 全域匯入對話框與拖曳檔案狀態管理
  const importDialogVisible = ref(false)
  const droppedFiles = ref<string[]>([])
  const confirmOverwrite = ref(true)

  /**
   * 初始化載入全域設定
   */
  async function initSettings(): Promise<void> {
    try {
      const s = await GetSeriesInfo() // ensure initialized
      const settings = await (await import('../services/api')).GetSettings()
      if (settings?.import) {
        confirmOverwrite.value = settings.import.confirmOverwrite ?? true
      }
    } catch (_) {}
  }

  /**
   * 更新並持久化覆蓋前確認選項設定
   * @param {boolean} val - 是否啟用覆蓋前確認
   */
  async function setConfirmOverwrite(val: boolean): Promise<void> {
    confirmOverwrite.value = val
    try {
      const { GetSettings, UpdateSettings } = await import('../services/api')
      const s = await GetSettings()
      if (s) {
        if (!s.import) {
          s.import = { confirmOverwrite: val, autoImportPreviousMatrix: false }
        } else {
          s.import.confirmOverwrite = val
        }
        await UpdateSettings(s)
      }
    } catch (err) {
      console.error('Failed to persist confirmOverwrite setting:', err)
    }
  }

  /**
   * 處理拖曳 Excel 檔案進應用程式事件
   * @param {string[]} paths - 拖曳進來的檔案完整路徑清單
   */
  function handleDroppedFiles(paths: string[]): void {
    if (!paths || paths.length === 0) return
    if (!isOpen.value) {
      logStore.addLogEntry('WARN', '請先建立或開啟系列專案，方可匯入 BOM 檔案')
      return
    }
    droppedFiles.value = [...paths]
    importDialogVisible.value = true
    logStore.addLogEntry('INFO', `偵測到拖曳 ${paths.length} 個 Excel 檔案，已自動載入至匯入對話框`)
  }

  /**
   * 清除暫存的拖曳檔案清單
   */
  function clearDroppedFiles(): void {
    droppedFiles.value = []
  }

  return {
    // State
    isOpen,
    seriesInfo,
    isLoading,
    error,
    currentTheme,
    workspaceView,
    importDialogVisible,
    droppedFiles,
    confirmOverwrite,
    // Getters
    isSeriesOpen,
    // Actions
    openSeries,
    createSeries,
    closeSeries,
    clearError,
    applyTheme,
    setWorkspaceView,
    handleDroppedFiles,
    clearDroppedFiles,
    initSettings,
    setConfirmOverwrite,
  }
})
