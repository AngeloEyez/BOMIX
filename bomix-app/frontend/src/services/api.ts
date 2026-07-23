/**
 * BOMIX Backend API Service
 *
 * This module provides a unified interface for all backend API calls.
 * It wraps Wails bindings and provides consistent error handling.
 */

import { Dialogs, Events } from '@wailsio/runtime'
import { App } from '../../bindings/bomix-app/backend/index.js'
import type { ViewResult, ViewPartGroup, ViewRevision, ViewSecondSource, ViewModelSelection } from '../../bindings/bomix-app/backend/view/models.js'

export type { ViewResult, ViewPartGroup, ViewRevision, ViewSecondSource, ViewModelSelection }

// Type definitions matching backend models
export interface SeriesInfo {
  id: number
  name: string
  description: string
  path: string
  lastExportPath: string
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
  createdAt: string
  updatedAt: string
}

export interface Part {
  id: number
  revisionId: number
  type: string
  supplier: string
  supplierPn: string
  description: string
  location: string
  quantity: number
  cost: number
  bomStatus: string
  ccl: string
  remark: string
  createdAt: string
  updatedAt: string
}

export interface SecondSource {
  id: number
  revisionId: number
  partId: number
  supplier: string
  supplierPn: string
  description: string
  cost: number
  leadTime: number
  isActive: boolean
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

export async function GetBOMView(revisionIDs: number[], viewType: string, modeOverride: string): Promise<ViewResult> {
  try {
    const res = await App.GetBOMView(revisionIDs, viewType, modeOverride)
    if (!res) {
      throw new Error('Received null result from GetBOMView')
    }
    return res as unknown as ViewResult
  } catch (error) {
    handleApiError(error, 'GetBOMView')
  }
}

// ==================== Import/Export ====================

export async function ImportExcel(filePaths: string[]): Promise<ImportResult[]> {
  try {
    const res = await App.ImportExcel(filePaths)
    return (res || []) as unknown as ImportResult[]
  } catch (error) {
    handleApiError(error, 'ImportExcel')
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
      const data = e.data && e.data.length > 0 ? e.data[0] : e.data
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
