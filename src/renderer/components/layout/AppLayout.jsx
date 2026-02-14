// ========================================
// 主佈局元件
// 包含側邊導航列、頂部標題列、主內容區域、底部狀態列
// 參考 Windows 11 Fluent Design 風格
// ========================================

import Sidebar from './Sidebar'
import StatusBar from './StatusBar'
import { Sun, Moon, Menu } from 'lucide-react'
import useSettingsStore from '../../stores/useSettingsStore'
import useSeriesStore from '../../stores/useSeriesStore'
import { useEffect } from 'react'

/**
 * 應用程式主佈局元件。
 *
 * 提供完整的桌面應用程式框架，包含左側導航、頂部標題、中央內容區與底部狀態列。
 *
 * @param {Object} props
 * @param {Array} props.pages - 頁面定義陣列
 * @param {string} props.currentPage - 目前選中的頁面 ID
 * @param {Function} props.onNavigate - 頁面切換回呼函數
 * @param {React.ReactNode} props.children - 頁面內容
 * @returns {JSX.Element} 主佈局
 */
function AppLayout({ pages, currentPage, onNavigate, children }) {
    const { theme, toggleTheme, initSettings, isLoading } = useSettingsStore()
    const { isOpen, currentPath } = useSeriesStore()

    // 從路徑取得檔案名稱
    const seriesName = currentPath ? currentPath.split(/[\\/]/).pop()?.replace('.bomix', '') : null

    // 初始化設定
    useEffect(() => {
        initSettings()
    }, [initSettings])

    if (isLoading) {
        return <div className="flex items-center justify-center h-screen">Loading...</div>
    }

    return (
        <div className="flex flex-col h-screen overflow-hidden bg-white dark:bg-surface-950 text-slate-900 dark:text-slate-100 transition-colors duration-200">
            {/* --- 頂部標題列 --- */}
            <header className="flex items-center justify-between h-12 px-4
        bg-white/80 dark:bg-surface-900/80 backdrop-blur-sm
        border-b border-slate-200 dark:border-slate-800
        shrink-0 z-10">
                <div className="flex items-center gap-2">
                    <h1 className="text-sm font-semibold text-primary-600 dark:text-primary-400">
                        BOMIX
                    </h1>
                    {/* 開啟系列後顯示系列名稱 */}
                    {isOpen && seriesName && (
                        <>
                            <span className="text-slate-300 dark:text-slate-600">—</span>
                            <span className="text-sm text-slate-600 dark:text-slate-400 font-medium truncate max-w-[300px]">
                                {seriesName}
                            </span>
                        </>
                    )}
                </div>
                <div className="flex items-center gap-2">
                    <button
                        onClick={toggleTheme}
                        className="p-2 rounded-lg hover:bg-slate-100 dark:hover:bg-surface-800 text-slate-600 dark:text-slate-400 transition-colors"
                        title={theme === 'light' ? '切換至深色模式' : '切換至淺色模式'}
                    >
                        {theme === 'light' ? <Moon size={18} /> : <Sun size={18} />}
                    </button>
                    <button className="p-2 rounded-lg hover:bg-slate-100 dark:hover:bg-surface-800 text-slate-600 dark:text-slate-400">
                        <Menu size={18} />
                    </button>
                </div>
            </header>

            {/* --- 主體區域（側邊欄 + 內容） --- */}
            <div className="flex flex-1 overflow-hidden">
                <Sidebar
                    pages={pages}
                    currentPage={currentPage}
                    onNavigate={onNavigate}
                />
                <main className="flex-1 overflow-auto p-6 relative">
                   {children}
                </main>
            </div>

            {/* --- 底部狀態列 --- */}
            <StatusBar />
        </div>
    )
}

export default AppLayout
