import { useState } from 'react'
import AppLayout from './components/layout/AppLayout'
import HomePage from './pages/HomePage'
import ProjectPage from './pages/ProjectPage'
import BomPage from './pages/BomPage'
import ComparePage from './pages/ComparePage'
import SettingsPage from './pages/SettingsPage'

// ========================================
// BOMIX 主應用程式元件
// 管理頁面導航狀態與整體佈局
// ========================================

/** 所有頁面的定義，用於導航與動態渲染 */
const PAGES = [
    { id: 'home', label: '首頁', icon: '🏠', component: HomePage },
    { id: 'project', label: '專案', icon: '📁', component: ProjectPage },
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
function App() {
    // 預設顯示首頁
    const [currentPage, setCurrentPage] = useState('home')

    // 取得目前頁面的元件
    const ActivePage = PAGES.find(p => p.id === currentPage)?.component || HomePage

    return (
        <AppLayout
            pages={PAGES}
            currentPage={currentPage}
            onNavigate={setCurrentPage}
        >
            <ActivePage />
        </AppLayout>
    )
}

export default App
