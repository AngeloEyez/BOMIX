import { useState, useEffect } from 'react'
import { LayoutDashboard, FileSpreadsheet, Grid3X3, ArrowRightLeft, Database, Settings } from 'lucide-react'
import AppLayout from './components/layout/AppLayout'
import Dashboard from './pages/Dashboard'
import BomPage from './pages/BomPage'
import ComparePage from './pages/ComparePage'
import SettingsPage from './pages/SettingsPage'
import useBomStore from './stores/useBomStore'

// ========================================
// BOMIX 主應用程式元件
// 管理頁面導航狀態與整體佈局
// ========================================

/** 所有頁面的定義，用於導航與動態渲染 */
const PAGES = [
    { id: 'dashboard', label: 'Dashboard', icon: <LayoutDashboard size={18} />, component: Dashboard },
    { id: 'bom', label: 'BOM', icon: <FileSpreadsheet size={18} />, component: BomPage },
    { id: 'matrix', label: 'Matrix', icon: <Grid3X3 size={18} />, component: BomPage },
    { id: 'bigbom', label: 'BigBOM', icon: <Database size={18} />, component: BomPage }, // Placeholder
    { id: 'compare', label: 'Compare', icon: <ArrowRightLeft size={18} />, component: ComparePage },
    { id: 'settings', label: '設定', icon: <Settings size={18} />, component: SettingsPage }, // Handled in layout
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
    const { setBomMode } = useBomStore()

    // 處理導航切換時的狀態設定
    const handleNavigate = (pageId) => {
        setCurrentPage(pageId)

        // 根據頁面設定 BomMode
        if (pageId === 'bom') setBomMode('BOM')
        else if (pageId === 'matrix') setBomMode('MATRIX')
        else if (pageId === 'bigbom') setBomMode('BIGBOM')
    }

    // 取得目前頁面的元件
    const ActivePage = PAGES.find(p => p.id === currentPage)?.component || Dashboard

    return (
        <ErrorBoundary>
            <AppLayout
                pages={PAGES}
                currentPage={currentPage}
                onNavigate={handleNavigate}
            >
                <ErrorBoundary>
                    <ActivePage onNavigate={handleNavigate} />
                </ErrorBoundary>
            </AppLayout>
        </ErrorBoundary>
    )
}

export default App
