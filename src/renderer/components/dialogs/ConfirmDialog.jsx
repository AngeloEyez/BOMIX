import Dialog from './Dialog'
import { AlertTriangle, CheckCircle, Info, HelpCircle } from 'lucide-react'

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
 * @param {boolean} [props.danger=false] - 是否為危險操作（紅色按鈕）- Deprecated, use variant='danger'
 * @param {'primary'|'danger'|'warning'|'success'|'info'} [props.variant='primary'] - 對話框樣式
 * @param {React.ReactNode} [props.icon] - 自訂圖標 (若未提供則根據 variant 自動選擇)
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
    variant = 'primary', // primary, danger, success, warning, info
    icon = null
}) {
    // 兼容舊版 danger prop
    const finalVariant = danger ? 'danger' : variant;

    const getButtonColor = () => {
        switch (finalVariant) {
            case 'danger': return 'bg-red-600 hover:bg-red-700 focus:ring-red-500';
            case 'warning': return 'bg-orange-600 hover:bg-orange-700 focus:ring-orange-500';
            case 'success': return 'bg-emerald-600 hover:bg-emerald-700 focus:ring-emerald-500';
            case 'info': return 'bg-blue-600 hover:bg-blue-700 focus:ring-blue-500';
            default: return 'bg-primary-600 hover:bg-primary-700 focus:ring-primary-500';
        }
    };

    const getDefaultIcon = () => {
        if (icon) return icon;

        switch (finalVariant) {
            case 'danger':
                return (
                    <div className="shrink-0 p-2 bg-red-100 dark:bg-red-900/30 text-red-600 dark:text-red-400 rounded-lg">
                        <AlertTriangle size={20} />
                    </div>
                );
            case 'warning':
                return (
                    <div className="shrink-0 p-2 bg-orange-100 dark:bg-orange-900/30 text-orange-600 dark:text-orange-400 rounded-lg">
                        <AlertTriangle size={20} />
                    </div>
                );
            case 'success':
                return (
                    <div className="shrink-0 p-2 bg-emerald-100 dark:bg-emerald-900/30 text-emerald-600 dark:text-emerald-400 rounded-lg">
                        <CheckCircle size={20} />
                    </div>
                );
            case 'info':
                return (
                    <div className="shrink-0 p-2 bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 rounded-lg">
                        <Info size={20} />
                    </div>
                );
            case 'primary':
            default:
                 // Default confirm icon? Maybe HelpCircle or just Info?
                 // Or allow null if no icon desired for primary.
                 return (
                    <div className="shrink-0 p-2 bg-primary-100 dark:bg-primary-900/30 text-primary-600 dark:text-primary-400 rounded-lg">
                        <HelpCircle size={20} />
                    </div>
                );
        }
    }

    return (
        <Dialog isOpen={isOpen} onClose={onClose} title={title} className="max-w-sm" modal={true}>
            <div className="flex flex-col gap-4">
                {/* 圖標與訊息 */}
                <div className="flex items-start gap-3">
                    {getDefaultIcon()}
                    <p className="text-sm text-slate-600 dark:text-slate-300 leading-relaxed pt-0.5 whitespace-pre-line">
                        {message}
                    </p>
                </div>

                {/* 操作按鈕 */}
                <div className="flex justify-end gap-2 pt-2">
                    <button
                        onClick={onClose}
                        className="px-4 py-2 text-sm font-medium text-slate-600 dark:text-slate-300
                            bg-slate-100 dark:bg-surface-700 hover:bg-slate-200 dark:hover:bg-surface-600
                            rounded-lg transition-colors focus:ring-2 focus:ring-offset-1 dark:focus:ring-offset-surface-800"
                    >
                        {cancelText}
                    </button>
                    <button
                        onClick={() => {
                            onConfirm()
                            onClose()
                        }}
                        className={`px-4 py-2 text-sm font-medium text-white rounded-lg transition-colors shadow-sm focus:ring-2 focus:ring-offset-1 dark:focus:ring-offset-surface-800 ${getButtonColor()}`}
                    >
                        {confirmText}
                    </button>
                </div>
            </div>
        </Dialog>
    )
}

export default ConfirmDialog
