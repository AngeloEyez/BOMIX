
import { useState, useEffect } from 'react'
import useTaskStore from '../../stores/useTaskStore'
import useSeriesStore from '../../stores/useSeriesStore'
import useAppStore from '../../stores/useAppStore'

/**
 * 應用程式狀態列元件
 *
 * 整合原有的 StatusBar 功能與任務排程進度顯示。
 * 左側：資料庫連線狀態、路徑
 * 中間：任務排程狀態面板（永遠顯示，idle / 任務名稱 + 進度 + 最後 log + 排隊數）
 * 右側：應用程式版本
 * 
 * @param {Object} props
 * @param {Function} props.onNavigate - 路由跳轉方法，用於點擊版本號跳轉至關於頁面
 */
function AppStatusLine({ onNavigate }) {
    const [version, setVersion] = useState('')
    const { isOpen, currentPath } = useSeriesStore()
    const { isDbBusy } = useAppStore()

    // 啟動時取得應用程式版本號
    useEffect(() => {
        if (window.api?.getVersion) {
            window.api.getVersion().then(setVersion).catch(() => setVersion('dev'))
        } else {
            // eslint-disable-next-line react-hooks/set-state-in-effect
            setVersion('dev')  // API 不存在時（開發環境備用），一次性初始化
        }
    }, [])

    // --- 任務排程狀態 ---
    const sessions = useTaskStore(state => state.sessions)
    const queueLength = useTaskStore(state => state.queueLength)
    const toggleDialog = useTaskStore(state => state.toggleDialog)
    
    // 找出最近更新的活躍任務（優先顯示 RUNNING，其次 QUEUED）
    const activeSession = (() => {
        const allSessions = Array.from(sessions.values())
        // 優先找 RUNNING 中的任務
        const running = allSessions.find(s => s.status === 'RUNNING')
        if (running) return running
        // 退而求其次找最近更新的任務
        return allSessions.sort((a, b) => new Date(b.updatedAt) - new Date(a.updatedAt))[0]
    })()

    // 取得最後一條 log 訊息
    const lastLog = activeSession?.logs?.length > 0
        ? activeSession.logs[activeSession.logs.length - 1]?.message
        : activeSession?.message

    // 狀態顏色與圖示對應
    const getStatusIndicator = (status) => {
        switch (status) {
            case 'RUNNING': return { color: 'bg-blue-500', animate: 'animate-pulse', icon: '📋' }
            case 'QUEUED': return { color: 'bg-amber-400', animate: '', icon: '⏳' }
            case 'COMPLETED': return { color: 'bg-green-500', animate: '', icon: '✅' }
            case 'FAILED': return { color: 'bg-red-500', animate: '', icon: '❌' }
            case 'CANCELLED': return { color: 'bg-gray-500', animate: '', icon: '⊘' }
            default: return { color: 'bg-gray-400', animate: '', icon: '💤' }
        }
    }

    return (
        <footer className="flex items-center justify-between h-7 px-4
            bg-muted border-t border-border
            text-xs text-muted-foreground select-none shrink-0"
        >
            {/* Left: Database Status (Fixed Width) */}
            <div className="flex items-center gap-3 w-64 shrink-0">
                 {/* 連線狀態指示點 */}
                 <span className="flex items-center gap-1.5">
                    <span className={`w-1.5 h-1.5 rounded-full transition-all duration-300 ${
                        isOpen
                            ? isDbBusy
                                ? 'bg-emerald-400 animate-ping'  // 忙碌時閃爍
                                : 'bg-emerald-500'               // 連線時恆亮
                            : 'bg-border'                        // 未連線（使用語義化變數）
                    }`} />
                    {isOpen ? (isDbBusy ? '存取中...' : '已連線') : '未連線'}
                </span>
                {/* 資料庫路徑 */}
                {isOpen && currentPath && (
                    <span className="text-muted-foreground/60 truncate font-mono" title={currentPath}>
                        {currentPath}
                    </span>
                )}
            </div>

            {/* Middle: 任務排程狀態面板（永遠顯示，可點擊） */}
            <div className="flex-1 flex justify-start items-center px-4 overflow-hidden">
                <div
                    className="flex items-center space-x-2 cursor-pointer hover:bg-accent px-2 py-0.5 rounded transition-colors max-w-full"
                    onClick={() => toggleDialog(true)}
                    title="點擊查看任務排程詳情"
                >
                    {activeSession ? (
                        <>
                            {/* 狀態指示點 */}
                            {(() => {
                                const indicator = getStatusIndicator(activeSession.status)
                                return (
                                    <div className={`w-1.5 h-1.5 rounded-full shrink-0 ${indicator.animate} ${indicator.color}`} />
                                )
                            })()}

                            {/* 任務名稱 */}
                            <span className="font-medium text-foreground truncate max-w-[150px]">
                                {activeSession.title}
                            </span>

                            {/* 進度條（RUNNING 時顯示） */}
                            {activeSession.status === 'RUNNING' && (
                                <div className="w-20 h-1.5 bg-border rounded-full overflow-hidden shrink-0">
                                    <div
                                        className="h-full bg-primary transition-all duration-300"
                                        style={{ width: `${activeSession.progress}%` }}
                                    />
                                </div>
                            )}

                            {/* 進度百分比（RUNNING 時顯示） */}
                            {activeSession.status === 'RUNNING' && (
                                <span className="text-muted-foreground shrink-0 w-8 text-right">
                                    {activeSession.progress}%
                                </span>
                            )}

                            {/* 最後一條 log */}
                            <span className="text-muted-foreground/70 truncate max-w-[250px]">
                                {lastLog || ''}
                            </span>

                            {/* 排隊數（有等待任務時顯示） */}
                            {queueLength > 0 && (
                                <span className="text-amber-500 shrink-0 font-medium">
                                    ({queueLength} 排隊)
                                </span>
                            )}
                        </>
                    ) : (
                        // 無任務時顯示 idle
                        <>
                            <span className="text-muted-foreground/50">💤</span>
                            <span className="text-muted-foreground/50 italic">idle</span>
                        </>
                    )}
                </div>
            </div>

            {/* Right: Version info */}
            <div 
                className="text-right shrink-0 cursor-pointer hover:bg-accent hover:text-foreground px-2 py-0.5 rounded transition-colors"
                onClick={() => onNavigate && onNavigate('about')}
                title="點擊查看關於與更新紀錄"
            >
                <span>v{version}</span>
            </div>
        </footer>
    )
}

export default AppStatusLine
