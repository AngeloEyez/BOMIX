import { create } from 'zustand'

// ========================================
// 應用程式全域狀態管理 (Zustand)
// 管理全域 UI 狀態，如 Loading 狀態等
// (Theme moved to useSettingsStore)
// ========================================

const useAppStore = create((set) => ({
    // --- 資料庫狀態 ---
    /** 資料庫是否忙碌中 (讀寫操作) */
    isDbBusy: false,
    /** 設定資料庫忙碌狀態 */
    setDbBusy: (isBusy) => set({ isDbBusy: isBusy }),
}))

export default useAppStore
