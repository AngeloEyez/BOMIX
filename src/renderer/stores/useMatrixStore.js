import { create } from 'zustand'

/**
 * Matrix BOM 狀態管理
 */
const useMatrixStore = create((set, get) => ({
    isLoading: false,
    error: null,

    /**
     * Matrix 資料快取
     * 結構: { [bomRevisionId]: { models: [], selections: [], summary: {} } }
     */
    matrixData: {},

    /**
     * 取得指定 BOM (或多個 BOM) 的 Matrix 資料
     * @param {number|Array<number>} bomRevisionIdOrIds
     * @param {string} cacheKey - 用於儲存資料的鍵值 (通常是 ID 或 Hash)
     */
    fetchMatrixData: async (bomRevisionIdOrIds, cacheKey) => {
        set({ isLoading: true, error: null })
        try {
            const result = await window.api.matrix.getData(bomRevisionIdOrIds)
            if (result.success) {
                // 如果是陣列，cacheKey 通常是 'multi' 或特定的 key，這裡簡化處理
                // 為了支援多 BOM 模式，我們可能需要一個更靈活的結構
                // 暫時使用傳入的 cacheKey 或第一個 ID
                const key = cacheKey || (Array.isArray(bomRevisionIdOrIds) ? 'multi' : bomRevisionIdOrIds);

                set(state => ({
                    matrixData: {
                        ...state.matrixData,
                        [key]: result.data
                    },
                    isLoading: false
                }))
            } else {
                set({ error: result.error, isLoading: false })
            }
        } catch (err) {
            console.error('[MatrixStore] fetch error:', err)
            set({ error: err.message, isLoading: false })
        }
    },

    /**
     * 建立 Matrix Models
     * @param {number} bomRevisionId
     * @param {Array} models
     */
    createModels: async (bomRevisionId, models) => {
        set({ isLoading: true })
        try {
            const result = await window.api.matrix.createModels(bomRevisionId, models)
            if (result.success) {
                await get().fetchMatrixData(bomRevisionId)
            } else {
                set({ error: result.error, isLoading: false })
            }
        } catch (err) {
            set({ error: err.message, isLoading: false })
        }
    },

    /**
     * 刪除 Matrix Model
     * @param {number} bomRevisionId
     * @param {number} modelId
     */
    deleteModel: async (bomRevisionId, modelId) => {
        set({ isLoading: true })
        try {
            const result = await window.api.matrix.deleteModel(modelId)
            if (result.success) {
                await get().fetchMatrixData(bomRevisionId)
            } else {
                set({ error: result.error, isLoading: false })
            }
        } catch (err) {
            set({ error: err.message, isLoading: false })
        }
    },

    /**
     * 更新 Matrix Model
     * @param {number} bomRevisionId
     * @param {number} modelId
     * @param {Object} updates
     */
    updateModel: async (bomRevisionId, modelId, updates) => {
        // Optimistic update is possible but fetch is safer
        try {
            const result = await window.api.matrix.updateModel(modelId, updates)
            if (result.success) {
                await get().fetchMatrixData(bomRevisionId)
            } else {
                set({ error: result.error })
            }
        } catch (err) {
            set({ error: err.message })
        }
    },

    /**
     * 刪除選擇
     * @param {number} bomRevisionId
     * @param {number} matrixModelId
     * @param {string} groupKey
     */
    deleteSelection: async (bomRevisionId, matrixModelId, groupKey) => {
        try {
            const result = await window.api.matrix.deleteSelection(matrixModelId, groupKey)
            if (result.success) {
                await get().fetchMatrixData(bomRevisionId)
            } else {
                set({ error: result.error })
            }
        } catch (err) {
            set({ error: err.message })
        }
    },

    /**
     * 儲存選擇
     * @param {number} bomRevisionId
     * @param {Object} selectionData
     */
    saveSelection: async (bomRevisionId, selectionData) => {
        try {
            // Optimistic update for UI responsiveness could be added here
            const result = await window.api.matrix.saveSelection(selectionData)
            if (result.success) {
                // Background refresh to sync implicit/explicit states if needed
                // Or just update local state if we want to be fancy.
                // For now, fetch to ensure correctness.
                // To avoid flickering, maybe we can silently update?
                const { matrixData } = get()
                const currentData = matrixData[bomRevisionId]

                // Manually update local state to reflect change immediately (Optimistic-ish)
                // Need to find if selection exists and update, or push new
                // This is complex because of 'implicit' conversion to 'explicit'.
                // Simple re-fetch is robust.
                await get().fetchMatrixData(bomRevisionId)
            } else {
                set({ error: result.error })
            }
        } catch (err) {
            set({ error: err.message })
        }
    },

    clearError: () => set({ error: null })
}))

export default useMatrixStore
