import Dialog from './Dialog'
import { AlertTriangle } from 'lucide-react'

// ========================================
// 確認對話框元件 (ConfirmDialog)
// 用於刪除確認等需要使用者二次確認的場景
// ========================================

/**
 * 確認對話框。
 *
 * @param {Object} props
 * @param {boolean} props.isOpen - 是否開啟
 * @param {Function} props.onClose - 關閉事件
 * @param {Function} props.onConfirm - 確認事件
 * @param {string} props.title - 標題
 * @param {string} props.message - 確認訊息
 * @param {string} [props.confirmText='確認'] - 確認按鈕文字
 * @param {string} [props.cancelText='取消'] - 取消按鈕文字
 * @param {boolean} [props.danger=false] - 是否為危險操作（紅色按鈕）
 * @returns {JSX.Element}
 */
function ConfirmDialog({
    isOpen,
    onClose,
    onConfirm,
    title = '確認',
    message,
    confirmText = '確認',
    cancelText = '取消',
    danger = false,
}) {
    return (
        <Dialog isOpen={isOpen} onClose={onClose} title={title} className="max-w-sm">
            <div className="flex flex-col gap-4">
                {/* 圖標與訊息 */}
                <div className="flex items-start gap-3">
                    {danger && (
                        <div className="shrink-0 p-2 bg-red-100 dark:bg-red-900/30 text-red-600 dark:text-red-400 rounded-lg">
                            <AlertTriangle size={20} />
                        </div>
                    )}
                    <p className="text-sm text-slate-600 dark:text-slate-300 leading-relaxed">
                        {message}
                    </p>
                </div>

                {/* 操作按鈕 */}
                <div className="flex justify-end gap-2 pt-2">
                    <button
                        onClick={onClose}
                        className="px-4 py-2 text-sm font-medium text-slate-600 dark:text-slate-300
                            bg-slate-100 dark:bg-surface-700 hover:bg-slate-200 dark:hover:bg-surface-600
                            rounded-lg transition-colors"
                    >
                        {cancelText}
                    </button>
                    <button
                        onClick={() => {
                            onConfirm()
                            onClose()
                        }}
                        className={`px-4 py-2 text-sm font-medium text-white rounded-lg transition-colors ${
                            danger
                                ? 'bg-red-600 hover:bg-red-700'
                                : 'bg-primary-600 hover:bg-primary-700'
                        }`}
                    >
                        {confirmText}
                    </button>
                </div>
            </div>
        </Dialog>
    )
}

export default ConfirmDialog
