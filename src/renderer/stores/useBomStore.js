import { create } from 'zustand'
import useAppStore from './useAppStore'
import useToastStore from './useToastStore'

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
    /** 目前選取的 BOM Revision ID (單選) - 用於傳統 BOM 檢視 */
    selectedRevisionId: null,
    /** 目前選取的 BOM Revision ID 集合 (多選) - 用於 Matrix/BigBOM */
    selectedRevisionIds: new Set(),
    /** 目前選取的 BOM Revision 物件 */
    selectedRevision: null,
    /** BOM 頁面模式: 'BOM' | 'MATRIX' | 'BIGBOM' */
    bomMode: 'BOM',
    /** 目前選取的視圖 ID */
    currentViewId: 'all_view',
    /** BOM 檢視資料 */
    bomView: [],
    /** 視圖資料快取 { [viewId]: data[] } */
    viewCache: {},
    /**
     * CCL Filter 狀態（使用者切換，跨 View 和頁面保留）
     * true 表示勾選，將向後端加入 ccl=Y 的 filter 條件
     */
    cclFilter: false,

    /**
     * 設定 BOM 模式
     * @param {'BOM'|'MATRIX'|'BIGBOM'} mode
     */
    setBomMode: (mode) => set({ bomMode: mode }),

    /**
     * 切換視圖
     * 根據 View 的預設 filters 與目前的 cclFilter 狀態組裝完整條件陣列。
     * @param {string} viewId
     * @param {Object} [viewsMap] - View 定義對照表（若不傳入則從 BomPage 的 views state 取得）
     */
    selectView: async (viewId, viewsMap = null) => {
        const { selectedRevisionIds, cclFilter } = get()
        if (selectedRevisionIds.size === 0) return

        // 立即更新 UI
        set({ currentViewId: viewId })

        set({ isLoading: true, error: null })
        const { setDbBusy } = useAppStore.getState()
        setDbBusy(true)

        try {
            const idsArray = Array.from(selectedRevisionIds)

            // 從 View 定義取得預設 filters
            // viewsMap 如果未傳入則透過 IPC 即時取得
            let viewDef = viewsMap ? viewsMap[Object.keys(viewsMap).find(k => viewsMap[k].id === viewId)] : null
            let baseFilters = viewDef?.filters || []

            // 如果視圖未在 viewsMap 中找到，透過 API 取得 filters
            if (baseFilters.length === 0) {
                const viewsResult = await window.api.bom.getViews()
                if (viewsResult.success) {
                    const viewsObj = viewsResult.data
                    const foundView = Object.values(viewsObj).find(v => v.id === viewId)
                    baseFilters = foundView?.filters || []
                }
            }

            // 組裝最終 filters：View 預設 + CCL checkbox 額外條件
            const extraFilters = cclFilter ? [{ field: 'ccl', operator: 'eq', value: 'Y' }] : []
            const finalFilters = [...baseFilters, ...extraFilters]

            const result = await window.api.bom.query(idsArray, finalFilters, {})
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
     * 選擇專案並載入其 BOM 版本列表。
     */
    selectProject: async (projectId) => {
        const { setDbBusy } = useAppStore.getState()
        set({
            selectedProjectId: projectId,
            selectedRevisionId: null,
            selectedRevisionIds: new Set(),
            selectedRevision: null,
            bomView: [],
            currentViewId: 'all_view',
            viewCache: {},
            isLoading: true,
            error: null,
        })
        // Revisions loading happens in reloadRevisions usually called by component or here?
        // Let's call it here to be safe.
        setDbBusy(true)
        try {
            const result = await window.api.bom.getRevisions(projectId)
            if (result.success) {
                set({ revisions: result.data })
            }
        } finally {
            setDbBusy(false)
        }
    },

    /**
     * 選擇 BOM 版本 (單選/多選切換)
     * @param {number} revisionId
     * @param {boolean} multiSelect - 是否為多選模式 (Ctrl/Cmd click)
     * @param {Object} [revisionObj] - 選填，BOM Revision 物件 (若提供則不需從現有 revisions 尋找)
     */
    toggleRevisionSelection: async (revisionId, multiSelect = false, revisionObj = null) => {
        const { revisions, selectedRevisionIds, bomMode } = get()
        let newIds = new Set(multiSelect ? selectedRevisionIds : [])

        if (multiSelect) {
            if (newIds.has(revisionId)) {
                newIds.delete(revisionId)
            } else {
                newIds.add(revisionId)
            }
        } else {
            newIds = new Set([revisionId])
        }

        // 根據選擇數量與當前模式自動切換
        let newMode = bomMode
        if (newIds.size > 1 && newMode === 'BOM') {
            // 多選時，若在 BOM 模式則切換到 BIGBOM (Placeholder) or stay in current if Matrix
            newMode = 'BIGBOM'
        } else if (newIds.size === 1 && newMode === 'BIGBOM') {
            newMode = 'BOM'
        }

        // 優先使用傳入的物件，若無則在目前的 revisions 列表中尋找
        const selectedRevision = revisionObj || revisions.find(r => r.id === revisionId) || null
        
        set({
            selectedRevisionId: newIds.size === 1 ? Array.from(newIds)[0] : null,
            selectedRevisionIds: newIds,
            selectedRevision: newIds.size === 1 ? selectedRevision : null,
            // 如果只有一個選擇且我們有專案 ID，則同步更新選取的專案 ID (確保 BomPage 能找到正確的專案名稱)
            selectedProjectId: (newIds.size === 1 && selectedRevision) ? selectedRevision.project_id : get().selectedProjectId,
            bomMode: newMode,
            viewCache: {}, // Clear cache on selection change
        })

        // Trigger Data Fetch
        await get().fetchData()
    },

    /**
     * 根據當前選擇與模式，抄取對應資料。
     * 組裝當前 View 的預設 filters 及額外的 CCL filter，透過 bom:query 查詢後端。
     */
    fetchData: async () => {
        const { selectedRevisionIds, currentViewId, cclFilter } = get()
        if (selectedRevisionIds.size === 0) {
            set({ bomView: [] })
            return
        }

        const { setDbBusy } = useAppStore.getState()
        set({ isLoading: true, error: null })
        setDbBusy(true)

        try {
            const idsArray = Array.from(selectedRevisionIds)

            // 取得當前 View 的預設 filters
            // MatrixTable 所需的 BOM rows 也透過此處取得
            const viewsResult = await window.api.bom.getViews()
            let baseFilters = []
            if (viewsResult.success) {
                const viewsObj = viewsResult.data
                const foundView = Object.values(viewsObj).find(v => v.id === currentViewId)
                baseFilters = foundView?.filters || []
            }

            // 組裝最終 filters：View 預設 + CCL checkbox 額外條件
            const extraFilters = cclFilter ? [{ field: 'ccl', operator: 'eq', value: 'Y' }] : []
            const finalFilters = [...baseFilters, ...extraFilters]

            const result = await window.api.bom.query(idsArray, finalFilters, {})
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
     * 設定 CCL Filter 狀態，並重新載入 BOM 資料。
     *
     * @param {boolean} enabled - true 表示套用 ccl=Y filter
     */
    setCclFilter: async (enabled) => {
        set({ cclFilter: enabled })
        await get().fetchData()
    },

    /**
     * 重新載入目前選取的 BOM 版本視圖。
     */
    reloadBomView: async () => {
        // reloadBomView 目前瞼軟，不演算任何操作 (revisionId 已在 fetchData 中處理)
        await get().fetchData()
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

    importExcel: async (filePaths) => {
        try {
            const result = await window.api.excel.import(filePaths)
            if (result.success) {
                return { success: true, data: result.data }
            } else {
                useToastStore.getState().addToast(`加入匯入排程失敗: ${result.error}`, 'error')
                return { success: false, error: result.error }
            }
        } catch (error) {
            useToastStore.getState().addToast(`匯入發生錯誤: ${error.message}`, 'error')
            return { success: false, error: error.message }
        }
    },

    exportExcel: async (revisionId) => {
        try {
            const currentRevision = get().selectedRevision
            const baseName = currentRevision?.filename?.replace(/\.[^/.]+$/, "") || 'BOM_Export'
            const defaultName = `${baseName}.xlsx`
            
            const dialogResult = await window.api.dialog.showSave({
                title: '匯出 BOM Excel',
                defaultPath: defaultName,
                filters: [{ name: 'Excel Files', extensions: ['xlsx'] }]
            })
            if (dialogResult.canceled || !dialogResult.data) {
                return { success: false, error: '使用者取消' }
            }

            const result = await window.api.excel.export(revisionId, dialogResult.data)

            if (result.success) {
                return { success: true }
            } else {
                useToastStore.getState().addToast(`加入匯出排程失敗: ${result.error}`, 'error')
                return { success: false, error: result.error }
            }
        } catch (error) {
            useToastStore.getState().addToast(`匯出發生錯誤: ${error.message}`, 'error')
            return { success: false, error: error.message }
        }
    },

    /**
     * 更新 BOM 版本資料 (Metadata)
     * 
     * @param {number} id - BOM Revision ID
     * @param {Object} updates - 更新內容
     * @returns {Promise<{success: boolean, error?: string}>}
     */
    updateRevision: async (id, updates) => {
        const { setDbBusy } = useAppStore.getState()
        setDbBusy(true)
        try {
            const result = await window.api.bom.updateRevision(id, updates)
            if (result.success) {
                // 更新列表
                set(state => ({
                    revisions: state.revisions.map(r => 
                        r.id === id ? result.data : r
                    ),
                    selectedRevision: state.selectedRevision?.id === id
                        ? result.data
                        : state.selectedRevision
                }))
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

    /** 清空所有狀態（關閉系列時呼叫） */
    reset: () => {
        set({
            selectedProjectId: null,
            revisions: [],
            selectedRevisionId: null,
            selectedRevision: null,
            bomView: [],
            currentViewId: 'all_view',
            viewCache: {},
            cclFilter: false,
            isLoading: false,
            error: null,
        })
    },

    /** 清除錯誤訊息 */
    clearError: () => set({ error: null }),
}))

export default useBomStore
