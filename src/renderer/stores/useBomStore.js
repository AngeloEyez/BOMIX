import { create } from 'zustand'
import useAppStore from './useAppStore'

// ========================================
// BOM 狀態管理 (Zustand)
// 管理 BOM 頁面的專案/版本選取、聚合資料與 CRUD 操作
// ========================================

const useBomStore = create((set, get) => ({
    // --- 狀態 ---
    /** 目前選取的專案 ID */
    selectedProjectId: null,
    /** 該專案下的 BOM Revision 列表 */
    revisions: [],
    /** 目前選取的 BOM Revision ID */
    selectedRevisionId: null,
    /** 目前選取的 BOM Revision 物件 */
    selectedRevision: null,
    /** 聚合 BOM 資料 (Main Items + 2nd Sources) */
    bomView: [],
    /** 是否正在載入中 */
    isLoading: false,
    /** 錯誤訊息 */
    error: null,

    // --- 動作 ---

    /**
     * 選擇專案並載入其 BOM 版本列表。
     *
     * @param {number} projectId - 專案 ID
     */
    selectProject: async (projectId) => {
        const { setDbBusy } = useAppStore.getState()
        set({
            selectedProjectId: projectId,
            selectedRevisionId: null,
            selectedRevision: null,
            bomView: [],
            isLoading: true,
            error: null,
        })
        setDbBusy(true)
        try {
            const result = await window.api.bom.getRevisions(projectId)
            if (result.success) {
                set({ revisions: result.data, isLoading: false })
            } else {
                set({ error: result.error, isLoading: false })
            }
        } catch (error) {
            set({ error: error.message, isLoading: false })
        } finally {
            setDbBusy(false)
        }
    },

    /**
     * 選擇 BOM 版本並載入聚合視圖。
     *
     * @param {number} revisionId - BOM Revision ID
     */
    selectRevision: async (revisionId) => {
        const { setDbBusy } = useAppStore.getState()
        const revision = get().revisions.find(r => r.id === revisionId) || null
        set({
            selectedRevisionId: revisionId,
            selectedRevision: revision,
            isLoading: true,
            error: null,
        })
        setDbBusy(true)
        try {
            const result = await window.api.bom.getView(revisionId)
            if (result.success) {
                set({ bomView: result.data, isLoading: false })
            } else {
                set({ error: result.error, isLoading: false })
            }
        } catch (error) {
            set({ error: error.message, isLoading: false })
        } finally {
            setDbBusy(false)
        }
    },

    /**
     * 重新載入目前選取的 BOM 版本視圖。
     */
    reloadBomView: async () => {
        const revisionId = get().selectedRevisionId
        if (!revisionId) return
        await get().selectRevision(revisionId)
    },

    /**
     * 重新載入 Revision 列表 (不變更選取狀態)。
     */
    reloadRevisions: async () => {
        const { setDbBusy } = useAppStore.getState()
        const projectId = get().selectedProjectId
        if (!projectId) return
        setDbBusy(true)
        try {
            const result = await window.api.bom.getRevisions(projectId)
            if (result.success) {
                set({ revisions: result.data })
            }
        } catch (error) {
            console.error('重新載入 Revision 失敗:', error)
        } finally {
            setDbBusy(false)
        }
    },

    /**
     * 刪除 BOM 版本。
     *
     * @param {number} revisionId - BOM Revision ID
     * @returns {Promise<{success: boolean, error?: string}>}
     */
    deleteBom: async (revisionId) => {
        const { setDbBusy } = useAppStore.getState()
        set({ error: null })
        setDbBusy(true)
        try {
            const result = await window.api.bom.delete(revisionId)
            if (result.success) {
                // 若刪除的是目前選取的版本，清除狀態
                const state = get()
                if (state.selectedRevisionId === revisionId) {
                    set({
                        selectedRevisionId: null,
                        selectedRevision: null,
                        bomView: [],
                    })
                }
                // 重新載入列表
                await get().reloadRevisions()
                return { success: true }
            } else {
                set({ error: result.error })
                return { success: false, error: result.error }
            }
        } catch (error) {
            set({ error: error.message })
            return { success: false, error: error.message }
        } finally {
            setDbBusy(false)
        }
    },

    /**
     * 匯入 Excel BOM。
     *
     * @param {string} filePath - Excel 檔案路徑
     * @param {number} projectId - 專案 ID
     * @param {string} phaseName - Phase 名稱
     * @param {string} version - 版本號
     * @returns {Promise<{success: boolean, data?: Object, error?: string}>}
     */
    importExcel: async (filePath, projectId, phaseName, version) => {
        const { setDbBusy } = useAppStore.getState()
        set({ isLoading: true, error: null })
        setDbBusy(true)
        try {
            const result = await window.api.excel.import(filePath, projectId, phaseName, version)
            if (result.success) {
                // 重新載入 Revision 列表
                await get().reloadRevisions()
                // 自動選取新匯入的版本
                if (result.data?.bomRevisionId) {
                    await get().selectRevision(result.data.bomRevisionId)
                }
                set({ isLoading: false })
                return { success: true, data: result.data }
            } else {
                set({ error: result.error, isLoading: false })
                return { success: false, error: result.error }
            }
        } catch (error) {
            set({ error: error.message, isLoading: false })
            return { success: false, error: error.message }
        } finally {
            setDbBusy(false)
        }
    },

    /**
     * 匯出 BOM 為 Excel。
     *
     * @param {number} revisionId - BOM Revision ID
     * @returns {Promise<{success: boolean, error?: string}>}
     */
    exportExcel: async (revisionId) => {
        const { setDbBusy } = useAppStore.getState()
        try {
            // 開啟儲存對話框
            const dialogResult = await window.api.dialog.showSave({
                title: '匯出 BOM Excel',
                defaultPath: 'BOM_Export.xlsx',
            })
            if (dialogResult.canceled || !dialogResult.data) {
                return { success: false, error: '使用者取消' }
            }

            set({ isLoading: true, error: null })
            setDbBusy(true)
            const result = await window.api.excel.export(revisionId, dialogResult.data)
            set({ isLoading: false })

            if (result.success) {
                return { success: true }
            } else {
                set({ error: result.error })
                return { success: false, error: result.error }
            }
        } catch (error) {
            set({ error: error.message, isLoading: false })
            return { success: false, error: error.message }
        } finally {
            setDbBusy(false)
        }
    },

    /** 清空所有狀態（關閉系列時呼叫） */
    reset: () => {
        set({
            selectedProjectId: null,
            revisions: [],
            selectedRevisionId: null,
            selectedRevision: null,
            bomView: [],
            isLoading: false,
            error: null,
        })
    },

    /** 清除錯誤訊息 */
    clearError: () => set({ error: null }),
}))

export default useBomStore
