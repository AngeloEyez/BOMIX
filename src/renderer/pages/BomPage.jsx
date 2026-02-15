import { useEffect, useState, useCallback } from 'react'
import {
    FileSpreadsheet, Upload, Download, Trash2,
    FolderOpen, ChevronDown, Info, X,
} from 'lucide-react'
import useSeriesStore from '../stores/useSeriesStore'
import useProjectStore from '../stores/useProjectStore'
import useBomStore from '../stores/useBomStore'
import useProgressStore from '../stores/useProgressStore'
import BomTable from '../components/tables/BomTable'
import ImportDialog from '../components/dialogs/ImportDialog'
import ConfirmDialog from '../components/dialogs/ConfirmDialog'

// ========================================
// BOM 檢視頁面
// 提供專案/版本選取、BOM 表格檢視、Excel 匯入匯出功能
// ========================================

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
    const { projects, loadProjects } = useProjectStore()
    const {
        selectedProjectId, revisions, selectedRevisionId, selectedRevision,
        bomView, isLoading, error,
        selectProject, selectRevision, reloadBomView,
        deleteBom, importExcel, exportExcel,
        clearError, reset,
    } = useBomStore()

    // Check if any export task is running
    const isExporting = useProgressStore(state => 
        Array.from(state.sessions.values()).some(s => s.type === 'EXPORT_BOM' && s.status === 'RUNNING')
    )

    // 匯入對話框
    const [isImportOpen, setIsImportOpen] = useState(false)
    const [importFile, setImportFile] = useState(null)
    // 刪除確認對話框
    const [deleteTarget, setDeleteTarget] = useState(null)
    // 頁面層級拖曳狀態
    const [isDragOver, setIsDragOver] = useState(false)

    // 開啟系列後載入專案列表
    useEffect(() => {
        if (isOpen) {
            loadProjects()
        } else {
            reset()
        }
    }, [isOpen, loadProjects, reset])

    // ========================================
    // 事件處理
    // ========================================

    /**
     * 專案選擇變更
     * @param {Event} e - 下拉選單事件
     */
    const handleProjectChange = (e) => {
        const projectId = Number(e.target.value)
        if (projectId) {
            selectProject(projectId)
        }
    }

    /**
     * BOM 版本選擇變更
     * @param {Event} e - 下拉選單事件
     */
    const handleRevisionChange = (e) => {
        const revisionId = Number(e.target.value)
        if (revisionId) {
            selectRevision(revisionId)
        }
    }

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
        // 僅在已選擇專案時允許拖曳
        if (selectedProjectId) {
            setIsDragOver(true)
        }
    }, [selectedProjectId])

    const handlePageDragLeave = useCallback((e) => {
        e.preventDefault()
        e.stopPropagation()
        setIsDragOver(false)
    }, [])

    const handlePageDrop = useCallback((e) => {
        e.preventDefault()
        e.stopPropagation()
        setIsDragOver(false)

        if (!selectedProjectId) return

        const files = e.dataTransfer?.files
        if (files && files.length > 0) {
            const file = files[0]
            const ext = file.name.split('.').pop()?.toLowerCase()
            if (ext === 'xls' || ext === 'xlsx') {
                // 使用 webUtils 取得檔案真實路徑 (解決 Context Isolation 問題)
                const path = window.api.utils.getPathForFile(file)

                // 開啟匯入對話框，並預填檔案
                setImportFile({ name: file.name, path })
                setIsImportOpen(true)
            }
        }
    }, [selectedProjectId])

    // ========================================
    // 未開啟系列 — 提示畫面
    // ========================================
    if (!isOpen) {
        return (
            <div className="flex flex-col items-center justify-center h-full gap-4 text-slate-400 dark:text-slate-500 animate-fade-in">
                <FolderOpen size={48} className="text-slate-300 dark:text-slate-600" />
                <h2 className="text-xl font-semibold">尚未開啟系列</h2>
                <p className="text-sm">請先從首頁建立或開啟系列資料庫，再檢視 BOM。</p>
            </div>
        )
    }

    // 計算目前選取的版本顯示名稱
    const selectedRevisionLabel = selectedRevision
        ? `${selectedRevision.phase_name} ${selectedRevision.version}`
        : null

    // ========================================
    // 頁面主體
    // ========================================
    return (
        <div
            className="flex flex-col h-full p-3 gap-2 animate-fade-in"
            onDragOver={handlePageDragOver}
            onDragLeave={handlePageDragLeave}
            onDrop={handlePageDrop}
        >
            {/* ========================================
                工具列
             ======================================== */}
            <div className="flex items-center gap-2 flex-wrap">
                {/* 專案選擇器 */}
                <div className="relative">
                    <select
                        value={selectedProjectId || ''}
                        onChange={handleProjectChange}
                        className="appearance-none pl-3 pr-8 py-1.5 text-sm
                            bg-white dark:bg-surface-800
                            border border-slate-200 dark:border-slate-700
                            rounded-lg text-slate-800 dark:text-slate-200
                            focus:outline-none focus:ring-2 focus:ring-primary-500
                            min-w-[140px] cursor-pointer"
                    >
                        <option value="">選擇專案...</option>
                        {projects.map((p) => (
                            <option key={p.id} value={p.id}>{p.project_code}</option>
                        ))}
                    </select>
                    <ChevronDown size={14} className="absolute right-2.5 top-1/2 -translate-y-1/2 text-slate-400 pointer-events-none" />
                </div>

                {/* BOM 版本選擇器 */}
                <div className="relative">
                    <select
                        value={selectedRevisionId || ''}
                        onChange={handleRevisionChange}
                        disabled={!selectedProjectId || revisions.length === 0}
                        className="appearance-none pl-3 pr-8 py-1.5 text-sm
                            bg-white dark:bg-surface-800
                            border border-slate-200 dark:border-slate-700
                            rounded-lg text-slate-800 dark:text-slate-200
                            focus:outline-none focus:ring-2 focus:ring-primary-500
                            min-w-[150px] cursor-pointer
                            disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                        <option value="">
                            {!selectedProjectId ? '先選擇專案' : revisions.length === 0 ? '無 BOM 版本' : '選擇版本...'}
                        </option>
                        {revisions.map((rev) => (
                            <option key={rev.id} value={rev.id}>
                                {rev.phase_name} {rev.version}{rev.suffix ? `-${rev.suffix}` : ''} {rev.mode ? `(${rev.mode})` : ''}
                            </option>
                        ))}
                    </select>
                    <ChevronDown size={14} className="absolute right-2.5 top-1/2 -translate-y-1/2 text-slate-400 pointer-events-none" />
                </div>

                {/* 分隔線 */}
                <div className="h-6 w-px bg-slate-200 dark:bg-slate-700 mx-1" />

                {/* 匯入按鈕 */}
                <button
                    onClick={() => setIsImportOpen(true)}
                    disabled={!selectedProjectId}
                    className="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium
                        bg-primary-600 hover:bg-primary-700 text-white
                        rounded-lg shadow-sm transition-colors
                        disabled:opacity-50 disabled:cursor-not-allowed"
                    title="匯入 Excel BOM"
                >
                    <Upload size={14} />
                    匯入
                </button>

                {/* 匯出按鈕 */}
                <button
                    onClick={handleExport}
                    disabled={!selectedRevisionId || isExporting}
                    className="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium
                        text-slate-600 dark:text-slate-300
                        bg-slate-100 dark:bg-surface-700 hover:bg-slate-200 dark:hover:bg-surface-600
                        rounded-lg transition-colors
                        disabled:opacity-50 disabled:cursor-not-allowed"
                    title={isExporting ? "匯出中..." : "匯出 Excel BOM"}
                >
                    <Download size={14} className={isExporting ? "animate-bounce" : ""} />
                    {isExporting ? "匯出中..." : "匯出"}
                </button>

                {/* 刪除版本按鈕 */}
                <button
                    onClick={() => setDeleteTarget(selectedRevisionId)}
                    disabled={!selectedRevisionId}
                    className="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium
                        text-red-600 dark:text-red-400
                        bg-red-50 dark:bg-red-900/20 hover:bg-red-100 dark:hover:bg-red-900/30
                        rounded-lg transition-colors
                        disabled:opacity-50 disabled:cursor-not-allowed"
                    title="刪除此 BOM 版本"
                >
                    <Trash2 size={14} />
                </button>

                {/* 版本資訊 */}
                {selectedRevision && (
                    <div className="ml-auto flex items-center gap-2 text-xs text-slate-400 dark:text-slate-500">
                        <Info size={13} />
                        <span>Mode: <strong className="text-slate-600 dark:text-slate-300">{selectedRevision.mode || 'NPI'}</strong></span>
                        {selectedRevision.suffix && (
                            <span>- {selectedRevision.suffix}</span>
                        )}
                        {selectedRevision.schematic_version && (
                            <span>| Sch: {selectedRevision.schematic_version}</span>
                        )}
                        {selectedRevision.pcb_version && (
                            <span>| PCB: {selectedRevision.pcb_version}</span>
                        )}
                        {selectedRevision.bom_date && (
                            <span>| {selectedRevision.bom_date}</span>
                        )}
                    </div>
                )}
            </div>

            {/* 錯誤提示 */}
            {error && (
                <div className="flex items-center gap-2 px-3 py-1.5 bg-red-50 dark:bg-red-900/20
                    border border-red-200 dark:border-red-800 rounded-lg text-xs text-red-600 dark:text-red-400">
                    <span>{error}</span>
                    <button onClick={clearError} className="ml-auto hover:text-red-800">
                        <X size={12} />
                    </button>
                </div>
            )}

            {/* ========================================
                BOM 表格 — 主要內容區
             ======================================== */}
            <div className="flex-1 min-h-0 overflow-hidden">
                {selectedRevisionId ? (
                    <BomTable data={bomView} isLoading={isLoading} />
                ) : (
                    <div className="flex flex-col items-center justify-center h-full gap-3 text-slate-400 dark:text-slate-500">
                        <FileSpreadsheet size={40} className="text-slate-300 dark:text-slate-600" />
                        <p className="text-sm">
                            {selectedProjectId
                                ? '請選擇 BOM 版本，或匯入 Excel 檔案。'
                                : '請先選擇專案。'}
                        </p>
                    </div>
                )}
            </div>

            {/* 拖曳覆蓋層 */}
            {isDragOver && selectedProjectId && (
                <div className="fixed inset-0 z-50 flex items-center justify-center
                    bg-primary-500/10 dark:bg-primary-400/10
                    border-4 border-dashed border-primary-400 dark:border-primary-500
                    pointer-events-none">
                    <div className="bg-white dark:bg-surface-800 rounded-xl shadow-lg px-8 py-6 text-center">
                        <Upload size={32} className="mx-auto mb-2 text-primary-500" />
                        <p className="text-lg font-semibold text-slate-700 dark:text-white">放開以匯入 Excel</p>
                        <p className="text-sm text-slate-400 mt-1">支援 .xls / .xlsx 格式</p>
                    </div>
                </div>
            )}

            {/* ========================================
                匯入對話框
             ======================================== */}
            <ImportDialog
                isOpen={isImportOpen}
                onClose={() => {
                    setIsImportOpen(false)
                    setImportFile(null)
                }}
                projectId={selectedProjectId}
                onImport={importExcel}
                initialFile={importFile}
            />

            {/* 刪除確認對話框 */}
            <ConfirmDialog
                isOpen={!!deleteTarget}
                onClose={() => setDeleteTarget(null)}
                onConfirm={handleDeleteConfirm}
                title="刪除 BOM 版本"
                message={`確定要刪除 BOM 版本「${selectedRevisionLabel || ''}」？此操作將一併刪除所有零件資料，且無法復原。`}
                confirmText="刪除"
                danger
            />
        </div>
    )
}

export default BomPage
