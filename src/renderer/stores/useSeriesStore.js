import { create } from 'zustand'

// ========================================
// 系列狀態管理 (Zustand)
// 管理目前開啟的系列資訊與最近開啟記錄
// ========================================

/** 最近開啟記錄的最大數量 */
const MAX_RECENT_FILES = 5

const useSeriesStore = create((set, get) => ({
    // --- 狀態 ---
    /** 是否已開啟系列 */
    isOpen: false,
    /** 目前系列的元資料 (含 bomCount) */
    currentSeries: null,
    /** 目前系列的檔案路徑 */
    currentPath: null,
    /** 最近開啟的系列路徑清單 */
    recentFiles: [],
    /** 是否正在載入中 */
    isLoading: false,
    /** 錯誤訊息 */
    error: null,

    // --- 動作 ---

    /**
     * 初始化最近開啟記錄。
     * 從 settings.json 中讀取先前儲存的路徑清單。
     */
    initRecentFiles: async () => {
        try {
            if (!window.api?.settings) return
            const result = await window.api.settings.get()
            if (result.success && result.data.recentFiles) {
                set({ recentFiles: result.data.recentFiles.slice(0, MAX_RECENT_FILES) })
            }
        } catch (error) {
            console.error('讀取最近開啟記錄失敗:', error)
        }
    },

    /**
     * 建立新系列。
     *
     * @param {string} [description] - 系列描述
     */
    createSeries: async (description) => {
        set({ isLoading: true, error: null })
        try {
            const dialogResult = await window.api.dialog.showSave({
                title: '建立新系列',
                defaultPath: 'untitled.bomix',
            })

            if (dialogResult.canceled || !dialogResult.data) {
                set({ isLoading: false })
                return false
            }

            const filePath = dialogResult.data
            const result = await window.api.series.create(filePath, description || '')

            if (result.success) {
                set({
                    isOpen: true,
                    currentSeries: result.data,
                    currentPath: filePath,
                    isLoading: false,
                })
                await get().addToRecentFiles(filePath)
                return true
            } else {
                set({ error: result.error, isLoading: false })
                return false
            }
        } catch (error) {
            set({ error: error.message, isLoading: false })
            return false
        }
    },

    /**
     * 開啟現有系列。
     *
     * @param {string} [filePath] - 直接指定路徑（跳過對話框）
     */
    openSeries: async (filePath) => {
        set({ isLoading: true, error: null })
        try {
            if (!filePath) {
                const dialogResult = await window.api.dialog.showOpen({
                    title: '開啟系列檔案',
                })

                if (dialogResult.canceled || !dialogResult.data) {
                    set({ isLoading: false })
                    return false
                }

                filePath = dialogResult.data
            }

            const result = await window.api.series.open(filePath)

            if (result.success) {
                set({
                    isOpen: true,
                    currentSeries: result.data,
                    currentPath: filePath,
                    isLoading: false,
                })
                await get().addToRecentFiles(filePath)
                return true
            } else {
                set({ error: result.error, isLoading: false })
                return false
            }
        } catch (error) {
            set({ error: error.message, isLoading: false })
            return false
        }
    },

    /**
     * 關閉目前系列。
     */
    closeSeries: () => {
        set({
            isOpen: false,
            currentSeries: null,
            currentPath: null,
            error: null,
        })
    },

    /**
     * 更新目前系列的描述。
     *
     * @param {string} description - 新的描述
     */
    updateDescription: async (description) => {
        try {
            const result = await window.api.series.updateMeta(description)
            if (result.success) {
                set({ currentSeries: result.data })
            }
            return result
        } catch (error) {
            return { success: false, error: error.message }
        }
    },

    /**
     * 重新命名系列 (檔案)。
     *
     * @param {string} newName - 新的系列名稱 (不含副檔名)
     */
    renameSeries: async (newName) => {
        set({ isLoading: true, error: null })
        try {
            const result = await window.api.series.rename(newName)
            if (result.success) {
                const newPath = result.data.newPath
                
                // 更新路徑與最近開啟記錄
                set({ currentPath: newPath, isLoading: false })

                // 更新 Recent Files (移除舊的，加入新的)
                // 注意: get().currentPath 在這裡還是舊的嗎? 應該用上一步的舊路徑做移除
                // 此處簡化: 直接呼叫 addToRecentFiles 會把新的加到最前。
                // 為了保持乾淨，我們應該先移除舊的。但這邊沒有存舊路徑的變數。
                // 實際上 addToRecentFiles 會處理去重，但這是一個更名操作。
                // 我們應該把舊路徑從 recentFiles 替換成新路徑。
                
                // 修正 Recent Files：先取得舊路徑，再更新狀態
                // 1. 取得舊路徑
                const oldPathForRecent = get().currentPath
                
                // 2. 更新狀態
                set({ currentPath: newPath, isLoading: false })
                
                // 3. 更新 Recent Files
                const currentRecents = get().recentFiles
                const newRecents = currentRecents.map(p => p === oldPathForRecent ? newPath : p)
                // 確保新路徑在最前
                 const finalRecents = [newPath, ...newRecents.filter(p => p !== newPath)].slice(0, MAX_RECENT_FILES)
                 
                set({ recentFiles: finalRecents })
                 
                // 寫入 settings
                try {
                    await window.api.settings.save({ recentFiles: finalRecents })
                } catch (e) {
                   console.error(e)
                }

                return { success: true }
            } else {
                set({ error: result.error, isLoading: false })
                return { success: false, error: result.error }
            }
        } catch (error) {
            set({ error: error.message, isLoading: false })
            return { success: false, error: error.message }
        }
    },

    /**
     * 將路徑加入最近開啟記錄。
     */
    addToRecentFiles: async (filePath) => {
        const current = get().recentFiles
        const updated = [filePath, ...current.filter(p => p !== filePath)]
            .slice(0, MAX_RECENT_FILES)
        set({ recentFiles: updated })

        try {
            await window.api.settings.save({ recentFiles: updated })
        } catch (error) {
            console.error('儲存最近開啟記錄失敗:', error)
        }
    },

    /**
     * 從最近開啟記錄中移除指定路徑。
     */
    removeFromRecentFiles: async (filePath) => {
        const updated = get().recentFiles.filter(p => p !== filePath)
        set({ recentFiles: updated })
        try {
            await window.api.settings.save({ recentFiles: updated })
        } catch (error) {
            console.error('更新最近開啟記錄失敗:', error)
        }
    },

    /** 清除錯誤訊息 */
    clearError: () => set({ error: null }),
}))

export default useSeriesStore
