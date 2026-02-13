import Dialog from './Dialog'
import ReactMarkdown from 'react-markdown'
import { useState, useEffect } from 'react'

// ========================================
// 更新記錄對話框 (ChangelogDialog)
// 讀取並渲染 CHANGELOG.md
// ========================================

function ChangelogDialog({ isOpen, onClose }) {
    const [content, setContent] = useState('')
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState(null)

    useEffect(() => {
        if (isOpen) {
            setLoading(true)
            setError(null)
            window.api.app.getChangelog()
                .then(result => {
                    if (result.success) {
                        setContent(result.data)
                    } else {
                        setError(result.error)
                    }
                })
                .catch(err => setError(err.message))
                .finally(() => setLoading(false))
        }
    }, [isOpen])

    return (
        <Dialog isOpen={isOpen} onClose={onClose} title="更新記錄" className="max-w-2xl">
            <div className="min-h-[300px]">
                {loading ? (
                    <div className="flex items-center justify-center h-full py-20 text-slate-400">
                        載入中...
                    </div>
                ) : error ? (
                    <div className="flex items-center justify-center h-full py-20 text-red-500">
                        {error}
                    </div>
                ) : (
                    <div className="prose prose-sm dark:prose-invert max-w-none">
                        <ReactMarkdown>{content}</ReactMarkdown>
                    </div>
                )}
            </div>
        </Dialog>
    )
}

export default ChangelogDialog
