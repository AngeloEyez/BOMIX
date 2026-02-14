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
 * @param {Function} props.onImport - 匯入回呼 (filePath, projectId, phaseName, version) => Promise
 * @returns {JSX.Element}
 */
function ImportDialog({ isOpen, onClose, projectId, onImport, initialFile }) {
    const [filePath, setFilePath] = useState('')
    const [fileName, setFileName] = useState('')
    const [phaseName, setPhaseName] = useState('')
    const [version, setVersion] = useState('')
    const [suffix, setSuffix] = useState('')
    const [error, setError] = useState('')
    const [isImporting, setIsImporting] = useState(false)
    const [isDragOver, setIsDragOver] = useState(false)

    // 處理初始檔案 (來自頁面拖曳)
    useEffect(() => {
        if (isOpen && initialFile) {
            if (initialFile.path) {
                setFilePath(initialFile.path)
                setFileName(initialFile.name)
            }
        }
    }, [isOpen, initialFile])

    // 重置表單
    const resetForm = useCallback(() => {
        setFilePath('')
        setFileName('')
        setPhaseName('')
        setVersion('')
        setSuffix('')
        setError('')
        setIsImporting(false)
        setIsDragOver(false)
    }, [])

    // 開啟檔案選擇對話框
    const handleSelectFile = async () => {
        try {
            const result = await window.api.dialog.showOpen({
                title: '選擇 BOM Excel 檔案',
                filters: [
                    { name: 'Excel Files', extensions: ['xls', 'xlsx'] },
                ],
            })
            if (!result.canceled && result.data) {
                const path = result.data
                setFilePath(path)
                setFileName(path.split(/[\\/]/).pop())
                setError('')
            }
        } catch (e) {
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
            const file = files[0]
            const ext = file.name.split('.').pop()?.toLowerCase()
            if (ext === 'xls' || ext === 'xlsx') {
                const path = window.api.utils.getPathForFile(file)
                setFilePath(path)
                setFileName(file.name)
                setError('')
            } else {
                setError('僅支援 .xls 或 .xlsx 格式')
            }
        }
    }

    // 送出匯入
    const handleSubmit = async () => {
        if (!filePath) {
            setError('請選擇 Excel 檔案')
            return
        }
        if (!phaseName.trim()) {
            setError('請輸入 Phase 名稱')
            return
        }
        if (!version.trim()) {
            setError('請輸入版本號')
            return
        }

        setIsImporting(true)
        setError('')

        try {
            const result = await onImport(filePath, projectId, phaseName.trim(), version.trim())
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
        <Dialog isOpen={isOpen} onClose={handleClose} title="匯入 BOM Excel" className="max-w-md">
            <div className="space-y-4">
                {/* 檔案選擇 / 拖曳區域 */}
                <div
                    onDragOver={handleDragOver}
                    onDragLeave={handleDragLeave}
                    onDrop={handleDrop}
                    onClick={!filePath ? handleSelectFile : undefined}
                    className={`border-2 border-dashed rounded-lg p-4 text-center cursor-pointer transition-colors
                        ${isDragOver
                            ? 'border-primary-400 bg-primary-50 dark:bg-primary-900/20'
                            : filePath
                                ? 'border-green-300 bg-green-50/50 dark:border-green-700 dark:bg-green-900/10'
                                : 'border-slate-300 dark:border-slate-600 hover:border-primary-300 dark:hover:border-primary-600'
                        }`}
                >
                    {filePath ? (
                        <div className="flex items-center justify-center gap-2">
                            <FileSpreadsheet size={20} className="text-green-600" />
                            <span className="text-sm text-slate-700 dark:text-slate-300 font-medium truncate max-w-[260px]">
                                {fileName}
                            </span>
                            <button
                                onClick={(e) => {
                                    e.stopPropagation()
                                    setFilePath('')
                                    setFileName('')
                                }}
                                className="p-0.5 rounded hover:bg-slate-200 dark:hover:bg-surface-600 text-slate-400"
                            >
                                <X size={14} />
                            </button>
                        </div>
                    ) : (
                        <div className="space-y-1 text-slate-400">
                            <Upload size={24} className="mx-auto" />
                            <p className="text-sm">點擊選擇檔案或拖曳 .xls / .xlsx</p>
                        </div>
                    )}
                </div>

                {/* Phase 名稱 */}
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                        Phase 名稱 <span className="text-red-500">*</span>
                    </label>
                    <input
                        type="text"
                        value={phaseName}
                        onChange={(e) => setPhaseName(e.target.value.toUpperCase())}
                        placeholder="例：EVT, SI, DB"
                        className="w-full px-3 py-2 text-sm
                            bg-white dark:bg-surface-900
                            border border-slate-300 dark:border-slate-600
                            rounded-lg text-slate-800 dark:text-slate-200
                            focus:outline-none focus:ring-2 focus:ring-primary-500
                            placeholder:text-slate-400"
                    />
                </div>

                {/* 版本號 */}
                <div>
                    <div className="space-y-1">
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                            版本 (Version) <span className="text-red-500">*</span>
                        </label>
                        <input
                            type="text"
                            value={version}
                            onChange={(e) => setVersion(e.target.value)}
                            placeholder="e.g. 0.1, 1.0"
                            className="w-full px-3 py-2 text-sm
                                bg-white dark:bg-surface-800
                                border border-slate-300 dark:border-slate-600
                                rounded-lg text-slate-800 dark:text-slate-200
                                focus:outline-none focus:ring-2 focus:ring-primary-500"
                        />
                    </div>

                    <div className="space-y-1 mt-4"> {/* Added mt-4 for spacing between Version and Suffix */}
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                            後綴 (Suffix) <span className="text-xs text-slate-400 font-normal">(選填)</span>
                        </label>
                        <input
                            type="text"
                            value={suffix}
                            onChange={(e) => setSuffix(e.target.value)}
                            placeholder="e.g. A, B, Test"
                            className="w-full px-3 py-2 text-sm
                                bg-white dark:bg-surface-800
                                border border-slate-300 dark:border-slate-600
                                rounded-lg text-slate-800 dark:text-slate-200
                                focus:outline-none focus:ring-2 focus:ring-primary-500"
                        />
                    </div>
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
