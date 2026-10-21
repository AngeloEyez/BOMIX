import { defineStore } from 'pinia'
import { ref, computed, reactive } from 'vue'
import { ListenToEvents, GetLogs, LogFrontend } from '../services/api'
import { useAppStore } from './app'
import { useProjectStore } from './project'
import { useBOMTableStore } from './bomTable'

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
  function triggerProjectReload() {
    try {
      const appStore = useAppStore()
      const projectStore = useProjectStore()
      const bomTableStore = useBOMTableStore()
      bomTableStore.triggerReload()
      if (appStore.seriesInfo?.id) {
        projectStore.loadProjects(appStore.seriesInfo.id).catch(err => {
          console.error('Failed to auto-reload projects after task completion:', err)
        })
      }
    } catch (e) {
      console.error('Error triggering project reload:', e)
    }
  }

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

  /**
   * 安全解析 Wails v3 事件資料（處理陣列包裝情況）
   */
  function extractPayload(data: any): any {
    if (Array.isArray(data) && data.length > 0) {
      return data[0]
    }
    return data || {}
  }

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
          attrs: entry.attrs ? { ...entry.attrs } : undefined,
          status: 'queued',
          history: []
        }
        taskMap.set(taskId, tracker)
        logs.value.push(tracker)
      } else if (entry.attrs) {
        // 合併新進屬性，確保 name、type 等重要中繼資料不被後續子日誌沖刷掉
        tracker.attrs = { ...(tracker.attrs || {}), ...entry.attrs }
      }

      // 將原始 Log 記錄在 Task 的 history 裡 (包含它原本的時間與內容)
      tracker.history?.push({ ...entry, id: entry.id || crypto.randomUUID() })

      const terminalStates = ['done', 'completed', 'error', 'failed', 'warning', 'cancelled']
      const isCurrentTerminal = terminalStates.includes(tracker.status?.toLowerCase() || '')

      // 根據 Log 內容更新狀態 (完全由確定的 taskStatus 變數與 entry.level 辨識，不使用文字關鍵字比對)
      if (entry.attrs?.taskStatus) {
        const ts = entry.attrs.taskStatus.toLowerCase()
        const isIncomingNonTerminal = ['queued', 'running', 'created'].includes(ts)

        // 終結狀態保護：若任務已為終結狀態，不允許非終結狀態覆蓋
        if (isCurrentTerminal && isIncomingNonTerminal) {
          return
        }

        if (ts === 'error' || ts === 'failed') {
          tracker.status = 'error'
          tracker.level = 'ERROR'
          tracker.message = entry.message
          tracker.timestamp = entry.timestamp
        } else if (ts === 'warning' || ts === 'warn') {
          tracker.status = 'warning'
          tracker.level = 'WARN'
          tracker.message = entry.message
          tracker.timestamp = entry.timestamp
          triggerProjectReload()
        } else if (ts === 'done' || ts === 'completed') {
          tracker.status = 'done'
          if (tracker.level !== 'ERROR' && tracker.level !== 'WARN') {
            tracker.level = 'INFO'
          }
          tracker.message = entry.message
          tracker.timestamp = entry.timestamp
          triggerProjectReload()
        } else if (ts === 'running') {
          tracker.status = 'running'
          tracker.message = entry.message
          tracker.timestamp = entry.timestamp
        } else if (ts === 'queued') {
          tracker.status = 'queued'
          tracker.message = entry.message
          tracker.timestamp = entry.timestamp
        } else if (ts === 'cancelled') {
          tracker.status = 'cancelled'
          tracker.level = 'WARN'
          tracker.message = entry.message
          tracker.timestamp = entry.timestamp
        }
      } else {
        // 若無 taskStatus：僅在非終結狀態下才更新主顯示訊息與時間戳記，避免子日誌沖刷掉已完成的任務狀態
        if (!isCurrentTerminal) {
          tracker.message = entry.message
          tracker.timestamp = entry.timestamp
        }

        // 依據標準 entry.level (ERROR / WARN) 提升 Tracker 的層級
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
   * 更新或建立指定 Task Tracker 的狀態與訊息
   * 具備終結狀態保護：若已經處於完成/失敗/警告/取消狀態，不允許被舊的 running/queued 事件覆蓋。
   *
   * @param taskId - 任務 ID
   * @param status - 任務狀態 ('queued' | 'running' | 'done' | 'completed' | 'warning' | 'error' | 'failed' | 'cancelled' | 'waiting_confirm')
   * @param message - 最新訊息 (可選)
   * @param name - 任務名稱 (可選)
   */
  function updateTaskStatus(taskId: string, status: string, message?: string, name?: string): void {
    if (!taskId) return
    const normStatus = status.toLowerCase()
    let tracker = taskMap.get(taskId)

    const terminalStates = ['done', 'completed', 'error', 'failed', 'warning', 'cancelled']
    const isIncomingTerminal = terminalStates.includes(normStatus)
    const isIncomingNonTerminal = ['queued', 'running', 'created'].includes(normStatus)

    if (!tracker) {
      const level = (normStatus === 'error' || normStatus === 'failed')
        ? 'ERROR'
        : (normStatus === 'warning' || normStatus === 'cancelled')
          ? 'WARN'
          : 'INFO'

      tracker = {
        id: `task-${taskId}`,
        isTaskTracker: true,
        level,
        message: message || '',
        timestamp: new Date().toISOString(),
        attrs: { taskID: taskId, ...(name ? { name } : {}) },
        status: normStatus === 'completed' ? 'done' : normStatus,
        history: message ? [{
          id: crypto.randomUUID(),
          level,
          message,
          timestamp: new Date().toISOString(),
          attrs: { taskID: taskId, taskStatus: normStatus, ...(name ? { name } : {}) }
        }] : []
      }
      taskMap.set(taskId, tracker)
      logs.value.push(tracker)
      if (isIncomingTerminal) {
        triggerProjectReload()
      }
      return
    }

    const isCurrentTerminal = terminalStates.includes(tracker.status?.toLowerCase() || '')
    if (isCurrentTerminal && isIncomingNonTerminal) {
      return
    }

    const finalStatus = normStatus === 'completed' ? 'done' : normStatus
    tracker.status = finalStatus
    if (message) {
      tracker.message = message
    }
    tracker.timestamp = new Date().toISOString()
    if (name && tracker.attrs) {
      tracker.attrs.name = name
    }

    let statusLevel = tracker.level
    if (normStatus === 'error' || normStatus === 'failed') {
      tracker.level = 'ERROR'
      statusLevel = 'ERROR'
    } else if (normStatus === 'warning' || normStatus === 'warn') {
      tracker.level = 'WARN'
      statusLevel = 'WARN'
      triggerProjectReload()
    } else if (normStatus === 'done' || normStatus === 'completed') {
      if (tracker.level !== 'ERROR' && tracker.level !== 'WARN') {
        tracker.level = 'INFO'
      }
      statusLevel = 'INFO'
      triggerProjectReload()
    }

    // 將狀態變更記錄加入 history
    if (message) {
      tracker.history?.push({
        id: crypto.randomUUID(),
        level: statusLevel,
        message,
        timestamp: new Date().toISOString(),
        attrs: { taskID: taskId, taskStatus: finalStatus, ...(name ? { name } : {}) }
      })
    }
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

  let isListening = false

  // Listen to log and task events from backend
  function startListening(): void {
    if (isListening) return
    isListening = true

    ListenToEvents('log:new', (data) => {
      const entry = extractPayload(data) as LogEntry
      if (entry && entry.level && entry.message) {
        addLog(entry)
      }
    })

    // 同步監聽任務生命週期事件，確保 Log Panel 的 Task Tracker 狀態即時更新
    ListenToEvents('task:created', (data) => {
      const payload = extractPayload(data)
      const id = payload.taskID || payload.id
      if (id) {
        updateTaskStatus(id, payload.status || 'queued', payload.message || 'Queued', payload.name)
      }
    })

    ListenToEvents('task:running', (data) => {
      const payload = extractPayload(data)
      const id = payload.taskID || payload.id
      if (id) {
        updateTaskStatus(id, 'running', payload.message || 'Running...', payload.name)
      }
    })

    ListenToEvents('task:progress', (data) => {
      const payload = extractPayload(data)
      const id = payload.taskID || payload.id
      if (id) {
        updateTaskStatus(id, payload.status || 'running', payload.message)
      }
    })

    ListenToEvents('task:waiting_confirm', (data) => {
      const payload = extractPayload(data)
      const id = payload.taskID || payload.id
      if (id) {
        updateTaskStatus(id, 'waiting_confirm', payload.message || '發現既有版本，等待確認是否覆蓋...')
      }
    })

    ListenToEvents('task:complete', (data) => {
      const payload = extractPayload(data)
      const id = payload.taskID || payload.id
      if (id) {
        updateTaskStatus(id, 'done', payload.message || '任務已完成')
      }
    })

    ListenToEvents('task:failed', (data) => {
      const payload = extractPayload(data)
      const id = payload.taskID || payload.id
      if (id) {
        const isWarning = payload.status === 'warning'
        updateTaskStatus(id, isWarning ? 'warning' : 'error', payload.error || payload.message || '任務失敗')
      }
    })

    ListenToEvents('task:cancelled', (data) => {
      const payload = extractPayload(data)
      const id = payload.taskID || payload.id
      if (id) {
        updateTaskStatus(id, 'cancelled', payload.message || '任務已取消')
      }
    })
  }

  /**
   * 即時更新當前執行中的 AI 對話 Task Tracker 的狀態訊息，使其直接顯示在 Log Panel 最新一行
   * @param message 即時狀態訊息（如「AI 正在分析您的提問...」、「正在執行工具: get_series_overview」）
   * @param taskId 可選指定的 Task ID
   */
  function updateActiveAIChatMessage(message: string, taskId?: string): void {
    if (!message) return

    if (taskId) {
      const tracker = taskMap.get(taskId)
      if (tracker) {
        tracker.message = message
        return
      }
    }

    // 若未指定 taskId 或未由 map 找到，尋找當前 status 為 running 的 AIChat 任務
    for (let i = logs.value.length - 1; i >= 0; i--) {
      const log = logs.value[i]
      if (
        log.isTaskTracker &&
        log.status === 'running' &&
        (log.attrs?.type === 'AIChat' || log.attrs?.name?.includes('AI 對話'))
      ) {
        log.message = message
        return
      }
    }
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
    updateTaskStatus,
    clearLogs,
    setFilterLevel,
    removeLog,
    loadLogs,
    startListening,
    updateActiveAIChatMessage,
  }
})
