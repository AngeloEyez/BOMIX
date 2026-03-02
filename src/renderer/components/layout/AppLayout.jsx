// ========================================
// 主佈局元件
// 包含側邊導航列、頂部標題列、主內容區域、底部狀態列
// 參考 Windows 11 Fluent Design 風格
// ========================================

import { useState } from 'react'
import { Sun, Moon, Settings, FolderPlus, FileDown } from 'lucide-react'
import useSettingsStore from '../../stores/useSettingsStore'
import useSeriesStore from '../../stores/useSeriesStore'
import useTaskStore from '../../stores/useTaskStore'
import useProjectStore from '../../stores/useProjectStore'
import { useEffect } from 'react'

// Components
import AppStatusLine from './AppStatusLine'
import ProgressDialog from '../dialogs/ProgressDialog'
import ToastContainer from './ToastContainer'
import ProjectDialog from '../dialogs/ProjectDialog'
import ImportDialog from '../dialogs/ImportDialog'

/**
 * 應用程式主佈局元件。
 *
 * 提供完整的桌面應用程式框架，包含左側導航、頂部標題、中央內容區與底部狀態列。
 * 全域動作（新增專案、匯入 BOM）統一放置於此，避免各頁面重複實作。
 *
 * @param {Object} props
 * @param {Array} props.pages - 頁面定義陣列
 * @param {string} props.currentPage - 目前選中的頁面 ID
 * @param {Function} props.onNavigate - 頁面切換回呼函數
 * @param {React.ReactNode} props.children - 頁面內容
 * @returns {JSX.Element} 主佈局
 */
function AppLayout({ pages, currentPage, onNavigate, children }) {
    const { loadSettings, isLoading, theme, toggleTheme } = useSettingsStore()
    const { currentPath, isOpen: isSeriesOpen } = useSeriesStore()
    const initTaskListeners = useTaskStore(state => state.initListeners)
    const { createProject } = useProjectStore()

    // 全域「新增專案」對話框狀態
    const [projectDialogOpen, setProjectDialogOpen] = useState(false)
    // 全域「匯入 BOM」對話框狀態
    const [importDialogOpen, setImportDialogOpen] = useState(false)

    // 從路徑取得檔案名稱
    const seriesName = currentPath ? currentPath.split(/[\\/]/).pop()?.replace('.bomix', '') : null

    // 初始化設定
    useEffect(() => {
        loadSettings()
    }, [loadSettings])

    // 初始化任務排程監聽
    useEffect(() => {
        initTaskListeners()
    }, [initTaskListeners])

    // 更新視窗標題
    useEffect(() => {
        document.title = seriesName ? `BOMIX - ${seriesName}` : 'BOMIX'
    }, [seriesName])

    // Theme effect
    useEffect(() => {
        if (theme === 'dark') {
            document.documentElement.classList.add('dark')
        } else {
            document.documentElement.classList.remove('dark')
        }
    }, [theme])

    /**
     * 處理新增專案對話框送出。
     *
     * @param {string} code - 專案代碼
     * @param {string} desc - 專案描述
     */
    const handleProjectSave = async (code, desc) => {
        await createProject(code, desc)
        setProjectDialogOpen(false)
    }

    /**
     * 處理匯入 BOM 送出。
     *
     * @param {string[]} filePaths - 選取的檔案路徑
     * @returns {Promise<{success: boolean, error?: string}>}
     */
    const handleImportSubmit = async (filePaths) => {
        const result = await window.api.excel.import(filePaths)
        if (result.success) {
            setImportDialogOpen(false)
            return { success: true }
        }
        return { success: false, error: result.error }
    }

    if (isLoading) {
        return <div className="flex items-center justify-center h-screen">Loading...</div>
    }

    return (
        <div className="flex flex-col h-screen overflow-hidden bg-white dark:bg-surface-950 text-slate-900 dark:text-slate-100 transition-colors duration-200">
            {/* --- 頂部標題列 --- */}
            <header className="flex items-center justify-between h-12 px-4
        bg-white/80 dark:bg-surface-900/80 backdrop-blur-sm
        border-b border-slate-200 dark:border-slate-800
        shrink-0 z-10 app-drag-region">
                <div className="flex items-center gap-1">
                    {/* 功能導航 */}
                    {pages.filter(p => p.id !== 'settings').map((page) => {
                        const isActive = page.id === currentPage
                        return (
                            <button
                                key={page.id}
                                onClick={() => onNavigate(page.id)}
                                className={`
                                    p-2 rounded-lg transition-all duration-200 flex items-center gap-2
                                    ${isActive
                                        ? 'bg-primary-100 dark:bg-primary-900/40 text-primary-600 dark:text-primary-400 font-medium'
                                        : 'text-slate-500 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800 hover:text-slate-700 dark:hover:text-slate-200'
                                    }
                                `}
                                title={page.label}
                            >
                                {page.icon}
                                <span className="text-sm">{page.label}</span>
                            </button>
                        )
                    })}
                </div>

                <div className="flex items-center gap-1">
                    {/* ========================================
                        全域動作區：僅在系列開啟時顯示
                    ======================================== */}
                    {isSeriesOpen && (
                        <>
                            {/* 新增專案按鈕 */}
                            <button
                                onClick={() => setProjectDialogOpen(true)}
                                className="p-2 rounded-lg text-slate-500 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800 hover:text-slate-700 dark:hover:text-slate-200 transition-colors"
                                title="新增專案"
                            >
                                <FolderPlus size={18} />
                            </button>

                            {/* 匯入 BOM 按鈕 */}
                            <button
                                onClick={() => setImportDialogOpen(true)}
                                className="p-2 rounded-lg text-slate-500 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800 hover:text-slate-700 dark:hover:text-slate-200 transition-colors"
                                title="匯入 Excel BOM"
                            >
                                <FileDown size={18} />
                            </button>

                            {/* 分隔線 */}
                            <div className="w-px h-5 bg-slate-200 dark:bg-slate-700 mx-1" />
                        </>
                    )}

                    {/* 主題切換 */}
                    <button
                        onClick={toggleTheme}
                        className="p-2 rounded-lg hover:bg-slate-100 dark:hover:bg-surface-800 text-slate-600 dark:text-slate-400 transition-colors"
                        title={theme === 'dark' ? '切換至淺色模式' : '切換至深色模式'}
                    >
                        {theme === 'dark' ? <Sun size={18} /> : <Moon size={18} />}
                    </button>

                    {/* 設定按鈕 */}
                    <button
                        onClick={() => onNavigate('settings')}
                        className={`p-2 rounded-lg transition-all duration-200
                            ${currentPage === 'settings'
                                ? 'bg-primary-100 dark:bg-primary-900/40 text-primary-600 dark:text-primary-400'
                                : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-surface-800'
                            }`}
                        title="設定"
                    >
                        <Settings size={18} />
                    </button>
                </div>
            </header>

            {/* --- 主體區域 --- */}
            <main className="flex-1 overflow-hidden relative flex flex-col">
               {children}
            </main>

            {/* --- 底部狀態列 --- */}
            <AppStatusLine />

            {/* Progress Dialog */}
            <ProgressDialog />

            {/* Global Toasts */}
            <ToastContainer />

            {/* ========================================
                全域對話框（與頁面無關，掛在頂層）
            ======================================== */}

            {/* 新增專案對話框 */}
            <ProjectDialog
                isOpen={projectDialogOpen}
                onClose={() => setProjectDialogOpen(false)}
                mode="create"
                initialData={null}
                onSave={handleProjectSave}
            />

            {/* 匯入 BOM 對話框 */}
            <ImportDialog
                isOpen={importDialogOpen}
                onClose={() => setImportDialogOpen(false)}
                onImport={handleImportSubmit}
            />
        </div>
    )
}

export default AppLayout
