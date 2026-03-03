import { useEffect, useState } from 'react'
import {
    FolderPlus, FolderOpen, Clock, X, Database, ChevronRight,
    Pencil, FileText, Trash2, AlertCircle
} from 'lucide-react'
import useSeriesStore from '../stores/useSeriesStore'
import useProjectStore from '../stores/useProjectStore'
import useBomStore from '../stores/useBomStore'
import useTaskStore from '../stores/useTaskStore'
import useToastStore from '../stores/useToastStore'
import Dialog from '../components/dialogs/Dialog'
import ProjectDialog from '../components/dialogs/ProjectDialog'
import BomMetaDialog from '../components/dialogs/BomMetaDialog'
import ConfirmDialog from '../components/dialogs/ConfirmDialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent } from '@/components/ui/card'

// ========================================
// 儀表板 (Dashboard)
// 整合系列管理與專案/BOM 樹狀檢視
// ========================================

function Dashboard({ onNavigate }) {
    // --- Stores ---
    const {
        isOpen, currentSeries, currentPath, recentFiles, isLoading: isSeriesLoading, error: seriesError,
        initRecentFiles, createSeries, openSeries, closeSeries,
        updateDescription, renameSeries, removeFromRecentFiles, clearError: clearSeriesError,
    } = useSeriesStore()

    const { 
        projects, loadProjects, createProject, updateProject, deleteProject, 
        reset: resetProjects 
    } = useProjectStore()

    const { 
        selectProject, toggleRevisionSelection, updateRevision, deleteBom
    } = useBomStore()

    const registerCompletedCallback = useTaskStore(state => state.registerCompletedCallback)
    const addToast = useToastStore(state => state.addToast)

    // --- Local State ---
    // Series Description Edit
    const [isEditingDesc, setIsEditingDesc] = useState(false)
    const [editDesc, setEditDesc] = useState('')

    // Series Rename
    const [isRenameOpen, setIsRenameOpen] = useState(false)
    const [renameValue, setRenameValue] = useState('')
    const [renameError, setRenameError] = useState('')

    // Project Dialog (Create/Edit)
    const [projectDialog, setProjectDialog] = useState({ isOpen: false, mode: 'create', data: null })

    // BOM Meta Dialog (Edit)
    const [bomDialog, setBomDialog] = useState({ isOpen: false, data: null })

    // Delete Confirm
    const [deleteConfirm, setDeleteConfirm] = useState({ isOpen: false, type: '', data: null })

    // Tree Data: BOMs per Project { [projectId]: [revisions] }
    const [projectBoms, setProjectBoms] = useState({})
    // Expanded Projects { [projectId]: boolean }
    const [expandedProjects, setExpandedProjects] = useState({})

    // --- Effects ---

    // Init Recent Files
    useEffect(() => {
        initRecentFiles()
    }, [initRecentFiles])

    // 系列開啟時載入專案，關閉時重置
    useEffect(() => {
        if (isOpen) {
            loadProjects()
        } else {
            resetProjects()
            // eslint-disable-next-line react-hooks/set-state-in-effect
            setProjectBoms({})  // 系列關閉時一次性清除本地 BOM 快取
            setExpandedProjects({})  // 同步清除幕開狀態
        }
    }, [isOpen, loadProjects, resetProjects])

    // 載入所有專案的 BOM 資料並建立樹狀結構
    useEffect(() => {
        if (isOpen && projects.length > 0) {
            projects.forEach(p => {
                window.api.bom.getRevisions(p.id).then(async (res) => {
                    if (res.success) {
                        const boms = res.data;
                        // 取得每個 BOM 的 Matrix 要約
                        const bomsWithMatrix = await Promise.all(boms.map(async (bom) => {
                             try {
                                 const summaryRes = await window.api.matrix.getSummary(bom.id);
                                 return {
                                     ...bom,
                                     matrixSummary: summaryRes.success ? summaryRes.data : null
                                 };
                             } catch (_e) {
                                 return bom;
                             }
                        }));
                        setProjectBoms(prev => ({ ...prev, [p.id]: bomsWithMatrix }))
                    }
                })
            })
            // 預設展開所有專案：在待機任務中執行，避免在 Effect 同步路徑中 setState
            Promise.resolve().then(() => {
                const initialExpanded = {}
                projects.forEach(p => initialExpanded[p.id] = true)
                setExpandedProjects(prev => ({ ...initialExpanded, ...prev }))
            })
        }
    }, [isOpen, projects])

    // 註冊 BATCH_IMPORT 完成時的 Callback
    useEffect(() => {
        if (!isOpen) return

        const unsubscribe = registerCompletedCallback('BATCH_IMPORT', async (data) => {
            const { result } = data
            // Batch Import 會自行處理 DB 更新
            // 當它完成時，直接重新載入所有的 Projects 和 BOMs
            if (result && !result.error) {
                // 重新刷新左側樹狀結構
                loadProjects()
            } else {
                const errMsg = result ? result.error : '未知錯誤'
                addToast(`匯入失敗：${errMsg}`, 'error')
            }
        })

        return () => unsubscribe()
    }, [isOpen, registerCompletedCallback, loadProjects, addToast])

    // 每個 IMPORT_BOM 子任務完成後，靜默刷新專案/BOM 樹
    // BATCH_IMPORT 完成時僅代表「排佇列完成」，IMPORT_BOM 才是真正的檔案匯入
    useEffect(() => {
        if (!isOpen) return

        const unsubscribe = registerCompletedCallback('IMPORT_BOM', async (data) => {
            const { result } = data
            if (result?.success !== false) {
                // 靜默重新載入專案列表；React diff 只更新有變化的節點，不會閃爍
                loadProjects()
            }
        })

        return () => unsubscribe()
    }, [isOpen, registerCompletedCallback, loadProjects])

    // --- Handlers: Series ---

    const handleSeriesCreate = async () => createSeries()
    const handleSeriesOpen = async (filePath) => openSeries(filePath)
    const handleSeriesClose = () => closeSeries()

    const handleStartEditDesc = () => {
        setEditDesc(currentSeries?.description || '')
        setIsEditingDesc(true)
    }

    const handleSaveDesc = async () => {
        await updateDescription(editDesc)
        setIsEditingDesc(false)
    }

    const handleOpenRename = () => {
        const name = currentPath?.split(/[\\/]/).pop().replace('.bomix', '') || ''
        setRenameValue(name)
        setRenameError('')
        setIsRenameOpen(true)
    }

    const handleRenameSubmit = async () => {
        if (!renameValue.trim()) return
        const result = await renameSeries(renameValue.trim())
        if (result.success) setIsRenameOpen(false)
        else setRenameError(result.error)
    }

    // --- Handlers: Project ---

    const handleCreateProject = () => {
        setProjectDialog({ isOpen: true, mode: 'create', data: null })
    }

    const handleEditProject = (project) => {
        setProjectDialog({ isOpen: true, mode: 'edit', data: project })
    }

    const handleProjectSave = async (code, desc) => {
        if (projectDialog.mode === 'create') {
            // create 模式對应到 AppLayout 的全域按鈕，此處保留 handler 以防萬一
            await createProject(code, desc)
        } else {
            await updateProject(projectDialog.data.id, { project_code: code, description: desc })
        }
        setProjectDialog({ ...projectDialog, isOpen: false })
    }

    const handleDeleteProject = (project) => {
        setDeleteConfirm({ 
            isOpen: true, 
            type: 'project', 
            data: project,
            title: '刪除專案',
            message: `確定要刪除專案「${project.project_code}」？此操作將刪除該專案下所有 BOM 資料且無法復原。`,
            danger: true,
            confirmText: "刪除"
        })
    }

    const toggleProjectExpand = (projectId) => {
        setExpandedProjects(prev => ({ ...prev, [projectId]: !prev[projectId] }))
    }

    // --- Handlers: BOM ---

    const handleEditBom = (bom, projectCode) => {
        setBomDialog({ isOpen: true, data: bom, projectCode })
    }

    const handleBomSave = async (id, updates) => {
        const result = await updateRevision(id, updates)
        if (result.success) {
            // Refresh local BOM list for this project
            // We need to know which project this BOM belongs to.
            // Updates return the updated object.
            // Or we just re-fetch revisions for that project.
            // Manual update:
            // updateRevision returns { success: true } in store, BUT I should have it return data.
            // In Store:
            /*
            if (result.success) {
                // ...
                return { success: true } // It doesn't return data in my modification!
            }
            */
            // I should re-fetch revisions for the project.
            // I'll re-fetch all for simplicity or find projectId from bom.
            const projectId = bomDialog.data.project_id
            const res = await window.api.bom.getRevisions(projectId)
            if (res.success) {
                setProjectBoms(prev => ({ ...prev, [projectId]: res.data }))
            }
        }
        setBomDialog({ ...bomDialog, isOpen: false })
    }

    const handleBomClick = async (bom) => {
        await selectProject(bom.project_id)
        await toggleRevisionSelection(bom.id, false) // false for single select
        onNavigate('bom')
    }

    const handleDeleteBom = (bom) => {
        setDeleteConfirm({ 
            isOpen: true, 
            type: 'bom', 
            data: bom,
            title: '刪除 BOM 版本',
            message: `確定要刪除 BOM 版本「${bom.phase_name} ${bom.version}」？`,
            danger: true,
            confirmText: "刪除"
        })
    }

    // --- Handlers: Import ---
    // (已移至 AppLayout 全域套且，此處保留空白）
    
    const handleConfirmAction = async () => {
        const { type, data } = deleteConfirm
        
        // Original Delete Logic
        if (type === 'project') {
             // ...
             await deleteProject(data.id)
        } else if (type === 'bom') {
             await deleteBom(data.id)
             // Refresh BOM list
             const res = await window.api.bom.getRevisions(data.project_id)
             if (res.success) {
                 setProjectBoms(prev => ({ ...prev, [data.project_id]: res.data }))
             }
        }
        setDeleteConfirm({ ...deleteConfirm, isOpen: false })
    }


    // ========================================
    // A. 歡迎畫面 (未開啟系列)
    // ========================================
    if (!isOpen) {
        return (
            <div className="flex flex-col items-center justify-center h-full gap-6 p-6 overflow-auto animate-fade-in relative">
                {/* 錯誤提示 */}
                {seriesError && (
                    <div className="absolute top-4 left-1/2 -translate-x-1/2 flex items-center gap-2 px-3 py-2 bg-destructive/10 border border-destructive/30 rounded-lg text-xs text-destructive shadow-md">
                        <AlertCircle size={13} />
                        <span>{seriesError}</span>
                        <Button variant="ghost" size="icon" className="h-5 w-5 ml-1" onClick={clearSeriesError}><X size={12} /></Button>
                    </div>
                )}

                <div className="text-center">
                    <h1 className="text-3xl font-bold text-primary mb-1">BOMIX</h1>
                    <p className="text-sm text-muted-foreground">BOM 變化管理與追蹤工具</p>
                </div>

                <div className="flex gap-3">
                    <button onClick={handleSeriesCreate} disabled={isSeriesLoading}
                        className="flex flex-col items-center gap-2 px-8 py-5 bg-card rounded-xl shadow-sm hover:shadow-md border border-border hover:border-primary/50 transition-all duration-200 cursor-pointer group disabled:opacity-50">
                        <span className="text-primary group-hover:scale-110 transition-transform"><FolderPlus size={28} /></span>
                        <span className="text-xs font-medium text-foreground">建立新系列</span>
                    </button>
                    <button onClick={() => handleSeriesOpen()} disabled={isSeriesLoading}
                        className="flex flex-col items-center gap-2 px-8 py-5 bg-card rounded-xl shadow-sm hover:shadow-md border border-border hover:border-primary/50 transition-all duration-200 cursor-pointer group disabled:opacity-50">
                        <span className="text-primary group-hover:scale-110 transition-transform"><FolderOpen size={28} /></span>
                        <span className="text-xs font-medium text-foreground">開啟系列</span>
                    </button>
                </div>

                <div className="w-full max-w-md">
                    <h2 className="text-xs font-semibold text-muted-foreground mb-2 flex items-center gap-1.5">
                        <Clock size={12} /> 最近開啟
                    </h2>
                    {recentFiles.length > 0 ? (
                        <Card>
                            <CardContent className="p-0 divide-y divide-border">
                                {recentFiles.map((filePath) => {
                                    const name = filePath.split(/[\/\\]/).pop()?.replace('.bomix', '') || filePath
                                    return (
                                        <div key={filePath} className="flex items-center justify-between px-3 py-2 hover:bg-muted/50 transition-colors group">
                                            <button onClick={() => handleSeriesOpen(filePath)} className="flex-1 flex items-center gap-2 text-left min-w-0">
                                                <Database size={14} className="shrink-0 text-primary" />
                                                <div className="min-w-0">
                                                    <p className="text-xs font-medium text-foreground truncate">{name}</p>
                                                    <p className="text-xs text-muted-foreground truncate font-mono">{filePath}</p>
                                                </div>
                                                <ChevronRight size={12} className="shrink-0 text-muted-foreground/40 group-hover:text-primary transition-colors" />
                                            </button>
                                            <Button variant="ghost" size="icon" className="h-6 w-6 opacity-0 group-hover:opacity-100 transition-opacity" onClick={() => removeFromRecentFiles(filePath)}>
                                                <X size={12} />
                                            </Button>
                                        </div>
                                    )
                                })}
                            </CardContent>
                        </Card>
                    ) : (
                        <div className="p-4 text-center text-xs text-muted-foreground border border-border rounded-xl">尚無開啟記錄</div>
                    )}
                </div>
            </div>
        )
    }

    // ========================================
    // B. Dashboard (已開啟系列)
    // ========================================
    const displayName = currentPath?.split(/[\/\\]/).pop().replace('.bomix', '') || '未命名系列'

    return (
        <div className="flex flex-col h-full animate-fade-in bg-muted/30">
            {/* 1. Header Area: Series Info */}
            <div className="bg-background border-b border-border px-4 py-3 flex items-center justify-between shrink-0">
                <div className="flex items-center gap-3 min-w-0">
                    <div className="p-2 bg-primary/10 text-primary rounded-lg shrink-0">
                        <Database size={18} />
                    </div>
                    <div className="min-w-0">
                        <div className="flex items-center gap-1.5">
                            <h2 className="text-sm font-bold text-foreground truncate">{displayName}</h2>
                            <Button variant="ghost" size="icon" className="h-6 w-6" onClick={handleOpenRename}>
                                <Pencil size={12} />
                            </Button>
                        </div>
                        {isEditingDesc ? (
                            <div className="flex gap-2 mt-0.5">
                                <Input
                                    value={editDesc}
                                    onChange={(e) => setEditDesc(e.target.value)}
                                    className="h-7 text-xs w-64"
                                    autoFocus
                                    onKeyDown={(e) => e.key === 'Enter' && handleSaveDesc()}
                                />
                                <Button size="sm" className="h-7 text-xs" onClick={handleSaveDesc}>儲存</Button>
                            </div>
                        ) : (
                            <div className="flex items-center gap-1.5 group cursor-pointer" onClick={handleStartEditDesc}>
                                <p className="text-xs text-muted-foreground truncate max-w-xs">{currentSeries.description || '點擊新增描述...'}</p>
                                <Pencil size={11} className="opacity-0 group-hover:opacity-100 text-muted-foreground" />
                            </div>
                        )}
                    </div>
                </div>
                <Button variant="ghost" size="icon" className="h-8 w-8" onClick={handleSeriesClose} title="關閉系列">
                    <X size={16} />
                </Button>
            </div>

            {/* 2. Content Area: Tree View */}
            <div className="flex-1 overflow-auto p-4">
                <div className="max-w-5xl mx-auto">
                    {projects.length === 0 ? (
                        <div className="text-center py-16 text-muted-foreground">
                            <FolderOpen size={40} className="mx-auto mb-3 opacity-20" />
                            <p className="text-sm">尚無專案，請點擊右上方「新增專案」</p>
                        </div>
                    ) : (
                        <div className="space-y-3">
                            {projects.map(project => (
                                <div key={project.id} className="bg-background rounded-lg border border-border shadow-sm overflow-hidden">
                                    {/* Project Header */}
                                    <div
                                        className="flex items-center justify-between px-3 py-2.5 cursor-pointer hover:bg-muted/50 transition-colors"
                                        onClick={() => toggleProjectExpand(project.id)}
                                    >
                                        <div className="flex items-center gap-2">
                                            <ChevronRight size={15} className={`text-muted-foreground/60 transition-transform ${expandedProjects[project.id] ? 'rotate-90' : ''}`} />
                                            <FolderOpen size={16} className="text-sky-400" />
                                            <div>
                                                <h3 className="text-sm font-medium text-foreground">{project.project_code}</h3>
                                                {project.description && <p className="text-xs text-muted-foreground">{project.description}</p>}
                                            </div>
                                        </div>
                                        <div className="flex items-center gap-1" onClick={e => e.stopPropagation()}>
                                            <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => handleEditProject(project)} title="編輯專案">
                                                <Pencil size={13} />
                                            </Button>
                                            <Button variant="ghost" size="icon" className="h-7 w-7 text-destructive/60 hover:text-destructive" onClick={() => handleDeleteProject(project)} title="刪除專案">
                                                <Trash2 size={13} />
                                            </Button>
                                        </div>
                                    </div>

                                    {/* BOM List (Children) */}
                                    {expandedProjects[project.id] && (
                                        <div className="border-t border-border bg-muted/20 px-3 py-1.5">
                                            <div className="ml-8 space-y-0.5">
                                                {projectBoms[project.id]?.length > 0 ? (
                                                    projectBoms[project.id].map(bom => (
                                                        <div
                                                            key={bom.id}
                                                            className="flex items-center justify-between px-2 py-1.5 rounded-md hover:bg-background border border-transparent hover:border-border transition-all cursor-pointer group"
                                                            onClick={() => handleBomClick(bom)}
                                                        >
                                                            <div className="flex items-center gap-2">
                                                                <FileText size={13} className="text-emerald-400" />
                                                                <span className="text-xs font-medium text-foreground">
                                                                    {bom.phase_name} {bom.version}
                                                                    {bom.suffix && <span className="ml-1 text-muted-foreground">-{bom.suffix}</span>}
                                                                </span>
                                                                <span className="text-[10px] px-1.5 py-0.5 bg-muted text-muted-foreground rounded">
                                                                    {bom.mode || 'NPI'}
                                                                </span>
                                                                {bom.matrixSummary?.hasMatrix && (
                                                                    <div className="flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] border border-border" title={bom.matrixSummary.isSafe ? 'Matrix Complete' : 'Matrix Incomplete'}>
                                                                        <span className="font-semibold text-muted-foreground">M</span>
                                                                        <div className={`w-1.5 h-1.5 rounded-full ${bom.matrixSummary.isSafe ? 'bg-emerald-500' : 'bg-amber-400'}`} />
                                                                    </div>
                                                                )}
                                                                {bom.bom_date && <span className="text-[10px] text-muted-foreground">{bom.bom_date}</span>}
                                                            </div>
                                                            <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity" onClick={e => e.stopPropagation()}>
                                                                <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => handleEditBom(bom, project.project_code)} title="編輯 BOM 屬性">
                                                                    <Pencil size={12} />
                                                                </Button>
                                                                <Button variant="ghost" size="icon" className="h-6 w-6 text-destructive/60 hover:text-destructive" onClick={() => handleDeleteBom(bom)} title="刪除 BOM">
                                                                    <Trash2 size={12} />
                                                                </Button>
                                                            </div>
                                                        </div>
                                                    ))
                                                ) : (
                                                    <div className="text-xs text-muted-foreground py-1.5 pl-2 italic">尚無 BOM 版本</div>
                                                )}
                                            </div>
                                        </div>
                                    )}
                                </div>
                            ))}
                        </div>
                    )}
                </div>
            </div>

            {/* 對話框 */}
            <Dialog isOpen={isRenameOpen} onClose={() => setIsRenameOpen(false)} title="重新命名系列" className="max-w-sm">
                <div className="space-y-3">
                    <Input
                        value={renameValue}
                        onChange={(e) => setRenameValue(e.target.value)}
                        placeholder="輸入新的系列名稱"
                        className="h-8 text-xs"
                        autoFocus
                        onKeyDown={(e) => e.key === 'Enter' && handleRenameSubmit()}
                    />
                    {renameError && <p className="text-xs text-destructive">{renameError}</p>}
                    <div className="flex justify-end gap-2">
                        <Button variant="outline" size="sm" onClick={() => setIsRenameOpen(false)}>取消</Button>
                        <Button size="sm" onClick={handleRenameSubmit}>確定</Button>
                    </div>
                </div>
            </Dialog>

            <ProjectDialog
                isOpen={projectDialog.isOpen}
                onClose={() => setProjectDialog({ ...projectDialog, isOpen: false })}
                mode={projectDialog.mode}
                initialData={projectDialog.data}
                onSave={handleProjectSave}
            />

            <BomMetaDialog 
                isOpen={bomDialog.isOpen}
                onClose={() => setBomDialog({ ...bomDialog, isOpen: false })}
                bom={bomDialog.data}
                projectCode={bomDialog.projectCode}
                onSave={handleBomSave}
            />

            <ConfirmDialog 
                isOpen={deleteConfirm.isOpen}
                onClose={() => setDeleteConfirm({ ...deleteConfirm, isOpen: false })}
                onConfirm={handleConfirmAction}
                title={deleteConfirm.title}
                message={deleteConfirm.message}
                confirmText={deleteConfirm.confirmText || "確認"}
                danger={deleteConfirm.danger} // Keeping for backward compatibility if mixed usage
                variant={deleteConfirm.variant || (deleteConfirm.danger ? 'danger' : 'primary')}
            />
        </div>
    )
}

export default Dashboard
