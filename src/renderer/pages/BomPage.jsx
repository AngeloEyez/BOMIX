import { useEffect, useState, useCallback, useMemo, useRef } from 'react'
import {
    FileSpreadsheet, Download,
    FolderOpen, ChevronDown, X, RotateCcw, Settings
} from 'lucide-react'
import useSeriesStore from '../stores/useSeriesStore'
import useProjectStore from '../stores/useProjectStore'
import useBomStore from '../stores/useBomStore'
import useTaskStore from '../stores/useTaskStore'
import useToastStore from '../stores/useToastStore'
import useMatrixStore from '../stores/useMatrixStore'
import BomTable from '../components/tables/BomTable'
import BomSidebar from '../components/layout/BomSidebar'
import ConfirmDialog from '../components/dialogs/ConfirmDialog'
import MatrixModelDialog from '../components/dialogs/MatrixModelDialog'
import { Button } from '@/components/ui/button'

// ========================================
// BOM 檢視頁面
// 提供專案/版本選取、BOM 表格檢視、Excel 匯入匯出功能
// ========================================

const DEFAULT_SEARCH_FIELDS = ['hhpn', 'description', 'supplier', 'supplier_pn', 'location']
const ALL_SEARCH_FIELDS = ['hhpn', 'description', 'supplier', 'supplier_pn', 'location', 'remark']

const FIELD_LABELS = {
    hhpn: 'HHPN',
    description: 'Description',
    supplier: 'Supplier',
    supplier_pn: 'Supplier PN',
    location: 'Location',
    remark: 'Remark'
}

/**
 * BOM 檢視頁面元件。
 *
 * 包含工具列 (專案選擇、版本選擇、匯入/匯出按鈕)
 * 及 BOM 聚合表格。使用緊湊佈局以最大化資料展示面積。
 *
 * @returns {JSX.Element} BOM 檢視頁面
 */
function BomPage() {
    const { isOpen } = useSeriesStore()
    const { projects, allBoms, loadProjects } = useProjectStore()
    const {
        selectedProjectId, selectedRevisionId, selectedRevisionIds, selectedRevision,
        bomView, isLoading, error,
        deleteBom, exportExcel,
        clearError, reset,
        currentViewId, selectView, 
        bomMode 
    } = useBomStore()

    // 檢查是否有匯出任務正在執行
    const isExporting = useTaskStore(state => 
        Array.from(state.sessions.values()).some(s => s.type === 'EXPORT_BOM' && s.status === 'RUNNING')
    )
    const registerCompletedCallback = useTaskStore(state => state.registerCompletedCallback)
    const addToast = useToastStore(state => state.addToast)

    const { fetchMatrixData } = useMatrixStore()

    // 删除確認對話框
    const [deleteTarget, setDeleteTarget] = useState(null)
    // Matrix Model 管理對話框
    const [isMatrixModelDialogOpen, setIsMatrixModelDialogOpen] = useState(false)
    // 頁面層級拖曳狀態
    const [isDragOver, setIsDragOver] = useState(false)

    // 視圖狀態 (Definitions)
    const [views, setViews] = useState({})
    
    // 搜尋狀態
    const [searchTerm, setSearchTerm] = useState('')
    const [isSearchOptionsOpen, setIsSearchOptionsOpen] = useState(false)
    const [searchFields, setSearchFields] = useState(new Set(DEFAULT_SEARCH_FIELDS))

    // Refs for click outside detection
    const searchOptionsPanelRef = useRef(null)
    const searchToggleRef = useRef(null)

    // Handle click outside to close search options
    useEffect(() => {
        const handleClickOutside = (event) => {
            if (isSearchOptionsOpen && 
                searchOptionsPanelRef.current && 
                !searchOptionsPanelRef.current.contains(event.target) &&
                searchToggleRef.current &&
                !searchToggleRef.current.contains(event.target)) {
                setIsSearchOptionsOpen(false)
            }
        }

        document.addEventListener('mousedown', handleClickOutside)
        return () => {
            document.removeEventListener('mousedown', handleClickOutside)
        }
    }, [isSearchOptionsOpen])

    // 初始化：載入視圖定義
    useEffect(() => {
        const loadViews = async () => {
            try {
                const result = await window.api.bom.getViews()
                if (result.success) {
                    setViews(result.data)
                }
            } catch (err) {
                console.error('Failed to load BOM views:', err)
            }
        }
        loadViews()
    }, [])

    // 載入 Matrix 資料
    useEffect(() => {
        // Support multi IDs for Matrix Mode
        if (bomMode === 'MATRIX' && selectedRevisionIds.size > 0) {
            const ids = Array.from(selectedRevisionIds);
            fetchMatrixData(ids);
        }
    }, [selectedRevisionIds, bomMode, fetchMatrixData])

    // 關鍵字過濾邏輯
    const filteredBom = useMemo(() => {
        let data = bomView

        // Matrix Mode 僅顯示 CCL=Y
        if (bomMode === 'MATRIX' && Array.isArray(data)) {
            data = data.filter(item => item.ccl === 'Y')
        }

        if (!Array.isArray(data) || !searchTerm || !searchTerm.trim()) return data || []

        const term = searchTerm.toLowerCase().trim()
        
        // 檢查單一項目是否符合
        const checkMatch = (item) => {
            return Array.from(searchFields).some(field => {
                // Map 'location' key to 'locations' property in source data
                const value = field === 'location' ? item.locations : item[field]
                return value && String(value).toLowerCase().includes(term)
            })
        }
        
        // ... (filter logic unchanged)
        return data.filter(mainItem => {
            if (checkMatch(mainItem)) return true
            if (mainItem.second_sources && mainItem.second_sources.length > 0) {
                if (mainItem.second_sources.some(ss => checkMatch(ss))) return true
            }
            return false
        })
    }, [bomView, searchTerm, searchFields, bomMode])


    // 開啟系列後載入專案列表
    useEffect(() => {
        if (isOpen) {
            loadProjects()
        } else {
            reset()
        }
    }, [isOpen, loadProjects, reset])

    // 註冊 BATCH_IMPORT 完成的 Callback
    useEffect(() => {
        if (!isOpen) return
        
        const unsubscribe = registerCompletedCallback('BATCH_IMPORT', async (data) => {
            const { result } = data

            if (result && !result.error) {
                // 重新載入 Revision 列表
                await useBomStore.getState().reloadRevisions()
                // 如果有匯入新的 BOM，會在此刷新
            } else {
                const errMsg = result ? result.error : '未知錯誤'
                addToast(`匯入失敗：${errMsg}`, 'error')
            }
        })
        
        return () => unsubscribe()
    }, [isOpen, registerCompletedCallback, addToast])

    // 每個 IMPORT_BOM 子任務完成後，靜默刷新專案列表與所有 BOM 快取
    // 呼叫 loadProjects 可同時涵蓋：
    //   (1) 新專案被自動建立的情境（useProjectStore.projects 更新）
    //   (2) 現有專案新增版本的情境（loadAllBoms 自動更新 allBoms）
    // 不清除已選取的 BOM 或搜尋狀態（避免閃爍）
    useEffect(() => {
        if (!isOpen) return

        const unsubscribe = registerCompletedCallback('IMPORT_BOM', async (data) => {
            const { result } = data
            if (result?.success !== false) {
                // 刷新專案列表（含 allBoms）→ BomSidebar 自動反映最新狀態
                await loadProjects()
                // 同時刷新 useBomStore.revisions（保持 BomPage 選取狀態與側邊欄一致）
                await useBomStore.getState().reloadRevisions()
            }
        })

        return () => unsubscribe()
    }, [isOpen, registerCompletedCallback, loadProjects])

    /**
     * 處理刪除 BOM 版本確認
     */
    const handleDeleteConfirm = async () => {
        if (deleteTarget) {
            await deleteBom(deleteTarget)
            setDeleteTarget(null)
        }
    }

    /**
     * 處理匯出 Excel
     */
    const handleExport = () => {
        if (selectedRevisionId) {
            exportExcel(selectedRevisionId) // Async fire-and-forget, handled by progress system
        }
    }

    // ========================================
    // 頁面層級拖曳匯入
    // ========================================
    const handlePageDragOver = useCallback((e) => {
        e.preventDefault()
        e.stopPropagation()
        // 僅在已選擇專案時允許拖曳 (Wait, sidebar selects BOMs, what about project?)
        // If we have selected IDs, we probably can infer project.
        // Or if we have `selectedProjectId` in store.
        // `useBomStore` keeps `selectedProjectId`.
        // Let's assume user must select a project via sidebar to enable drop?
        // Actually, drop on sidebar project item would be cool, but page drop needs context.
        setIsDragOver(true)
    }, [])

    const handlePageDragLeave = useCallback((e) => {
        e.preventDefault()
        e.stopPropagation()
        setIsDragOver(false)
    }, [])

    const handlePageDrop = useCallback((e) => {
        e.preventDefault()
        e.stopPropagation()
        setIsDragOver(false)

        // Find active project if any
        // We need a target project ID. `selectedProjectId` in BomStore is updated when clicking project in sidebar?
        // Let's assume BomStore tracks last selected project.
        // If no project selected, we can't import easily unless we parse filename or ask user.
        // For now, require selectedProjectId.

        // Warning: BomSidebar updates `selectedRevisionIds`. Does it update `selectedProjectId`?
        // Yes, `BomSidebar` uses `selectProject` or user might click project.

        // const files = e.dataTransfer?.files
        // ... (Import logic needs valid projectId)
    }, [])

    // ========================================
    // 未開啟系列 — 提示畫面
    // ========================================
    if (!isOpen) {
        return (
            <div className="flex flex-col items-center justify-center h-full gap-4 text-muted-foreground animate-fade-in">
                <FolderOpen size={40} className="text-border" />
                <h2 className="text-lg font-semibold">尚未開啟系列</h2>
                <p className="text-sm">請先從首頁建立或開啟系列資料庫，再檢視 BOM。</p>
            </div>
        )
    }

    // 取得當前選取 BOM 的所屬專案名稱
    const selectedProjectCode = useMemo(() => {
        // 優先從 selectedRevision 取得 project_id，若無則使用 store 的 selectedProjectId
        const pid = selectedRevision?.project_id || selectedProjectId
        if (pid && projects.length > 0) {
            // 使用 == 以相容 string/number 型別差異
            const project = projects.find(p => p.id == pid)
            return project ? project.project_code : ''
        }
        return ''
    }, [selectedRevision, selectedProjectId, projects])

    // 計算選取的專案數量 (跨專案多選)
    const selectedProjectCount = useMemo(() => {
        if (selectedRevisionIds.size === 0) return 0
        const pIds = new Set()
        Object.entries(allBoms).forEach(([projectId, revs]) => {
            if (revs.some(r => selectedRevisionIds.has(r.id))) {
                pIds.add(projectId)
            }
        })
        return pIds.size
    }, [selectedRevisionIds, allBoms])

    // 計算目前選取的版本顯示名稱 (Multi-select handling?)
    const selectionCount = selectedRevisionIds.size;
    const headerTitle = selectionCount === 0
        ? <span className="text-sm font-semibold text-muted-foreground">未選擇 BOM</span>
        : selectionCount === 1
            ? (
                <div className="flex flex-col leading-tight">
                    <span className="text-[10px] font-medium text-muted-foreground uppercase tracking-wider">
                        {selectedProjectCode || 'Unknown Project'}
                    </span>
                    <span className="text-sm font-bold text-foreground">
                        {selectedRevision?.phase_name}-{selectedRevision?.version}{selectedRevision?.suffix ? `-${selectedRevision.suffix}` : ''}
                    </span>
                </div>
            )
            : (
                <div className="flex flex-col leading-tight">
                    <span className="text-[10px] font-medium text-muted-foreground uppercase tracking-wider">
                        {selectedProjectCount} Projects
                    </span>
                    <span className="text-sm font-bold text-foreground">
                        {selectionCount} BOMs
                    </span>
                </div>
            )

    // ========================================
    // 頁面主體
    // ========================================
    return (
        <div className="flex h-full animate-fade-in">
            {/* 側邊欄 */}
            <BomSidebar />

            {/* 主要內容區 */}
            <div
                className="flex-1 flex flex-col min-w-0 p-3 gap-2"
                onDragOver={handlePageDragOver}
                onDragLeave={handlePageDragLeave}
                onDrop={handlePageDrop}
            >
                {/* 工具列 */}
                <div className="flex items-center gap-2 flex-wrap bg-background p-2 rounded-lg shadow-sm border border-border">

                    {/* 標題/狀態 */}
                    <div className="flex items-center gap-2 px-2 mr-2 border-r border-border min-h-[32px]">
                        {headerTitle}
                    </div>

                    {/* 視圖切換 (View Switcher) */}
                    {selectionCount > 0 && (
                        <div className="flex items-center bg-muted rounded-md p-0.5">
                            {['ALL', 'SMD', 'PTH', 'BOTTOM'].map(key => {
                                const view = views[key]
                                if (!view) return null
                                const isActive = currentViewId === view.id
                                return (
                                    <button
                                        key={key}
                                        onClick={() => selectView(view.id)}
                                        className={`px-2.5 py-1 text-xs font-medium rounded transition-all
                                            ${isActive
                                                ? 'bg-background text-primary shadow-sm'
                                                : 'text-muted-foreground hover:text-foreground'
                                            }`}
                                    >
                                        {key}
                                    </button>
                                )
                            })}
                        </div>
                    )}

                    {/* Matrix 設定按鈕 (Only in Matrix Mode) */}
                    {selectionCount > 0 && bomMode === 'MATRIX' && (
                        <Button
                            variant="ghost"
                            size="icon"
                            className="h-7 w-7 text-muted-foreground hover:text-indigo-500"
                            onClick={() => setIsMatrixModelDialogOpen(true)}
                            title="管理 Matrix Models"
                        >
                            <Settings size={15} />
                        </Button>
                    )}

                    {/* 搜尋框 (Search) */}
                    {selectionCount > 0 && (
                        <div className="relative ml-auto flex flex-col">
                            <div className="relative">
                                <input
                                    type="text"
                                    value={searchTerm}
                                    onChange={(e) => setSearchTerm(e.target.value)}
                                    onDoubleClick={() => setIsSearchOptionsOpen(prev => !prev)}
                                    onFocus={() => setIsSearchOptionsOpen(false)}
                                    onKeyDown={(e) => {
                                        if (e.key === 'Escape') {
                                            setSearchTerm('')
                                            setIsSearchOptionsOpen(false)
                                        }
                                    }}
                                    placeholder="Search..."
                                    className={`pl-3 pr-14 py-1.5 text-xs h-8 transition-all
                                        bg-background border rounded-md
                                        text-foreground placeholder:text-muted-foreground
                                        focus:outline-none focus:ring-1 focus:ring-ring
                                        ${searchFields.size !== DEFAULT_SEARCH_FIELDS.length ? 'border-amber-400' : 'border-input'}
                                        ${isSearchOptionsOpen ? 'w-72' : 'focus:w-56 w-36'}`}
                                />

                                <div className="absolute right-2 top-1/2 -translate-y-1/2 flex items-center gap-1">
                                    {searchTerm && (
                                        <button
                                            onClick={() => setSearchTerm('')}
                                            className="text-muted-foreground hover:text-foreground"
                                            title="Clear search"
                                        >
                                            <X size={12} />
                                        </button>
                                    )}
                                    {searchFields.size !== DEFAULT_SEARCH_FIELDS.length && (
                                        <button
                                            onClick={() => setSearchFields(new Set(DEFAULT_SEARCH_FIELDS))}
                                            className="text-amber-500 hover:text-amber-600"
                                            title="Reset search fields"
                                        >
                                            <RotateCcw size={12} />
                                        </button>
                                    )}
                                    <button
                                        ref={searchToggleRef}
                                        onClick={() => setIsSearchOptionsOpen(prev => !prev)}
                                        className={`text-muted-foreground hover:text-foreground transition-transform ${isSearchOptionsOpen ? 'rotate-180' : ''}`}
                                        title="Toggle search options"
                                    >
                                        <ChevronDown size={13} />
                                    </button>
                                </div>
                            </div>

                            {/* Search Options Panel */}
                            {isSearchOptionsOpen && (
                                <div
                                    ref={searchOptionsPanelRef}
                                    className="absolute top-full right-0 mt-1 w-72 p-2
                                        bg-background rounded-lg shadow-xl border border-border
                                        z-50 flex flex-wrap gap-1.5 animate-in fade-in slide-in-from-top-1"
                                >
                                    {ALL_SEARCH_FIELDS.map(field => {
                                        const isSelected = searchFields.has(field)
                                        return (
                                            <button
                                                key={field}
                                                onClick={() => {
                                                    setSearchFields(prev => {
                                                        const next = new Set(prev)
                                                        if (next.has(field)) next.delete(field)
                                                        else next.add(field)
                                                        if (next.size === 0) return prev
                                                        return next
                                                    })
                                                }}
                                                onDoubleClick={(e) => {
                                                    e.stopPropagation()
                                                    setSearchFields(new Set([field]))
                                                }}
                                                className={`px-2 py-0.5 text-[11px] rounded border transition-colors select-none
                                                    ${isSelected
                                                        ? 'bg-primary/10 text-primary border-primary/30 font-medium'
                                                        : 'bg-muted text-muted-foreground border-border hover:bg-muted/70'
                                                    }`}
                                                title="Click to toggle, Double-click to select only this"
                                            >
                                                {FIELD_LABELS[field]}
                                            </button>
                                        )
                                    })}
                                </div>
                            )}
                        </div>
                    )}

                    {/* 分隔線 */}
                    <div className="h-5 w-px bg-border mx-1" />

                    {/* 匠出按鈕 */}
                    <Button
                        size="sm"
                        onClick={handleExport}
                        disabled={selectionCount === 0 || isExporting}
                        title={isExporting ? '匯出中...' : '匯出 Excel BOM'}
                    >
                        <Download size={13} className={isExporting ? 'animate-bounce' : ''} />
                        {isExporting ? '匯出中...' : '匯出'}
                    </Button>
                </div>

                {/* 錯誤提示 */}
                {error && (
                    <div className="flex items-center gap-2 px-3 py-1.5 bg-destructive/10
                        border border-destructive/30 rounded-lg text-xs text-destructive">
                        <span>{error}</span>
                        <button onClick={clearError} className="ml-auto hover:opacity-70"><X size={12} /></button>
                    </div>
                )}

                {/* ========================================
                    BOM 表格 — 主要內容區
                 ======================================== */}
                <div className="flex-1 min-h-0 overflow-hidden">
                    {selectionCount > 0 ? (
                        <BomTable
                            data={filteredBom}
                            isLoading={isLoading}
                            searchTerm={searchTerm}
                            searchFields={searchFields}
                            mode={bomMode}
                            viewContextIds={Array.from(selectedRevisionIds)}
                        />
                    ) : (
                        <div className="flex flex-col items-center justify-center h-full gap-3 text-muted-foreground">
                            <FileSpreadsheet size={36} className="text-border" />
                            <p className="text-sm">請從左側選擇 BOM 版本。</p>
                        </div>
                    )}
                </div>
            </div>

            {/* 拖曳覆蓋層 */}
            {isDragOver && (
                <div className="fixed inset-0 z-50 flex items-center justify-center
                    bg-primary/10 border-4 border-dashed border-primary
                    pointer-events-none">
                    <div className="bg-background rounded-xl shadow-lg px-8 py-6 text-center">
                        <p className="text-base font-semibold text-foreground">放開以匯入 Excel</p>
                        <p className="text-xs text-muted-foreground mt-1">支援 .xls / .xlsx 格式</p>
                    </div>
                </div>
            )}

            {/* Matrix Model 管理對話框 */}
            {/* Use first selected ID for model management? Or disable for multi-select? */}
            {/* User requirement: "MATRIX 模式支援一到多個BOM... UI 根據 Matrix_Models 的數量動態增加 DataGrid 的橫向欄位" */}
            {/* But management usually is per BOM. Multi-BOM model management is complex. */}
            {/* Let's disable for multi-select or use the first one. */}
            {selectedRevisionId && (
                <MatrixModelDialog
                    isOpen={isMatrixModelDialogOpen}
                    onClose={() => setIsMatrixModelDialogOpen(false)}
                    bomRevisionId={selectedRevisionId}
                />
            )}

            {/* 刪除確認對話框 */}
            <ConfirmDialog
                isOpen={!!deleteTarget}
                onClose={() => setDeleteTarget(null)}
                onConfirm={handleDeleteConfirm}
                title="刪除 BOM 版本"
                message={`確定要刪除 BOM 版本？此操作將一併刪除所有零件資料，且無法復原。`}
                confirmText="刪除"
                danger
            />
        </div>
    )
}

export default BomPage
