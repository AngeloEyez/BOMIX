import { create } from 'zustand'

// ========================================
// 設定狀態管理 (Zustand)
// 管理應用程式全域設定，如主題、側邊欄等
// ========================================

const useSettingsStore = create((set, get) => ({
    // --- 狀態 ---
    theme: 'light', // 'light' | 'dark'
    activeThemeId: 'default', // Current active theme ID
    availableThemes: [], // List of available themes
    isLoading: true,

    // 側邊欄設定
    bomSidebarWidth: 250,
    isBomSidebarCollapsed: false,

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

            let targetTheme = 'light'
            let targetThemeId = 'default'
            let sidebarWidth = 250
            let sidebarCollapsed = false

            if (result.success) {
                if (result.data.theme) targetTheme = result.data.theme
                if (result.data.themeId) targetThemeId = result.data.themeId
                if (result.data.bomSidebarWidth) sidebarWidth = result.data.bomSidebarWidth
                if (result.data.isBomSidebarCollapsed) sidebarCollapsed = result.data.isBomSidebarCollapsed
            }

            // 3. 套用設定
            set({
                theme: targetTheme,
                activeThemeId: targetThemeId,
                bomSidebarWidth: sidebarWidth,
                isBomSidebarCollapsed: sidebarCollapsed
            })

            // 套用 Light/Dark 模式
            get().toggleDarkMode(targetTheme === 'dark')

            // 套用主題配色
            await get().applyTheme(targetThemeId)

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

        // Re-apply theme variables for the new mode
        await get().applyTheme(get().activeThemeId)

        // 儲存至後端
        await get().saveSettings()
    },

    // 切換配色主題
    setThemeId: async (themeId) => {
        set({ activeThemeId: themeId })
        await get().applyTheme(themeId)
        await get().saveSettings()
    },

    // 更新並儲存其他設定 (如側邊欄)
    updateSettings: async (newSettings) => {
        // Update local state
        set(newSettings)
        // Persist
        // We use settings property for persistence in saveSettings?
        // Wait, saveSettings reads top-level props: theme, activeThemeId, bomSidebarWidth, isBomSidebarCollapsed.
        // And updateSettings spreads `newSettings` into store.
        // So if newSettings contains `bomSidebarWidth`, it updates store, then saveSettings reads it.
        // This is correct.
        await get().saveSettings()
    },

    // 儲存設定至後端 (Centralized Save)
    saveSettings: async () => {
        try {
            const { theme, activeThemeId, bomSidebarWidth, isBomSidebarCollapsed } = get()
            await window.api.settings.save({
                theme,
                themeId: activeThemeId,
                bomSidebarWidth,
                isBomSidebarCollapsed
            })
        } catch (error) {
            console.error('儲存設定失敗:', error)
        }
    },

    // 內部：切換 DOM class
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
