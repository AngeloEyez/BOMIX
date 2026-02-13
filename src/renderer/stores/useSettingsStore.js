import { create } from 'zustand'

// ========================================
// 設定狀態管理 (Zustand)
// 管理應用程式全域設定，如主題等
// ========================================

const useSettingsStore = create((set, get) => ({
    theme: 'light', // 'light' | 'dark' | 'system' (暫簡化為 light/dark)
    isLoading: true,

    // 初始化設定：從後端讀取
    initSettings: async () => {
        try {
            // 從後端讀取設定
            const result = await window.api.settings.get()
            if (result.success && result.data.theme) {
                set({ theme: result.data.theme })
                get().applyTheme(result.data.theme)
            } else {
                // 預設 Light，或者可偵測系統主題
                set({ theme: 'light' })
                get().applyTheme('light')
            }
        } catch (error) {
            console.error('初始化設定失敗:', error)
        } finally {
            set({ isLoading: false })
        }
    },

    // 切換主題
    toggleTheme: async () => {
        const currentTheme = get().theme
        const newTheme = currentTheme === 'light' ? 'dark' : 'light'

        set({ theme: newTheme })
        get().applyTheme(newTheme)

        // 儲存至後端
        try {
            await window.api.settings.save({ theme: newTheme })
        } catch (error) {
            console.error('儲存設定失敗:', error)
        }
    },

    // 套用主題樣式
    applyTheme: (theme) => {
        const root = window.document.documentElement
        if (theme === 'dark') {
            root.classList.add('dark')
        } else {
            root.classList.remove('dark')
        }
    },
}))

export default useSettingsStore
