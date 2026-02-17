// ========================================
// 主佈局元件
// 包含側邊導航列、頂部標題列、主內容區域、底部狀態列
// 參考 Windows 11 Fluent Design 風格
// ========================================

import { Sun, Moon, Menu, Settings } from 'lucide-react'
import useAppStore from '../../stores/useAppStore' // useAppStore has toggleTheme
import useSettingsStore from '../../stores/useSettingsStore' // SettingStore handles sidebar state etc.
import useSeriesStore from '../../stores/useSeriesStore'
import useProgressStore from '../../stores/useProgressStore'
import { useEffect } from 'react'

// Components
import AppStatusLine from './AppStatusLine'
import ProgressDialog from '../dialogs/ProgressDialog'

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
    const { isDarkMode, toggleTheme } = useAppStore() // Correct store for theme
    const { loadSettings, isLoading } = useSettingsStore()
    const { isOpen, currentPath } = useSeriesStore()
    const initProgressListeners = useProgressStore(state => state.initListeners)

    // 從路徑取得檔案名稱
    const seriesName = currentPath ? currentPath.split(/[\\/]/).pop()?.replace('.bomix', '') : null

    // 初始化設定
    useEffect(() => {
        loadSettings()
    }, [loadSettings])

    // 初始化進度監聽
    useEffect(() => {
        initProgressListeners()
    }, [initProgressListeners])

    // 更新視窗標題
    useEffect(() => {
        document.title = seriesName ? `BOMIX - ${seriesName}` : 'BOMIX'
    }, [seriesName])

    // Theme effect
    useEffect(() => {
        if (isDarkMode) {
            document.documentElement.classList.add('dark')
        } else {
            document.documentElement.classList.remove('dark')
        }
    }, [isDarkMode])

    if (isLoading) {
        return <div className="flex items-center justify-center h-screen">Loading...</div>
    }

    return (
        <div className="flex flex-col h-screen overflow-hidden bg-white dark:bg-surface-950 text-slate-900 dark:text-slate-100 transition-colors duration-200">
            {/* --- 頂部標題列 --- */}
            <header className="flex items-center justify-between h-12 px-4
        bg-white/80 dark:bg-surface-900/80 backdrop-blur-sm
        border-b border-slate-200 dark:border-slate-800
        shrink-0 z-10 app-drag-region">
                <div className="flex items-center gap-1">
                    {/* 功能導航 */}
                    {pages.filter(p => p.id !== 'settings').map((page) => {
                        const isActive = page.id === currentPage
                        return (
                            <button
                                key={page.id}
                                onClick={() => onNavigate(page.id)}
                                className={`
                                    p-2 rounded-lg transition-all duration-200 flex items-center gap-2
                                    ${isActive
                                        ? 'bg-primary-100 dark:bg-primary-900/40 text-primary-600 dark:text-primary-400 font-medium'
                                        : 'text-slate-500 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800 hover:text-slate-700 dark:hover:text-slate-200'
                                    }
                                `}
                                title={page.label}
                            >
                                {page.icon}
                                <span className="text-sm">{page.label}</span>
                            </button>
                        )
                    })}
                </div>
                <div className="flex items-center gap-2">
                    <button
                        onClick={toggleTheme}
                        className="p-2 rounded-lg hover:bg-slate-100 dark:hover:bg-surface-800 text-slate-600 dark:text-slate-400 transition-colors"
                        title={isDarkMode ? '切換至淺色模式' : '切換至深色模式'}
                    >
                        {isDarkMode ? <Sun size={18} /> : <Moon size={18} />}
                    </button>

                    {/* Settings Button Moved Here */}
                    <button
                        onClick={() => onNavigate('settings')}
                        className={`p-2 rounded-lg transition-all duration-200
                            ${currentPage === 'settings'
                                ? 'bg-primary-100 dark:bg-primary-900/40 text-primary-600 dark:text-primary-400'
                                : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-surface-800'
                            }`}
                        title="設定"
                    >
                        <Settings size={18} />
                    </button>
                </div>
            </header>

            {/* --- 主體區域 --- */}
            <main className="flex-1 overflow-hidden relative flex flex-col">
               {children}
            </main>

            {/* --- 底部狀態列 --- */}
            <AppStatusLine />

            {/* --- 全域對話框 --- */}
            <ProgressDialog />
        </div>
    )
}

export default AppLayout
