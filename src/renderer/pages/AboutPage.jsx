import { useState, useEffect } from 'react'
import { Package } from 'lucide-react'
import { Separator } from '@/components/ui/separator'
import ReactMarkdown from 'react-markdown'

// ========================================
// 關於頁面 (AboutPage)
// 顯示應用程式版本、技術資訊與授權，以及更新紀錄
// ========================================

/**
 * 關於頁面元件。
 *
 * 全頁面顯示 BOMIX 應用程式的版本號、Electron/Node.js 技術資訊、GitHub 連結，
 * 以及從 CHANGELOG.md 讀取的更新紀錄。
 *
 * @returns {JSX.Element}
 */
function AboutPage() {
    // ---- 關於資訊狀態 ----
    const [version, setVersion] = useState('Unknown')
    const [versions] = useState(() => {
        try {
            return window.api.getVersions() ?? { electron: '', chrome: '', node: '' }
        } catch (_e) {
            return { electron: '', chrome: '', node: '' }
        }
    })

    // ---- 更新紀錄狀態 ----
    const [changelogContent, setChangelogContent] = useState('')
    const [changelogLoading, setChangelogLoading] = useState(true)
    const [changelogError, setChangelogError] = useState(null)

    // 初始化：取得版本號與更新紀錄
    useEffect(() => {
        // 取得版本號
        if (window.api?.getVersion) {
            window.api.getVersion().then(setVersion).catch(() => setVersion('dev'))
        }

        // 取得更新紀錄
        if (window.api?.app?.getChangelog) {
            // eslint-disable-next-line react-hooks/set-state-in-effect
            setChangelogLoading(true)
            setChangelogError(null)
            window.api.app.getChangelog()
                .then(result => {
                    if (result.success) {
                        setChangelogContent(result.data)
                    } else {
                        setChangelogError(result.error)
                    }
                })
                .catch(err => setChangelogError(err.message))
                .finally(() => setChangelogLoading(false))
        } else {
            setChangelogLoading(false)
            setChangelogError('無法取得更新紀錄 API')
        }
    }, [])

    return (
        <div className="flex flex-col h-full overflow-hidden bg-background text-foreground animate-fade-in">
            {/* 頂部標題與裝飾背景 - 置中緊湊佈局 */}
            <div className="shrink-0 py-4 px-8 flex flex-col items-center justify-center border-b border-border bg-muted/20">
                <div className="flex items-center gap-3">
                    <div className="w-10 h-10 bg-primary/10 rounded-xl flex items-center justify-center text-primary shadow-sm shrink-0">
                        <Package size={40} />
                    </div>
                    <div className="flex flex-col items-left gap-0.5">
                        <div className="flex items-center gap-2">
                            <h1 className="text-xl font-bold tracking-tight">BOMIX</h1>
                            <span className="text-[12px] text-muted-foreground bg-background/50 px-1.5 py-0.5 rounded-full border border-border/50">
                                v {version}
                            </span>
                        </div>
                        <p className="text-xs text-muted-foreground">
                            電子 BOM 版本變化管理與追蹤工具
                        </p>
                    </div>
                </div>
            </div>

            {/* 可滾動的內容區塊：使用原生 overflow-y-auto 確保卷軸出現 */}
            <div className="flex-1 overflow-y-auto px-8 py-6">
                <div className="max-w-3xl mx-auto space-y-10">
                    
                    {/* 系統資訊區塊 */}
                    <section>
                        <h2 className="text-lg font-semibold mb-4">系統資訊</h2>
                        <div className="border border-border rounded-md overflow-hidden bg-card text-card-foreground">
                            {[
                                { label: 'BOMIX 版本', value: version },
                                { label: 'Electron', value: versions.electron || '未知' },
                                { label: 'Chrome', value: versions.chrome || '未知' },
                                { label: 'Node.js', value: versions.node || '未知' },
                            ].map((row, i, arr) => (
                                <div key={row.label}>
                                    <div className="flex justify-between px-4 py-3 text-sm">
                                        <span className="text-muted-foreground">{row.label}</span>
                                        <span className="font-mono">{row.value}</span>
                                    </div>
                                    {i < arr.length - 1 && <Separator />}
                                </div>
                            ))}
                        </div>
                        <div className="mt-4 text-xs text-muted-foreground text-center">
                            © {new Date().getFullYear()} BOMIX Team. All rights reserved.
                        </div>
                    </section>

                    {/* 更新紀錄區塊 */}
                    <section>
                        <h2 className="text-lg font-semibold mb-4">更新紀錄</h2>
                        <div className="bg-card text-card-foreground border border-border rounded-md p-6">
                            {changelogLoading ? (
                                <div className="flex items-center justify-center py-10 text-muted-foreground text-sm">
                                    載入中...
                                </div>
                            ) : changelogError ? (
                                <div className="flex items-center justify-center py-10 text-destructive text-sm">
                                    {changelogError}
                                </div>
                            ) : (
                                <div className="prose prose-sm dark:prose-invert max-w-none">
                                    <ReactMarkdown>{changelogContent}</ReactMarkdown>
                                </div>
                            )}
                        </div>
                    </section>
                </div>
            </div>
        </div>
    )
}

export default AboutPage
