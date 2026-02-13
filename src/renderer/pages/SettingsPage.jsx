// ========================================
// 設定頁面
// 提供主題切換、About 對話框、Change Log 等功能
// ========================================

/**
 * 設定頁面元件。
 *
 * 提供應用程式設定入口，包含主題切換與相關資訊。
 *
 * @returns {JSX.Element} 設定頁面
 */
function SettingsPage() {
    return (
        <div className="max-w-2xl mx-auto">
            <h2 className="text-xl font-semibold text-slate-800 dark:text-slate-200 mb-6">
                設定
            </h2>

            {/* 外觀設定 */}
            <div className="bg-white dark:bg-surface-800 rounded-xl
        border border-slate-200 dark:border-slate-700
        p-4 mb-4">
                <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-3">
                    外觀
                </h3>
                <div className="flex items-center justify-between">
                    <span className="text-sm text-slate-600 dark:text-slate-400">
                        深色模式
                    </span>
                    <span className="text-xs text-slate-400">即將推出</span>
                </div>
            </div>

            {/* 關於 */}
            <div className="bg-white dark:bg-surface-800 rounded-xl
        border border-slate-200 dark:border-slate-700
        p-4">
                <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-3">
                    關於
                </h3>
                <div className="space-y-2 text-sm text-slate-600 dark:text-slate-400">
                    <p>BOMIX — BOM 變化管理與追蹤工具</p>
                    <p>授權：MIT License</p>
                </div>
            </div>
        </div>
    )
}

export default SettingsPage
