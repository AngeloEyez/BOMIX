import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { GetSeriesInfo, CloseSeries, OpenSeries, CreateSeries } from '../services/api'
import { useLogStore } from './log'

export interface SeriesInfo {
  id: number
  name: string
  description: string
  path: string
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

  return {
    // State
    isOpen,
    seriesInfo,
    isLoading,
    error,
    currentTheme,
    // Getters
    isSeriesOpen,
    // Actions
    openSeries,
    createSeries,
    closeSeries,
    clearError,
    applyTheme,
  }
})
