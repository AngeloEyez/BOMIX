import { X, CheckCircle, AlertCircle, Info } from 'lucide-react'
import useToastStore from '../../stores/useToastStore'

function ToastContainer() {
    const toasts = useToastStore(state => state.toasts)
    const removeToast = useToastStore(state => state.removeToast)

    if (toasts.length === 0) return null

    return (
        <div className="fixed bottom-12 right-6 z-[100] flex flex-col gap-2 pointer-events-none">
            {toasts.map(toast => {
                let bg, border, text, Icon
                switch (toast.type) {
                    case 'success':
                        bg = 'bg-green-50 dark:bg-green-900/80'
                        border = 'border-green-200 dark:border-green-800'
                        text = 'text-green-800 dark:text-green-100'
                        Icon = <CheckCircle size={18} className="text-green-500 dark:text-green-400 shrink-0" />
                        break
                    case 'error':
                        bg = 'bg-red-50 dark:bg-red-900/80'
                        border = 'border-red-200 dark:border-red-800'
                        text = 'text-red-800 dark:text-red-100'
                        Icon = <AlertCircle size={18} className="text-red-500 dark:text-red-400 shrink-0" />
                        break
                    case 'info':
                    default:
                        bg = 'bg-blue-50 dark:bg-blue-900/80'
                        border = 'border-blue-200 dark:border-blue-800'
                        text = 'text-blue-800 dark:text-blue-100'
                        Icon = <Info size={18} className="text-blue-500 dark:text-blue-400 shrink-0" />
                        break
                }

                return (
                    <div 
                        key={toast.id}
                        className={`pointer-events-auto flex items-start gap-3 w-80 p-4 border rounded-xl shadow-lg transition-all duration-300 opacity-100 scale-100 ${bg} ${border}`}
                    >
                        {Icon}
                        <div className={`flex-1 text-sm font-medium ${text}`}>
                            {toast.message}
                        </div>
                        <button 
                            onClick={() => removeToast(toast.id)}
                            className={`p-0.5 rounded-md hover:bg-black/5 dark:hover:bg-white/10 transition-colors ${text}`}
                        >
                            <X size={16} />
                        </button>
                    </div>
                )
            })}
        </div>
    )
}

export default ToastContainer
