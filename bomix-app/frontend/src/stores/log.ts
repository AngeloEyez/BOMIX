import { defineStore } from 'pinia'
import { ref, computed, reactive } from 'vue'
import { ListenToEvents, GetLogs, LogFrontend } from '../services/api'

export interface LogEntry {
  id?: string
  level: string
  message: string
  timestamp: string
  attrs?: Record<string, string>
  
  // Task Tracker extensions
  isTaskTracker?: boolean
  status?: string // 'queued', 'running', 'error', 'done'
  history?: LogEntry[]
}

export type LogLevel = 'ALL' | 'DEBUG' | 'INFO' | 'WARN' | 'ERROR'

export const useLogStore = defineStore('log', () => {
  // State
  const logs = ref<LogEntry[]>([])
  const filterLevel = ref<LogLevel>('ALL')
  const globalLogLevel = ref<string>('INFO')

  const SEVERITY: Record<string, number> = {
    'DEBUG': 10,
    'INFO': 20,
    'WARN': 30,
    'ERROR': 40
  }

  // Getters
  const filteredLogs = computed(() => {
    const minSeverity = SEVERITY[globalLogLevel.value.toUpperCase()] || 20

    if (filterLevel.value === 'ALL') {
      return logs.value.filter(log => (SEVERITY[log.level.toUpperCase()] || 0) >= minSeverity)
    }
    return logs.value.filter(log => log.level === filterLevel.value && (SEVERITY[log.level.toUpperCase()] || 0) >= minSeverity)
  })

  const errorCount = computed(() =>
    logs.value.filter(log => log.level === 'ERROR').length
  )

  const warnCount = computed(() =>
    logs.value.filter(log => log.level === 'WARN').length
  )

  const infoCount = computed(() =>
    logs.value.filter(log => log.level === 'INFO').length
  )

  const debugCount = computed(() =>
    logs.value.filter(log => log.level === 'DEBUG').length
  )

  // 暫存已被追蹤的 task 索引，以便快速查找
  const taskMap = reactive(new Map<string, LogEntry>())

  // Actions
  function addLog(entry: LogEntry): void {
    const taskId = entry.attrs?.taskID
    
    // 如果是 Task 相關的 log，則進行群組化處理
    if (taskId) {
      let tracker = taskMap.get(taskId)
      
      if (!tracker) {
        // 建立一個全新的 Task Tracker Row，並加入主要 logs 陣列中
        tracker = {
          id: `task-${taskId}`,
          isTaskTracker: true,
          level: entry.level,
          message: entry.message,
          timestamp: entry.timestamp,
          attrs: entry.attrs,
          status: 'queued',
          history: []
        }
        taskMap.set(taskId, tracker)
        logs.value.push(tracker)
      }
      
      // 將原始 Log 記錄在 Task 的 history 裡 (包含它原本的時間與內容)
      tracker.history?.push({ ...entry, id: entry.id || crypto.randomUUID() })
      
      // 更新 Tracker 的狀態與最新訊息
      tracker.message = entry.message
      tracker.timestamp = entry.timestamp
      
      // 根據 Log 內容更新狀態 (完全由確定的 taskStatus 變數與 entry.level 辨識，不使用文字關鍵字比對)
      if (entry.attrs?.taskStatus) {
        const ts = entry.attrs.taskStatus.toLowerCase()
        if (ts === 'error' || ts === 'failed') {
          tracker.status = 'error'
          tracker.level = 'ERROR'
        } else if (ts === 'warning' || ts === 'warn') {
          tracker.status = 'warning'
          tracker.level = 'WARN'
        } else if (ts === 'done' || ts === 'completed') {
          tracker.status = 'done'
          if (tracker.level !== 'ERROR' && tracker.level !== 'WARN') {
            tracker.level = 'INFO'
          }
        } else if (ts === 'running') {
          tracker.status = 'running'
        } else if (ts === 'queued') {
          tracker.status = 'queued'
        }
      } else {
        // 若無 taskStatus，僅依據標準 entry.level (ERROR / WARN) 提升 Tracker 的層級
        const currentSeverity = SEVERITY[tracker.level.toUpperCase()] || 0
        const newSeverity = SEVERITY[entry.level.toUpperCase()] || 0
        if (newSeverity > currentSeverity) {
          tracker.level = entry.level
        }
      }
      
      return // 不再將 Task 的子日誌放進 main logs 陣列
    }

    // 一般日誌
    logs.value.push({
      id: entry.id || crypto.randomUUID(),
      level: entry.level,
      message: entry.message,
      timestamp: entry.timestamp,
      attrs: entry.attrs,
    })
  }

  /**
   * 發布一則日誌 (提供給前端各組件使用)
   * 
   * **如何發布一般日誌：**
   * ```ts
   * logStore.addLogEntry('INFO', '這是一般日誌')
   * ```
   * 
   * **如何發布 Task 日誌：**
   * 只要在 attrs 中帶上 `taskID`，日誌系統就會自動將其轉換為 Task Tracker 模式。
   * 它會在主視窗中固定顯示為一行，隨後相同 taskID 的日誌都會即時更新該行，
   * 並且使用者可以雙擊該行展開查看詳細歷史日誌。
   * ```ts
   * logStore.addLogEntry('INFO', 'Task started', { taskID: '12345', name: 'Import Excel' })
   * logStore.addLogEntry('DEBUG', 'Processing...', { taskID: '12345' })
   * logStore.addLogEntry('ERROR', 'Failed', { taskID: '12345', error: 'Format error' })
   * ```
   * 
   * @param level 日誌等級 (INFO, DEBUG, WARN, ERROR)
   * @param message 訊息內容
   * @param attrs 附加屬性 (若包含 taskID，則視為 task 日誌)
   */
  function addLogEntry(level: string, message: string, attrs?: Record<string, string>): void {
    // 若有 attrs，我們可將其格式化後附在 message 後方，或者後端其實不支援 attrs？
    // 注意：目前 backend LogFrontend 只吃 (level, message)。若需要 taskID 可以在這裡拼裝。
    // 但因為純前端的 Log 都是一般 Log (不是 Task Tracker)，所以我們可以直接傳送文字即可。
    LogFrontend(level, message)
  }

  function clearLogs(): void {
    logs.value = []
    taskMap.clear()
  }

  function setFilterLevel(level: LogLevel): void {
    filterLevel.value = level
  }

  function removeLog(logId: string): void {
    const index = logs.value.findIndex(log => log.id === logId)
    if (index !== -1) {
      logs.value.splice(index, 1)
    }
  }

  // Load logs from backend
  async function loadLogs(limit: number = 100): Promise<void> {
    try {
      const entries = await GetLogs('ALL', limit)
      logs.value = entries
    } catch (err) {
      console.error('Failed to load logs:', err)
    }
  }

  // Listen to log events from backend
  function startListening(): void {
    ListenToEvents('log:new', (data) => {
      const entry = data as LogEntry
      addLog(entry)
    })
  }

  return {
    // State
    logs,
    filterLevel,
    globalLogLevel,
    // Getters
    filteredLogs,
    errorCount,
    warnCount,
    infoCount,
    debugCount,
    // Actions
    addLog,
    addLogEntry,
    clearLogs,
    setFilterLevel,
    removeLog,
    loadLogs,
    startListening,
  }
})
