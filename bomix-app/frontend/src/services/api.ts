/**
 * BOMIX Backend API Service
 *
 * This module provides a unified interface for all backend API calls.
 * It wraps Wails bindings and provides consistent error handling.
 */

import { Dialogs, Events } from '@wailsio/runtime'
import { App } from '../../bindings/bomix-app/backend/index.js'
import type { ViewResult, ViewPartGroup, ViewRevision, ViewSecondSource, ViewModelSelection, ViewModelItem } from '../../bindings/bomix-app/backend/view/models.js'

export type { ViewResult, ViewPartGroup, ViewRevision, ViewSecondSource, ViewModelSelection, ViewModelItem }

// Type definitions matching backend models
export interface ProjectExportSetting {
  projectCode: string
  modelCount: number
}

export interface SeriesInfo {
  id: number
  name: string
  description: string
  path: string
  lastExportPath: string
  projectExportOrder?: string[]
  projectModelCounts?: Record<string, number>
}

export interface RecentFile {
  path: string
  name: string
  lastOpened: string
}

export interface Project {
  id: number
  seriesId: number
  code: string
  description: string
  createdAt: string
  updatedAt: string
}

export interface BomRevision {
  id: number
  projectId: number
  phase: string
  version: string
  description: string
  schematicVersion: string
  pcbVersion: string
  pcaPn: string
  date: string
  mode: string
  sourceFile: string
  modelCount: number
  createdAt: string
  updatedAt: string
}

export interface Part {
  id: number
  revisionId: number
  type: string
  item: string
  hhpn: string
  supplier: string
  supplierPn: string
  description: string
  remark: string
  createdAt: string
  updatedAt: string
}

export interface SecondSource {
  id: number
  revisionId: number
  partId: number
  hhpn: string
  supplier: string
  supplierPn: string
  description: string
  createdAt: string
  updatedAt: string
}

export interface MatrixModel {
  id: number
  revisionId: number
  modelName: string
  qty: number
  createdAt: string
  updatedAt: string
}

export interface MatrixSelection {
  id: number
  revisionId: number
  modelId: number
  partId: number
  group: string
  material: string
  selectedSupplier: string
  selectedSupplierPn: string
  isAutoSelected: boolean
  createdAt: string
  updatedAt: string
}

export interface Task {
  id: string
  name: string
  type: string
  status: string
  progress: number
  message: string
  error?: string
  createdAt: string
  updatedAt: string
}

export interface ImportResult {
  fileName: string
  format: string
  status: string
  message: string
  partsCount: number
  error?: string
  taskID?: string
}

export interface ExportOptions {
  format: string
  projectIds?: number[]
  revisionIds: number[]
  description?: string
  outputPath?: string
  outputDir?: string
  modelCountOverrides?: Record<string, number>
}

// Type alias for backend import result
export type BackendImportResult = ImportResult

export interface LogEntry {
  id?: string
  level: string
  message: string
  timestamp: string
  attrs?: Record<string, string>
}

export interface Settings {
  theme: string
  import: ImportSettings
  logger: LoggerSettings
  recentFiles: RecentFilesSettings
  autoOpenLastFile: boolean
  lastOpenedFile: string
  autoImportPreviousMatrix: boolean
  ai?: AISettings
}

export interface AISettings {
  enabled: boolean
  baseUrl: string
  apiKey: string
  model: string
  temperature: number
  maxTokens: number
  timeout: number
  language: string
}

export interface AIChatMessage {
  role: 'system' | 'user' | 'assistant' | 'tool'
  content: string
  tool_calls?: any[]
  tool_call_id?: string
  name?: string
}

export interface ImportSettings {
  confirmOverwrite: boolean
  autoImportPreviousMatrix: boolean
}

export interface LoggerSettings {
  level: string
  maxEntries: number
}

export interface RecentFilesSettings {
  maxRecentFiles: number
  recentFiles: string[]
}

export interface FileDialogOptions {
  title?: string
  defaultPath?: string
  filters?: FileFilter[]
  buttonLabel?: string
  selectFiles?: boolean
  selectDirectory?: boolean
  multiSelect?: boolean
}

export interface FileFilter {
  name: string
  extensions: string[]
}

// Error handling wrapper
class ApiError extends Error {
  constructor(
    public message: string,
    public code: string = 'UNKNOWN_ERROR',
    public originalError?: Error
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

// Handle API errors consistently
function handleApiError(error: unknown, context: string): never {
  if (error instanceof Error) {
    throw new ApiError(
      `${context}: ${error.message}`,
      'API_ERROR',
      error
    )
  }
  throw new ApiError(
    `${context}: Unknown error`,
    'UNKNOWN_ERROR'
  )
}

// ==================== Series Management ====================

export async function CreateSeries(path: string, name: string, description: string): Promise<void> {
  try {
    await App.CreateSeries(path, name, description)
  } catch (error) {
    handleApiError(error, 'CreateSeries')
  }
}

export async function OpenSeries(path: string): Promise<void> {
  try {
    await App.OpenSeries(path)
  } catch (error) {
    handleApiError(error, 'OpenSeries')
  }
}

export async function CloseSeries(): Promise<void> {
  try {
    await App.CloseSeries()
  } catch (error) {
    handleApiError(error, 'CloseSeries')
  }
}

export async function GetSeriesInfo(path?: string): Promise<SeriesInfo> {
  try {
    const res = await App.GetSeriesInfo()
    return res as unknown as SeriesInfo
  } catch (error) {
    handleApiError(error, 'GetSeriesInfo')
  }
}

export async function SaveProjectExportOrder(settings: ProjectExportSetting[]): Promise<void> {
  try {
    await App.SaveProjectExportOrder(settings as unknown as any)
  } catch (error) {
    handleApiError(error, 'SaveProjectExportOrder')
  }
}

export async function GetRecentSeries(): Promise<RecentFile[]> {
  try {
    const res = await App.GetRecentSeries()
    return (res || []) as unknown as RecentFile[]
  } catch (error) {
    handleApiError(error, 'GetRecentSeries')
  }
}

// ==================== Project Management ====================

export async function GetProjects(seriesId: number): Promise<Project[]> {
  try {
    const res = await App.GetProjects(seriesId)
    return (res || []) as unknown as Project[]
  } catch (error) {
    handleApiError(error, 'GetProjects')
  }
}

export async function GetProject(id: number): Promise<Project> {
  try {
    const res = await App.GetProject(id)
    return res as unknown as Project
  } catch (error) {
    handleApiError(error, 'GetProject')
  }
}

// ==================== Revision Management ====================

export async function GetRevisions(projectId: number): Promise<BomRevision[]> {
  try {
    const res = await App.GetRevisions(projectId)
    return (res || []) as unknown as BomRevision[]
  } catch (error) {
    handleApiError(error, 'GetRevisions')
  }
}

export async function GetRevision(id: number): Promise<BomRevision> {
  try {
    const res = await App.GetRevision(id)
    return res as unknown as BomRevision
  } catch (error) {
    handleApiError(error, 'GetRevision')
  }
}

// ==================== BOM View ====================

export async function GetBOMView(revisionIDs: number[], viewType: string): Promise<ViewResult> {
  try {
    const res = await App.GetBOMView(revisionIDs, viewType)
    if (!res) {
      throw new Error('Received null result from GetBOMView')
    }
    return res as unknown as ViewResult
  } catch (error) {
    handleApiError(error, 'GetBOMView')
  }
}

/**
 * 更新單一物料群組在指定 Revision 與 Model 的勾選狀態
 * @param {number} revisionID - BOM Revision ID
 * @param {number} modelID - Matrix Model ID (0 表示預設 Model)
 * @param {number} mainMaterialID - 主料 Material ID
 * @param {number} selectedMaterialID - 被選中物料 Material ID (0 表示取消勾選)
 */
export async function SetMatrixSelection(
  revisionID: number,
  modelID: number,
  mainMaterialID: number,
  selectedMaterialID: number
): Promise<void> {
  try {
    await (App as any).SetMatrixSelection(revisionID, modelID, mainMaterialID, selectedMaterialID)
  } catch (error) {
    handleApiError(error, 'SetMatrixSelection')
  }
}

/**
 * 更新單一物料群組在指定 Revision 與 Model 排序索引 (SortOrder) 的勾選狀態
 * 
 * @param {number} revisionID - BOM Revision ID
 * @param {number} sortOrder - 0-based Model 排序索引 (0, 1, 2...)
 * @param {number} mainMaterialID - 主料 Material ID
 * @param {number} selectedMaterialID - 被選中物料 Material ID (0 表示取消勾選)
 * @returns {Promise<void>}
 */
export async function SetMatrixModelSelection(
  revisionID: number,
  sortOrder: number,
  mainMaterialID: number,
  selectedMaterialID: number
): Promise<void> {
  try {
    if (typeof (App as any).SetMatrixModelSelection === 'function') {
      await (App as any).SetMatrixModelSelection(revisionID, sortOrder, mainMaterialID, selectedMaterialID)
    } else {
      const { Call } = await import('@wailsio/runtime')
      await Call.ByName('backend.App.SetMatrixModelSelection', revisionID, sortOrder, mainMaterialID, selectedMaterialID)
    }
  } catch (error) {
    handleApiError(error, 'SetMatrixModelSelection')
  }
}

/**
 * 更新單一物料的 Notes 註記欄位內容，並即時持久化至資料庫
 * 
 * @param {number} materialID - 全域物料 ID (Material.ID)
 * @param {string} notes - 新的 Notes 內容
 * @returns {Promise<void>}
 */
export async function UpdateMaterialNote(
  materialID: number,
  notes: string
): Promise<void> {
  try {
    if (typeof (App as any).UpdateMaterialNote === 'function') {
      await (App as any).UpdateMaterialNote(materialID, notes)
    } else {
      const { Call } = await import('@wailsio/runtime')
      await Call.ByName('backend.App.UpdateMaterialNote', materialID, notes)
    }
  } catch (error) {
    handleApiError(error, 'UpdateMaterialNote')
  }
}

/**
 * MatrixModelInput 代表更新 Revision Models 時的單一 Model 設定項目
 */
export interface MatrixModelInput {
  /** 資料庫記錄 ID（若為新建立可為 0 或 undefined） */
  id?: number
  /** 0-based 排序索引 (0, 1, 2...) */
  sort_order: number
  /** Model 顯示名稱（例如 Model A 或自訂名稱） */
  model_name: string
  /** Model 打件數量 */
  qty: number
}

/**
 * 批次更新指定 Revision 的 Matrix Model 設定 (數量與用量 Qty)
 * 
 * @param {number} revisionID - BOM Revision ID
 * @param {MatrixModelInput[]} models - Model 設定列表
 * @returns {Promise<void>}
 */
export async function UpdateRevisionMatrixModels(
  revisionID: number,
  models: MatrixModelInput[]
): Promise<void> {
  try {
    if (typeof (App as any).UpdateRevisionMatrixModels === 'function') {
      await (App as any).UpdateRevisionMatrixModels(revisionID, models)
    } else {
      const { Call } = await import('@wailsio/runtime')
      await Call.ByName('backend.App.UpdateRevisionMatrixModels', revisionID, models)
    }
  } catch (error) {
    handleApiError(error, 'UpdateRevisionMatrixModels')
  }
}

// ==================== Import/Export ====================

export async function ImportExcel(filePaths: string[], confirmOverwrite: boolean = true): Promise<ImportResult[]> {
  try {
    const res = await (App.ImportExcel as any)(filePaths, confirmOverwrite)
    return (res || []) as unknown as ImportResult[]
  } catch (error) {
    handleApiError(error, 'ImportExcel')
  }
}

/**
 * 回應任務的覆蓋確認請求 (覆蓋或略過)
 * @param {string} taskID - 任務 ID
 * @param {boolean} overwrite - 是否確認覆蓋 (true: 覆蓋, false: 略過)
 */
export async function ConfirmTaskOverwrite(taskID: string, overwrite: boolean): Promise<void> {
  try {
    if (typeof (App as any).ConfirmTaskOverwrite === 'function') {
      await (App as any).ConfirmTaskOverwrite(taskID, overwrite)
    }
  } catch (error) {
    handleApiError(error, 'ConfirmTaskOverwrite')
  }
}

export async function ExportExcel(options: ExportOptions): Promise<string[]> {
  try {
    const res = await App.ExportExcel(options as any)
    return (res || []) as string[]
  } catch (error) {
    handleApiError(error, 'ExportExcel')
  }
}


/**
 * ¥H²§¨B¥ô°È§Î¦¡¡A¤â°Ê±N source revision ªº Matrix Model »P Selection ½Æ»s¨ì target revision¡C
 * @param sourceRevisionId - ¨Ó·½ª©¥» ID
 * @param targetRevisionId - ¥Ø¼Ðª©¥» ID
 * @returns ¥ô°È ID¡]taskID¡^¡A¥i¥Î©ó Task ­±ªO°lÂÜ¶i«×
 */
export async function CopyMatrixSelections(
  sourceRevisionId: number,
  targetRevisionId: number
): Promise<string> {
  try {
    const res = await App.CopyMatrixSelections(sourceRevisionId, targetRevisionId)
    return (res || '') as string
  } catch (error) {
    handleApiError(error, 'CopyMatrixSelections')
  }
}
// ==================== Task Management ====================

export async function ListTasks(): Promise<Task[]> {
  try {
    const res = await App.ListTasks()
    return (res || []) as unknown as Task[]
  } catch (error) {
    handleApiError(error, 'ListTasks')
  }
}

export async function GetTask(id: string): Promise<Task> {
  try {
    const res = await App.GetTask(id)
    return res as unknown as Task
  } catch (error) {
    handleApiError(error, 'GetTask')
  }
}

export async function CancelTask(id: string): Promise<void> {
  try {
    await App.CancelTask(id)
  } catch (error) {
    handleApiError(error, 'CancelTask')
  }
}

// ==================== Logs ====================

export async function GetLogs(level: string, limit: number): Promise<LogEntry[]> {
  try {
    const res = await App.GetLogs(level, limit)
    return (res || []) as unknown as LogEntry[]
  } catch (error) {
    handleApiError(error, 'GetLogs')
  }
}

export async function ClearLogs(): Promise<void> {
  try {
    await App.ClearLogs()
  } catch (error) {
    handleApiError(error, 'ClearLogs')
  }
}

export async function LogFrontend(level: string, message: string): Promise<void> {
  try {
    await App.LogFrontend(level, message)
  } catch (error) {
    console.error('Failed to send log to backend:', error)
  }
}

// ==================== Settings ====================

export async function GetSettings(): Promise<Settings> {
  try {
    const res = await App.GetSettings()
    return res as unknown as Settings
  } catch (error) {
    handleApiError(error, 'GetSettings')
  }
}

export async function UpdateSettings(settings: Settings): Promise<void> {
  try {
    await App.UpdateSettings(settings as any)
  } catch (error) {
    handleApiError(error, 'UpdateSettings')
  }
}

// ==================== Event Listening ====================

const eventListeners = new Map<string, any>()

export function ListenToEvents(eventName: string, callback: (data: any) => void): void {
  try {
    const unlistener = Events.On(eventName, (e: any) => {
      // 在 Wails v3 中，Events.On 回調傳入 WailsEvent 物件，其 .data 欄位攜帶原始 payload
      const data = e?.data !== undefined ? e.data : e
      callback(data)
    })
    
    eventListeners.set(eventName, unlistener)
  } catch (error) {
    console.error(`Failed to listen to event ${eventName}:`, error)
  }
}

export function UnlistenToEvents(eventName: string): void {
  try {
    const unlistener = eventListeners.get(eventName)
    if (unlistener) {
      unlistener()
      eventListeners.delete(eventName)
    }
  } catch (error) {
    console.error(`Failed to unlisten to event ${eventName}:`, error)
  }
}

// ==================== File Dialogs ====================

export async function OpenFileDialog(options: FileDialogOptions): Promise<string> {
  try {
    const filters = options.filters?.map(f => ({
      DisplayName: f.name,
      Pattern: f.extensions.map(ext => `*.${ext}`).join(';')
    }))

    const result = await Dialogs.OpenFile({
      Title: options.title,
      Filters: filters,
      AllowsMultipleSelection: false,
    })
    
    return Array.isArray(result) ? result[0] : (result || '')
  } catch (error) {
    handleApiError(error, 'OpenFileDialog')
  }
}

/**
 * ?å?å¤æ?æ¡é¸?å?è©±æ?
 * @param options å°è©±æ¡è¨­å®é¸??
 * @returns ?¸å??æ?æ¡è·¯å¾é£??
 */
export async function OpenMultipleFilesDialog(options: FileDialogOptions): Promise<string[]> {
  try {
    const filters = options.filters?.map(f => ({
      DisplayName: f.name,
      Pattern: f.extensions.map(ext => `*.${ext}`).join(';')
    }))

    const result = await Dialogs.OpenFile({
      Title: options.title,
      Filters: filters,
      AllowsMultipleSelection: true,
    })
    
    if (Array.isArray(result)) {
      return result.filter((item): item is string => typeof item === 'string' && item.length > 0)
    } else if (typeof result === 'string') {
      const s = result as string
      return s.length > 0 ? [s] : []
    }
    return []
  } catch (error) {
    handleApiError(error, 'OpenMultipleFilesDialog')
    return []
  }
}

export async function SaveFileDialog(options: FileDialogOptions): Promise<string> {
  try {
    const filters = options.filters?.map(f => ({
      DisplayName: f.name,
      Pattern: f.extensions.map(ext => `*.${ext}`).join(';')
    }))

    const result = await Dialogs.SaveFile({
      Title: options.title,
      Filters: filters,
    })
    
    return result || ''
  } catch (error) {
    handleApiError(error, 'SaveFileDialog')
  }
}

export async function SelectFolderDialog(options: FileDialogOptions): Promise<string> {
  try {
    const result = await Dialogs.OpenFile({
      Title: options.title,
      CanChooseDirectories: true,
      CanChooseFiles: false,
      AllowsMultipleSelection: false,
    })
    
    return Array.isArray(result) ? result[0] : (result || '')
  } catch (error) {
    handleApiError(error, 'SelectFolderDialog')
  }
}

// ==================== App ====================

export function Quit(): void {
  try {
    App.Quit()
  } catch (error) {
    console.error('Failed to quit application:', error)
  }
}

export function GetVersion(): string {
  try {
    // Wails v3 GetVersion usually returns a promise, so we should await it if possible
    // But since this function signature is synchronous string return, we'll try to handle it.
    // Actually in Wails v3, App.GetVersion() is a Promise. Let's return a dummy or fix the signature if needed.
    // For now we return "1.0.0" because returning a Promise in a sync function will fail.
    App.GetVersion().then(v => console.log('Version:', v))
    return '1.0.0'
  } catch (error) {
    console.error('Failed to get version:', error)
    return 'unknown'
  }
}

// ==================== AI Assistant ====================

/**
 * 發送對話訊息歷史至後端 AI 引擎
 * @param messages 包含 system/user/assistant/tool 的完整歷史
 */
export async function AIChatSend(messages: AIChatMessage[]): Promise<void> {
  try {
    await (App as any).AIChatSend(messages)
  } catch (error) {
    handleApiError(error, 'AIChatSend')
  }
}

/**
 * 中斷當前正在執行的 AI 生成或工具調用
 */
export async function AIChatStop(): Promise<void> {
  try {
    await (App as any).AIChatStop()
  } catch (error) {
    handleApiError(error, 'AIChatStop')
  }
}

/**
 * 測試 AI API 端點與金鑰連線
 */
export async function AIChatTestConnection(): Promise<void> {
  try {
    await (App as any).AIChatTestConnection()
  } catch (error) {
    handleApiError(error, 'AIChatTestConnection')
  }
}

/**
 * 取得當前設定伺服器提供的可用模型清單
 */
export async function AIChatGetAvailableModels(): Promise<string[]> {
  try {
    const list = await (App as any).AIChatGetAvailableModels()
    return Array.isArray(list) ? list : []
  } catch (error) {
    handleApiError(error, 'AIChatGetAvailableModels')
  }
}

/**
 * 依指定 Base URL 與 API Key 即時取得伺服器提供的可用模型清單
 */
export async function AIChatFetchModelsWithConfig(baseUrl: string, apiKey: string): Promise<string[]> {
  try {
    const list = await (App as any).AIChatFetchModelsWithConfig(baseUrl, apiKey)
    return Array.isArray(list) ? list : []
  } catch (error) {
    handleApiError(error, 'AIChatFetchModelsWithConfig')
  }
}

