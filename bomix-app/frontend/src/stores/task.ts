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

  function updateTask(taskId: string, updates: Partial<Task>): void {
    const index = tasks.value.findIndex(t => t.id === taskId)
    if (index !== -1) {
      tasks.value[index] = { ...tasks.value[index], ...updates }
    } else {
      tasks.value.push({
        id: taskId,
        name: updates.name || 'Import Task',
        type: updates.type || 'Import',
        status: updates.status || 'queued',
        progress: updates.progress || 0,
        message: updates.message || '',
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        ...updates
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
      const payload = data as any
      const id = payload.taskID || payload.id
      if (!id) return
      const existing = getTask(id)
      if (!existing) {
        addTask({
          id,
          name: payload.name || 'Task',
          type: payload.type || 'Import',
          status: payload.status || 'queued',
          progress: Math.round((payload.progress || 0) * 100),
          message: payload.message || 'Queued',
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
        })
      } else {
        updateTask(id, {
          status: payload.status || 'queued',
          updatedAt: new Date().toISOString(),
        })
      }
    })

    ListenToEvents('task:running', (data) => {
      const payload = data as any
      const id = payload.taskID || payload.id
      if (!id) return
      updateTask(id, {
        status: 'running',
        message: payload.message || 'Running...',
        updatedAt: new Date().toISOString(),
      })
    })

    ListenToEvents('task:progress', (data) => {
      const payload = data as any
      const id = payload.taskID || payload.id
      if (!id) return
      const rawProg = payload.progress || 0
      const progressVal = rawProg <= 1.0 ? Math.round(rawProg * 100) : Math.round(rawProg)
      updateTask(id, {
        status: payload.status || 'running',
        progress: progressVal,
        message: payload.message || '',
        updatedAt: new Date().toISOString(),
      })
    })

    ListenToEvents('task:complete', (data) => {
      const payload = data as any
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
      const payload = data as any
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
      const payload = data as any
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
