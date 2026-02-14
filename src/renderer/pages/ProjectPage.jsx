import { useEffect, useState } from 'react'
import { Plus, Pencil, Trash2, FolderOpen, Search, X } from 'lucide-react'
import useSeriesStore from '../stores/useSeriesStore'
import useProjectStore from '../stores/useProjectStore'
import Dialog from '../components/dialogs/Dialog'
import ConfirmDialog from '../components/dialogs/ConfirmDialog'

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
    const [editingProject, setEditingProject] = useState(null) // null = 新增模式
    const [formCode, setFormCode] = useState('')
    const [formDesc, setFormDesc] = useState('')
    const [formError, setFormError] = useState('')

    // 刪除確認對話框
    const [deleteTarget, setDeleteTarget] = useState(null)

    // 搜尋
    const [searchQuery, setSearchQuery] = useState('')

    // 開啟系列後自動載入
    useEffect(() => {
        if (isOpen) {
            loadProjects()
        }
    }, [isOpen, loadProjects])

    // ========================================
    // 未開啟系列 — 提示畫面
    // ========================================
    if (!isOpen) {
        return (
            <div className="flex flex-col items-center justify-center h-full gap-4 text-slate-400 dark:text-slate-500 animate-fade-in">
                <FolderOpen size={48} className="text-slate-300 dark:text-slate-600" />
                <h2 className="text-xl font-semibold">尚未開啟系列</h2>
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
     * 開啟編輯對話框
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
        // 驗證
        if (!editingProject && !formCode.trim()) {
            setFormError('請輸入專案代碼')
            return
        }

        let result
        if (editingProject) {
            // 編輯模式
            result = await updateProject(editingProject.id, formDesc)
        } else {
            // 新增模式
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
        <div className="h-full overflow-auto p-6 scroll-smooth">
            <div className="max-w-4xl mx-auto space-y-6 animate-fade-in">
            {/* 頁面標頭 */}
            <div className="flex items-center justify-between">
                <div>
                    <h2 className="text-2xl font-bold text-slate-800 dark:text-white">專案管理</h2>
                    <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
                        共 {projects.length} 個專案
                    </p>
                </div>
                <button
                    onClick={handleOpenCreate}
                    className="flex items-center gap-2 px-4 py-2 text-sm font-medium
                        bg-primary-600 hover:bg-primary-700 text-white
                        rounded-lg shadow-sm transition-colors"
                >
                    <Plus size={16} />
                    新增專案
                </button>
            </div>

            {/* 搜尋列 */}
            {projects.length > 0 && (
                <div className="relative">
                    <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
                    <input
                        type="text"
                        placeholder="搜尋專案代碼或描述..."
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                        className="w-full pl-10 pr-4 py-2 text-sm
                            bg-white dark:bg-surface-800
                            border border-slate-200 dark:border-slate-700
                            rounded-lg text-slate-800 dark:text-slate-200
                            focus:outline-none focus:ring-2 focus:ring-primary-500
                            placeholder:text-slate-400"
                    />
                    {searchQuery && (
                        <button
                            onClick={() => setSearchQuery('')}
                            className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600"
                        >
                            <X size={14} />
                        </button>
                    )}
                </div>
            )}

            {/* 錯誤提示 */}
            {error && (
                <div className="flex items-center gap-2 px-4 py-2 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg text-sm text-red-600 dark:text-red-400">
                    <span>{error}</span>
                    <button onClick={clearError} className="ml-auto hover:text-red-800">
                        <X size={14} />
                    </button>
                </div>
            )}

            {/* 載入中 */}
            {isLoading && (
                <div className="text-center py-12 text-slate-400 animate-pulse">
                    載入中...
                </div>
            )}

            {/* 專案列表 */}
            {!isLoading && filteredProjects.length === 0 && (
                <div className="text-center py-16 text-slate-400 dark:text-slate-500">
                    {searchQuery ? (
                        <p>找不到符合「{searchQuery}」的專案</p>
                    ) : (
                        <div className="space-y-2">
                            <p className="text-lg">尚無專案</p>
                            <p className="text-sm">點擊「新增專案」開始建立您的第一個專案。</p>
                        </div>
                    )}
                </div>
            )}

            {!isLoading && filteredProjects.length > 0 && (
                <div className="grid gap-3">
                    {filteredProjects.map((project) => (
                        <div
                            key={project.id}
                            className="flex items-center justify-between p-4
                                bg-white dark:bg-surface-800 rounded-xl
                                border border-slate-200 dark:border-slate-700
                                hover:border-primary-300 dark:hover:border-primary-600
                                shadow-sm hover:shadow-md
                                transition-all duration-200 group"
                        >
                            {/* 專案資訊 */}
                            <div className="min-w-0 flex-1">
                                <h3 className="text-base font-bold text-slate-800 dark:text-white">
                                    {project.project_code}
                                </h3>
                                <p className="text-sm text-slate-500 dark:text-slate-400 mt-0.5 truncate">
                                    {project.description || '（無描述）'}
                                </p>
                                <p className="text-xs text-slate-400 mt-1 font-mono">
                                    建立於 {project.created_at?.split('T')[0] || project.created_at?.split(' ')[0] || '—'}
                                </p>
                            </div>

                            {/* 操作按鈕 */}
                            <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                                <button
                                    onClick={() => handleOpenEdit(project)}
                                    className="p-2 rounded-lg hover:bg-slate-100 dark:hover:bg-surface-700 text-slate-400 hover:text-primary-600 transition-colors"
                                    title="編輯專案"
                                >
                                    <Pencil size={16} />
                                </button>
                                <button
                                    onClick={() => setDeleteTarget(project)}
                                    className="p-2 rounded-lg hover:bg-red-50 dark:hover:bg-red-900/20 text-slate-400 hover:text-red-600 transition-colors"
                                    title="刪除專案"
                                >
                                    <Trash2 size={16} />
                                </button>
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
                <div className="space-y-4">
                    {/* 專案代碼 */}
                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                            專案代碼 {!editingProject && <span className="text-red-500">*</span>}
                        </label>
                        <input
                            type="text"
                            value={formCode}
                            onChange={(e) => setFormCode(e.target.value.toUpperCase())}
                            disabled={!!editingProject}
                            placeholder="例：TANGLED"
                            className="w-full px-3 py-2 text-sm
                                bg-white dark:bg-surface-900
                                border border-slate-300 dark:border-slate-600
                                rounded-lg text-slate-800 dark:text-slate-200
                                focus:outline-none focus:ring-2 focus:ring-primary-500
                                disabled:opacity-50 disabled:cursor-not-allowed
                                placeholder:text-slate-400"
                            autoFocus={!editingProject}
                        />
                        {editingProject && (
                            <p className="text-xs text-slate-400 mt-1">專案代碼建立後不可修改</p>
                        )}
                    </div>

                    {/* 專案描述 */}
                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                            描述
                        </label>
                        <textarea
                            value={formDesc}
                            onChange={(e) => setFormDesc(e.target.value)}
                            placeholder="專案描述（選填）"
                            rows={3}
                            className="w-full px-3 py-2 text-sm
                                bg-white dark:bg-surface-900
                                border border-slate-300 dark:border-slate-600
                                rounded-lg text-slate-800 dark:text-slate-200
                                focus:outline-none focus:ring-2 focus:ring-primary-500
                                resize-none placeholder:text-slate-400"
                            autoFocus={!!editingProject}
                        />
                    </div>

                    {/* 表單錯誤 */}
                    {formError && (
                        <p className="text-sm text-red-600 dark:text-red-400">{formError}</p>
                    )}

                    {/* 操作按鈕 */}
                    <div className="flex justify-end gap-2 pt-2">
                        <button
                            onClick={() => setIsDialogOpen(false)}
                            className="px-4 py-2 text-sm font-medium text-slate-600 dark:text-slate-300
                                bg-slate-100 dark:bg-surface-700 hover:bg-slate-200 dark:hover:bg-surface-600
                                rounded-lg transition-colors"
                        >
                            取消
                        </button>
                        <button
                            onClick={handleSubmit}
                            className="px-4 py-2 text-sm font-medium text-white
                                bg-primary-600 hover:bg-primary-700
                                rounded-lg transition-colors"
                        >
                            {editingProject ? '儲存' : '建立'}
                        </button>
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
