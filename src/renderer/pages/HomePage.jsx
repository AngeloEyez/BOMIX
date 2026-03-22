// ========================================
// 首頁（歡迎頁面）
// 顯示應用程式名稱、快速操作、系列資訊
// 統一使用 shadcn Button/Input 與語義化顏色變數
// ========================================

import { useEffect, useState } from 'react'
import { FolderPlus, FolderOpen, Clock, X, Database, ChevronRight, Pencil, Check, FileText, AlertCircle } from 'lucide-react'
import useSeriesStore from '../stores/useSeriesStore'
import useProjectStore from '../stores/useProjectStore'
import Dialog from '../components/dialogs/Dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader } from '@/components/ui/card'

/**
 * 首頁元件。
 *
 * 狀態一：已開啟系列 → 顯示系列資訊卡片與統計。
 * 狀態二：未開啟系列 → 顯示歡迎畫面與最近開啟清單。
 *
 * @returns {JSX.Element}
 */
function HomePage() {
    const {
        isOpen, currentSeries, currentPath, recentFiles, isLoading, error,
        initRecentFiles, createSeries, openSeries, closeSeries,
        updateDescription, renameSeries, removeFromRecentFiles, clearError,
    } = useSeriesStore()

    const { loadProjects, projects, reset: resetProjects } = useProjectStore()

    const [isEditingDesc, setIsEditingDesc] = useState(false)
    const [editDesc, setEditDesc] = useState('')
    const [isRenameOpen, setIsRenameOpen] = useState(false)
    const [renameValue, setRenameValue] = useState('')
    const [renameError, setRenameError] = useState('')

    // 初始化最近開啟記錄
    useEffect(() => { initRecentFiles() }, [initRecentFiles])

    // 開啟系列時載入專案資料，關閉時重置
    useEffect(() => {
        if (isOpen) loadProjects()
        else resetProjects()
    }, [isOpen, loadProjects, resetProjects])

    /** 建立新系列 */
    const handleCreate = () => createSeries()
    /** 開啟指定路徑系列，不傳路徑則彈出系統對話框 */
    const handleOpen = (filePath) => openSeries(filePath)
    /** 關閉目前系列 */
    const handleClose = () => closeSeries()

    /** 進入描述編輯模式 */
    const handleStartEditDesc = () => {
        setEditDesc(currentSeries?.description || '')
        setIsEditingDesc(true)
    }

    /** 儲存描述變更 */
    const handleSaveDesc = async () => {
        await updateDescription(editDesc)
        setIsEditingDesc(false)
    }

    /** 開啟重新命名對話框，預填目前档名 */
    const handleOpenRename = () => {
        const name = currentPath?.split(/[/\\]/).pop().replace('.bomix', '') || ''
        setRenameValue(name)
        setRenameError('')
        setIsRenameOpen(true)
    }

    /**
     * 驗證並送出重新命名請求。
     */
    const handleRenameSubmit = async () => {
        if (!renameValue.trim()) { setRenameError('請輸入系列名稱'); return }
        if (/[\\/:*?"<>|]/.test(renameValue)) { setRenameError('名稱包含無效字元'); return }
        const result = await renameSeries(renameValue.trim())
        if (result.success) setIsRenameOpen(false)
        else setRenameError(result.error)
    }

    // ========================================
    // 狀態一：已開啟系列
    // ========================================
    if (isOpen && currentSeries) {
        const displayName = currentPath?.split(/[/\\]/).pop().replace('.bomix', '') || '未命名系列'

        return (
            <div className="max-w-xl mx-auto py-6 px-4 space-y-4 animate-fade-in">
                {/* 系列資訊卡片 */}
                <Card>
                    {/* 卡片標頭 */}
                    <CardHeader className="px-4 py-3 border-b border-border flex-row items-center justify-between space-y-0">
                        <div className="flex items-center gap-3 min-w-0">
                            <div className="p-1.5 bg-primary/10 text-primary rounded-md shrink-0">
                                <Database size={16} />
                            </div>
                            <div className="min-w-0">
                                <div className="flex items-center gap-1.5">
                                    <h2 className="text-sm font-bold text-foreground truncate">{displayName}</h2>
                                    <Button variant="ghost" size="icon" className="h-6 w-6" onClick={handleOpenRename} title="重新命名">
                                        <Pencil size={12} />
                                    </Button>
                                </div>
                                <p className="text-xs text-muted-foreground font-mono truncate max-w-xs">{currentPath}</p>
                            </div>
                        </div>
                        <Button variant="ghost" size="icon" className="h-7 w-7 shrink-0" onClick={handleClose} title="關閉系列">
                            <X size={15} />
                        </Button>
                    </CardHeader>

                    <CardContent className="p-4 space-y-4">
                        {/* 描述欄位 */}
                        <div>
                            <div className="flex items-center gap-1.5 mb-1">
                                <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">系列描述</span>
                                {!isEditingDesc && (
                                    <Button variant="ghost" size="icon" className="h-5 w-5" onClick={handleStartEditDesc} title="編輯描述">
                                        <Pencil size={11} />
                                    </Button>
                                )}
                            </div>
                            {isEditingDesc ? (
                                <div className="flex gap-2">
                                    <Input
                                        value={editDesc}
                                        onChange={(e) => setEditDesc(e.target.value)}
                                        className="h-8 text-xs flex-1"
                                        autoFocus
                                        onKeyDown={(e) => e.key === 'Enter' && handleSaveDesc()}
                                    />
                                    <Button size="icon" className="h-8 w-8" onClick={handleSaveDesc}>
                                        <Check size={14} />
                                    </Button>
                                </div>
                            ) : (
                                <p className="text-sm text-foreground/80 leading-relaxed">
                                    {currentSeries.description || '（無描述）'}
                                </p>
                            )}
                        </div>

                        {/* 統計數量 */}
                        <div className="grid grid-cols-2 gap-3">
                            <div className="bg-muted/50 rounded-lg p-3 flex items-center justify-between">
                                <div>
                                    <p className="text-xl font-bold text-primary">{projects.length}</p>
                                    <p className="text-xs text-muted-foreground">專案總數</p>
                                </div>
                                <FolderOpen className="text-border" size={28} />
                            </div>
                            <div className="bg-muted/50 rounded-lg p-3 flex items-center justify-between">
                                <div>
                                    <p className="text-xl font-bold text-sky-500">{currentSeries.bomCount || 0}</p>
                                    <p className="text-xs text-muted-foreground">BOM 版本數</p>
                                </div>
                                <FileText className="text-border" size={28} />
                            </div>
                        </div>

                        {/* 日期資訊 */}
                        <div className="pt-2 border-t border-border flex justify-between text-xs text-muted-foreground font-mono">
                            <span>建立於: {currentSeries.created_at?.split('T')[0] || '-'}</span>
                            <span>最後修改: {currentSeries.updated_at?.split('T')[0] || '-'}</span>
                        </div>
                    </CardContent>
                </Card>

                {/* 快速操作 */}
                <div className="flex gap-2">
                    <Button variant="outline" size="sm" onClick={handleCreate} className="flex items-center gap-1.5">
                        <FolderPlus size={14} /> 建立新系列
                    </Button>
                    <Button variant="outline" size="sm" onClick={() => handleOpen()} className="flex items-center gap-1.5">
                        <FolderOpen size={14} /> 開啟其他系列
                    </Button>
                </div>

                {/* 重新命名對話框 */}
                <Dialog isOpen={isRenameOpen} onClose={() => setIsRenameOpen(false)} title="重新命名系列" className="max-w-sm">
                    <div className="space-y-3">
                        <p className="text-xs text-muted-foreground">這將會修改資料庫檔案名稱，請確認後操作。</p>
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
            </div>
        )
    }

    // ========================================
    // 狀態二：未開啟系列（歡迎畫面）
    // ========================================
    return (
        <div className="flex flex-col items-center justify-center h-full gap-6 p-6 overflow-auto animate-fade-in">
            {/* 標題 */}
            <div className="text-center">
                <h1 className="text-3xl font-bold text-primary mb-1">BOMIX</h1>
                <p className="text-sm text-muted-foreground">BOM 變化管理與追蹤工具</p>
            </div>

            {/* 錯誤提示 */}
            {error && (
                <div className="flex items-center gap-2 px-3 py-2 bg-destructive/10 border border-destructive/30 rounded-lg text-xs text-destructive">
                    <AlertCircle size={13} />
                    <span>{error}</span>
                    <Button variant="ghost" size="icon" className="h-5 w-5 ml-auto" onClick={clearError}>
                        <X size={12} />
                    </Button>
                </div>
            )}

            {/* 主要操作按鈕 */}
            <div className="flex gap-3">
                <button
                    onClick={handleCreate}
                    disabled={isLoading}
                    className="flex flex-col items-center gap-2 px-8 py-5
                        bg-card rounded-xl shadow-sm hover:shadow-md
                        border border-border hover:border-primary/50
                        transition-all duration-200 cursor-pointer
                        group disabled:opacity-50 disabled:cursor-not-allowed"
                >
                    <span className="text-primary group-hover:scale-110 transition-transform">
                        <FolderPlus size={28} />
                    </span>
                    <span className="text-xs font-medium text-foreground">建立新系列</span>
                </button>

                <button
                    onClick={() => handleOpen()}
                    disabled={isLoading}
                    className="flex flex-col items-center gap-2 px-8 py-5
                        bg-card rounded-xl shadow-sm hover:shadow-md
                        border border-border hover:border-primary/50
                        transition-all duration-200 cursor-pointer
                        group disabled:opacity-50 disabled:cursor-not-allowed"
                >
                    <span className="text-primary group-hover:scale-110 transition-transform">
                        <FolderOpen size={28} />
                    </span>
                    <span className="text-xs font-medium text-foreground">開啟系列</span>
                </button>
            </div>

            {isLoading && <p className="text-xs text-muted-foreground animate-pulse">處理中...</p>}

            {/* 最近開啟的系列 */}
            <div className="w-full max-w-md">
                <h2 className="text-xs font-semibold text-muted-foreground mb-2 flex items-center gap-1.5">
                    <Clock size={12} /> 最近開啟
                </h2>
                {recentFiles.length > 0 ? (
                    <Card>
                        <CardContent className="p-0 divide-y divide-border">
                            {recentFiles.map((filePath) => {
                                const name = filePath.split(/[/\\]/).pop()?.replace('.bomix', '') || filePath
                                return (
                                    <div
                                        key={filePath}
                                        className="flex items-center justify-between px-3 py-2 hover:bg-muted/50 transition-colors group"
                                    >
                                        <button
                                            onClick={() => handleOpen(filePath)}
                                            className="flex-1 flex items-center gap-2 text-left min-w-0"
                                        >
                                            <Database size={14} className="shrink-0 text-primary" />
                                            <div className="min-w-0">
                                                <p className="text-xs font-medium text-foreground truncate">{name}</p>
                                                <p className="text-xs text-muted-foreground truncate font-mono">{filePath}</p>
                                            </div>
                                            <ChevronRight size={12} className="shrink-0 text-muted-foreground/50 group-hover:text-primary transition-colors" />
                                        </button>
                                        <Button
                                            variant="ghost"
                                            size="icon"
                                            className="h-6 w-6 shrink-0 opacity-0 group-hover:opacity-100 transition-opacity"
                                            onClick={() => removeFromRecentFiles(filePath)}
                                            title="從列表中移除"
                                        >
                                            <X size={12} />
                                        </Button>
                                    </div>
                                )
                            })}
                        </CardContent>
                    </Card>
                ) : (
                    <div className="bg-card border border-border rounded-xl p-4 text-center text-xs text-muted-foreground">
                        尚無開啟記錄
                    </div>
                )}
            </div>
        </div>
    )
}

export default HomePage
