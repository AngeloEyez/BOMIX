import { Component } from 'react'
import { AlertTriangle } from 'lucide-react'

class ErrorBoundary extends Component {
    constructor(props) {
        super(props)
        this.state = { hasError: false, error: null, errorInfo: null }
    }

    static getDerivedStateFromError(error) {
        return { hasError: true, error }
    }

    componentDidCatch(error, errorInfo) {
        console.error("UI Error Caught:", error, errorInfo)
        this.setState({ errorInfo })
    }

    render() {
        if (this.state.hasError) {
            return (
                <div className="flex flex-col items-center justify-center h-full p-8 text-center bg-red-50 dark:bg-red-900/10 rounded-lg border border-red-100 dark:border-red-900/30">
                    <div className="p-3 bg-red-100 dark:bg-red-900/30 text-red-600 dark:text-red-400 rounded-full mb-4">
                        <AlertTriangle size={32} />
                    </div>
                    <h2 className="text-xl font-bold text-red-700 dark:text-red-400 mb-2">
                        發生錯誤
                    </h2>
                    <p className="text-sm text-red-600 dark:text-red-300 mb-4 max-w-md">
                        頁面載入時發生未預期的錯誤。請嘗試重新載入，或聯絡開發人員。
                    </p>
                    <div className="w-full max-w-lg overflow-auto bg-white dark:bg-slate-900 p-4 rounded text-left border border-red-200 dark:border-red-800">
                        <code className="text-xs font-mono text-red-800 dark:text-red-300 block mb-2">
                            {this.state.error && this.state.error.toString()}
                        </code>
                        <details className="text-xs text-slate-500 cursor-pointer">
                            <summary>查看詳細堆疊 (Stack Trace)</summary>
                            <pre className="mt-2 whitespace-pre-wrap">
                                {this.state.errorInfo && this.state.errorInfo.componentStack}
                            </pre>
                        </details>
                    </div>
                    <button
                        onClick={() => window.location.reload()}
                        className="mt-6 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg text-sm font-medium transition-colors"
                    >
                        重新載入應用程式
                    </button>
                </div>
            )
        }

        return this.props.children
    }
}

export default ErrorBoundary
