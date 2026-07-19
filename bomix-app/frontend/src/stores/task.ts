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
  function startListening(): void {
    ListenToEvents('task:progress', (data) => {
      const task = data as Task
      updateTask(task.id, {
        status: task.status,
        progress: task.progress,
        message: task.message,
        updatedAt: task.updatedAt,
      })
    })

    ListenToEvents('task:complete', (data) => {
      const task = data as Task
      updateTask(task.id, {
        status: 'completed',
        progress: 100,
        message: task.message || 'Completed',
        updatedAt: task.updatedAt,
      })
    })

    ListenToEvents('task:failed', (data) => {
      const task = data as Task
      updateTask(task.id, {
        status: 'failed',
        error: task.error || 'Unknown error',
        updatedAt: task.updatedAt,
      })
    })

    ListenToEvents('task:cancelled', (data) => {
      const task = data as Task
      updateTask(task.id, {
        status: 'cancelled',
        message: 'Cancelled',
        updatedAt: task.updatedAt,
      })
    })

    ListenToEvents('task:created', (data) => {
      const task = data as Task
      addTask(task)
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
