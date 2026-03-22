// ========================================
// 通用對話框包裝元件 (Dialog)
// 包裝 shadcn Dialog 元件，保留原有 API 以利向後相容
// ========================================

import {
    Dialog as ShadDialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
} from '@/components/ui/dialog'

/**
 * 通用對話框元件（向後相容包裝）。
 *
 * 保留原有的 isOpen/onClose/title/children API，
 * 內部使用 shadcn Dialog 實作，自動支援 Dark/Light 主題。
 *
 * @param {Object} props
 * @param {boolean} props.isOpen - 是否開啟
 * @param {Function} props.onClose - 關閉事件
 * @param {string} props.title - 標題
 * @param {React.ReactNode} props.children - 內容
 * @param {string} [props.className] - 自訂 DialogContent 樣式
 * @param {boolean} [props.modal=true] - 是否為強制對話框（false: 允許點背景關閉）
 * @returns {JSX.Element}
 */
function Dialog({ isOpen, onClose, title, children, className = '', modal = true }) {
    return (
        <ShadDialog
            open={isOpen}
            onOpenChange={(open) => {
                // 當 dialog 被關閉時（shadcn 內建的 ESC / 背景點擊）
                if (!open) onClose()
            }}
            // modal=false 時允許與背景互動（shadcn Dialog 預設 modal=true）
        >
            <DialogContent
                className={`max-h-[85vh] flex flex-col gap-0 p-0 ${className}`}
                // 消除 Radix UI 的 Missing Description 警告
                aria-describedby={undefined}
                // 當 modal=true 時禁止點背景關閉
                onInteractOutside={(e) => {
                    if (modal) e.preventDefault()
                }}
            >
                {/* 標題列 */}
                <DialogHeader className="px-4 py-3 border-b border-border shrink-0">
                    <DialogTitle className="text-sm font-semibold">{title}</DialogTitle>
                </DialogHeader>

                {/* 內容區（可捲動） */}
                <div className="overflow-y-auto p-4 flex-1">
                    {children}
                </div>
            </DialogContent>
        </ShadDialog>
    )
}

export default Dialog
