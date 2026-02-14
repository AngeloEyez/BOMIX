import { create } from 'zustand'

// ========================================
// 應用程式全域狀態管理 (Zustand)
// 管理全域 UI 狀態，如主題、Loading 狀態等
// ========================================

const useAppStore = create((set) => ({
    // --- 主題設定 ---
    isDarkMode: false,
    toggleTheme: () => set((state) => ({ isDarkMode: !state.isDarkMode })),

    // --- 資料庫狀態 ---
    /** 資料庫是否忙碌中 (讀寫操作) */
    isDbBusy: false,
    /** 設定資料庫忙碌狀態 */
    setDbBusy: (isBusy) => set({ isDbBusy: isBusy }),
}))

export default useAppStore
