// ========================================
// 主佈局元件
// 包含頂部標題列(含導航)、主內容區域、底部狀態列
// 使用 shadcn UI 元件統一風格
// ========================================

import { useState, useEffect, useCallback } from 'react'
import { Sun, Moon, Settings, FolderPlus, FileDown } from 'lucide-react'
import useSettingsStore from '../../stores/useSettingsStore'
import useSeriesStore from '../../stores/useSeriesStore'
import useTaskStore from '../../stores/useTaskStore'
import useProjectStore from '../../stores/useProjectStore'
import useToastStore from '../../stores/useToastStore'

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
import PhaseOrderDialog from '../dialogs/PhaseOrderDialog'

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
    const addSystemLog = useTaskStore(state => state.addSystemLog)
    const { createProject } = useProjectStore()

    // 全域「新增專案」對話框狀態
    const [projectDialogOpen, setProjectDialogOpen] = useState(false)
    // 全域「匯入 BOM」對話框狀態
    const [importDialogOpen, setImportDialogOpen] = useState(false)
    // 全域拖曳狀態
    const [isDragOver, setIsDragOver] = useState(false)
    // Phase 驗證用的狀態
    const [phaseDialog, setPhaseDialog] = useState({ isOpen: false, requiredPhase: '', pendingPaths: [] })

    const addToast = useToastStore(state => state.addToast)

    // 從路徑取得檔案名稱（不含副檔名）
    const seriesName = currentPath ? currentPath.split(/[\\/]/).pop()?.replace('.bomix', '') : null
    const currentSeries = useSeriesStore(state => state.currentSeries)

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
     * 在送出前驗證 Phase 是否在定義中，若無則開啟 PhaseOrderDialog。
     *
     * @param {string[]} filePaths - 選取的檔案路徑
     * @returns {Promise<{success: boolean, error?: string}>}
     */
    const handleImportSubmit = async (filePaths) => {
        // 先解析這些檔案的 MetaData
        const analysis = await window.api.excel.analyzeFiles(filePaths)
        if (!analysis.success) {
            return { success: false, error: analysis.error }
        }

        const validFiles = analysis.data.validFiles
        if (validFiles.length === 0) {
            return { success: false, error: '沒有合法且包含有效 Phase 的 BOM 檔案可供匯入' }
        }

        // 檢查 Phase 是否在 currentPhaseOrder 中
        let phaseOrderStr = currentSeries?.phase_order;
        let orderArray = [
            'RFI', 'RFP', 'RFQ, RFx', 'DB, EVT', 'SI, DVT', 'PV, PVT', 'TLD, PRD', 'MVB, MP'
        ];
        if (phaseOrderStr) {
            try {
                const parsed = JSON.parse(phaseOrderStr);
                if (Array.isArray(parsed) && parsed.length > 0) orderArray = parsed;
            } catch (e) {
               // ignore
            }
        }

        // 找出第一個不符合的 Phase
        let missingPhase = null;
        for (const file of validFiles) {
            const phaseName = file.phase;
            if (!phaseName) continue;

            // 解析出 base phase (e.g., DB1 -> DB)
            const match = phaseName.match(/^([A-Za-z]+)(\d*)$/);
            const basePhase = match ? match[1].toUpperCase() : phaseName.toUpperCase();

            // 檢查是否存在於 orderArray 中
            const exists = orderArray.some(def => {
                const parts = def.split(',').map(s => s.trim().toUpperCase());
                return parts.includes(basePhase);
            });

            if (!exists) {
                missingPhase = basePhase; // 使用 base phase 讓 user 加
                break;
            }
        }

        if (missingPhase) {
            setPhaseDialog({ isOpen: true, requiredPhase: missingPhase, pendingPaths: filePaths });
            addSystemLog(`發現未定義的 Phase: ${missingPhase}`, 'error')
            return { success: false, error: `發現未定義的 Phase: ${missingPhase}` }; // Let the ImportDialog stay open or handle this
        }

        // 如果全部檢查通過，進行實際的匯入排程
        const result = await window.api.excel.import(filePaths)
        if (result.success) {
            addSystemLog('已送出匯入任務，請於進度視窗追蹤狀態', 'info')
            setImportDialogOpen(false)
            return { success: true }
        }
        addSystemLog(`匯入排程失敗：${result.error}`, 'error')
        return { success: false, error: result.error }
    }

    const handleSavePhaseOrder = async (orderArray) => {
        const result = await useSeriesStore.getState().updatePhaseOrder(orderArray);
        if (result.success) {
            setPhaseDialog(prev => ({ ...prev, isOpen: false }));
            // 如果有 pendingPaths，則自動重試匯入
            if (phaseDialog.pendingPaths.length > 0) {
                const retryResult = await handleImportSubmit(phaseDialog.pendingPaths);
                if (retryResult.success) {
                    addSystemLog('Phase 新增成功，已自動開始匯入', 'success')
                    addToast('Phase 新增成功，已自動開始匯入', 'success');
                } else if (retryResult.error && retryResult.error.includes('發現未定義的 Phase')) {
                    // It means there was ANOTHER missing phase, handleImportSubmit already opened the dialog again.
                    addSystemLog('發現另一個未定義的 Phase，請繼續新增', 'warn')
                    addToast('發現另一個未定義的 Phase，請繼續新增', 'warning');
                } else {
                    addSystemLog(`自動匯入失敗：${retryResult.error}`, 'error')
                    addToast(`自動匯入失敗：${retryResult.error}`, 'error');
                }
            }
        } else {
            addSystemLog(`儲存 Phase 排序失敗：${result.error}`, 'error')
            addToast(`儲存 Phase 排序失敗：${result.error}`, 'error');
        }
    }

    // ========================================
    // 全域拖曳匯入處理
    // ========================================
    const handleDragOver = useCallback((e) => {
        e.preventDefault()
        e.stopPropagation()
        if (!isSeriesOpen) return // 未開啟系列不允許匯入
        setIsDragOver(true)
    }, [isSeriesOpen])

    const handleDragLeave = useCallback((e) => {
        e.preventDefault()
        e.stopPropagation()
        // 避免子元素觸發 dragleave 導致閃爍
        if (e.currentTarget && e.relatedTarget && e.currentTarget.contains(e.relatedTarget)) {
            return
        }
        setIsDragOver(false)
    }, [])

    const handleDrop = useCallback(async (e) => {
        e.preventDefault()
        e.stopPropagation()
        setIsDragOver(false)

        if (!isSeriesOpen) return

        const files = e.dataTransfer?.files
        if (!files || files.length === 0) return

        const validPaths = []
        let hasInvalid = false

        for (let i = 0; i < files.length; i++) {
            const file = files[i]
            const ext = file.name.split('.').pop()?.toLowerCase()
            if (ext === 'xls' || ext === 'xlsx') {
                validPaths.push(window.api.utils.getPathForFile(file))
            } else {
                hasInvalid = true
            }
        }

        if (hasInvalid) {
            addToast('已忽略不支援的檔案格式，僅接受 .xls 與 .xlsx。', 'warning')
        }

        if (validPaths.length > 0) {
            const result = await handleImportSubmit(validPaths)
            if (!result.success && result.error) {
                addToast(`匯入失敗：${result.error}`, 'error')
            }
        } else if (!hasInvalid) {
             addToast('未找到有效的 Excel 檔案。', 'warning')
        }
    }, [isSeriesOpen, addToast, handleImportSubmit])

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
                        {pages.filter(p => p.id !== 'settings' && p.id !== 'about').map((page) => {
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
                <main 
                    className="flex-1 overflow-hidden relative flex flex-col"
                    onDragOver={handleDragOver}
                    onDragLeave={handleDragLeave}
                    onDrop={handleDrop}
                >
                    {children}

                    {/* 拖曳覆蓋層 */}
                    {isDragOver && (
                        <div className="absolute inset-0 z-[100] flex items-center justify-center
                            bg-primary/5 backdrop-blur-[1px] border-4 border-dashed border-primary
                            pointer-events-none transition-all duration-200">
                            <div className="bg-background/95 rounded-xl shadow-2xl px-10 py-8 text-center transform scale-105 border border-primary/30">
                                <FileDown className="size-12 mx-auto mb-4 text-primary animate-bounce" />
                                <p className="text-xl font-bold text-foreground">放開以匯入 Excel BOM</p>
                                <p className="text-sm text-muted-foreground mt-2">支援多選 .xls / .xlsx 格式檔案</p>
                                <p className="text-xs text-muted-foreground mt-1">匯入任務將於背景執行</p>
                            </div>
                        </div>
                    )}
                </main>

                {/* --- 底部狀態列 --- */}
                <AppStatusLine onNavigate={onNavigate} />

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

                <PhaseOrderDialog
                    isOpen={phaseDialog.isOpen}
                    onClose={() => setPhaseDialog(prev => ({ ...prev, isOpen: false }))}
                    currentPhaseOrder={currentSeries?.phase_order}
                    onSave={handleSavePhaseOrder}
                    requiredPhase={phaseDialog.requiredPhase}
                />
            </div>
        </TooltipProvider>
    )
}

export default AppLayout
