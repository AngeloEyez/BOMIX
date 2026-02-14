import { create } from 'zustand'

// ========================================
// 專案狀態管理 (Zustand)
// 管理目前系列下的專案列表與 CRUD 操作
// ========================================

const useProjectStore = create((set, get) => ({
    // --- 狀態 ---
    /** 專案列表 */
    projects: [],
    /** 目前選取的專案 */
    selectedProject: null,
    /** 是否正在載入中 */
    isLoading: false,
    /** 錯誤訊息 */
    error: null,

    // --- 動作 ---

    /**
     * 載入目前系列下的所有專案。
     */
    loadProjects: async () => {
        set({ isLoading: true, error: null })
        try {
            const result = await window.api.project.getAll()
            if (result.success) {
                set({ projects: result.data, isLoading: false })
            } else {
                set({ error: result.error, isLoading: false })
            }
        } catch (error) {
            set({ error: error.message, isLoading: false })
        }
    },

    /**
     * 建立新專案。
     *
     * @param {string} projectCode - 專案代碼（唯一）
     * @param {string} [description] - 專案描述
     * @returns {Promise<{success: boolean, error?: string}>}
     */
    createProject: async (projectCode, description) => {
        set({ error: null })
        try {
            const result = await window.api.project.create(projectCode, description)
            if (result.success) {
                // 重新載入列表以確保順序一致
                await get().loadProjects()
                return { success: true, data: result.data }
            } else {
                set({ error: result.error })
                return { success: false, error: result.error }
            }
        } catch (error) {
            set({ error: error.message })
            return { success: false, error: error.message }
        }
    },

    /**
     * 更新專案描述。
     *
     * @param {number} id - 專案 ID
     * @param {string} description - 新的描述
     * @returns {Promise<{success: boolean, error?: string}>}
     */
    updateProject: async (id, data) => {
        set({ error: null })
        try {
            const result = await window.api.project.update(id, data)
            if (result.success) {
                // 更新列表中對應的項目
                set(state => ({
                    projects: state.projects.map(p =>
                        p.id === id ? result.data : p
                    ),
                    // 若選取的專案被更新，同步更新
                    selectedProject: state.selectedProject?.id === id
                        ? result.data
                        : state.selectedProject,
                }))
                return { success: true }
            } else {
                set({ error: result.error })
                return { success: false, error: result.error }
            }
        } catch (error) {
            set({ error: error.message })
            return { success: false, error: error.message }
        }
    },

    /**
     * 刪除專案。
     *
     * @param {number} id - 專案 ID
     * @returns {Promise<{success: boolean, error?: string}>}
     */
    deleteProject: async (id) => {
        set({ error: null })
        try {
            const result = await window.api.project.delete(id)
            if (result.success) {
                set(state => ({
                    projects: state.projects.filter(p => p.id !== id),
                    // 若刪除的是目前選取的專案，清除選取
                    selectedProject: state.selectedProject?.id === id
                        ? null
                        : state.selectedProject,
                }))
                return { success: true }
            } else {
                set({ error: result.error })
                return { success: false, error: result.error }
            }
        } catch (error) {
            set({ error: error.message })
            return { success: false, error: error.message }
        }
    },

    /**
     * 選取專案。
     *
     * @param {Object|null} project - 專案物件
     */
    selectProject: (project) => {
        set({ selectedProject: project })
    },

    /** 清除狀態（關閉系列時呼叫） */
    reset: () => {
        set({
            projects: [],
            selectedProject: null,
            isLoading: false,
            error: null,
        })
    },

    /** 清除錯誤訊息 */
    clearError: () => set({ error: null }),
}))

export default useProjectStore
