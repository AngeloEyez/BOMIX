// ========================================
// 關於對話框 (AboutDialog)
// 顯示應用程式版本、技術資訊與授權
// ========================================

import Dialog from './Dialog'
import { Package, Github } from 'lucide-react'
import { useState, useEffect } from 'react'
import { Separator } from '@/components/ui/separator'

/**
 * 關於對話框元件。
 *
 * 顯示 BOMIX 應用程式的版本號、Electron/Node.js 技術資訊以及 GitHub 連結。
 *
 * @param {Object} props
 * @param {boolean} props.isOpen - 是否開啟
 * @param {Function} props.onClose - 關閉事件
 * @returns {JSX.Element}
 */
function AboutDialog({ isOpen, onClose }) {
    const [version, setVersion] = useState('Unknown')
    // getVersions 是同步呼叫，且 app 啟動後值不變，使用惰性初始化一次取得即可
    const [versions] = useState(() => {
        try {
            return window.api.getVersions() ?? { electron: '', chrome: '', node: '' }
        } catch (_e) {
            return { electron: '', chrome: '', node: '' }
        }
    })

    // 非同步取得應用程式版本號，僅在對話框開啟時觸發
    useEffect(() => {
        if (isOpen) {
            window.api.getVersion().then(setVersion)
        }
    }, [isOpen])

    return (
        <Dialog isOpen={isOpen} onClose={onClose} title="關於 BOMIX" modal={false}>
            <div className="flex flex-col items-center space-y-4 text-center">
                {/* App 圖標 */}
                <div className="w-14 h-14 bg-primary/10 rounded-2xl flex items-center justify-center text-primary">
                    <Package size={28} />
                </div>

                <div>
                    <h2 className="text-base font-bold text-foreground">BOMIX</h2>
                    <p className="text-xs text-muted-foreground mt-0.5">
                        電子 BOM 版本變化管理與追蹤工具
                    </p>
                </div>

                {/* 版本資訊表格 */}
                <div className="w-full border border-border rounded-md overflow-hidden">
                    {[
                        { label: '版本', value: version },
                        { label: 'Electron', value: versions.electron },
                        { label: 'Chrome', value: versions.chrome },
                        { label: 'Node.js', value: versions.node },
                    ].map((row, i, arr) => (
                        <div key={row.label}>
                            <div className="flex justify-between px-3 py-1.5 text-xs">
                                <span className="text-muted-foreground">{row.label}</span>
                                <span className="font-mono text-foreground">{row.value}</span>
                            </div>
                            {i < arr.length - 1 && <Separator />}
                        </div>
                    ))}
                </div>

                <div className="pt-1 text-xs text-muted-foreground">
                    <p>© 2026 BOMIX Team. All rights reserved.</p>
                </div>

                {/* GitHub 連結 */}
                <a
                    href="https://github.com/Start-Hero/BOMIX"
                    target="_blank"
                    rel="noreferrer"
                    className="flex items-center gap-1.5 text-xs text-primary hover:text-primary/80 transition-colors"
                >
                    <Github size={14} />
                    <span>View on GitHub</span>
                </a>
            </div>
        </Dialog>
    )
}

export default AboutDialog
