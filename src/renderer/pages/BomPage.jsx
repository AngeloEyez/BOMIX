import { useEffect, useState, useMemo } from 'react'
import { FolderOpen, FileSpreadsheet } from 'lucide-react'
import useSeriesStore from '../stores/useSeriesStore'
import useProjectStore from '../stores/useProjectStore'
import useBomStore from '../stores/useBomStore'
import useTaskStore from '../stores/useTaskStore'
import useToastStore from '../stores/useToastStore'
import useMatrixStore from '../stores/useMatrixStore'

import BomSidebar from '../components/layout/BomSidebar'
import ConfirmDialog from '../components/dialogs/ConfirmDialog'
import MatrixModelDialog from '../components/dialogs/MatrixModelDialog'
import BomToolbar from '../components/toolbar/BomToolbar'
import StandardBomView from '../components/views/StandardBomView'
import MatrixBomView from '../components/views/MatrixBomView'

import {
    ResizablePanelGroup,
    ResizablePanel,
    ResizableHandle
} from '@/components/ui/resizable'

// ========================================
// BOM 檢視頁面
// 提供專案/版本選取、BOM 表格檢視、Excel 匯入匯出功能
// ========================================

const DEFAULT_SEARCH_FIELDS = ['hhpn', 'description', 'supplier', 'supplier_pn', 'location']

/**
 * BOM 檢視頁面元件。
 *
 * 包含工具列 (專案選擇、版本選擇、匯入/匯出按鈕)
 * 及 BOM 聚合表格。使用緊湊佈局以最大化資料展示面積。
 * 支援多種顯示模式：標準 BOM、MATRIX 等。
 */
export default function BomPage() {
    const { isOpen } = useSeriesStore()
    const { projects, allBoms, loadProjects } = useProjectStore()
    const {
        selectedProjectId, selectedRevisionId, selectedRevisionIds, selectedRevision,
        bomView, isLoading, error,
        deleteBom, exportExcel,
        clearError, reset,
        currentViewId, selectView, 
        bomMode,
        cclFilter, setCclFilter
    } = useBomStore()

    const isExporting = useTaskStore(state => 
        Array.from(state.sessions.values()).some(s => s.type === 'EXPORT_BOM' && s.status === 'RUNNING')
    )
    const registerCompletedCallback = useTaskStore(state => state.registerCompletedCallback)
    const addToast = useToastStore(state => state.addToast)
    const { fetchMatrixData } = useMatrixStore()

    const [deleteTarget, setDeleteTarget] = useState(null)
    const [isMatrixModelDialogOpen, setIsMatrixModelDialogOpen] = useState(false)
    const [views, setViews] = useState({})
    
    // 搜尋狀態
    const [searchTerm, setSearchTerm] = useState('')
    const [searchFields, setSearchFields] = useState(new Set(DEFAULT_SEARCH_FIELDS))

    const defaultLayout = useMemo(() => {
        try {
            const saved = window.localStorage.getItem('bom-layout')
            if (saved) return JSON.parse(saved)
        } catch(_e) { /* ignore */ }
        return undefined
    }, [])

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

    useEffect(() => {
        if (bomMode === 'MATRIX' && selectedRevisionIds.size > 0) {
            const ids = Array.from(selectedRevisionIds)
            fetchMatrixData(ids)
        }
    }, [selectedRevisionIds, bomMode, fetchMatrixData])

    // 過濾邏輯
    const filteredBom = useMemo(() => {
        let data = bomView
        if (!Array.isArray(data) || !searchTerm || !searchTerm.trim()) return data || []
        const term = searchTerm.toLowerCase().trim()
        
        const checkMatch = (item) => {
            return Array.from(searchFields).some(field => {
                const value = field === 'location' ? item.locations : item[field]
                return value && String(value).toLowerCase().includes(term)
            })
        }
        
        return data.filter(mainItem => {
            if (checkMatch(mainItem)) return true
            if (mainItem.second_sources && mainItem.second_sources.length > 0) {
                if (mainItem.second_sources.some(ss => checkMatch(ss))) return true
            }
            return false
        })
    }, [bomView, searchTerm, searchFields])

    useEffect(() => {
        if (isOpen) {
            loadProjects()
        } else {
            reset()
        }
    }, [isOpen, loadProjects, reset])

    useEffect(() => {
        if (!isOpen) return
        const unsubscribe = registerCompletedCallback('BATCH_IMPORT', async (data) => {
            const { result } = data
            if (result && !result.error) {
                await useBomStore.getState().reloadRevisions()
            } else {
                const errMsg = result ? result.error : '未知錯誤'
                addToast(`匯入失敗：${errMsg}`, 'error')
            }
        })
        return () => unsubscribe()
    }, [isOpen, registerCompletedCallback, addToast])

    useEffect(() => {
        if (!isOpen) return
        const unsubscribe = registerCompletedCallback('IMPORT_BOM', async (data) => {
            const { result } = data
            if (result?.success !== false) {
                await loadProjects()
                await useBomStore.getState().reloadRevisions()
            }
        })
        return () => unsubscribe()
    }, [isOpen, registerCompletedCallback, loadProjects])

    const handleDeleteConfirm = async () => {
        if (deleteTarget) {
            await deleteBom(deleteTarget)
            setDeleteTarget(null)
        }
    }

    const handleExport = () => {
        if (selectedRevisionId) {
            exportExcel(selectedRevisionId)
        }
    }

    const selectedProjectCode = useMemo(() => {
        const pid = selectedRevision?.project_id || selectedProjectId
        if (pid && projects.length > 0) {
            const project = projects.find(p => p.id == pid)
            return project ? project.project_code : ''
        }
        return ''
    }, [selectedRevision, selectedProjectId, projects])

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

    if (!isOpen) {
        return (
            <div className="flex flex-col items-center justify-center h-full gap-4 text-muted-foreground animate-fade-in">
                <FolderOpen size={40} className="text-border" />
                <h2 className="text-lg font-semibold">尚未開啟系列</h2>
                <p className="text-sm">請先從首頁建立或開啟系列資料庫，再檢視 BOM。</p>
            </div>
        )
    }

    const selectionCount = selectedRevisionIds.size
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

    // View Routing
    let MainContentView = null
    if (selectionCount > 0) {
        if (bomMode === 'MATRIX') {
            MainContentView = (
                <MatrixBomView
                    data={filteredBom}
                    isLoading={isLoading}
                    searchTerm={searchTerm}
                    searchFields={searchFields}
                    viewContextIds={Array.from(selectedRevisionIds)}
                />
            )
        } else {
            MainContentView = (
                <StandardBomView
                    data={filteredBom}
                    isLoading={isLoading}
                    searchTerm={searchTerm}
                    searchFields={searchFields}
                    viewContextIds={Array.from(selectedRevisionIds)}
                />
            )
        }
    } else {
        MainContentView = (
            <div className="flex flex-col items-center justify-center h-full gap-3 text-muted-foreground">
                <FileSpreadsheet size={36} className="text-border" />
                <p className="text-sm">請從左側選擇 BOM 版本。</p>
            </div>
        )
    }

    return (
        <div className="flex h-full animate-fade-in">
            <ResizablePanelGroup 
                direction="horizontal" 
                className="h-full" 
                defaultLayout={defaultLayout}
                onLayoutChanged={(sizes) => {
                    window.localStorage.setItem('bom-layout', JSON.stringify(sizes))
                }}
            >
                <ResizablePanel
                    id="sidebar"
                    defaultSize="12%"
                    minSize="5%"
                    collapsible={true}
                    collapsedSize="0.7%"
                    className="min-w-0"
                >
                    <BomSidebar />
                </ResizablePanel>

                <ResizableHandle withHandle />

                <ResizablePanel id="main" defaultSize="88%" minSize="75%">
                    <div className="flex flex-1 flex-col h-full min-w-0 p-3 gap-2">
                        <BomToolbar
                            headerTitle={headerTitle}
                            selectionCount={selectionCount}
                            bomMode={bomMode}
                            views={views}
                            currentViewId={currentViewId}
                            selectView={selectView}
                            cclFilter={cclFilter}
                            setCclFilter={setCclFilter}
                            searchTerm={searchTerm}
                            setSearchTerm={setSearchTerm}
                            searchFields={searchFields}
                            setSearchFields={setSearchFields}
                            isExporting={isExporting}
                            onExport={handleExport}
                            onOpenMatrixSettings={() => setIsMatrixModelDialogOpen(true)}
                            error={error}
                            clearError={clearError}
                        />

                        <div className="flex-1 min-h-0 overflow-hidden bg-background rounded-lg shadow-sm">
                            {MainContentView}
                        </div>
                    </div>
                </ResizablePanel>
            </ResizablePanelGroup>

            {selectedRevisionId && (
                <MatrixModelDialog
                    isOpen={isMatrixModelDialogOpen}
                    onClose={() => setIsMatrixModelDialogOpen(false)}
                    bomRevisionId={selectedRevisionId}
                />
            )}

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
