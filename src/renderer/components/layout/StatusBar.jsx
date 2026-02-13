// ========================================
// 底部狀態列元件
// 顯示連線狀態、資料庫路徑、應用程式版本
// ========================================

import { useState, useEffect } from 'react'

/**
 * 底部狀態列元件。
 *
 * 顯示應用程式版本號與目前的連線狀態。
 * 後續將加入資料庫路徑與操作狀態提示。
 *
 * @returns {JSX.Element} 狀態列
 */
function StatusBar() {
    const [version, setVersion] = useState('')

    // 啟動時取得應用程式版本號
    useEffect(() => {
        if (window.api?.getVersion) {
            window.api.getVersion().then(setVersion).catch(() => setVersion('dev'))
        } else {
            setVersion('dev')
        }
    }, [])

    return (
        <footer className="flex items-center justify-between h-7 px-4
      bg-white dark:bg-surface-900
      border-t border-slate-200 dark:border-slate-700
      text-xs text-slate-500 dark:text-slate-400
      shrink-0">
            <div className="flex items-center gap-3">
                {/* 連線狀態指示 */}
                <span className="flex items-center gap-1">
                    <span className="w-2 h-2 rounded-full bg-slate-300 dark:bg-slate-600" />
                    未連線
                </span>
                {/* TODO: 開啟系列後顯示資料庫路徑 */}
            </div>
            <div>
                <span>BOMIX v{version}</span>
            </div>
        </footer>
    )
}

export default StatusBar
