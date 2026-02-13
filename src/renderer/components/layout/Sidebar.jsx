// ========================================
// 側邊導航列元件
// 固定於左側，顯示圖標與文字標籤
// ========================================

/**
 * 側邊導航列元件。
 *
 * 顯示所有可導航的頁面項目，高亮目前選中的頁面。
 *
 * @param {Object} props
 * @param {Array} props.pages - 頁面定義陣列（含 id, label, icon）
 * @param {string} props.currentPage - 目前選中的頁面 ID
 * @param {Function} props.onNavigate - 頁面切換回呼函數
 * @returns {JSX.Element} 側邊導航列
 */
function Sidebar({ pages, currentPage, onNavigate }) {
    return (
        <nav className="flex flex-col w-[72px] shrink-0
      bg-white dark:bg-surface-900
      border-r border-slate-200 dark:border-slate-700
      py-2">
            {pages.map((page) => {
                const isActive = page.id === currentPage
                return (
                    <button
                        key={page.id}
                        onClick={() => onNavigate(page.id)}
                        className={`
              flex flex-col items-center justify-center
              gap-0.5 py-3 mx-1 rounded-lg
              text-xs font-medium
              transition-all duration-150
              cursor-pointer
              ${isActive
                                ? 'bg-primary-100 dark:bg-primary-900/40 text-primary-700 dark:text-primary-300'
                                : 'text-slate-500 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800 hover:text-slate-700 dark:hover:text-slate-200'
                            }
            `}
                        title={page.label}
                    >
                        <span className="text-lg">{page.icon}</span>
                        <span>{page.label}</span>
                    </button>
                )
            })}
        </nav>
    )
}

export default Sidebar
