import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { GetSeriesInfo, CloseSeries } from '../services/api'

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

  // Getters
  const isSeriesOpen = computed(() => isOpen.value)

  // Actions
  async function openSeries(path: string): Promise<void> {
    isLoading.value = true
    error.value = null
    try {
      const info = await GetSeriesInfo(path)
      seriesInfo.value = {
        id: info.id,
        name: info.name,
        description: info.description,
        path,
      }
      isOpen.value = true
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to open series'
      isOpen.value = false
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function createSeries(path: string, name: string, description: string): Promise<void> {
    isLoading.value = true
    error.value = null
    try {
      // This will be implemented when CreateSeries API is ready
      seriesInfo.value = {
        id: 0,
        name,
        description,
        path,
      }
      isOpen.value = true
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to create series'
      isOpen.value = false
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function closeSeries(): Promise<void> {
    try {
      await CloseSeries()
    } catch (err) {
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
