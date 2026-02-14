import { useEffect, useState } from 'react'
import { FolderPlus, FolderOpen, Clock, X, Database, ChevronRight, Pencil, Check, FileText } from 'lucide-react'
import useSeriesStore from '../stores/useSeriesStore'
import useProjectStore from '../stores/useProjectStore'
import Dialog from '../components/dialogs/Dialog'

// ========================================
// 首頁（歡迎頁面）
// 顯示應用程式名稱、快速操作、系列資訊
// ========================================

function HomePage() {
    const {
        isOpen, currentSeries, currentPath, recentFiles, isLoading, error,
        initRecentFiles, createSeries, openSeries, closeSeries,
        updateDescription, renameSeries, removeFromRecentFiles, clearError,
    } = useSeriesStore()

    const { loadProjects, projects, reset: resetProjects } = useProjectStore()

    // 編輯描述
    const [isEditingDesc, setIsEditingDesc] = useState(false)
    const [editDesc, setEditDesc] = useState('')

    // 重新命名
    const [isRenameOpen, setIsRenameOpen] = useState(false)
    const [renameValue, setRenameValue] = useState('')
    const [renameError, setRenameError] = useState('')

    // 初始化與載入專案狀態
    useEffect(() => {
        initRecentFiles()
    }, [initRecentFiles])

    useEffect(() => {
        if (isOpen) {
            loadProjects() // 專案數量統計
        } else {
            resetProjects()
        }
    }, [isOpen, loadProjects, resetProjects])

    // --- Handlers ---
    const handleCreate = async () => createSeries()
    const handleOpen = async (filePath) => openSeries(filePath)
    const handleClose = () => closeSeries()

    const handleStartEditDesc = () => {
        setEditDesc(currentSeries?.description || '')
        setIsEditingDesc(true)
    }

    const handleSaveDesc = async () => {
        await updateDescription(editDesc)
        setIsEditingDesc(false)
    }

    const handleOpenRename = () => {
        // 從路徑解析目前檔名 (不含副檔名)
        const name = currentPath?.split(/[\\/]/).pop().replace('.bomix', '') || ''
        setRenameValue(name)
        setRenameError('')
        setIsRenameOpen(true)
    }

    const handleRenameSubmit = async () => {
        if (!renameValue.trim()) {
            setRenameError('請輸入系列名稱')
            return
        }
        // 前端初步驗證 Windows 檔名
        if (/[\\/:*?"<>|]/.test(renameValue)) {
            setRenameError('名稱包含無效字元 (\\/:*?"<>|)')
            return
        }

        const result = await renameSeries(renameValue.trim())
        if (result.success) {
            setIsRenameOpen(false)
        } else {
            setRenameError(result.error)
        }
    }

    // ========================================
    // 已開啟系列 — 系列資訊卡片
    // ========================================
    if (isOpen && currentSeries) {
        // 顯示名稱 (不含 .bomix)
        const displayName = currentPath?.split(/[\\/]/).pop().replace('.bomix', '') || '未命名系列'

        return (
            <div className="max-w-2xl mx-auto space-y-6 animate-fade-in">
                {/* 系列資訊卡片 */}
                <div className="bg-white dark:bg-surface-800 rounded-xl shadow-sm border border-slate-200 dark:border-slate-700 overflow-hidden">
                    {/* 卡片標頭 */}
                    <div className="flex items-center justify-between p-4 border-b border-slate-100 dark:border-slate-700">
                        <div className="flex items-center gap-3 min-w-0">
                            <div className="shrink-0 p-2 bg-primary-100 dark:bg-primary-900/30 text-primary-600 dark:text-primary-400 rounded-lg">
                                <Database size={20} />
                            </div>
                            <div className="min-w-0">
                                <div className="flex items-center gap-2">
                                    <h2 className="text-lg font-bold text-slate-800 dark:text-white truncate">
                                        {displayName}
                                    </h2>
                                    <button
                                        onClick={handleOpenRename}
                                        className="p-1 rounded-md hover:bg-slate-100 dark:hover:bg-surface-700 text-slate-400 hover:text-primary-600 transition-colors"
                                        title="重新命名系列"
                                    >
                                        <Pencil size={14} />
                                    </button>
                                </div>
                                <p className="text-xs text-slate-400 font-mono truncate max-w-md">
                                    {currentPath}
                                </p>
                            </div>
                        </div>
                        <button
                            onClick={handleClose}
                            className="p-2 rounded-lg hover:bg-slate-100 dark:hover:bg-surface-700 text-slate-400 transition-colors"
                            title="關閉系列"
                        >
                            <X size={18} />
                        </button>
                    </div>

                    {/* 卡片內容 */}
                    <div className="p-5 space-y-5">
                        {/* 描述 */}
                        <div>
                            <div className="flex items-center gap-2 mb-1">
                                <span className="text-xs font-semibold text-slate-500 uppercase tracking-wider">
                                    系列描述
                                </span>
                                {!isEditingDesc && (
                                    <button
                                        onClick={handleStartEditDesc}
                                        className="p-1 rounded hover:bg-slate-100 dark:hover:bg-surface-700 text-slate-400 transition-colors"
                                        title="編輯描述"
                                    >
                                        <Pencil size={12} />
                                    </button>
                                )}
                            </div>
                            {isEditingDesc ? (
                                <div className="flex gap-2">
                                    <input
                                        type="text"
                                        value={editDesc}
                                        onChange={(e) => setEditDesc(e.target.value)}
                                        className="flex-1 px-3 py-1.5 text-sm border border-slate-300 dark:border-slate-600
                                            bg-white dark:bg-surface-900 rounded-lg
                                            text-slate-800 dark:text-slate-200
                                            focus:outline-none focus:ring-2 focus:ring-primary-500"
                                        autoFocus
                                        onKeyDown={(e) => e.key === 'Enter' && handleSaveDesc()}
                                    />
                                    <button
                                        onClick={handleSaveDesc}
                                        className="p-1.5 bg-primary-600 hover:bg-primary-700 text-white rounded-lg transition-colors"
                                    >
                                        <Check size={16} />
                                    </button>
                                </div>
                            ) : (
                                <p className="text-sm text-slate-600 dark:text-slate-300 leading-relaxed">
                                    {currentSeries.description || '（無描述）'}
                                </p>
                            )}
                        </div>

                        {/* 統計資訊 */}
                        <div className="grid grid-cols-2 gap-4">
                            <div className="bg-slate-50 dark:bg-surface-900/50 rounded-lg p-3 flex items-center justify-between group">
                                <div>
                                    <p className="text-2xl font-bold text-primary-600 dark:text-primary-400">
                                        {projects.length}
                                    </p>
                                    <p className="text-xs text-slate-500">專案總數</p>
                                </div>
                                <FolderOpen className="text-slate-200 dark:text-slate-700" size={32} />
                            </div>
                            
                            <div className="bg-slate-50 dark:bg-surface-900/50 rounded-lg p-3 flex items-center justify-between group">
                                <div>
                                    <p className="text-2xl font-bold text-sky-600 dark:text-sky-400">
                                        {currentSeries.bomCount || 0}
                                    </p>
                                    <p className="text-xs text-slate-500">BOM 版本數</p>
                                </div>
                                <FileText className="text-slate-200 dark:text-slate-700" size={32} />
                            </div>
                        </div>

                        {/* 日期資訊 footer */}
                        <div className="pt-2 border-t border-slate-100 dark:border-slate-700 flex justify-between text-xs text-slate-400 font-mono">
                            <div>
                                <span className="mr-2">建立於:</span>
                                {currentSeries.created_at?.split('T')[0] || '-'}
                            </div>
                            <div>
                                <span className="mr-2">最後修改:</span>
                                {currentSeries.updated_at?.split('T')[0] || '-'}
                            </div>
                        </div>
                    </div>
                </div>

                {/* 快速操作區域 (維持原樣) */}
                <div className="flex gap-3">
                    <button
                        onClick={handleCreate}
                        className="flex items-center gap-2 px-4 py-2 text-sm
                            bg-white dark:bg-surface-800 rounded-lg shadow-sm
                            border border-slate-200 dark:border-slate-700
                            hover:border-primary-300 dark:hover:border-primary-600
                            text-slate-600 dark:text-slate-300
                            transition-all"
                    >
                        <FolderPlus size={16} />
                        <span>建立新系列</span>
                    </button>
                    <button
                        onClick={() => handleOpen()}
                        className="flex items-center gap-2 px-4 py-2 text-sm
                            bg-white dark:bg-surface-800 rounded-lg shadow-sm
                            border border-slate-200 dark:border-slate-700
                            hover:border-primary-300 dark:hover:border-primary-600
                            text-slate-600 dark:text-slate-300
                            transition-all"
                    >
                        <FolderOpen size={16} />
                        <span>開啟其他系列</span>
                    </button>
                </div>

                {/* --- 重新命名對話框 --- */}
                <Dialog
                    isOpen={isRenameOpen}
                    onClose={() => setIsRenameOpen(false)}
                    title="重新命名系列"
                    className="max-w-sm"
                >
                    <div className="space-y-4">
                        <p className="text-sm text-slate-500 dark:text-slate-400">
                            這將會修改資料庫檔案名稱。
                        </p>
                        <div>
                            <input
                                type="text"
                                value={renameValue}
                                onChange={(e) => setRenameValue(e.target.value)}
                                placeholder="輸入新的系列名稱"
                                className="w-full px-3 py-2 text-sm
                                    bg-white dark:bg-surface-900
                                    border border-slate-300 dark:border-slate-600
                                    rounded-lg text-slate-800 dark:text-slate-200
                                    focus:outline-none focus:ring-2 focus:ring-primary-500"
                                autoFocus
                            />
                            {renameError && (
                                <p className="text-xs text-red-500 mt-1">{renameError}</p>
                            )}
                        </div>
                        <div className="flex justify-end gap-2">
                             <button
                                onClick={() => setIsRenameOpen(false)}
                                className="px-4 py-2 text-sm text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-surface-700 rounded-lg transition-colors"
                            >
                                取消
                            </button>
                            <button
                                onClick={handleRenameSubmit}
                                className="px-4 py-2 text-sm text-white bg-primary-600 hover:bg-primary-700 rounded-lg transition-colors"
                            >
                                確定
                            </button>
                        </div>
                    </div>
                </Dialog>
            </div>
        )
    }

    // ========================================
    // 未開啟系列 — 歡迎畫面 (維持不變)
    // ========================================
    return (
        <div className="flex flex-col items-center justify-center h-full gap-8 p-6 overflow-auto animate-fade-in">
            {/* ... 原本的歡迎畫面內容 ... */}
            <div className="text-center">
                <h1 className="text-4xl font-bold text-primary-600 dark:text-primary-400 mb-2">
                    BOMIX
                </h1>
                <p className="text-lg text-slate-500 dark:text-slate-400">
                    BOM 變化管理與追蹤工具
                </p>
            </div>
             {/* 錯誤提示 */}
             {error && (
                <div className="flex items-center gap-2 px-4 py-2 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg text-sm text-red-600 dark:text-red-400">
                    <span>{error}</span>
                    <button onClick={clearError} className="ml-2 hover:text-red-800">
                        <X size={14} />
                    </button>
                </div>
            )}

            <div className="flex gap-4">
                <button
                    onClick={handleCreate}
                    disabled={isLoading}
                    className="flex flex-col items-center gap-2 px-8 py-6
                        bg-white dark:bg-surface-800 rounded-xl
                        shadow-sm hover:shadow-md
                        border border-slate-200 dark:border-slate-700
                        hover:border-primary-300 dark:hover:border-primary-600
                        transition-all duration-200 cursor-pointer
                        group disabled:opacity-50 disabled:cursor-not-allowed"
                >
                    <span className="text-primary-500 group-hover:scale-110 transition-transform">
                        <FolderPlus size={32} />
                    </span>
                    <span className="text-sm font-medium text-slate-700 dark:text-slate-300">
                        建立新系列
                    </span>
                </button>

                <button
                    onClick={() => handleOpen()}
                    disabled={isLoading}
                    className="flex flex-col items-center gap-2 px-8 py-6
                        bg-white dark:bg-surface-800 rounded-xl
                        shadow-sm hover:shadow-md
                        border border-slate-200 dark:border-slate-700
                        hover:border-primary-300 dark:hover:border-primary-600
                        transition-all duration-200 cursor-pointer
                        group disabled:opacity-50 disabled:cursor-not-allowed"
                >
                    <span className="text-primary-500 group-hover:scale-110 transition-transform">
                        <FolderOpen size={32} />
                    </span>
                    <span className="text-sm font-medium text-slate-700 dark:text-slate-300">
                        開啟系列
                    </span>
                </button>
            </div>
            
            {isLoading && (
                <p className="text-sm text-slate-400 animate-pulse">
                    處理中...
                </p>
            )}

           {/* 最近開啟的系列 */}
           <div className="w-full max-w-md">
                <h2 className="text-sm font-semibold text-slate-500 dark:text-slate-400 mb-3 flex items-center gap-2">
                    <Clock size={14} />
                    最近開啟
                </h2>
                {recentFiles.length > 0 ? (
                    <div className="bg-white dark:bg-surface-800 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden divide-y divide-slate-100 dark:divide-slate-700">
                        {recentFiles.map((filePath) => {
                            const name = filePath.split(/[\\/]/).pop()?.replace('.bomix', '') || name
                            return (
                                <div
                                    key={filePath}
                                    className="flex items-center justify-between p-3 hover:bg-slate-50 dark:hover:bg-surface-700/50 transition-colors group"
                                >
                                    <button
                                        onClick={() => handleOpen(filePath)}
                                        className="flex-1 flex items-center gap-3 text-left min-w-0"
                                    >
                                        <Database size={16} className="shrink-0 text-primary-500" />
                                        <div className="min-w-0">
                                            <p className="text-sm font-medium text-slate-700 dark:text-slate-200 truncate">
                                                {name}
                                            </p>
                                            <p className="text-xs text-slate-400 truncate font-mono">
                                                {filePath}
                                            </p>
                                        </div>
                                        <ChevronRight size={14} className="shrink-0 text-slate-300 group-hover:text-primary-500 transition-colors" />
                                    </button>
                                    <button
                                        onClick={() => removeFromRecentFiles(filePath)}
                                        className="shrink-0 ml-2 p-1 rounded hover:bg-slate-200 dark:hover:bg-surface-600 text-slate-300 hover:text-slate-500 opacity-0 group-hover:opacity-100 transition-all"
                                        title="從列表中移除"
                                    >
                                        <X size={14} />
                                    </button>
                                </div>
                            )
                        })}
                    </div>
                ) : (
                    <div className="bg-white dark:bg-surface-800 rounded-xl
                        border border-slate-200 dark:border-slate-700
                        p-4 text-center text-sm text-slate-400 dark:text-slate-500"
                    >
                        尚無開啟記錄
                    </div>
                )}
            </div>
        </div>
    )
}

export default HomePage
