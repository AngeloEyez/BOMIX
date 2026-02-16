import { create } from 'zustand'

// ========================================
// 設定狀態管理 (Zustand)
// 管理應用程式全域設定，如主題等
// ========================================

const useSettingsStore = create((set, get) => ({
    theme: 'light', // 'light' | 'dark'
    activeThemeId: 'default', // Current active theme ID
    availableThemes: [], // List of available themes
    isLoading: true,

    // 初始化設定：從後端讀取
    initSettings: async () => {
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

            if (result.success) {
                if (result.data.theme) targetTheme = result.data.theme
                if (result.data.themeId) targetThemeId = result.data.themeId
            }

            // 3. 套用設定
            set({ theme: targetTheme, activeThemeId: targetThemeId })
            
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

    // 切換 Light/Dark 模式
    toggleTheme: async () => {
        const currentTheme = get().theme
        const newTheme = currentTheme === 'light' ? 'dark' : 'light'

        set({ theme: newTheme })
        get().toggleDarkMode(newTheme === 'dark')
        
        // Re-apply theme variables for the new mode
        // Wait for state update to complete implicitly or fetch from get() inside
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

    // 儲存設定至後端
    saveSettings: async () => {
        try {
            const { theme, activeThemeId } = get()
            await window.api.settings.save({ 
                theme, 
                themeId: activeThemeId 
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
            // Need current theme mode (light/dark)
            const mode = get().theme

            const result = await window.api.theme.getAttributes(themeId)
            
            if (result.success && result.data && result.data.colors) {
                const allColors = result.data.colors
                
                // Separate common colors vs mode-specific
                // Structure: { ...baseColors, light: { ... }, dark: { ... } }
                
                const { light, dark, ...baseColors } = allColors
                
                // Merge base + mode specific
                const activeModeColors = mode === 'dark' ? (dark || {}) : (light || {})
                const finalColors = { ...baseColors, ...activeModeColors }
                
                // 建構 CSS 變數內容
                let css = ':root {\n'
                for (const [key, value] of Object.entries(finalColors)) {
                    // Skip nested objects just in case (though destructured above)
                    if (typeof value === 'object') continue
                    css += `  ${key}: ${value};\n`
                }
                css += '}\n'

                // 注入 style tag
                let styleTag = document.getElementById('theme-style')
                if (!styleTag) {
                    styleTag = document.createElement('style')
                    styleTag.id = 'theme-style'
                    document.head.appendChild(styleTag)
                }
                styleTag.textContent = css
                console.log(`[SettingsStore] Applied theme: ${themeId} (${mode} mode)`)
            }
        } catch (error) {
            console.error(`套用主題 ${themeId} 失敗:`, error)
        }
    },
}))

export default useSettingsStore
