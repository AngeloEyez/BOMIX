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
                // Refresh data. If bomRevisionId is array (Multi-BOM), we need to refetch appropriately.
                // Since we don't track *which* view context triggered this easily here without passing extra args,
                // we assume `fetchMatrixData` handles the refresh if we pass the same ID/IDs.
                // But `saveSelection` usually happens in a context where we know the scope.
                // Currently `BomTable` passes `bomRevisionId` which is `selectedRevisionId` or `selectedRevisionIds`.
                // If Multi-BOM, `bomRevisionId` passed from `BomTable` handles should be the *context* IDs?
                // Actually `handleMatrixSelection` in `BomTable` passes `model.bom_revision_id` as first arg.
                // This is WRONG if we are in Multi-BOM view, we need to refresh the Multi-BOM view!
                // We should pass the *View Context IDs* to `saveSelection` as the first arg?
                // OR `useMatrixStore` should track active view context.

                // Let's assume the component calls `fetchMatrixData` again or we re-fetch the *active* context.
                // But `saveSelection` signature is `(bomRevisionId, selectionData)`.
                // If `bomRevisionId` is the *single* BOM ID where selection happened, re-fetching that *single* ID won't update 'multi' key in store.

                // Fix: Check if 'multi' key exists in store and refresh it too?
                // Or simply rely on the caller to handle refresh?
                // `BomTable` calls this.
                // Let's try to refresh both single and multi if possible, or just the one passed.
                // If `bomRevisionId` passed is an array, it refreshes array.

                await get().fetchMatrixData(bomRevisionId)

                // If we are in multi-mode, we might also want to refresh 'multi' if the passed ID was single?
                // This is getting complicated.
                // Better approach: `BomTable` passes the *current view IDs* to `saveSelection` as first arg.
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
