// ========================================
// 確認對話框元件 (ConfirmDialog)
// 用於刪除確認等需要使用者二次確認的場景
// 使用 shadcn Button 統一按鈕樣式
// ========================================

import Dialog from './Dialog'
import { AlertTriangle, CheckCircle, Info, HelpCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'

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
 * @param {boolean} [props.danger=false] - 是否為危險操作（舊版相容）
 * @param {'primary'|'danger'|'warning'|'success'|'info'} [props.variant='primary'] - 對話框樣式
 * @param {React.ReactNode} [props.icon] - 自訂圖標
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
    variant = 'primary',
    icon = null
}) {
    // 兼容舊版 danger prop
    const finalVariant = danger ? 'danger' : variant

    /**
     * 取得確認按鈕的 shadcn variant。
     *
     * @returns {'destructive'|'default'} shadcn Button variant
     */
    const getButtonVariant = () => {
        switch (finalVariant) {
            case 'danger': return 'destructive'
            default: return 'default'
        }
    }

    /**
     * 取得圖標元素（依 variant 自動選擇）。
     *
     * @returns {JSX.Element|null} 圖標元素
     */
    const getDefaultIcon = () => {
        if (icon) return icon

        const iconMap = {
            danger: { Icon: AlertTriangle, cls: 'bg-destructive/10 text-destructive' },
            warning: { Icon: AlertTriangle, cls: 'bg-amber-100 text-amber-600 dark:bg-amber-900/30 dark:text-amber-400' },
            success: { Icon: CheckCircle, cls: 'bg-emerald-100 text-emerald-600 dark:bg-emerald-900/30 dark:text-emerald-400' },
            info: { Icon: Info, cls: 'bg-primary/10 text-primary' },
            primary: { Icon: HelpCircle, cls: 'bg-primary/10 text-primary' },
        }
        const { Icon, cls } = iconMap[finalVariant] ?? iconMap.primary
        return (
            <div className={`shrink-0 p-2 rounded-lg ${cls}`}>
                <Icon size={18} />
            </div>
        )
    }

    return (
        <Dialog isOpen={isOpen} onClose={onClose} title={title} className="max-w-sm" modal={true}>
            <div className="flex flex-col gap-4">
                {/* 圖標與訊息 */}
                <div className="flex items-start gap-3">
                    {getDefaultIcon()}
                    <p className="text-sm text-foreground leading-relaxed pt-0.5 whitespace-pre-line">
                        {message}
                    </p>
                </div>

                {/* 操作按鈕 */}
                <div className="flex justify-end gap-2">
                    <Button variant="outline" size="sm" onClick={onClose}>
                        {cancelText}
                    </Button>
                    <Button
                        variant={getButtonVariant()}
                        size="sm"
                        onClick={() => {
                            onConfirm()
                            onClose()
                        }}
                    >
                        {confirmText}
                    </Button>
                </div>
            </div>
        </Dialog>
    )
}

export default ConfirmDialog
