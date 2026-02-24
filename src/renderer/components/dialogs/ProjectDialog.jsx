import { useState, useEffect } from 'react'
import Dialog from './Dialog'
import { X } from 'lucide-react'

/**
 * 專案編輯對話框
 * 
 * @param {Object} props
 * @param {boolean} props.isOpen - 是否開啟
 * @param {Function} props.onClose - 關閉回呼
 * @param {string} props.mode - 'create' | 'edit'
 * @param {Object} props.initialData - 編輯時的初始資料
 * @param {Function} props.onSave - 儲存回呼 (code, desc) => Promise
 */
function ProjectDialog({ isOpen, onClose, mode = 'create', initialData, onSave }) {
    const [projectCode, setProjectCode] = useState('')
    const [description, setDescription] = useState('')
    const [error, setError] = useState('')
    const [isSaving, setIsSaving] = useState(false)

    useEffect(() => {
        if (isOpen) {
            setProjectCode(initialData?.project_code || '')
            setDescription(initialData?.description || '')
            setError('')
        }
    }, [isOpen, initialData, mode])

    const handleSave = async () => {
        if (!projectCode.trim()) {
            setError('請輸入專案代碼')
            return
        }

        setIsSaving(true)
        setError('')
        try {
            await onSave(projectCode.trim(), description.trim())
            // onClose is handled by parent usually, but we can verify success there.
        } catch (err) {
            setError(err.message || '儲存失敗')
        } finally {
            setIsSaving(false)
        }
    }

    return (
        <Dialog
            isOpen={isOpen}
            onClose={onClose}
            title={mode === 'create' ? '新增專案' : '編輯專案'}
            className="max-w-md"
        >
            <div className="space-y-4">
                {/* Project Code */}
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                        專案代碼 {mode === 'create' && <span className="text-red-500">*</span>}
                    </label>
                    <input
                        type="text"
                        value={projectCode}
                        onChange={(e) => setProjectCode(e.target.value.toUpperCase())}
                        placeholder="例：TANGLED"
                        className="w-full px-3 py-2 text-sm
                            bg-white dark:bg-surface-900
                            border border-slate-300 dark:border-slate-600
                            rounded-lg text-slate-800 dark:text-slate-200
                            focus:outline-none focus:ring-2 focus:ring-primary-500
                            placeholder:text-slate-400"
                        autoFocus={mode === 'create'}
                    />
                    {mode === 'edit' && (
                        <p className="text-xs text-amber-600 dark:text-amber-400 mt-1">
                            注意：修改專案代碼可能會影響關聯資料的識別 (視實作而定)。
                        </p>
                    )}
                </div>

                {/* Description */}
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                        描述
                    </label>
                    <textarea
                        value={description}
                        onChange={(e) => setDescription(e.target.value)}
                        placeholder="專案描述（選填）"
                        rows={3}
                        className="w-full px-3 py-2 text-sm
                            bg-white dark:bg-surface-900
                            border border-slate-300 dark:border-slate-600
                            rounded-lg text-slate-800 dark:text-slate-200
                            focus:outline-none focus:ring-2 focus:ring-primary-500
                            resize-none placeholder:text-slate-400"
                        autoFocus={mode === 'edit'}
                    />
                </div>

                {/* Error */}
                {error && (
                    <div className="text-sm text-red-500 flex items-center gap-1">
                        <X size={14} />
                        {error}
                    </div>
                )}

                {/* Buttons */}
                <div className="flex justify-end gap-2 pt-2">
                    <button
                        onClick={onClose}
                        className="px-4 py-2 text-sm text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-surface-700 rounded-lg transition-colors"
                    >
                        取消
                    </button>
                    <button
                        onClick={handleSave}
                        disabled={isSaving}
                        className="flex items-center gap-2 px-4 py-2 text-sm text-white bg-primary-600 hover:bg-primary-700 rounded-lg transition-colors disabled:opacity-50"
                    >
                        {isSaving ? '儲存中...' : <>{mode === 'create' ? '建立' : '儲存'}</>}
                    </button>
                </div>
            </div>
        </Dialog>
    )
}

export default ProjectDialog
