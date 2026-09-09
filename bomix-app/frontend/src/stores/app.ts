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

  // 全域匯入對話框與檔案清單狀態管理 (Single Source of Truth)
  const importDialogVisible = ref(false)
  const importFiles = ref<string[]>([])
  const confirmOverwrite = ref(true)

  // 全域 Matrix 複製對話框狀態管理
  const copyMatrixDialogVisible = ref(false)

  /**
   * 開啟 Matrix 複製對話框
   */
  function openCopyMatrixDialog(): void {
    if (!isOpen.value) {
      logStore.addLogEntry('WARN', '請先建立或開啟系列專案，方可複製 Matrix')
      return
    }
    copyMatrixDialogVisible.value = true
  }

  /**
   * 關閉 Matrix 複製對話框
   */
  function closeCopyMatrixDialog(): void {
    copyMatrixDialogVisible.value = false
  }

  /**
   * 初始化載入全域設定
   */
  async function initSettings(): Promise<void> {
    try {
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
   * 開啟匯入對話框
   * @param {string[]} [files] - 選擇性傳入之預設匯入檔案清單 (例如拖曳檔案時)，若未傳入則重置為空清單
   */
  function openImportDialog(files?: string[]): void {
    if (!isOpen.value) {
      logStore.addLogEntry('WARN', '請先建立或開啟系列專案，方可匯入 BOM 檔案')
      return
    }
    if (files && files.length > 0) {
      // 傳入新檔案時：合併進現有清單 (自動去重)
      const combined = Array.from(new Set([...importFiles.value, ...files]))
      importFiles.value = combined
      logStore.addLogEntry('INFO', `偵測到拖曳 ${files.length} 個 Excel 檔案，已自動載入至匯入對話框`)
    } else {
      // 手動點擊開啟時：重置為空清單
      importFiles.value = []
    }
    importDialogVisible.value = true
  }

  /**
   * 追加檔案至當前匯入檔案清單 (自動去重)
   * @param {string[]} files - 欲追加之檔案路徑清單
   */
  function addImportFiles(files: string[]): void {
    if (!files || files.length === 0) return
    const combined = Array.from(new Set([...importFiles.value, ...files]))
    importFiles.value = combined
  }

  /**
   * 從當前匯入檔案清單中移除指定索引之項目
   * @param {number} index - 欲移除之項目索引
   */
  function removeImportFile(index: number): void {
    if (index >= 0 && index < importFiles.value.length) {
      importFiles.value.splice(index, 1)
    }
  }

  /**
   * 清空當前匯入檔案清單
   */
  function clearImportFiles(): void {
    importFiles.value = []
  }

  /**
   * 關閉匯入對話框並重置檔案清單
   */
  function closeImportDialog(): void {
    importDialogVisible.value = false
    importFiles.value = []
  }

  /**
   * 處理拖曳 Excel 檔案進應用程式事件 (相容別名)
   * @param {string[]} paths - 拖曳進來的檔案完整路徑清單
   */
  function handleDroppedFiles(paths: string[]): void {
    openImportDialog(paths)
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
    importFiles,
    confirmOverwrite,
    copyMatrixDialogVisible,
    // Getters
    isSeriesOpen,
    // Actions
    openSeries,
    createSeries,
    closeSeries,
    clearError,
    applyTheme,
    setWorkspaceView,
    openImportDialog,
    addImportFiles,
    removeImportFile,
    clearImportFiles,
    closeImportDialog,
    openCopyMatrixDialog,
    closeCopyMatrixDialog,
    handleDroppedFiles,
    initSettings,
    setConfirmOverwrite,
  }
})
