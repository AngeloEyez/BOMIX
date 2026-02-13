// ========================================
// 首頁（歡迎頁面）
// 顯示應用程式名稱、快速操作、系列資訊
// ========================================

/**
 * 首頁元件。
 *
 * 提供快速操作入口（建立新系列、開啟系列）與最近開啟的系列列表。
 *
 * @returns {JSX.Element} 首頁
 */
function HomePage() {
    return (
        <div className="flex flex-col items-center justify-center h-full gap-8">
            {/* 應用程式標題 */}
            <div className="text-center">
                <h1 className="text-4xl font-bold text-primary-600 dark:text-primary-400 mb-2">
                    BOMIX
                </h1>
                <p className="text-lg text-slate-500 dark:text-slate-400">
                    BOM 變化管理與追蹤工具
                </p>
            </div>

            {/* 快速操作按鈕 */}
            <div className="flex gap-4">
                <button className="flex flex-col items-center gap-2 px-8 py-6
          bg-white dark:bg-surface-800 rounded-xl
          shadow-sm hover:shadow-md
          border border-slate-200 dark:border-slate-700
          hover:border-primary-300 dark:hover:border-primary-600
          transition-all duration-200 cursor-pointer
          group">
                    <span className="text-3xl group-hover:scale-110 transition-transform">📁</span>
                    <span className="text-sm font-medium text-slate-700 dark:text-slate-300">
                        建立新系列
                    </span>
                </button>

                <button className="flex flex-col items-center gap-2 px-8 py-6
          bg-white dark:bg-surface-800 rounded-xl
          shadow-sm hover:shadow-md
          border border-slate-200 dark:border-slate-700
          hover:border-primary-300 dark:hover:border-primary-600
          transition-all duration-200 cursor-pointer
          group">
                    <span className="text-3xl group-hover:scale-110 transition-transform">📂</span>
                    <span className="text-sm font-medium text-slate-700 dark:text-slate-300">
                        開啟系列
                    </span>
                </button>
            </div>

            {/* 最近開啟的系列 */}
            <div className="w-full max-w-md">
                <h2 className="text-sm font-semibold text-slate-500 dark:text-slate-400 mb-3">
                    最近開啟
                </h2>
                <div className="bg-white dark:bg-surface-800 rounded-xl
          border border-slate-200 dark:border-slate-700
          p-4 text-center text-sm text-slate-400 dark:text-slate-500">
                    尚無開啟記錄
                </div>
            </div>
        </div>
    )
}

export default HomePage
