import { useState, useCallback, useEffect } from 'react'
import { Upload, FileSpreadsheet, X } from 'lucide-react'
import Dialog from './Dialog'

// ========================================
// Excel 匯入對話框
// 選擇檔案 + 填寫 Phase / Version
// ========================================

/**
 * Excel 匯入對話框元件。
 *
 * 支援透過按鈕選擇檔案或拖曳 .xls/.xlsx 檔案。
 * 使用者需填入 Phase 名稱與版本號後送出匯入。
 *
 * @param {Object} props
 * @param {boolean} props.isOpen - 是否開啟
 * @param {Function} props.onClose - 關閉回呼
 * @param {number} props.projectId - 目標專案 ID
 * @param {string} props.projectCode - 目標專案代碼 (用於顯示與驗證)
 * @param {Function} props.onImport - 匯入回呼 (filePath, projectId, phaseName, version) => Promise
 * @returns {JSX.Element}
 */
function ImportDialog({ isOpen, onClose, onImport, initialFile }) {
    const [filePaths, setFilePaths] = useState([])
    const [error, setError] = useState('')
    const [isImporting, setIsImporting] = useState(false)
    const [isDragOver, setIsDragOver] = useState(false)

    // 處理初始檔案 (來自頁面拖曳)，在對話框開啟時同步帶入預設路徑
    useEffect(() => {
        if (isOpen && initialFile) {
            if (initialFile.path) {
                // eslint-disable-next-line react-hooks/set-state-in-effect
                setFilePaths([initialFile.path])  // 初始化拖曳帶入的檔案路徑，非循環觸發
            }
        }
    }, [isOpen, initialFile])

    // 重置表單
    const resetForm = useCallback(() => {
        setFilePaths([])
        setError('')
        setIsImporting(false)
        setIsDragOver(false)
    }, [])

    // 開啟檔案選擇對話框
    const handleSelectFile = async () => {
        try {
            const result = await window.api.dialog.showOpen({
                title: '選擇 BOM Excel 檔案 (可多選)',
                properties: ['openFile', 'multiSelections'],
                filters: [
                    { name: 'Excel Files', extensions: ['xls', 'xlsx'] },
                ],
            })
            if (!result.canceled && result.data && result.data.length > 0) {
                setFilePaths(result.data)
                setError('')
            }
        } catch (_e) {
            setError('開啟檔案失敗')
        }
    }

    // 拖曳處理
    const handleDragOver = (e) => {
        e.preventDefault()
        e.stopPropagation()
        setIsDragOver(true)
    }

    const handleDragLeave = (e) => {
        e.preventDefault()
        e.stopPropagation()
        setIsDragOver(false)
    }

    const handleDrop = (e) => {
        e.preventDefault()
        e.stopPropagation()
        setIsDragOver(false)

        const files = e.dataTransfer?.files
        if (files && files.length > 0) {
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
                setError(hasInvalid ? '僅支援 .xls 或 .xlsx 格式，已過濾非支援檔案' : '')
            } else {
                setError('僅支援 .xls 或 .xlsx 格式')
            }
        }
    }

    // 送出匯入
    const handleSubmit = async () => {
        if (!filePaths || filePaths.length === 0) {
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

    // 關閉時重置
    const handleClose = () => {
        resetForm()
        onClose()
    }

    return (
        <Dialog 
            isOpen={isOpen} 
            onClose={handleClose} 
            title={
                <div className="flex flex-col">
                    <span>匯入 Excel</span>
                    <span className="text-xs font-normal text-slate-500 dark:text-slate-400 mt-0.5">
                        BOM, Matrix
                    </span>
                </div>
            }
            className="max-w-md"
        >
            <div className="space-y-4">
                {/* 檔案選擇 / 拖曳區域 */}
                <div
                    onDragOver={handleDragOver}
                    onDragLeave={handleDragLeave}
                    onDrop={handleDrop}
                    onClick={filePaths.length === 0 ? handleSelectFile : undefined}
                    className={`border-2 border-dashed rounded-lg p-4 text-center cursor-pointer transition-colors
                        ${isDragOver
                            ? 'border-primary-400 bg-primary-50 dark:bg-primary-900/20'
                            : filePaths.length > 0
                                ? 'border-green-300 bg-green-50/50 dark:border-green-700 dark:bg-green-900/10'
                                : 'border-slate-300 dark:border-slate-600 hover:border-primary-300 dark:hover:border-primary-600'
                        }`}
                >
                    {filePaths.length > 0 ? (
                        <div className="flex flex-col items-center justify-center w-full">
                            <div className="flex items-center gap-2 mb-2">
                                <FileSpreadsheet size={20} className="text-green-600" />
                                <span className="text-sm font-bold text-slate-700 dark:text-slate-200">
                                    已選取 {filePaths.length} 個檔案
                                </span>
                            </div>
                            
                            {/* 檔案連結列表 */}
                            <div className="w-full max-h-40 overflow-y-auto bg-white/50 dark:bg-black/20 rounded border border-green-200 dark:border-green-900/50 p-2 space-y-1">
                                {filePaths.map((path, index) => (
                                    <div key={index} className="flex items-center gap-2 text-xs text-slate-600 dark:text-slate-400 group">
                                        <div className="w-1 h-1 rounded-full bg-green-500 shrink-0" />
                                        <span className="truncate flex-1 text-left" title={path}>
                                            {path.split(/[/\\]/).pop()}
                                        </span>
                                    </div>
                                ))}
                            </div>

                            <button
                                onClick={(e) => {
                                    e.stopPropagation()
                                    setFilePaths([])
                                }}
                                className="mt-3 text-xs text-red-500 hover:text-red-700 font-medium px-2 py-1 rounded bg-red-50 hover:bg-red-100 dark:bg-red-900/20 dark:hover:bg-red-900/40 transition-colors"
                            >
                                清除重新選擇
                            </button>
                        </div>
                    ) : (
                        <div className="space-y-1 text-slate-400">
                            <Upload size={24} className="mx-auto" />
                            <p className="text-sm">點擊選擇檔案或拖曳 .xls / .xlsx</p>
                        </div>
                    )}
                </div>



                {/* 錯誤訊息 */}
                {error && (
                    <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
                )}

                {/* 操作按鈕 */}
                <div className="flex justify-end gap-2 pt-1">
                    <button
                        onClick={handleClose}
                        disabled={isImporting}
                        className="px-4 py-2 text-sm font-medium text-slate-600 dark:text-slate-300
                            bg-slate-100 dark:bg-surface-700 hover:bg-slate-200 dark:hover:bg-surface-600
                            rounded-lg transition-colors disabled:opacity-50"
                    >
                        取消
                    </button>
                    <button
                        onClick={handleSubmit}
                        disabled={isImporting}
                        className="px-4 py-2 text-sm font-medium text-white
                            bg-primary-600 hover:bg-primary-700
                            rounded-lg transition-colors disabled:opacity-50"
                    >
                        {isImporting ? '匯入中...' : '匯入'}
                    </button>
                </div>
            </div>
        </Dialog>
    )
}

export default ImportDialog
