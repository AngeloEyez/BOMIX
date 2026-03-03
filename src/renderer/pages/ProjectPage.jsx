import { useEffect, useState } from 'react'
import { Plus, Pencil, Trash2, FolderOpen, Search, X } from 'lucide-react'
import useSeriesStore from '../stores/useSeriesStore'
import useProjectStore from '../stores/useProjectStore'
import Dialog from '../components/dialogs/Dialog'
import ConfirmDialog from '../components/dialogs/ConfirmDialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

// ========================================
// 專案管理頁面
// 顯示專案列表，支援 CRUD 操作
// ========================================

/**
 * 專案管理頁面元件。
 *
 * 提供完整的專案 CRUD 功能：新增、編輯、刪除，以及專案卡片視圖。
 *
 * @returns {JSX.Element} 專案管理頁面
 */
function ProjectPage() {
    const { isOpen } = useSeriesStore()
    const {
        projects, isLoading, error,
        loadProjects, createProject, updateProject, deleteProject, clearError,
    } = useProjectStore()

    // 新增/編輯對話框狀態
    const [isDialogOpen, setIsDialogOpen] = useState(false)
    const [editingProject, setEditingProject] = useState(null)
    const [formCode, setFormCode] = useState('')
    const [formDesc, setFormDesc] = useState('')
    const [formError, setFormError] = useState('')

    // 刪除確認對話框
    const [deleteTarget, setDeleteTarget] = useState(null)

    // 搜尋
    const [searchQuery, setSearchQuery] = useState('')

    // 開啟系列後自動載入
    useEffect(() => {
        if (isOpen) loadProjects()
    }, [isOpen, loadProjects])

    // ========================================
    // 未開啟系列 — 提示畫面
    // ========================================
    if (!isOpen) {
        return (
            <div className="flex flex-col items-center justify-center h-full gap-4 text-muted-foreground animate-fade-in">
                <FolderOpen size={40} className="text-border" />
                <h2 className="text-lg font-semibold">尚未開啟系列</h2>
                <p className="text-sm">請先從首頁建立或開啟系列資料庫，再管理專案。</p>
            </div>
        )
    }

    // ========================================
    // 對話框操作
    // ========================================

    /**
     * 開啟新增對話框
     */
    const handleOpenCreate = () => {
        setEditingProject(null)
        setFormCode('')
        setFormDesc('')
        setFormError('')
        setIsDialogOpen(true)
    }

    /**
     * 開啟編輯對話框。
     * @param {Object} project - 要編輯的專案
     */
    const handleOpenEdit = (project) => {
        setEditingProject(project)
        setFormCode(project.project_code)
        setFormDesc(project.description || '')
        setFormError('')
        setIsDialogOpen(true)
    }

    /**
     * 處理表單提交（新增或編輯）
     */
    const handleSubmit = async () => {
        if (!editingProject && !formCode.trim()) {
            setFormError('請輸入專案代碼')
            return
        }

        let result
        if (editingProject) {
            result = await updateProject(editingProject.id, formDesc)
        } else {
            result = await createProject(formCode.trim(), formDesc.trim())
        }

        if (result.success) {
            setIsDialogOpen(false)
        } else {
            setFormError(result.error || '操作失敗')
        }
    }

    /**
     * 處理刪除專案
     */
    const handleDelete = async () => {
        if (deleteTarget) {
            await deleteProject(deleteTarget.id)
            setDeleteTarget(null)
        }
    }

    // 篩選專案
    const filteredProjects = projects.filter(p => {
        const q = searchQuery.toLowerCase()
        return p.project_code.toLowerCase().includes(q)
            || (p.description || '').toLowerCase().includes(q)
    })

    // ========================================
    // 已開啟系列 — 專案列表
    // ========================================
    return (
        <div className="h-full overflow-auto p-4 scroll-smooth">
            <div className="max-w-3xl mx-auto space-y-4 animate-fade-in">
                {/* 頁面標頭 */}
                <div className="flex items-center justify-between">
                    <div>
                        <h2 className="text-base font-bold text-foreground">專案管理</h2>
                        <p className="text-xs text-muted-foreground mt-0.5">共 {projects.length} 個專案</p>
                    </div>
                    <Button size="sm" onClick={handleOpenCreate} className="gap-1.5">
                        <Plus size={14} />
                        新增專案
                    </Button>
                </div>

                {/* 搜尋列 */}
                {projects.length > 0 && (
                    <div className="relative">
                        <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
                        <Input
                            placeholder="搜尋專案代碼或描述..."
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                            className="h-8 text-xs pl-8 pr-8"
                        />
                        {searchQuery && (
                            <button
                                onClick={() => setSearchQuery('')}
                                className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                            >
                                <X size={13} />
                            </button>
                        )}
                    </div>
                )}

                {/* 錯誤提示 */}
                {error && (
                    <div className="flex items-center gap-2 px-3 py-2 bg-destructive/10 border border-destructive/30 rounded-lg text-xs text-destructive">
                        <span>{error}</span>
                        <button onClick={clearError} className="ml-auto hover:opacity-70"><X size={12} /></button>
                    </div>
                )}

                {/* 載入中 */}
                {isLoading && (
                    <div className="text-center py-10 text-muted-foreground text-sm animate-pulse">載入中...</div>
                )}

                {/* 空狀態 */}
                {!isLoading && filteredProjects.length === 0 && (
                    <div className="text-center py-12 text-muted-foreground">
                        {searchQuery ? (
                            <p className="text-sm">找不到符合「{searchQuery}」的專案</p>
                        ) : (
                            <div className="space-y-1">
                                <p className="text-base">尚無專案</p>
                                <p className="text-xs">點擊「新增專案」開始建立您的第一個專案。</p>
                            </div>
                        )}
                    </div>
                )}

                {/* 專案列表 */}
                {!isLoading && filteredProjects.length > 0 && (
                    <div className="grid gap-2">
                        {filteredProjects.map((project) => (
                            <div
                                key={project.id}
                                className="flex items-center justify-between px-4 py-3
                                    bg-background rounded-lg
                                    border border-border hover:border-primary/40
                                    shadow-sm hover:shadow-md
                                    transition-all duration-200 group"
                            >
                                {/* 專案資訊 */}
                                <div className="min-w-0 flex-1">
                                    <h3 className="text-sm font-bold text-foreground">{project.project_code}</h3>
                                    <p className="text-xs text-muted-foreground mt-0.5 truncate">
                                        {project.description || '（無描述）'}
                                    </p>
                                    <p className="text-[10px] text-muted-foreground/70 mt-1 font-mono">
                                        建立於 {project.created_at?.split('T')[0] || project.created_at?.split(' ')[0] || '—'}
                                    </p>
                                </div>

                                {/* 操作按鈕 */}
                                <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                                    <Button
                                        variant="ghost"
                                        size="icon"
                                        className="h-7 w-7"
                                        onClick={() => handleOpenEdit(project)}
                                        title="編輯專案"
                                    >
                                        <Pencil size={14} />
                                    </Button>
                                    <Button
                                        variant="ghost"
                                        size="icon"
                                        className="h-7 w-7 text-destructive/60 hover:text-destructive"
                                        onClick={() => setDeleteTarget(project)}
                                        title="刪除專案"
                                    >
                                        <Trash2 size={14} />
                                    </Button>
                                </div>
                            </div>
                        ))}
                    </div>
                )}

                {/* ========================================
                    新增/編輯專案對話框
                 ======================================== */}
                <Dialog
                    isOpen={isDialogOpen}
                    onClose={() => setIsDialogOpen(false)}
                    title={editingProject ? '編輯專案' : '新增專案'}
                    className="max-w-md"
                >
                    <div className="space-y-3">
                        {/* 專案代碼 */}
                        <div className="space-y-1">
                            <Label className="text-xs">
                                專案代碼 {!editingProject && <span className="text-destructive">*</span>}
                            </Label>
                            <Input
                                value={formCode}
                                onChange={(e) => setFormCode(e.target.value.toUpperCase())}
                                disabled={!!editingProject}
                                placeholder="例：TANGLED"
                                className="h-8 text-xs"
                                autoFocus={!editingProject}
                                onKeyDown={(e) => e.key === 'Enter' && handleSubmit()}
                            />
                            {editingProject && (
                                <p className="text-xs text-muted-foreground">專案代碼建立後不可修改</p>
                            )}
                        </div>

                        {/* 專案描述 */}
                        <div className="space-y-1">
                            <Label className="text-xs">描述</Label>
                            <textarea
                                value={formDesc}
                                onChange={(e) => setFormDesc(e.target.value)}
                                placeholder="專案描述（選填）"
                                rows={3}
                                className="w-full px-3 py-1.5 text-xs
                                    bg-background border border-input rounded-md
                                    text-foreground placeholder:text-muted-foreground
                                    focus:outline-none focus:ring-1 focus:ring-ring
                                    resize-none transition-colors"
                                autoFocus={!!editingProject}
                            />
                        </div>

                        {/* 表單錯誤 */}
                        {formError && <p className="text-xs text-destructive">{formError}</p>}

                        {/* 操作按鈕 */}
                        <div className="flex justify-end gap-2 pt-1 border-t border-border">
                            <Button variant="outline" size="sm" onClick={() => setIsDialogOpen(false)}>取消</Button>
                            <Button size="sm" onClick={handleSubmit}>
                                {editingProject ? '儲存' : '建立'}
                            </Button>
                        </div>
                    </div>
                </Dialog>

                {/* 刪除確認對話框 */}
                <ConfirmDialog
                    isOpen={!!deleteTarget}
                    onClose={() => setDeleteTarget(null)}
                    onConfirm={handleDelete}
                    title="刪除專案"
                    message={`確定要刪除專案「${deleteTarget?.project_code}」？此操作將一併刪除該專案下所有的 BOM 版本資料，且無法復原。`}
                    confirmText="刪除"
                    danger
                />
            </div>
        </div>
    )
}

export default ProjectPage
