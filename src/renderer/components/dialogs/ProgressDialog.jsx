import { useEffect, useRef, useState } from 'react'
import useProgressStore, { MAX_SESSIONS } from '../../stores/useProgressStore'

/**
 * 詳細進度對話框
 * 顯示所有 Sessions 的列表與 Logs
 */
function ProgressDialog() {
    const isDialogOpen = useProgressStore(state => state.isDialogOpen)
    const toggleDialog = useProgressStore(state => state.toggleDialog)
    const sessions = useProgressStore(state => state.sessions)
    const activeSessionId = useProgressStore(state => state.activeSessionId)
    const setActiveSession = useProgressStore(state => state.setActiveSession)
    const clearCompletedSessions = useProgressStore(state => state.clearCompletedSessions)

    const scrollRef = useRef(null)
    const [sortBy, setSortBy] = useState('startTime') // 'startTime' | 'lastUpdate'

    // Auto-scroll to TOP of logs when active session updates (since logs are reversed)
    useEffect(() => {
        if (scrollRef.current) {
            scrollRef.current.scrollTop = 0
        }
    }, [sessions.get(activeSessionId)?.logs?.length, activeSessionId])

    if (!isDialogOpen) return null

    const sessionList = Array.from(sessions.values()).sort((a, b) => {
        if (sortBy === 'startTime') {
            return new Date(b.createdAt) - new Date(a.createdAt)
        } else {
            return new Date(b.updatedAt) - new Date(a.updatedAt)
        }
    })

    const activeSession = sessions.get(activeSessionId) || sessionList[0]

    return (
        <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center backdrop-blur-sm">
            <div className="bg-white dark:bg-surface-900 w-[900px] h-[600px] rounded-lg shadow-xl flex flex-col overflow-hidden border border-slate-200 dark:border-slate-700">
                
                {/* Header */}
                <div className="h-12 bg-gray-100 dark:bg-surface-800 border-b border-gray-200 dark:border-slate-700 flex items-center justify-between px-4">
                    <h3 className="font-semibold text-gray-700 dark:text-slate-200">工作進度監控</h3>
                    <button 
                        onClick={() => toggleDialog(false)}
                        className="text-gray-500 hover:text-gray-700 dark:text-slate-400 dark:hover:text-slate-200"
                    >
                        ✕
                    </button>
                </div>

                {/* Content Area */}
                <div className="flex-1 flex overflow-hidden">
                    
                    {/* Left: Session List */}
                    <div className="w-[350px] border-r border-gray-200 dark:border-slate-700 bg-gray-50 dark:bg-surface-950 flex flex-col">
                         <div className="p-2 border-b border-gray-200 dark:border-slate-700 flex flex-col gap-2 bg-gray-100 dark:bg-surface-800">
                            <div className="flex justify-between items-center">
                                <span className="text-xs font-bold text-gray-500 dark:text-slate-400 uppercase">Sessions ({sessions.size} / {MAX_SESSIONS})</span>
                                <button 
                                    onClick={clearCompletedSessions}
                                    className="text-xs text-blue-600 dark:text-blue-400 hover:underline"
                                >
                                    清除已完成
                                </button>
                            </div>
                            {/* Sort Controls */}
                            <div className="flex bg-gray-200 dark:bg-surface-700 rounded p-0.5">
                                <button 
                                    className={`flex-1 text-xs py-1 rounded transition-colors ${sortBy === 'startTime' ? 'bg-white dark:bg-surface-600 shadow-sm text-slate-800 dark:text-slate-200' : 'text-gray-600 dark:text-slate-400 hover:bg-gray-300 dark:hover:bg-surface-600'}`}
                                    onClick={() => setSortBy('startTime')}
                                >
                                    依開始時間
                                </button>
                                <button 
                                    className={`flex-1 text-xs py-1 rounded transition-colors ${sortBy === 'lastUpdate' ? 'bg-white dark:bg-surface-600 shadow-sm text-slate-800 dark:text-slate-200' : 'text-gray-600 dark:text-slate-400 hover:bg-gray-300 dark:hover:bg-surface-600'}`}
                                    onClick={() => setSortBy('lastUpdate')}
                                >
                                    依更新時間
                                </button>
                            </div>
                        </div>
                        <div className="overflow-y-auto flex-1 p-2 space-y-2">
                            {sessionList.map(session => (
                                <div 
                                    key={session.id}
                                    onClick={() => setActiveSession(session.id)}
                                    className={`p-3 rounded cursor-pointer border transition-colors ${
                                        activeSessionId === session.id 
                                            ? 'bg-blue-50 dark:bg-blue-900/20 border-blue-300 dark:border-blue-700 shadow-sm' 
                                            : 'bg-white dark:bg-surface-800 border-gray-200 dark:border-slate-700 hover:bg-gray-100 dark:hover:bg-surface-700'
                                    }`}
                                >
                                    <div className="flex justify-between items-start mb-1">
                                        <span className="font-bold text-sm truncate text-gray-800 dark:text-slate-200">{session.title || 'Untitled Task'}</span>
                                        <StatusBadge status={session.status} />
                                    </div>
                                    <div className="text-xs text-gray-600 dark:text-slate-400 mb-2 truncate" title={session.message}>
                                        {session.message || 'Initializing...'}
                                    </div>
                                    <div className="flex items-center justify-between text-[10px] text-gray-400 dark:text-slate-500 mb-2">
                                        <span>Logs: {session.logs?.length || 0}</span>
                                        <span>{new Date(session.updatedAt).toLocaleTimeString()}</span>
                                    </div>
                                    {session.status === 'RUNNING' && (
                                        <div className="h-1.5 bg-gray-200 dark:bg-surface-600 rounded-full overflow-hidden">
                                            <div 
                                                className="h-full bg-blue-500 transition-all duration-300"
                                                style={{ width: `${session.progress}%` }} 
                                            />
                                        </div>
                                    )}
                                </div>
                            ))}
                            {sessionList.length === 0 && (
                                <div className="text-center text-gray-400 dark:text-slate-600 mt-10 text-sm">
                                    無活動紀錄
                                </div>
                            )}
                        </div>
                    </div>

                    {/* Right: Detailed Logs */}
                    <div className="flex-1 flex flex-col bg-white dark:bg-surface-900 min-w-0">
                         {/* ... Right pane remains similar but uses activeSession determined above ... */}
                        {activeSession ? (
                            <>
                                <div className="p-4 border-b border-gray-200 dark:border-slate-700 bg-gray-50 dark:bg-surface-800">
                                    <h4 className="font-bold text-lg text-slate-800 dark:text-slate-100">{activeSession.title}</h4>
                                    <div className="text-sm text-slate-500 mt-1 font-mono text-xs select-all">
                                        ID: {activeSession.id}
                                    </div>
                                    <div className="text-sm text-slate-600 dark:text-slate-300 mt-1">
                                        狀態: <span className="font-medium">{activeSession.status}</span> ({activeSession.progress}%)
                                    </div>
                                     {activeSession.error && (
                                        <div className="mt-2 p-2 bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-300 text-sm rounded border border-red-200 dark:border-red-800">
                                            錯誤: {activeSession.error}
                                        </div>
                                    )}
                                    {activeSession.result && (
                                        <div className="mt-2 p-2 bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-300 text-sm rounded border border-green-200 dark:border-green-800 whitespace-pre-wrap font-mono text-xs overflow-x-auto scrollbar-thin">
                                            結果: {JSON.stringify(activeSession.result, null, 2)}
                                        </div>
                                    )}
                                </div>

                                <div className="flex-1 p-4 overflow-y-auto font-mono text-xs space-y-1 scrollbar-thin dark:scrollbar-track-surface-800 dark:scrollbar-thumb-slate-600" ref={scrollRef}>
                                    {activeSession.logs && activeSession.logs.length > 0 ? (
                                        [...activeSession.logs].reverse().map((log, index) => (
                                            <div key={index} className="flex space-x-2 hover:bg-gray-50 dark:hover:bg-surface-800 p-0.5 rounded group">
                                                <span className="text-gray-400 dark:text-slate-500 min-w-[70px] select-none">
                                                    {new Date(log.timestamp).toLocaleTimeString()}
                                                </span>
                                                <span className={`font-semibold ${getLevelColor(log.level)} min-w-[40px] uppercase text-center select-none`}>
                                                    {log.level}
                                                </span>
                                                <span className="text-gray-800 dark:text-slate-300 break-all group-hover:text-black dark:group-hover:text-white">
                                                    {log.message}
                                                </span>
                                            </div>
                                        ))
                                    ) : (
                                        <div className="text-gray-400 dark:text-slate-600 italic text-center mt-10">無日誌紀錄</div>
                                    )}
                                </div>
                            </>
                        ) : (
                            <div className="flex-1 flex items-center justify-center text-gray-400 dark:text-slate-600">
                                請選擇一個 Session 查看詳情
                            </div>
                        )}
                    </div>
                </div>
            </div>
        </div>
    )
}

function StatusBadge({ status }) {
    const colors = {
        PENDING: 'bg-gray-100 text-gray-600 dark:bg-surface-700 dark:text-slate-400',
        RUNNING: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300',
        COMPLETED: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300',
        FAILED: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300',
        CANCELLED: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-300'
    }
    return (
        <span className={`text-[10px] px-1.5 py-0.5 rounded font-bold ${colors[status] || colors.PENDING}`}>
            {status}
        </span>
    )
}

function getLevelColor(level) {
    switch (level) {
        case 'error': return 'text-red-600 dark:text-red-400'
        case 'warn': return 'text-yellow-600 dark:text-yellow-400'
        default: return 'text-blue-600 dark:text-blue-400'
    }
}

export default ProgressDialog
