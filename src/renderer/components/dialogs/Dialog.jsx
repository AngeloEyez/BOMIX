import { X } from 'lucide-react'
import { useEffect, useRef } from 'react'

// ========================================
// 通用對話框元件 (Dialog)
// 提供 Modal 基礎外觀與行為
// ========================================

/**
 * 
 * @param {Object} props
 * @param {boolean} props.isOpen - 是否開啟
 * @param {Function} props.onClose - 關閉事件
 * @param {string} props.title - 標題
 * @param {React.ReactNode} props.children - 內容
 * @param {string} props.className - 自訂樣式
 * @param {boolean} [props.modal=true] - 是否為強制對話框 (true: 點背景不關閉, false: 點背景關閉)
 */
function Dialog({ isOpen, onClose, title, children, className = '', modal = true }) {
    const dialogRef = useRef(null)

    // 點擊 Backdrop 關閉
    const handleBackdropClick = (e) => {
        if (e.target === e.currentTarget) {
            if (!modal) {
                onClose()
            } else {
                // Optional: Shake animation or visual cue?
                // For now, just do nothing.
                // Maybe focus the dialog?
                dialogRef.current?.focus()
            }
        }
    }

    // 按 ESC 關閉
    useEffect(() => {
        const handleEsc = (e) => {
            if (e.key === 'Escape' && isOpen) {
                onClose()
            }
        }
        window.addEventListener('keydown', handleEsc)
        return () => window.removeEventListener('keydown', handleEsc)
    }, [isOpen, onClose])

    if (!isOpen) return null

    return (
        <div 
            className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm animate-fade-in"
            onClick={handleBackdropClick}
        >
            <div 
                ref={dialogRef}
                className={`bg-white dark:bg-surface-800 rounded-xl shadow-xl w-full max-w-lg mx-4 flex flex-col max-h-[85vh] animate-scale-in ${className}`}
            >
                {/* Header */}
                <div className="flex items-center justify-between p-4 border-b border-slate-100 dark:border-slate-700">
                    <h3 className="text-lg font-semibold text-slate-800 dark:text-slate-100">
                        {title}
                    </h3>
                    <button 
                        onClick={onClose}
                        className="p-1 rounded-md hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-500 dark:text-slate-400"
                    >
                        <X size={20} />
                    </button>
                </div>

                {/* Content */}
                <div className="p-6 overflow-y-auto">
                    {children}
                </div>
            </div>
        </div>
    )
}

export default Dialog
