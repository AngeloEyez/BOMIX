import { create } from 'zustand'
import { getDefaults, getPersistKeys } from '../config/settingsConfig'

// ========================================
// 設定狀態管理 (Zustand)
// 管理應用程式全域設定，如主題、側邊欄等
//
// ⚠️ 設定的預設值與項目定義，請至 src/renderer/config/settingsConfig.js 修改
// ========================================

// 從 settingsConfig 取得所有設定的預設值（單一真實來源）
const SETTING_DEFAULTS = getDefaults()

const useSettingsStore = create((set, get) => ({
    // --- 狀態（設定初始值從 settingsConfig.js 統一管理）---
    ...SETTING_DEFAULTS,

    // 可用主題列表（動態從後端取得，不在 settingsConfig 中定義）
    availableThemes: [],
    isLoading: true,

    // 側邊欄設定（純 UI 狀態，不透過 settingsConfig 管理）
    bomSidebarWidth: 250,

    // 初始化設定：從後端讀取
    initSettings: async () => {
        set({ isLoading: true })
        try {
            // 1. 讀取可用主題列表
            const themesResult = await window.api.theme.getList()
            if (themesResult.success) {
                set({ availableThemes: themesResult.data })
            }

            // 2. 從後端讀取使用者設定
            const result = await window.api.settings.get()

            if (result.success && result.data) {
                // 僅套用有在 settingsConfig 中定義的 key，避免未知欄位污染 Store
                const persistKeys = getPersistKeys()
                const loadedSettings = {}
                for (const key of persistKeys) {
                    if (result.data[key] !== undefined) {
                        loadedSettings[key] = result.data[key]
                    }
                }
                // 側邊欄設定（不在 settingsConfig 中的額外欄位）
                if (result.data.bomSidebarWidth) loadedSettings.bomSidebarWidth = result.data.bomSidebarWidth

                set(loadedSettings)
            }

            // 3. 套用主題效果
            const { theme, activeThemeId } = get()
            get().toggleDarkMode(theme === 'dark')
            await get().applyTheme(activeThemeId)

        } catch (error) {
            console.error('初始化設定失敗:', error)
        } finally {
            set({ isLoading: false })
        }
    },

    // Alias for compatibility
    loadSettings: async () => {
        await get().initSettings()
    },

    // 切換 Light/Dark 模式
    toggleTheme: async () => {
        const currentTheme = get().theme
        const newTheme = currentTheme === 'light' ? 'dark' : 'light'

        set({ theme: newTheme })
        get().toggleDarkMode(newTheme === 'dark')

        // 重新套用色彩主題變數
        await get().applyTheme(get().activeThemeId)
        await get().saveSettings()
    },

    // 切換配色主題
    setThemeId: async (themeId) => {
        set({ activeThemeId: themeId })
        await get().applyTheme(themeId)
        await get().saveSettings()
    },

    // 更新並儲存通用設定（如側邊欄寬度）
    updateSettings: async (newSettings) => {
        set(newSettings)
        await get().saveSettings()
    },

    /**
     * 還原所有設定至 settingsConfig.js 中定義的預設值。
     * 會立即更新 Store 並重新套用主題效果，最後持久化儲存。
     */
    resetToDefaults: async () => {
        const defaults = getDefaults()
        set(defaults)

        // 重新套用預設主題效果
        get().toggleDarkMode(defaults.theme === 'dark')
        await get().applyTheme(defaults.activeThemeId)
        await get().saveSettings()
    },

    // 儲存設定至後端（自動根據 settingsConfig 中 persist:true 的欄位）
    saveSettings: async () => {
        try {
            const state = get()
            const persistKeys = getPersistKeys()

            // 從 Store 取出所有需持久化的設定值
            const payload = persistKeys.reduce((acc, key) => {
                acc[key] = state[key]
                return acc
            }, {})

            // 加入不在 settingsConfig 的側邊欄設定
            payload.bomSidebarWidth = state.bomSidebarWidth

            // 後端 API 使用 themeId 作為 activeThemeId 的鍵名
            if (payload.activeThemeId !== undefined) {
                payload.themeId = payload.activeThemeId
                delete payload.activeThemeId
            }

            await window.api.settings.save(payload)
        } catch (error) {
            console.error('儲存設定失敗:', error)
        }
    },

    // 內部：切換 DOM class 以套用 dark mode
    toggleDarkMode: (isDark) => {
        const root = window.document.documentElement
        if (isDark) {
            root.classList.add('dark')
        } else {
            root.classList.remove('dark')
        }
    },

    // 內部：套用主題 CSS 變數
    applyTheme: async (themeId) => {
        try {
            const mode = get().theme
            const result = await window.api.theme.getAttributes(themeId)

            if (result.success && result.data && result.data.colors) {
                const allColors = result.data.colors
                const { light, dark, ...baseColors } = allColors
                const activeModeColors = mode === 'dark' ? (dark || {}) : (light || {})
                const finalColors = { ...baseColors, ...activeModeColors }

                let css = ':root {\n'
                for (const [key, value] of Object.entries(finalColors)) {
                    if (typeof value === 'object') continue
                    css += `  ${key}: ${value};\n`
                }
                css += '}\n'

                let styleTag = document.getElementById('theme-style')
                if (!styleTag) {
                    styleTag = document.createElement('style')
                    styleTag.id = 'theme-style'
                    document.head.appendChild(styleTag)
                }
                styleTag.textContent = css
            }
        } catch (error) {
            console.error(`套用主題 ${themeId} 失敗:`, error)
        }
    },
}))

export default useSettingsStore
