import { create } from 'zustand'

// ========================================
// Settings 狀態管理
// 管理應用程式層級的設定與狀態持久化 (如側邊欄寬度、主題)
// ========================================

const AVAILABLE_THEMES = [
    { id: 'default', name: '預設 (藍色)' },
    { id: 'emerald', name: '翡翠綠' },
    { id: 'purple', name: '紫羅蘭' },
    { id: 'amber', name: '琥珀黃' },
]

const useSettingsStore = create((set, get) => ({
    // --- 狀態 ---
    settings: {
        bomSidebarWidth: 250,
        isBomSidebarCollapsed: false,
        theme: 'light', // 'light' | 'dark'
        activeThemeId: 'default',
    },

    availableThemes: AVAILABLE_THEMES,
    isLoading: false,
    error: null,

    // --- Selectors (Convenience) ---
    // Note: Zustand doesn't have computed props natively in the store definition like Vue/MobX,
    // but components can use selectors. Here we can't easily expose them as properties unless we sync them.
    // However, to satisfy SettingsPage destructuring: const { theme, ... } = useSettingsStore()
    // We can expose getters that proxy to settings, BUT Zustand stores are plain objects.
    // We should probably rely on components strictly using `settings.theme` OR update the store structure.
    // BUT SettingsPage.jsx uses `const { theme } = useSettingsStore()`.
    // This implies `theme` is a top-level property in the store state.
    // Let's flatten the critical UI state or provide hooks.
    // To match existing usage without rewriting components too much:

    // We will sync `theme` and `activeThemeId` to top level for easy access, OR
    // we rewrite `SettingsPage` to use `settings.theme`.
    // Let's rewrite `SettingsPage` accessors slightly later? No, Plan says fix Store.
    // I will expose `theme` and `activeThemeId` as top-level derived state is tricky.
    // I will simply duplicate them at top level for read access, and sync in actions.

    theme: 'light',
    activeThemeId: 'default',

    /**
     * 載入設定
     */
    loadSettings: async () => {
        set({ isLoading: true })
        try {
            const result = await window.api.settings.get()
            if (result.success) {
                const merged = { ...get().settings, ...result.data }
                set({
                    settings: merged,
                    theme: merged.theme || 'light',
                    activeThemeId: merged.activeThemeId || 'default',
                    isLoading: false
                })
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
        const current = get().settings
        const updated = { ...current, ...newSettings }

        // Update State
        set({
            settings: updated,
            // Sync top-level props if they changed
            ...(newSettings.theme !== undefined && { theme: newSettings.theme }),
            ...(newSettings.activeThemeId !== undefined && { activeThemeId: newSettings.activeThemeId }),
        })

        // Persist
        window.api.settings.save(updated).catch(console.error)
    },

    /**
     * 切換主題 (Light/Dark)
     */
    toggleTheme: () => {
        const currentTheme = get().theme
        const newTheme = currentTheme === 'light' ? 'dark' : 'light'
        get().updateSettings({ theme: newTheme })
    },

    /**
     * 設定配色 ID
     */
    setThemeId: (id) => {
        get().updateSettings({ activeThemeId: id })
    },
}))

export default useSettingsStore
