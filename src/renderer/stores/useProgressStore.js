import { create } from 'zustand'

// 最大 Session 保留數量
export const MAX_SESSIONS = 30

/**
 * Progress Store
 * 管理後端任務進度 (Sessions) 與相關 UI 狀態
 */
const useProgressStore = create((set, get) => ({
    // --- State ---
    sessions: new Map(),        // Map<taskId, sessionObj>
    activeSessionId: null,      // 當前 UI 關注的 Session ID
    isDialogOpen: false,        // 詳細進度對話框是否開啟
    ipcInitialized: false,      // 是否已初始化 IPC 監聽

    // --- Actions ---

    /**
     * 開關詳細進度對話框
     */
    toggleDialog: (isOpen) => {
        set({ isDialogOpen: isOpen !== undefined ? isOpen : !get().isDialogOpen })
    },

    /**
     * 設定當前關注的 Session
     */
    setActiveSession: (sessionId) => {
        set({ activeSessionId: sessionId })
    },

    /**
     * 清除已完成或失敗的 Sessions
     */
    clearCompletedSessions: () => {
        const { sessions, activeSessionId } = get()
        const newSessions = new Map(sessions)
        let newActiveId = activeSessionId

        for (const [id, session] of newSessions.entries()) {
            if (['COMPLETED', 'FAILED', 'CANCELLED'].includes(session.status)) {
                newSessions.delete(id)
                if (id === activeSessionId) {
                    newActiveId = null
                }
            }
        }
        
        // 如果目前沒有 Active Session，但還有其他執行中的 Session，自動切換過去
        if (!newActiveId && newSessions.size > 0) {
            newActiveId = newSessions.keys().next().value
        }

        set({ sessions: newSessions, activeSessionId: newActiveId })
    },

    /**
     * 初始化 IPC 監聽器 (只執行一次)
     */
    initListeners: () => {
        if (get().ipcInitialized) return

        const handleUpdate = (task) => {
            set((state) => {
                const newSessions = new Map(state.sessions)
                
                // 如果是新任務且當前沒有 Active Session，自動設為 Active
                let newActiveId = state.activeSessionId
                if (!newSessions.has(task.id) && !newActiveId) {
                    newActiveId = task.id
                }

                newSessions.set(task.id, task)

                // 限制最大 Session 數量 (若超過則移除最早的已完成任務，若無則移除最早的任務)
                if (newSessions.size > MAX_SESSIONS) {
                    let targetIdToDelete = null

                    // 1. 先找最早的已完成/失敗/取消任務
                    for (const [id, session] of newSessions.entries()) {
                        if (['COMPLETED', 'FAILED', 'CANCELLED'].includes(session.status)) {
                            targetIdToDelete = id
                            break // Map 依插入順序，找到的第一個就是最早的
                        }
                    }

                    // 2. 如果沒有已完成的，則強制移除最早的 (即便是 RUNNING)
                    if (!targetIdToDelete) {
                        targetIdToDelete = newSessions.keys().next().value
                    }

                    if (targetIdToDelete) {
                        newSessions.delete(targetIdToDelete)
                        if (targetIdToDelete === newActiveId) {
                            newActiveId = null
                        }
                    }
                }

                return { 
                    sessions: newSessions, 
                    activeSessionId: newActiveId 
                }
            })
        }

        if (window.api && window.api.progress) {
            window.api.progress.onUpdate(handleUpdate)
            set({ ipcInitialized: true })
            console.log('[ProgressStore] IPC listeners initialized')
        } else {
            console.warn('[ProgressStore] window.api.progress not found')
        }
    }
}))

export default useProgressStore
