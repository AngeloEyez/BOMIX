// ========================================
// BOM 檢視頁面（佔位）
// Phase 4 將實作 BOM 表格檢視與編輯功能
// ========================================

/**
 * BOM 檢視頁面元件。
 *
 * 目前為佔位頁面，待 Phase 4 實作 BOM 表格（使用 TanStack Table）。
 *
 * @returns {JSX.Element} BOM 檢視頁面
 */
function BomPage() {
    return (
        <div className="flex flex-col items-center justify-center h-full gap-4 text-slate-400 dark:text-slate-500">
            <span className="text-5xl">📊</span>
            <h2 className="text-xl font-semibold">BOM 檢視</h2>
            <p className="text-sm">此功能將在 Phase 4 實作</p>
        </div>
    )
}

export default BomPage
