
import { useState, useEffect } from 'react'
import useProgressStore from '../../stores/useProgressStore'
import useSeriesStore from '../../stores/useSeriesStore'
import useAppStore from '../../stores/useAppStore'

/**
 * 應用程式狀態列元件
 *
 * 整合原有的 StatusBar 功能與新的進度回饋顯示。
 * 左側：資料庫連線狀態、路徑
 * 中間：長時間任務進度回饋 (可點擊)
 * 右側：應用程式版本
 */
function AppStatusLine() {
    const [version, setVersion] = useState('')
    const { isOpen, currentPath } = useSeriesStore()
    const { isDbBusy } = useAppStore()

    // 啟動時取得應用程式版本號
    useEffect(() => {
        if (window.api?.getVersion) {
            window.api.getVersion().then(setVersion).catch(() => setVersion('dev'))
        } else {
            setVersion('dev')
        }
    }, [])

    // --- Progress Feedback ---
    const sessions = useProgressStore(state => state.sessions)
    const toggleDialog = useProgressStore(state => state.toggleDialog)
    
    // Find the session with the latest update time
    const activeSession = Array.from(sessions.values())
        .sort((a, b) => new Date(b.updatedAt) - new Date(a.updatedAt))[0]

    // Helper to get status color
    const getStatusColor = (status) => {
        switch (status) {
            case 'RUNNING': return 'bg-blue-500'
            case 'COMPLETED': return 'bg-green-500'
            case 'FAILED': return 'bg-red-500'
            case 'CANCELLED': return 'bg-gray-500'
            default: return 'bg-gray-400'
        }
    }

    return (
        <footer className="flex items-center justify-between h-8 px-4
            bg-gray-100 dark:bg-surface-900 border-t border-gray-300 dark:border-slate-700
            text-xs text-slate-500 dark:text-slate-400 select-none shrink-0"
        >
            {/* Left: Database Status (Fixed Width) */}
            <div className="flex items-center gap-3 w-64 shrink-0">
                 {/* 連線狀態指示 */}
                 <span className="flex items-center gap-1">
                    <span className={`w-2 h-2 rounded-full transition-all duration-300 ${
                        isOpen
                            ? isDbBusy
                                ? 'bg-emerald-400 animate-ping'  // 忙碌時閃爍
                                : 'bg-emerald-500'               // 連線時恆亮
                            : 'bg-slate-300 dark:bg-slate-600'   // 未連線
                    }`} />
                    {isOpen ? (isDbBusy ? '存取中...' : '已連線') : '未連線'}
                </span>
                {/* 資料庫路徑 */}
                {isOpen && currentPath && (
                    <span className="text-slate-400 dark:text-slate-500 truncate font-mono" title={currentPath}>
                        {currentPath}
                    </span>
                )}
            </div>

            {/* Middle: Progress Feedback (Elastic Width, Aligned Left) */}
            <div className="flex-1 flex justify-start items-center px-4 overflow-hidden">
                {activeSession ? (
                    <div 
                        className="flex items-center space-x-3 cursor-pointer hover:bg-gray-200 dark:hover:bg-slate-800 px-3 py-0.5 rounded transition-colors max-w-full"
                        onClick={() => toggleDialog(true)}
                        title="點擊查看詳細進度"
                    >
                        {/* Status Dot */}
                        <div className={`w-2 h-2 rounded-full shrink-0 ${activeSession.status === 'RUNNING' ? 'animate-pulse' : ''} ${getStatusColor(activeSession.status)}`} />
                        
                        {/* Activity Info */}
                        <span className="font-medium text-gray-700 dark:text-gray-300 truncate max-w-[200px]">
                            {activeSession.title}
                        </span>
                        <span className="text-gray-500 truncate max-w-[300px]">
                            - {activeSession.message || activeSession.currentTask}
                        </span>

                        {/* Progress Bar (Mini) */}
                        {activeSession.status === 'RUNNING' && (
                            <div className="w-20 h-1.5 bg-gray-300 dark:bg-slate-700 rounded-full overflow-hidden shrink-0">
                                <div 
                                    className="h-full bg-blue-500 transition-all duration-300"
                                    style={{ width: `${activeSession.progress}%` }}
                                />
                            </div>
                        )}
                    </div>
                ) : (
                    // Placeholder
                    <span className="text-gray-300 dark:text-slate-700 italic">
                        
                    </span>
                )}
            </div>

            {/* Right: Version info */}
            <div className="text-right shrink-0">
                <span>v{version}</span>
            </div>
        </footer>
    )
}

export default AppStatusLine
