import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { ListenToEvents } from '../services/api'

export interface Task {
  id: string
  name: string
  type: string
  status: string
  progress: number
  message: string
  createdAt: string
  updatedAt: string
  error?: string
}

export const useTaskStore = defineStore('task', () => {
  // State
  const tasks = ref<Task[]>([])

  // Getters
  const activeTasks = computed(() =>
    tasks.value.filter(t => t.status === 'created' || t.status === 'queued' || t.status === 'running')
  )

  const completedTasks = computed(() =>
    tasks.value.filter(t => t.status === 'completed' || t.status === 'failed')
  )

  const runningTasks = computed(() =>
    tasks.value.filter(t => t.status === 'running')
  )

  const queuedTasks = computed(() =>
    tasks.value.filter(t => t.status === 'queued')
  )

  // Actions
  function addTask(task: Task): void {
    tasks.value.push(task)
  }

  /**
   * 將後端各類 Task 狀態字串標準化為前端小寫統一格式
   */
  function normalizeTaskStatus(status?: string): string {
    if (!status) return 'queued'
    const lower = status.toLowerCase()
    if (lower === 'waitingconfirm' || lower === 'waiting_confirm') return 'waiting_confirm'
    if (lower === 'created' || lower === 'queued') return 'queued'
    if (lower === 'running') return 'running'
    if (lower === 'completed' || lower === 'done') return 'completed'
    if (lower === 'warning') return 'warning'
    if (lower === 'failed' || lower === 'error') return 'failed'
    if (lower === 'cancelled') return 'cancelled'
    return lower
  }

  /**
   * 安全解析 Wails v3 事件資料（處理陣列包裝情況）
   */
  function extractPayload(data: any): any {
    if (Array.isArray(data) && data.length > 0) {
      return data[0]
    }
    return data || {}
  }

  /**
   * 更新或新增 Task 狀態
   * 具備終結狀態保護機制：若當前任務已處於終結狀態（completed/failed/warning/cancelled），
   * 則不允許舊的或亂序的 queued/running 事件覆蓋狀態。
   */
  function updateTask(taskId: string, updates: Partial<Task>): void {
    const index = tasks.value.findIndex(t => t.id === taskId)
    const terminalStates = ['completed', 'failed', 'warning', 'cancelled']
    const safeStatus = updates.status ? normalizeTaskStatus(updates.status) : undefined
    
    if (index !== -1) {
      const currentTask = tasks.value[index]
      const isTerminal = terminalStates.includes(currentTask.status)
      const isIncomingNonTerminal = safeStatus && ['queued', 'running', 'created'].includes(safeStatus)

      // 若目前任務已完成/失敗/警告/取消，且傳入的新狀態為非終結狀態，則忽略 status 的更新
      if (isTerminal && isIncomingNonTerminal) {
        const { status, ...safeUpdates } = updates
        tasks.value[index] = { ...tasks.value[index], ...safeUpdates }
        return
      }
      tasks.value[index] = {
        ...tasks.value[index],
        ...updates,
        ...(safeStatus ? { status: safeStatus } : {})
      }
    } else {
      tasks.value.push({
        id: taskId,
        name: updates.name || 'Import Task',
        type: updates.type || 'Import',
        status: safeStatus || 'queued',
        progress: updates.progress || 0,
        message: updates.message || '',
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        ...updates,
        ...(safeStatus ? { status: safeStatus } : {})
      })
    }
  }

  function removeTask(taskId: string): void {
    const index = tasks.value.findIndex(t => t.id === taskId)
    if (index !== -1) {
      tasks.value.splice(index, 1)
    }
  }

  function clearAllTasks(): void {
    tasks.value = []
  }

  function clearCompletedTasks(): void {
    tasks.value = tasks.value.filter(t =>
      t.status !== 'completed' && t.status !== 'failed'
    )
  }

  function getTask(taskId: string): Task | undefined {
    return tasks.value.find(t => t.id === taskId)
  }

  // Listen to task events from backend
  let isListening = false

  function startListening(): void {
    if (isListening) return
    isListening = true

    ListenToEvents('task:created', (data) => {
      const payload = extractPayload(data)
      const id = payload.taskID || payload.id
      if (!id) return
      const existing = getTask(id)
      if (!existing) {
        addTask({
          id,
          name: payload.name || 'Task',
          type: payload.type || 'Import',
          status: normalizeTaskStatus(payload.status || 'queued'),
          progress: Math.round((payload.progress || 0) * 100),
          message: payload.message || 'Queued',
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
        })
      } else {
        updateTask(id, {
          status: normalizeTaskStatus(payload.status || 'queued'),
          updatedAt: new Date().toISOString(),
        })
      }
    })

    ListenToEvents('task:running', (data) => {
      const payload = extractPayload(data)
      const id = payload.taskID || payload.id
      if (!id) return
      updateTask(id, {
        status: 'running',
        message: payload.message || 'Running...',
        updatedAt: new Date().toISOString(),
      })
    })

    ListenToEvents('task:waiting_confirm', (data) => {
      const payload = extractPayload(data)
      const id = payload.taskID || payload.id
      if (!id) return
      updateTask(id, {
        status: 'waiting_confirm',
        message: payload.message || '發現既有版本，等待確認是否覆蓋...',
        updatedAt: new Date().toISOString(),
      })
    })

    ListenToEvents('task:progress', (data) => {
      const payload = extractPayload(data)
      const id = payload.taskID || payload.id
      if (!id) return
      const existing = getTask(id)
      // 若任務已屬於終結狀態，忽略後續的進度推送
      if (existing && ['completed', 'failed', 'warning', 'cancelled'].includes(existing.status)) {
        return
      }
      const rawProg = payload.progress || 0
      const progressVal = rawProg <= 1.0 ? Math.round(rawProg * 100) : Math.round(rawProg)
      const newStatus = payload.status ? normalizeTaskStatus(payload.status) : (existing?.status || 'running')
      updateTask(id, {
        status: newStatus,
        progress: progressVal,
        message: payload.message || '',
        updatedAt: new Date().toISOString(),
      })
    })

    ListenToEvents('task:complete', (data) => {
      const payload = extractPayload(data)
      const id = payload.taskID || payload.id
      if (!id) return
      updateTask(id, {
        status: 'completed',
        progress: 100,
        message: payload.message || 'Completed',
        updatedAt: new Date().toISOString(),
      })
    })

    ListenToEvents('task:failed', (data) => {
      const payload = extractPayload(data)
      const id = payload.taskID || payload.id
      if (!id) return
      const status = payload.status === 'warning' ? 'warning' : 'failed'
      updateTask(id, {
        status,
        error: payload.error || 'Task Failed',
        message: payload.error || 'Task Failed',
        updatedAt: new Date().toISOString(),
      })
    })

    ListenToEvents('task:cancelled', (data) => {
      const payload = extractPayload(data)
      const id = payload.taskID || payload.id
      if (!id) return
      updateTask(id, {
        status: 'cancelled',
        message: 'Cancelled',
        updatedAt: new Date().toISOString(),
      })
    })
  }

  return {
    // State
    tasks,
    // Getters
    activeTasks,
    completedTasks,
    runningTasks,
    queuedTasks,
    // Actions
    addTask,
    updateTask,
    removeTask,
    clearAllTasks,
    clearCompletedTasks,
    getTask,
    startListening,
  }
})
