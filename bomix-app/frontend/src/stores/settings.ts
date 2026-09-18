/**
 * @file settings.ts
 * @description 系統設定集中管理 Store，落實「單一真實來源 (SSOT)」架構。
 * 所有設定預設值統一自後端 GetDefaultSettings() (來自 defaults.go) 取得，
 * 嚴禁在前端任何元件或 Store 中硬編碼預設值與 fallback。
 */

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { GetSettings, GetDefaultSettings, UpdateSettings, type Settings, type ImportSettings, type AISettings } from '../services/api'
import { useAppStore } from './app'
import { useLogStore } from './log'
import { useAIChatStore } from './aiChat'

export const useSettingsStore = defineStore('settings', () => {
  // 後端 defaults.go 定義之唯一真實預設值
  const defaults = ref<Settings | null>(null)
  // 當前有效的設定值
  const currentSettings = ref<Settings | null>(null)
  // 是否已自後端成功載入設定
  const isLoaded = ref<boolean>(false)

  /**
   * 深層合併設定物件，確保任何在後端存檔中缺漏的欄位，均以 defaults 補齊
   * @param current 當前或後端回傳之設定
   * @param fallback 預設值物件 (SSOT)
   */
  function mergeWithDefaults(current: Partial<Settings> | null, fallback: Settings): Settings {
    if (!current) {
      return JSON.parse(JSON.stringify(fallback))
    }

    return {
      theme: current.theme || fallback.theme,
      autoOpenLastFile: current.autoOpenLastFile !== undefined ? current.autoOpenLastFile : fallback.autoOpenLastFile,
      lastOpenedFile: current.lastOpenedFile !== undefined ? current.lastOpenedFile : fallback.lastOpenedFile,
      autoImportPreviousMatrix: current.autoImportPreviousMatrix !== undefined ? current.autoImportPreviousMatrix : fallback.autoImportPreviousMatrix,
      import: {
        confirmOverwrite: current.import?.confirmOverwrite !== undefined
          ? current.import.confirmOverwrite
          : fallback.import.confirmOverwrite,
        autoImportPreviousMatrix: current.import?.autoImportPreviousMatrix !== undefined
          ? current.import.autoImportPreviousMatrix
          : (current.autoImportPreviousMatrix !== undefined ? current.autoImportPreviousMatrix : fallback.import.autoImportPreviousMatrix),
      },
      logger: {
        level: current.logger?.level || fallback.logger.level,
        maxEntries: current.logger?.maxEntries || fallback.logger.maxEntries,
      },
      recentFiles: {
        maxRecentFiles: current.recentFiles?.maxRecentFiles || fallback.recentFiles.maxRecentFiles,
        recentFiles: current.recentFiles?.recentFiles || fallback.recentFiles.recentFiles,
      },
      ai: {
        enabled: current.ai?.enabled !== undefined ? current.ai.enabled : (fallback.ai?.enabled ?? false),
        baseUrl: current.ai?.baseUrl !== undefined ? current.ai.baseUrl : (fallback.ai?.baseUrl || ''),
        apiKey: current.ai?.apiKey !== undefined ? current.ai.apiKey : (fallback.ai?.apiKey || ''),
        model: current.ai?.model !== undefined ? current.ai.model : (fallback.ai?.model || ''),
        temperature: current.ai?.temperature !== undefined ? current.ai.temperature : (fallback.ai?.temperature ?? 0.1),
        maxTokens: current.ai?.maxTokens !== undefined ? current.ai.maxTokens : (fallback.ai?.maxTokens ?? 4096),
        timeout: current.ai?.timeout !== undefined ? current.ai.timeout : (fallback.ai?.timeout ?? 60),
        language: current.ai?.language || fallback.ai?.language || 'zh-TW',
      },
    }
  }

  /**
   * 分發設定副作用至全域 stores (主題、日誌、AI 開啟狀態等)
   */
  function dispatchSideEffects(s: Settings): void {
    const appStore = useAppStore()
    const logStore = useLogStore()
    const aiChatStore = useAIChatStore()

    if (s.theme) {
      appStore.applyTheme(s.theme)
    }
    if (s.logger?.level) {
      logStore.globalLogLevel = s.logger.level
    }
    if (s.import) {
      appStore.confirmOverwrite = s.import.confirmOverwrite
    }
    if (s.ai) {
      aiChatStore.isEnabled = s.ai.enabled
      if (s.ai.model) {
        aiChatStore.currentModel = s.ai.model
      }
    }
  }

  /**
   * 初始化載入全域設定
   */
  async function initSettings(): Promise<Settings | null> {
    try {
      // 1. 同時取得後端 SSOT 預設值與當前設定
      const [backendDefaults, backendCurrent] = await Promise.all([
        GetDefaultSettings(),
        GetSettings(),
      ])

      if (backendDefaults) {
        defaults.value = backendDefaults
      }

      if (defaults.value) {
        const merged = mergeWithDefaults(backendCurrent, defaults.value)
        currentSettings.value = merged
        dispatchSideEffects(merged)
        isLoaded.value = true
        return merged
      }

      return null
    } catch (err) {
      console.error('Failed to init settings:', err)
      return null
    }
  }

  /**
   * 儲存設定至後端並同步更新本地狀態與副作用
   */
  async function saveSettings(newSettings: Settings): Promise<void> {
    try {
      await UpdateSettings(newSettings)
      currentSettings.value = JSON.parse(JSON.stringify(newSettings))
      dispatchSideEffects(newSettings)
    } catch (err) {
      console.error('Failed to save settings:', err)
      throw err
    }
  }

  /**
   * 提供給其他 Store 更新 Import 部分設定的專用方法 (自動以目前值或預設值補齊其餘欄位)
   */
  async function updateImportSettings(partial: Partial<ImportSettings>): Promise<void> {
    if (!currentSettings.value && defaults.value) {
      currentSettings.value = JSON.parse(JSON.stringify(defaults.value))
    }
    if (!currentSettings.value) {
      await initSettings()
    }
    if (!currentSettings.value) return

    const updated: Settings = {
      ...currentSettings.value,
      import: {
        ...currentSettings.value.import,
        ...partial,
      },
    }
    if (partial.autoImportPreviousMatrix !== undefined) {
      updated.autoImportPreviousMatrix = partial.autoImportPreviousMatrix
    }
    await saveSettings(updated)
  }

  /**
   * 提供給其他 Store 更新 AI 部分設定的專用方法
   */
  async function updateAISettings(partial: Partial<AISettings>): Promise<void> {
    if (!currentSettings.value && defaults.value) {
      currentSettings.value = JSON.parse(JSON.stringify(defaults.value))
    }
    if (!currentSettings.value) {
      await initSettings()
    }
    if (!currentSettings.value || !currentSettings.value.ai) return

    const updated: Settings = {
      ...currentSettings.value,
      ai: {
        ...currentSettings.value.ai,
        ...partial,
      },
    }
    await saveSettings(updated)
  }

  return {
    defaults,
    currentSettings,
    isLoaded,
    initSettings,
    saveSettings,
    updateImportSettings,
    updateAISettings,
    mergeWithDefaults,
  }
})
