// ========================================
// Excel 匯入對話框 (ImportDialog)
// 支援點擊選擇檔案或拖曳 .xls/.xlsx 檔案
// 使用 shadcn Button 統一按鈕風格
// ========================================

import { useState, useCallback, useEffect } from 'react'
import { Upload, FileSpreadsheet, AlertCircle } from 'lucide-react'
import Dialog from './Dialog'
import { Button } from '@/components/ui/button'

/**
 * Excel 匯入對話框元件。
 *
 * 支援透過按鈕選擇或拖曳 .xls/.xlsx 檔案，送出匯入任務。
 *
 * @param {Object} props
 * @param {boolean} props.isOpen - 是否開啟
 * @param {Function} props.onClose - 關閉回呼
 * @param {Function} props.onImport - 匯入回呼 (filePaths) => Promise<{success, error}>
 * @param {Object} [props.initialFile] - 拖曳帶入的初始檔案物件
 * @returns {JSX.Element}
 */
function ImportDialog({ isOpen, onClose, onImport, initialFile }) {
    const [filePaths, setFilePaths] = useState([])
    const [error, setError] = useState('')
    const [isImporting, setIsImporting] = useState(false)
    const [isDragOver, setIsDragOver] = useState(false)

    // 處理初始檔案（來自頁面拖曳），在對話框開啟時同步帶入預設路徑
    useEffect(() => {
        if (isOpen && initialFile?.path) {
            // eslint-disable-next-line react-hooks/set-state-in-effect
            setFilePaths([initialFile.path])
        }
    }, [isOpen, initialFile])

    /** 重置表單至初始狀態 */
    const resetForm = useCallback(() => {
        setFilePaths([])
        setError('')
        setIsImporting(false)
        setIsDragOver(false)
    }, [])

    /** 開啟系統檔案選擇對話框 */
    const handleSelectFile = async () => {
        try {
            const result = await window.api.dialog.showOpen({
                title: '選擇 BOM Excel 檔案 (可多選)',
                properties: ['openFile', 'multiSelections'],
                filters: [{ name: 'Excel Files', extensions: ['xls', 'xlsx'] }],
            })
            if (!result.canceled && result.data?.length > 0) {
                setFilePaths(result.data)
                setError('')
            }
        } catch (_e) {
            setError('開啟檔案失敗')
        }
    }

    /** 拖曳相關事件處理 */
    const handleDragOver = (e) => { e.preventDefault(); e.stopPropagation(); setIsDragOver(true) }
    const handleDragLeave = (e) => { e.preventDefault(); e.stopPropagation(); setIsDragOver(false) }

    /**
     * 處理拖放檔案，過濾非 Excel 格式。
     *
     * @param {DragEvent} e
     */
    const handleDrop = (e) => {
        e.preventDefault()
        e.stopPropagation()
        setIsDragOver(false)

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

        if (validPaths.length > 0) {
            setFilePaths(validPaths)
            setError(hasInvalid ? '已過濾不支援的格式，僅接受 .xls/.xlsx' : '')
        } else {
            setError('僅支援 .xls 或 .xlsx 格式')
        }
    }

    /** 送出匯入任務 */
    const handleSubmit = async () => {
        if (!filePaths.length) {
            setError('請選擇 Excel 檔案')
            return
        }
        setIsImporting(true)
        setError('')
        try {
            const result = await onImport(filePaths)
            if (result.success) {
                resetForm()
                onClose()
            } else {
                setError(result.error || '匯入失敗')
                setIsImporting(false)
            }
        } catch (e) {
            setError(e.message || '匯入失敗')
            setIsImporting(false)
        }
    }

    /** 關閉並重置表單 */
    const handleClose = () => {
        resetForm()
        onClose()
    }

    return (
        <Dialog
            isOpen={isOpen}
            onClose={handleClose}
            title="匯入 Excel BOM"
            className="max-w-md"
        >
            <div className="space-y-3">
                {/* 拖曳 / 點擊選擇區域 */}
                <div
                    onDragOver={handleDragOver}
                    onDragLeave={handleDragLeave}
                    onDrop={handleDrop}
                    onClick={filePaths.length === 0 ? handleSelectFile : undefined}
                    className={`border-2 border-dashed rounded-lg p-5 text-center transition-colors
                        ${isDragOver
                            ? 'border-primary bg-primary/5'
                            : filePaths.length > 0
                                ? 'border-emerald-400 bg-emerald-50/50 dark:bg-emerald-900/10'
                                : 'border-border hover:border-primary/50 cursor-pointer'
                        }`}
                >
                    {filePaths.length > 0 ? (
                        <div className="flex flex-col items-center">
                            <div className="flex items-center gap-2 mb-2">
                                <FileSpreadsheet size={18} className="text-emerald-500" />
                                <span className="text-sm font-medium text-foreground">
                                    已選取 {filePaths.length} 個檔案
                                </span>
                            </div>

                            {/* 已選取的檔案列表 */}
                            <div className="w-full max-h-36 overflow-y-auto bg-background/60 rounded border border-border p-2 space-y-1">
                                {filePaths.map((path, index) => (
                                    <div key={index} className="flex items-center gap-2 text-xs text-muted-foreground">
                                        <div className="w-1 h-1 rounded-full bg-emerald-500 shrink-0" />
                                        <span className="truncate flex-1 text-left" title={path}>
                                            {path.split(/[/\\]/).pop()}
                                        </span>
                                    </div>
                                ))}
                            </div>

                            {/* 清除重選按鈕 */}
                            <Button
                                variant="ghost"
                                size="sm"
                                className="mt-2 text-xs text-destructive hover:text-destructive"
                                onClick={(e) => { e.stopPropagation(); setFilePaths([]) }}
                            >
                                清除重新選擇
                            </Button>
                        </div>
                    ) : (
                        <div className="space-y-1 text-muted-foreground">
                            <Upload size={22} className="mx-auto" />
                            <p className="text-xs">點擊選擇或拖曳 .xls / .xlsx 至此</p>
                        </div>
                    )}
                </div>

                {/* 錯誤訊息 */}
                {error && (
                    <div className="flex items-center gap-1.5 text-xs text-destructive">
                        <AlertCircle size={13} />
                        {error}
                    </div>
                )}

                {/* 操作按鈕 */}
                <div className="flex justify-end gap-2">
                    <Button variant="outline" size="sm" onClick={handleClose} disabled={isImporting}>
                        取消
                    </Button>
                    <Button size="sm" onClick={handleSubmit} disabled={isImporting}>
                        {isImporting ? '匯入中...' : '匯入'}
                    </Button>
                </div>
            </div>
        </Dialog>
    )
}

export default ImportDialog
