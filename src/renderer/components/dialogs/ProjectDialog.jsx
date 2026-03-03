// ========================================
// 專案新增/編輯對話框 (ProjectDialog)
// 使用 shadcn Input 和 Button 統一風格
// ========================================

import { useState, useEffect } from 'react'
import Dialog from './Dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { AlertCircle } from 'lucide-react'

/**
 * 專案編輯對話框。
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

    // 開啟時重置表單
    useEffect(() => {
        if (isOpen) {
            setProjectCode(initialData?.project_code || '')
            setDescription(initialData?.description || '')
            setError('')
        }
    }, [isOpen, initialData, mode])

    /**
     * 處理儲存，驗證後呼叫 onSave 回呼。
     */
    const handleSave = async () => {
        if (!projectCode.trim()) {
            setError('請輸入專案代碼')
            return
        }
        setIsSaving(true)
        setError('')
        try {
            await onSave(projectCode.trim(), description.trim())
        } catch (err) {
            setError(err.message || '儲存失敗')
        } finally {
            setIsSaving(false)
        }
    }

    /**
     * 按下 Enter 時觸發儲存。
     *
     * @param {KeyboardEvent} e
     */
    const handleKeyDown = (e) => {
        if (e.key === 'Enter' && !e.shiftKey) handleSave()
    }

    return (
        <Dialog
            isOpen={isOpen}
            onClose={onClose}
            title={mode === 'create' ? '新增專案' : '編輯專案'}
            className="max-w-md"
        >
            <div className="space-y-3">
                {/* 專案代碼 */}
                <div className="space-y-1">
                    <Label htmlFor="project-code" className="text-xs">
                        專案代碼 {mode === 'create' && <span className="text-destructive">*</span>}
                    </Label>
                    <Input
                        id="project-code"
                        type="text"
                        value={projectCode}
                        onChange={(e) => setProjectCode(e.target.value.toUpperCase())}
                        onKeyDown={handleKeyDown}
                        placeholder="例：TANGLED"
                        className="h-8 text-xs"
                        autoFocus={mode === 'create'}
                    />
                    {mode === 'edit' && (
                        <p className="text-xs text-amber-600 dark:text-amber-400">
                            注意：修改專案代碼可能會影響關聯資料的識別。
                        </p>
                    )}
                </div>

                {/* 描述 */}
                <div className="space-y-1">
                    <Label htmlFor="project-desc" className="text-xs">描述</Label>
                    <textarea
                        id="project-desc"
                        value={description}
                        onChange={(e) => setDescription(e.target.value)}
                        placeholder="專案描述（選填）"
                        rows={3}
                        className="w-full px-3 py-1.5 text-xs selectable
                            bg-background border border-input rounded-md
                            text-foreground placeholder:text-muted-foreground
                            focus:outline-none focus:ring-1 focus:ring-ring
                            resize-none transition-colors"
                        autoFocus={mode === 'edit'}
                    />
                </div>

                {/* 錯誤訊息 */}
                {error && (
                    <div className="flex items-center gap-1.5 text-xs text-destructive">
                        <AlertCircle size={13} />
                        {error}
                    </div>
                )}

                {/* 操作按鈕 */}
                <div className="flex justify-end gap-2 pt-1">
                    <Button variant="outline" size="sm" onClick={onClose}>
                        取消
                    </Button>
                    <Button size="sm" onClick={handleSave} disabled={isSaving}>
                        {isSaving ? '儲存中...' : (mode === 'create' ? '建立' : '儲存')}
                    </Button>
                </div>
            </div>
        </Dialog>
    )
}

export default ProjectDialog
