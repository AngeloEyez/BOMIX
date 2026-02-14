import { useState } from 'react'
import AppLayout from './components/layout/AppLayout'
import Dashboard from './pages/Dashboard'
import BomPage from './pages/BomPage'
import ComparePage from './pages/ComparePage'
import SettingsPage from './pages/SettingsPage'

// ========================================
// BOMIX 主應用程式元件
// 管理頁面導航狀態與整體佈局
// ========================================

/** 所有頁面的定義，用於導航與動態渲染 */
const PAGES = [
    { id: 'dashboard', label: '儀表板', icon: '🏠', component: Dashboard },
    { id: 'bom', label: 'BOM', icon: '📊', component: BomPage },
    { id: 'compare', label: '比較', icon: '🔄', component: ComparePage },
    { id: 'settings', label: '設定', icon: '⚙️', component: SettingsPage },
]

/**
 * App 主元件。
 *
 * 維護目前的導航頁面狀態，並將頁面定義與切換函數傳給 AppLayout。
 *
 * @returns {JSX.Element} 應用程式根元件
 */
import ErrorBoundary from './components/ErrorBoundary'

// ...

function App() {
    // 預設顯示儀表板
    const [currentPage, setCurrentPage] = useState('dashboard')

    // 取得目前頁面的元件
    const ActivePage = PAGES.find(p => p.id === currentPage)?.component || Dashboard

    return (
        <ErrorBoundary>
            <AppLayout
                pages={PAGES}
                currentPage={currentPage}
                onNavigate={setCurrentPage}
            >
                <ErrorBoundary>
                    <ActivePage onNavigate={setCurrentPage} />
                </ErrorBoundary>
            </AppLayout>
        </ErrorBoundary>
    )
}

export default App
