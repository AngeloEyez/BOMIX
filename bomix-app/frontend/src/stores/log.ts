import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { ListenToEvents, GetLogs } from '../services/api'

export interface LogEntry {
  id?: string
  level: string
  message: string
  timestamp: string
  attrs?: Record<string, string>
}

export type LogLevel = 'ALL' | 'DEBUG' | 'INFO' | 'WARN' | 'ERROR'

export const useLogStore = defineStore('log', () => {
  // State
  const logs = ref<LogEntry[]>([])
  const filterLevel = ref<LogLevel>('ALL')

  // Getters
  const filteredLogs = computed(() => {
    if (filterLevel.value === 'ALL') {
      return logs.value
    }
    return logs.value.filter(log => log.level === filterLevel.value)
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

  // Actions
  function addLog(entry: LogEntry): void {
    logs.value.unshift({
      id: entry.id || crypto.randomUUID(),
      level: entry.level,
      message: entry.message,
      timestamp: entry.timestamp,
      attrs: entry.attrs,
    })
  }

  function addLogEntry(level: string, message: string, attrs?: Record<string, string>): void {
    addLog({
      level,
      message,
      timestamp: new Date().toISOString(),
      attrs,
    })
  }

  function clearLogs(): void {
    logs.value = []
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
    // Getters
    filteredLogs,
    errorCount,
    warnCount,
    infoCount,
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
