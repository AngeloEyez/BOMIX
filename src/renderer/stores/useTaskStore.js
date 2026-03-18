import { create } from 'zustand'

// 最大任務保留數量
export const MAX_TASKS = 30

/** 系統通知 Session 的固定 ID */
export const SYSTEM_LOG_SESSION_ID = 'system-logs'

/** 系統通知 Session 最多保留的日誌條數 */
export const MAX_SYSTEM_LOGS = 100

/**
 * Task Store
 *
 * 管理後端任務排程狀態與相關 UI 互動。
 * 取代舊版 useProgressStore，新增佇列狀態追蹤、任務完成 callback 機制。
 *
 * 主要功能：
 * - 透過 IPC 監聽任務狀態更新（task:update）
 * - 監聽任務完成事件（task:completed）以觸發 UI callback
 * - 提供佇列狀態（排隊數、當前任務）
 * - 管理 ProgressDialog 的開關與 Session 選擇
 */
const useTaskStore = create((set, get) => ({
    // --- State ---
    /** @type {Map<string, Object>} 所有任務（含歷史紀錄） */
    sessions: new Map(),
    /** @type {string|null} 當前 UI 關注的 Session ID */
    activeSessionId: null,
    /** @type {boolean} 詳細進度對話框是否開啟 */
    isDialogOpen: false,
    /** @type {boolean} 是否已初始化 IPC 監聽 */
    ipcInitialized: false,
    /** @type {number} 佇列中等待的任務數 */
    queueLength: 0,
    /**
     * 全域最新一條日誌（任務日誌或系統通知皆會更新）。
     * 供 AppStatusLine 快速讀取，無需掃描所有 sessions。
     * @type {{ message: string, level: string, timestamp: string } | null}
     */
    lastGlobalLog: null,

    // --- 任務完成 Callback 註冊 ---
    /** @type {Map<string, Set<Function>>} 任務完成時的 callback (依 type 分類) */
    _completedCallbacks: new Map(),

    // --- Actions ---

    /**
     * 開關詳細進度對話框
     * @param {boolean} [isOpen] - 指定開關狀態，不傳則 toggle
     */
    toggleDialog: (isOpen) => {
        set({ isDialogOpen: isOpen !== undefined ? isOpen : !get().isDialogOpen })
    },

    /**
     * 設定當前關注的 Session
     * @param {string} sessionId - 任務 ID
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
     * 註冊任務完成 callback。
     *
     * 當指定類型的任務完成時，自動呼叫 callback。
     * 用於匯入完成後重新載入 UI 資料等場景。
     *
     * @param {string} taskType - 任務類型，如 'IMPORT_BOM'
     * @param {Function} callback - callback 函數，接收 { id, type, result, metadata }
     * @returns {Function} 取消註冊的函數
     */
    registerCompletedCallback: (taskType, callback) => {
        const callbacks = get()._completedCallbacks
        const newCallbacks = new Map(callbacks)
        
        if (!newCallbacks.has(taskType)) {
            newCallbacks.set(taskType, new Set())
        }
        newCallbacks.get(taskType).add(callback)
        
        set({ _completedCallbacks: newCallbacks })

        // 返回取消註冊的函數
        return () => {
            const current = get()._completedCallbacks
            const updated = new Map(current)
            if (updated.has(taskType)) {
                updated.get(taskType).delete(callback)
                if (updated.get(taskType).size === 0) {
                    updated.delete(taskType)
                }
            }
            set({ _completedCallbacks: updated })
        }
    },

    /**
     * 新增一條系統通知日誌（非任務型日誌，如 UI 操作結果）。
     *
     * 所有訊息會累積在固定 ID 為 `system-logs` 的 Session 中，
     * 總數超過 MAX_SYSTEM_LOGS 時，自動移除最舊的那一筆。
     * 同時更新 `lastGlobalLog` 供狀態列即時顯示。
     *
     * @param {string} message - 日誌訊息
     * @param {'info'|'warn'|'error'|'success'} [level='info'] - 日誌等級
     */
    addSystemLog: (message, level = 'info') => {
        const now = new Date().toISOString()
        const newLog = { message, level, timestamp: now }

        set((state) => {
            const newSessions = new Map(state.sessions)

            // 取得或建立「系統通知」Session
            const existing = newSessions.get(SYSTEM_LOG_SESSION_ID)
            const prevLogs = existing?.logs ?? []

            // 超過上限時移除最舊的
            const trimmed = prevLogs.length >= MAX_SYSTEM_LOGS
                ? prevLogs.slice(prevLogs.length - MAX_SYSTEM_LOGS + 1)
                : prevLogs

            newSessions.set(SYSTEM_LOG_SESSION_ID, {
                id: SYSTEM_LOG_SESSION_ID,
                title: '系統通知',
                status: 'SYSTEM',
                progress: 0,
                message,
                logs: [...trimmed, newLog],
                createdAt: existing?.createdAt ?? now,
                updatedAt: now,
            })

            return {
                sessions: newSessions,
                lastGlobalLog: newLog,
            }
        })
    },

    /**
     * 取消排隊中的任務
     * @param {string} taskId - 任務 ID
     */
    cancelTask: async (taskId) => {
        if (window.api?.task) {
            await window.api.task.cancel(taskId)
        }
    },

    /**
     * 移除已結束的任務紀錄
     * @param {string} taskId - 任務 ID
     */
    removeTask: async (taskId) => {
        if (window.api?.task) {
            const result = await window.api.task.remove(taskId)
            if (result?.success) {
                const { sessions, activeSessionId } = get()
                const newSessions = new Map(sessions)
                newSessions.delete(taskId)
                let newActiveId = activeSessionId === taskId ? null : activeSessionId
                if (!newActiveId && newSessions.size > 0) {
                    newActiveId = newSessions.keys().next().value
                }
                set({ sessions: newSessions, activeSessionId: newActiveId })
            }
        }
    },

    /**
     * 初始化 IPC 監聽器 (只執行一次)。
     *
     * 監聽 task:update 與 task:completed 兩個通道。
     */
    initListeners: () => {
        if (get().ipcInitialized) return

        // --- task:update 監聽 ---
        const handleUpdate = (task) => {
            set((state) => {
                const newSessions = new Map(state.sessions)

                // 如果是新任務且當前沒有 Active Session，自動設為 Active
                let newActiveId = state.activeSessionId
                if (!newSessions.has(task.id) && !newActiveId) {
                    newActiveId = task.id
                }

                newSessions.set(task.id, task)

                // 計算排隊中的任務數
                let queueLen = 0
                for (const session of newSessions.values()) {
                    if (session.status === 'QUEUED') queueLen++
                }

                // 限制最大任務數量
                if (newSessions.size > MAX_TASKS) {
                    let targetIdToDelete = null

                    // 1. 先找最早的已完成/失敗/取消任務
                    for (const [id, session] of newSessions.entries()) {
                        if (['COMPLETED', 'FAILED', 'CANCELLED'].includes(session.status)) {
                            targetIdToDelete = id
                            break
                        }
                    }

                    // 2. 如果沒有已完成的，則強制移除最早的
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

                // 更新全域最新日誌（取目前任務最後一條 log）
                let newLastGlobalLog = state.lastGlobalLog
                const updatedSession = newSessions.get(task.id)
                if (updatedSession?.logs?.length > 0) {
                    newLastGlobalLog = updatedSession.logs[updatedSession.logs.length - 1]
                }

                return {
                    sessions: newSessions,
                    activeSessionId: newActiveId,
                    queueLength: queueLen,
                    lastGlobalLog: newLastGlobalLog,
                }
            })
        }

        // --- task:completed 監聽 —— 觸發 UI callback ---
        const handleCompleted = (data) => {
            const callbacks = get()._completedCallbacks
            const callbackSet = callbacks.get(data.type)
            if (callbackSet) {
                for (const callback of callbackSet) {
                    try {
                        callback(data)
                    } catch (err) {
                        console.error(`[TaskStore] callback 執行錯誤 (${data.type}):`, err)
                    }
                }
            }
        }

        if (window.api?.task) {
            window.api.task.onUpdate(handleUpdate)
            window.api.task.onCompleted(handleCompleted)
            set({ ipcInitialized: true })
            console.log('[TaskStore] IPC listeners initialized')
        } else {
            console.warn('[TaskStore] window.api.task not found')
        }
    }
}))

export default useTaskStore
