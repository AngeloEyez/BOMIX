// ========================================
// 主佈局元件
// 包含頂部標題列(含導航)、主內容區域、底部狀態列
// 使用 shadcn UI 元件統一風格
// ========================================

import { useState, useEffect } from 'react'
import { Sun, Moon, Settings, FolderPlus, FileDown } from 'lucide-react'
import useSettingsStore from '../../stores/useSettingsStore'
import useSeriesStore from '../../stores/useSeriesStore'
import useTaskStore from '../../stores/useTaskStore'
import useProjectStore from '../../stores/useProjectStore'

// shadcn UI 元件
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

// 自訂元件
import AppStatusLine from './AppStatusLine'
import ProgressDialog from '../dialogs/ProgressDialog'
import ToastContainer from './ToastContainer'
import ProjectDialog from '../dialogs/ProjectDialog'
import ImportDialog from '../dialogs/ImportDialog'

/**
 * 應用程式主佈局元件。
 *
 * 提供完整的桌面應用程式框架，包含頂部標題(含導航)、中央內容區與底部狀態列。
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

    // 從路徑取得檔案名稱（不含副檔名）
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

    // 同步 dark class 至 html 根元素（供 tailwind dark: variant 使用）
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
        return (
            <div className="flex items-center justify-center h-screen bg-background text-foreground">
                <span className="text-sm text-muted-foreground">載入中...</span>
            </div>
        )
    }

    return (
        // TooltipProvider 需包裹整個應用程式，讓所有 Tooltip 可正常運作
        <TooltipProvider delayDuration={400}>
            <div className="flex flex-col h-screen overflow-hidden bg-background text-foreground transition-colors duration-200">
                {/* --- 頂部標題列 --- */}
                <header className="flex items-center justify-between h-10 px-2
                    bg-background/95 backdrop-blur-sm
                    border-b border-border
                    shrink-0 z-10 app-drag-region">

                    {/* 左側：功能導航 */}
                    <div className="flex items-center gap-0.5">
                        {pages.filter(p => p.id !== 'settings').map((page) => {
                            const isActive = page.id === currentPage
                            return (
                                <Tooltip key={page.id}>
                                    <TooltipTrigger asChild>
                                        <Button
                                            variant={isActive ? 'secondary' : 'ghost'}
                                            size="sm"
                                            onClick={() => onNavigate(page.id)}
                                            className={`h-7 px-2.5 text-xs gap-1.5 ${
                                                isActive
                                                    ? 'text-foreground font-medium'
                                                    : 'text-muted-foreground'
                                            }`}
                                        >
                                            {/* 導航圖標（縮小） */}
                                            <span className="[&>svg]:size-3.5">{page.icon}</span>
                                            <span>{page.label}</span>
                                        </Button>
                                    </TooltipTrigger>
                                    <TooltipContent side="bottom" className="text-xs">
                                        {page.label}
                                    </TooltipContent>
                                </Tooltip>
                            )
                        })}
                    </div>

                    {/* 右側：全域動作與設定 */}
                    <div className="flex items-center gap-0.5">
                        {/* ========================================
                            全域動作區：僅在系列開啟時顯示
                        ======================================== */}
                        {isSeriesOpen && (
                            <>
                                {/* 新增專案按鈕 */}
                                <Tooltip>
                                    <TooltipTrigger asChild>
                                        <Button
                                            variant="ghost"
                                            size="icon"
                                            onClick={() => setProjectDialogOpen(true)}
                                            className="h-7 w-7 text-muted-foreground hover:text-foreground"
                                        >
                                            <FolderPlus className="size-4" />
                                        </Button>
                                    </TooltipTrigger>
                                    <TooltipContent side="bottom" className="text-xs">新增專案</TooltipContent>
                                </Tooltip>

                                {/* 匯入 BOM 按鈕 */}
                                <Tooltip>
                                    <TooltipTrigger asChild>
                                        <Button
                                            variant="ghost"
                                            size="icon"
                                            onClick={() => setImportDialogOpen(true)}
                                            className="h-7 w-7 text-muted-foreground hover:text-foreground"
                                        >
                                            <FileDown className="size-4" />
                                        </Button>
                                    </TooltipTrigger>
                                    <TooltipContent side="bottom" className="text-xs">匯入 Excel BOM</TooltipContent>
                                </Tooltip>

                                {/* 分隔線 */}
                                <Separator orientation="vertical" className="h-4 mx-1" />
                            </>
                        )}

                        {/* 主題切換（Light/Dark） */}
                        <Tooltip>
                            <TooltipTrigger asChild>
                                <Button
                                    variant="ghost"
                                    size="icon"
                                    onClick={toggleTheme}
                                    className="h-7 w-7 text-muted-foreground hover:text-foreground"
                                >
                                    {theme === 'dark' ? <Sun className="size-4" /> : <Moon className="size-4" />}
                                </Button>
                            </TooltipTrigger>
                            <TooltipContent side="bottom" className="text-xs">
                                {theme === 'dark' ? '切換至淺色模式' : '切換至深色模式'}
                            </TooltipContent>
                        </Tooltip>

                        {/* 設定按鈕 */}
                        <Tooltip>
                            <TooltipTrigger asChild>
                                <Button
                                    variant={currentPage === 'settings' ? 'secondary' : 'ghost'}
                                    size="icon"
                                    onClick={() => onNavigate('settings')}
                                    className="h-7 w-7 text-muted-foreground hover:text-foreground"
                                >
                                    <Settings className="size-4" />
                                </Button>
                            </TooltipTrigger>
                            <TooltipContent side="bottom" className="text-xs">設定</TooltipContent>
                        </Tooltip>
                    </div>
                </header>

                {/* --- 主體區域 --- */}
                <main className="flex-1 overflow-hidden relative flex flex-col">
                    {children}
                </main>

                {/* --- 底部狀態列 --- */}
                <AppStatusLine />

                {/* 進度對話框 */}
                <ProgressDialog />

                {/* 全域 Toast 通知 */}
                <ToastContainer />

                {/* ========================================
                    全域對話框（掛在頂層，與頁面無關）
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
        </TooltipProvider>
    )
}

export default AppLayout
