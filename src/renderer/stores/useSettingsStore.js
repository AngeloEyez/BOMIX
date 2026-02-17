import { create } from 'zustand'

// ========================================
// Settings 狀態管理
// 管理應用程式層級的設定與狀態持久化 (如側邊欄寬度)
// ========================================

const useSettingsStore = create((set, get) => ({
    // 預設設定
    settings: {
        bomSidebarWidth: 250,
        isBomSidebarCollapsed: false,
    },

    isLoading: false,
    error: null,

    /**
     * 載入設定
     */
    loadSettings: async () => {
        set({ isLoading: true })
        try {
            const result = await window.api.settings.get()
            if (result.success) {
                set(state => ({
                    settings: { ...state.settings, ...result.data },
                    isLoading: false
                }))
            }
        } catch (error) {
            console.error('Failed to load settings:', error)
            set({ isLoading: false })
        }
    },

    /**
     * 更新設定並持久化
     * @param {Object} newSettings
     */
    updateSettings: async (newSettings) => {
        set(state => {
            const updated = { ...state.settings, ...newSettings }
            // 非同步儲存
            window.api.settings.save(updated).catch(console.error)
            return { settings: updated }
        })
    },
}))

export default useSettingsStore
