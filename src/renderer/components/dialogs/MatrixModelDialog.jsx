// ========================================
// Matrix Model 管理對話框 (MatrixModelDialog)
// 改用 shadcn Dialog 封裝與 Button/Input 統一風格
// ========================================

import { useEffect } from 'react'
import useMatrixStore from '../../stores/useMatrixStore'
import { Trash2, Plus, RefreshCw, AlertCircle } from 'lucide-react'
import Dialog from './Dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

/**
 * Matrix Model 管理對話框。
 *
 * 提供 Model 的新增、編輯（編輯後 blur 時自動儲存）、刪除功能。
 * 當沒有任何 Model 時顯示初始化按鈕。
 *
 * @param {Object} props
 * @param {boolean} props.isOpen - 是否開啟
 * @param {Function} props.onClose - 關閉回呼
 * @param {number} props.bomRevisionId - 當前 BOM 版本 ID
 * @returns {JSX.Element}
 */
const MatrixModelDialog = ({ isOpen, onClose, bomRevisionId }) => {
    const {
        matrixData, fetchMatrixData, createModels,
        updateModel, deleteModel, isLoading, error
    } = useMatrixStore()

    const models = matrixData[bomRevisionId]?.models || []

    // 開啟時載入 Model 資料
    useEffect(() => {
        if (isOpen && bomRevisionId) fetchMatrixData(bomRevisionId)
    }, [isOpen, bomRevisionId, fetchMatrixData])

    /** 初始化預設 Models (A, B, C) */
    const handleCreateDefault = () => createModels(bomRevisionId)

    /** 新增一筆空白 Model */
    const handleAddModel = () => createModels(bomRevisionId, [{ name: 'New Model', description: '' }])

    /**
     * 更新單一 Model 的指定欄位（blur 時觸發）。
     *
     * @param {number} id - Model ID
     * @param {string} field - 欄位名稱
     * @param {string} value - 新值
     */
    const handleUpdate = (id, field, value) => updateModel(bomRevisionId, id, { [field]: value })

    /**
     * 刪除 Model（確認後執行）。
     *
     * @param {number} id - Model ID
     * @param {string} name - Model 名稱（顯示於確認訊息）
     */
    const handleDelete = (id, name) => {
        if (confirm(`確定要刪除 Model "${name}" 嗎？\n如果有相關聯的選擇紀錄，必須先清除選擇才能刪除。`)) {
            deleteModel(bomRevisionId, id)
        }
    }

    return (
        <Dialog isOpen={isOpen} onClose={onClose} title="Matrix Model 管理" className="max-w-2xl">
            <div className="space-y-3">
                {/* 錯誤訊息 */}
                {error && (
                    <div className="flex items-center gap-2 px-3 py-2 bg-destructive/10 border border-destructive/30 rounded-lg text-xs text-destructive">
                        <AlertCircle size={13} />
                        {error}
                    </div>
                )}

                {/* 載入中 */}
                {isLoading && (
                    <div className="flex justify-center py-6">
                        <RefreshCw size={20} className="animate-spin text-muted-foreground" />
                    </div>
                )}

                {/* 空狀態：初始化提示 */}
                {!isLoading && models.length === 0 && (
                    <div className="text-center py-8 space-y-3">
                        <p className="text-sm text-muted-foreground">目前沒有任何 Matrix Model</p>
                        <Button size="sm" onClick={handleCreateDefault} className="gap-1.5">
                            <RefreshCw size={13} />
                            初始化預設 Models (A, B, C)
                        </Button>
                    </div>
                )}

                {/* Model 列表 */}
                {models.length > 0 && (
                    <div className="space-y-2 max-h-80 overflow-y-auto -mr-1 pr-1">
                        {models.map((model) => (
                            <div
                                key={model.id}
                                className="flex items-center gap-3 p-3 bg-muted/40 rounded-lg border border-border"
                            >
                                <div className="flex-1 space-y-2">
                                    {/* 名稱輸入 */}
                                    <div className="flex items-center gap-2">
                                        <span className="text-xs font-medium text-muted-foreground w-10">名稱</span>
                                        <Input
                                            className="h-7 text-xs"
                                            defaultValue={model.name}
                                            onBlur={(e) => {
                                                if (e.target.value !== model.name) {
                                                    handleUpdate(model.id, 'name', e.target.value)
                                                }
                                            }}
                                        />
                                    </div>
                                    {/* 描述輸入 */}
                                    <div className="flex items-center gap-2">
                                        <span className="text-xs font-medium text-muted-foreground w-10">描述</span>
                                        <Input
                                            className="h-7 text-xs"
                                            defaultValue={model.description}
                                            placeholder="選填..."
                                            onBlur={(e) => {
                                                if (e.target.value !== model.description) {
                                                    handleUpdate(model.id, 'description', e.target.value)
                                                }
                                            }}
                                        />
                                    </div>
                                </div>
                                {/* 刪除按鈕 */}
                                <Button
                                    variant="ghost"
                                    size="icon"
                                    className="h-8 w-8 text-destructive hover:text-destructive hover:bg-destructive/10"
                                    onClick={() => handleDelete(model.id, model.name)}
                                    title="刪除"
                                >
                                    <Trash2 size={15} />
                                </Button>
                            </div>
                        ))}
                    </div>
                )}

                {/* 底部：Model 數量 + 新增按鈕 */}
                <div className="flex items-center justify-between pt-2 border-t border-border">
                    <span className="text-xs text-muted-foreground">共 {models.length} 個 Model</span>
                    <Button variant="outline" size="sm" onClick={handleAddModel} className="gap-1.5">
                        <Plus size={13} />
                        新增 Model
                    </Button>
                </div>
            </div>
        </Dialog>
    )
}

export default MatrixModelDialog
