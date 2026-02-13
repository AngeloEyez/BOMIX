// ========================================
// 主佈局元件
// 包含側邊導航列、頂部標題列、主內容區域、底部狀態列
// 參考 Windows 11 Fluent Design 風格
// ========================================

import Sidebar from './Sidebar'
import StatusBar from './StatusBar'

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
    return (
        <div className="flex flex-col h-screen overflow-hidden">
            {/* --- 頂部標題列 --- */}
            <header className="flex items-center justify-between h-12 px-4
        bg-white/80 dark:bg-surface-900/80 backdrop-blur-sm
        border-b border-slate-200 dark:border-slate-700
        shrink-0">
                <div className="flex items-center gap-2">
                    <h1 className="text-sm font-semibold text-primary-600 dark:text-primary-400">
                        BOMIX
                    </h1>
                    {/* TODO: 開啟系列後顯示系列名稱 */}
                </div>
                <div className="flex items-center gap-2">
                    {/* TODO: 主題切換按鈕 */}
                </div>
            </header>

            {/* --- 主體區域（側邊欄 + 內容） --- */}
            <div className="flex flex-1 overflow-hidden">
                <Sidebar
                    pages={pages}
                    currentPage={currentPage}
                    onNavigate={onNavigate}
                />
                <main className="flex-1 overflow-auto p-6 bg-surface-50 dark:bg-surface-950">
                    {children}
                </main>
            </div>

            {/* --- 底部狀態列 --- */}
            <StatusBar />
        </div>
    )
}

export default AppLayout
