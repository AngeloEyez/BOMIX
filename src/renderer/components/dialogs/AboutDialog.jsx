import Dialog from './Dialog'
import { Package, Github } from 'lucide-react'
import { useState, useEffect } from 'react'

// ========================================
// 關於對話框 (AboutDialog)
// 顯示應用程式資訊
// ========================================

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
                <div className="w-16 h-16 bg-primary-100 dark:bg-primary-900 rounded-2xl flex items-center justify-center text-primary-600 dark:text-primary-400">
                    <Package size={32} />
                </div>
                
                <div>
                    <h2 className="text-xl font-bold text-slate-900 dark:text-white">BOMIX</h2>
                    <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
                        電子 BOM 版本變化管理與追蹤工具
                    </p>
                </div>

                <div className="flex flex-col gap-1 text-sm text-slate-600 dark:text-slate-300 bg-slate-50 dark:bg-surface-700/50 p-4 rounded-lg w-full">
                    <div className="flex justify-between">
                        <span>版本</span>
                        <span className="font-mono">{version}</span>
                    </div>
                    <div className="flex justify-between">
                        <span>Electron</span>
                        <span className="font-mono">{versions.electron}</span>
                    </div>
                    <div className="flex justify-between">
                        <span>Chrome</span>
                        <span className="font-mono">{versions.chrome}</span>
                    </div>
                    <div className="flex justify-between">
                        <span>Node.js</span>
                        <span className="font-mono">{versions.node}</span>
                    </div>
                </div>

                <div className="pt-2 text-xs text-slate-400">
                    <p>© 2026 BOMIX Team. All rights reserved.</p>
                </div>

                <a 
                    href="https://github.com/Start-Hero/BOMIX" 
                    target="_blank" 
                    rel="noreferrer"
                    className="flex items-center gap-2 text-sm text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 transition-colors"
                >
                    <Github size={16} />
                    <span>View on GitHub</span>
                </a>
            </div>
        </Dialog>
    )
}

export default AboutDialog
