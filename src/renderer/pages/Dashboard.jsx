import { useEffect, useState, useCallback } from 'react'
import { 
    FolderPlus, FolderOpen, Clock, X, Database, ChevronRight, ChevronDown, 
    Pencil, Check, FileText, Settings, Trash2, MoreVertical 
} from 'lucide-react'
import useSeriesStore from '../stores/useSeriesStore'
import useProjectStore from '../stores/useProjectStore'
import useBomStore from '../stores/useBomStore'
import Dialog from '../components/dialogs/Dialog'
import ProjectDialog from '../components/dialogs/ProjectDialog' // Assuming this exists or I'll need to check location
import BomMetaDialog from '../components/dialogs/BomMetaDialog'
import ConfirmDialog from '../components/dialogs/ConfirmDialog'

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
        isLoading: isProjectLoading, reset: resetProjects 
    } = useProjectStore()

    const { 
        selectProject, selectRevision, updateRevision, deleteBom
    } = useBomStore()

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

    // Load Projects when Series Opens
    useEffect(() => {
        if (isOpen) {
            loadProjects()
        } else {
            resetProjects()
            setProjectBoms({})
            setExpandedProjects({})
        }
    }, [isOpen, loadProjects, resetProjects])

    // Fetch BOMs for all projects (or lazy load)
    // Here we fetch all to show the full tree structure capabilities
    useEffect(() => {
        if (isOpen && projects.length > 0) {
            projects.forEach(p => {
                window.api.bom.getRevisions(p.id).then(res => {
                    if (res.success) {
                        setProjectBoms(prev => ({ ...prev, [p.id]: res.data }))
                    }
                })
            })
            // Default expand all? Or collapse? Let's expand all for visibility.
            const initialExpanded = {}
            projects.forEach(p => initialExpanded[p.id] = true)
            setExpandedProjects(prev => ({ ...initialExpanded, ...prev }))
        }
    }, [isOpen, projects])

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
            await createProject(code, desc)
        } else {
            // Check project.service.js updateProject(id, data)
            // It expects { project_code, description }
            // If code is not changed, maybe don't send it? Or service handles unique check.
            // Since we upgraded service to support project_code update.
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
            message: `確定要刪除專案「${project.project_code}」？此操作將刪除該專案下所有 BOM 資料且無法復原。`
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
            const projectIds = Object.keys(projectBoms)
            // It's inefficient to find owner, but we can iterate.
            // Or use result.success (stores updated).
            // But we need to update `projectBoms` state to reflect changes in Tree.
            // `updateRevision` updates `useBomStore` revisions, but NOT `projectBoms` local state.
            // So we need to re-fetch or manual update.
            // Manual update:
            const updatedBom = result.access ? result.data : null // Wait, updateRevision returns { success, data }?
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
        await selectRevision(bom.id)
        onNavigate('bom')
    }

    const handleDeleteBom = (bom) => {
        setDeleteConfirm({ 
            isOpen: true, 
            type: 'bom', 
            data: bom,
            title: '刪除 BOM 版本',
            message: `確定要刪除 BOM 版本「${bom.phase_name} ${bom.version}」？`
        })
    }

    const handleConfirmDelete = async () => {
        const { type, data } = deleteConfirm
        if (type === 'project') {
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
            <div className="flex flex-col items-center justify-center h-full gap-8 p-6 overflow-auto animate-fade-in relative">
                 {/* 錯誤提示 */}
                 {seriesError && (
                    <div className="absolute top-4 left-1/2 -translate-x-1/2 flex items-center gap-2 px-4 py-2 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg text-sm text-red-600 dark:text-red-400 shadow-md">
                        <span>{seriesError}</span>
                        <button onClick={clearSeriesError} className="ml-2 hover:text-red-800">
                            <X size={14} />
                        </button>
                    </div>
                )}
            
                <div className="text-center">
                    <h1 className="text-4xl font-bold text-primary-600 dark:text-primary-400 mb-2">BOMIX</h1>
                    <p className="text-lg text-slate-500 dark:text-slate-400">BOM 變化管理與追蹤工具</p>
                </div>

                <div className="flex gap-4">
                    <button onClick={handleSeriesCreate} disabled={isSeriesLoading}
                        className="flex flex-col items-center gap-2 px-8 py-6 bg-white dark:bg-surface-800 rounded-xl shadow-sm hover:shadow-md border border-slate-200 dark:border-slate-700 hover:border-primary-300 dark:hover:border-primary-600 transition-all duration-200 cursor-pointer group disabled:opacity-50">
                        <span className="text-primary-500 group-hover:scale-110 transition-transform"><FolderPlus size={32} /></span>
                        <span className="text-sm font-medium text-slate-700 dark:text-slate-300">建立新系列</span>
                    </button>
                    <button onClick={() => handleSeriesOpen()} disabled={isSeriesLoading}
                        className="flex flex-col items-center gap-2 px-8 py-6 bg-white dark:bg-surface-800 rounded-xl shadow-sm hover:shadow-md border border-slate-200 dark:border-slate-700 hover:border-primary-300 dark:hover:border-primary-600 transition-all duration-200 cursor-pointer group disabled:opacity-50">
                        <span className="text-primary-500 group-hover:scale-110 transition-transform"><FolderOpen size={32} /></span>
                        <span className="text-sm font-medium text-slate-700 dark:text-slate-300">開啟系列</span>
                    </button>
                </div>

                <div className="w-full max-w-md">
                    <h2 className="text-sm font-semibold text-slate-500 dark:text-slate-400 mb-3 flex items-center gap-2">
                        <Clock size={14} /> 最近開啟
                    </h2>
                    {recentFiles.length > 0 ? (
                        <div className="bg-white dark:bg-surface-800 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden divide-y divide-slate-100 dark:divide-slate-700">
                            {recentFiles.map((filePath) => {
                                const name = filePath.split(/[\\/]/).pop()?.replace('.bomix', '') || filePath
                                return (
                                    <div key={filePath} className="flex items-center justify-between p-3 hover:bg-slate-50 dark:hover:bg-surface-700/50 transition-colors group">
                                        <button onClick={() => handleSeriesOpen(filePath)} className="flex-1 flex items-center gap-3 text-left min-w-0">
                                            <Database size={16} className="shrink-0 text-primary-500" />
                                            <div className="min-w-0">
                                                <p className="text-sm font-medium text-slate-700 dark:text-slate-200 truncate">{name}</p>
                                                <p className="text-xs text-slate-400 truncate font-mono">{filePath}</p>
                                            </div>
                                            <ChevronRight size={14} className="shrink-0 text-slate-300 group-hover:text-primary-500 transition-colors" />
                                        </button>
                                        <button onClick={() => removeFromRecentFiles(filePath)} className="shrink-0 ml-2 p-1 rounded hover:bg-slate-200 dark:hover:bg-surface-600 text-slate-300 hover:text-slate-500 opacity-0 group-hover:opacity-100 transition-all">
                                            <X size={14} />
                                        </button>
                                    </div>
                                )
                            })}
                        </div>
                    ) : (
                        <div className="p-4 text-center text-sm text-slate-400 border border-slate-200 dark:border-slate-700 rounded-xl">尚無開啟記錄</div>
                    )}
                </div>
            </div>
        )
    }

    // ========================================
    // B. Dashboard (已開啟系列)
    // ========================================
    const displayName = currentPath?.split(/[\\/]/).pop().replace('.bomix', '') || '未命名系列'

    return (
        <div className="flex flex-col h-full animate-fade-in bg-slate-50 dark:bg-surface-950">
            {/* 1. Header Area: Series Info */}
            <div className="bg-white dark:bg-surface-900 border-b border-slate-200 dark:border-slate-800 px-6 py-4 flex items-center justify-between shrink-0">
                <div className="flex items-center gap-4 min-w-0">
                    <div className="p-3 bg-primary-100 dark:bg-primary-900/30 text-primary-600 dark:text-primary-400 rounded-xl">
                        <Database size={24} />
                    </div>
                    <div className="min-w-0">
                        <div className="flex items-center gap-2">
                            <h2 className="text-xl font-bold text-slate-800 dark:text-white truncate">{displayName}</h2>
                            <button onClick={handleOpenRename} className="p-1 rounded-md text-slate-400 hover:text-primary-600 hover:bg-slate-100 dark:hover:bg-surface-800 transition-colors">
                                <Pencil size={14} />
                            </button>
                        </div>
                        {isEditingDesc ? (
                            <div className="flex gap-2 mt-1">
                                <input
                                    type="text"
                                    value={editDesc}
                                    onChange={(e) => setEditDesc(e.target.value)}
                                    className="px-2 py-0.5 text-sm border rounded bg-white dark:bg-surface-800 dark:border-slate-700 dark:text-slate-200 focus:ring-1 focus:ring-primary-500"
                                    autoFocus
                                    onKeyDown={(e) => e.key === 'Enter' && handleSaveDesc()}
                                />
                                <button onClick={handleSaveDesc} className="px-2 py-0.5 bg-primary-600 text-white rounded text-xs">儲存</button>
                            </div>
                        ) : (
                            <div className="flex items-center gap-2 group cursor-pointer mt-1" onClick={handleStartEditDesc}>
                                <p className="text-sm text-slate-500 truncate max-w-lg">{currentSeries.description || '點擊新增描述...'}</p>
                                <Pencil size={12} className="opacity-0 group-hover:opacity-100 text-slate-400" />
                            </div>
                        )}
                    </div>
                </div>
                <div className="flex items-center gap-3">
                     <button onClick={handleCreateProject} className="flex items-center gap-1.5 px-3 py-1.5 bg-primary-600 text-white text-sm font-medium rounded-lg hover:bg-primary-700 shadow-sm transition-colors">
                        <FolderPlus size={16} />
                        新增專案
                    </button>
                    <button onClick={handleSeriesClose} className="p-2 text-slate-400 hover:bg-slate-100 dark:hover:bg-surface-800 rounded-lg transition-colors" title="關閉系列">
                        <X size={20} />
                    </button>
                </div>
            </div>

            {/* 2. Content Area: Tree View */}
            <div className="flex-1 overflow-auto p-6">
                <div className="max-w-5xl mx-auto">
                    {projects.length === 0 ? (
                        <div className="text-center py-20 text-slate-400">
                            <FolderOpen size={48} className="mx-auto mb-4 opacity-20" />
                            <p>尚無專案，請點擊右上方「新增專案」</p>
                        </div>
                    ) : (
                        <div className="space-y-4">
                            {projects.map(project => (
                                <div key={project.id} className="bg-white dark:bg-surface-900 rounded-xl border border-slate-200 dark:border-slate-800 shadow-sm overflow-hidden">
                                    {/* Project Header */}
                                    <div 
                                        className="flex items-center justify-between p-3 cursor-pointer hover:bg-slate-50 dark:hover:bg-surface-800/50 transition-colors"
                                        onClick={() => toggleProjectExpand(project.id)}
                                    >
                                        <div className="flex items-center gap-3">
                                            <ChevronRight size={18} className={`text-slate-400 transition-transform ${expandedProjects[project.id] ? 'rotate-90' : ''}`} />
                                            <FolderOpen size={20} className="text-sky-500" />
                                            <div>
                                                <h3 className="font-medium text-slate-800 dark:text-slate-200">{project.project_code}</h3>
                                                {project.description && <p className="text-xs text-slate-400">{project.description}</p>}
                                            </div>
                                        </div>
                                        <div className="flex items-center gap-2" onClick={e => e.stopPropagation()}>
                                            <button 
                                                onClick={() => handleEditProject(project)}
                                                className="p-1.5 text-slate-400 hover:text-primary-600 hover:bg-primary-50 dark:hover:bg-primary-900/20 rounded transition-colors"
                                                title="編輯專案"
                                            >
                                                <Pencil size={15} />
                                            </button>
                                            <button 
                                                onClick={() => handleDeleteProject(project)}
                                                className="p-1.5 text-slate-400 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 rounded transition-colors"
                                                title="刪除專案"
                                            >
                                                <Trash2 size={15} />
                                            </button>
                                        </div>
                                    </div>

                                    {/* BOM List (Children) */}
                                    {expandedProjects[project.id] && (
                                        <div className="border-t border-slate-100 dark:border-slate-800 bg-slate-50/50 dark:bg-surface-950/30 px-3 py-2">
                                            <div className="ml-9 space-y-1">
                                                {projectBoms[project.id]?.length > 0 ? (
                                                    projectBoms[project.id].map(bom => (
                                                        <div 
                                                            key={bom.id} 
                                                            className="flex items-center justify-between p-2 rounded-lg hover:bg-white dark:hover:bg-surface-800 border border-transparent hover:border-slate-200 dark:hover:border-slate-700 transition-all cursor-pointer group"
                                                            onClick={() => handleBomClick(bom)}
                                                        >
                                                            <div className="flex items-center gap-3">
                                                                <FileText size={16} className="text-emerald-500" />
                                                                <span className="text-sm font-medium text-slate-700 dark:text-slate-300">
                                                                    {bom.phase_name} {bom.version}
                                                                    {bom.suffix && <span className="ml-1 text-slate-500">-{bom.suffix}</span>}
                                                                </span>
                                                                <span className="text-xs px-1.5 py-0.5 bg-slate-100 dark:bg-surface-700 text-slate-500 rounded text-[10px]">
                                                                    {bom.mode || 'NPI'}
                                                                </span>
                                                                {bom.bom_date && <span className="text-xs text-slate-400">{bom.bom_date}</span>}
                                                            </div>
                                                            <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity" onClick={e => e.stopPropagation()}>
                                                                <button 
                                                                    onClick={() => handleEditBom(bom, project.project_code)}
                                                                    className="p-1 text-slate-400 hover:text-primary-600 rounded"
                                                                    title="編輯 BOM 屬性"
                                                                >
                                                                    <Pencil size={14} />
                                                                </button>
                                                                <button 
                                                                    onClick={() => handleDeleteBom(bom)}
                                                                    className="p-1 text-slate-400 hover:text-red-600 rounded"
                                                                    title="刪除 BOM"
                                                                >
                                                                    <Trash2 size={14} />
                                                                </button>
                                                            </div>
                                                        </div>
                                                    ))
                                                ) : (
                                                    <div className="text-xs text-slate-400 py-2 pl-2 italic">尚無 BOM 版本</div>
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

            {/* Dialogs */}
            <Dialog
                isOpen={isRenameOpen}
                onClose={() => setIsRenameOpen(false)}
                title="重新命名系列"
                className="max-w-sm"
            >
                <div className="space-y-4">
                    <input
                        type="text"
                        value={renameValue}
                        onChange={(e) => setRenameValue(e.target.value)}
                        placeholder="輸入新的系列名稱"
                        className="w-full px-3 py-2 text-sm border rounded-lg dark:bg-surface-900 dark:border-slate-600"
                    />
                    {renameError && <p className="text-xs text-red-500">{renameError}</p>}
                    <div className="flex justify-end gap-2">
                        <button onClick={() => setIsRenameOpen(false)} className="px-4 py-2 text-sm hover:bg-slate-100 dark:hover:bg-surface-700 rounded-lg">取消</button>
                        <button onClick={handleRenameSubmit} className="px-4 py-2 text-sm bg-primary-600 text-white rounded-lg">確定</button>
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
                onConfirm={handleConfirmDelete}
                title={deleteConfirm.title}
                message={deleteConfirm.message}
                confirmText="刪除"
                danger
            />
        </div>
    )
}

export default Dashboard
