import React, { useState, useCallback } from 'react'
import {
    ChevronRight, ChevronDown, FileText, FolderOpen,
    PanelLeftClose, PanelLeftOpen, ChevronsDownUp, ChevronsUpDown
} from 'lucide-react'
import useProjectStore from '../../stores/useProjectStore'
import useBomStore from '../../stores/useBomStore'
import useSettingsStore from '../../stores/useSettingsStore'

// ========================================
// BOM 側邊欄
// 樹狀顯示系列中所有專案與 BOM Revisions
// 支援單選/多選、拖曳調整寬度、收合展開
//
// 資料來源：
//   - 專案列表    → useProjectStore.projects
//   - 各專案 BOM  → useProjectStore.allBoms（由 loadProjects/loadAllBoms 統一管理）
//   不再維護本地快取，避免與 Store 資料不同步
// ========================================

/**
 * BOM 側邊欄元件。
 *
 * 提供專案與 BOM Revision 的樹狀列表，支援：
 * - 拖曳右側邊界調整寬度（放開後才持久化）
 * - 點擊收合按鈕收合至最小寬度（狀態持久化）
 * - 再次點擊展開按鈕恢復至收合前的寬度
 * - 標頭的「全部展開/收合」切換按鈕
 * - 資料由 useProjectStore.allBoms 統一管理，IMPORT_BOM 後由 Dashboard/BomPage 觸發 loadProjects 自動更新
 *
 * @returns {JSX.Element} BOM 側邊欄
 */
const BomSidebar = () => {
    // ========================================
    // Store — 直接讀取 Zustand，不維護本地快取
    // ========================================
    const { projects, allBoms } = useProjectStore()
    const { selectedRevisionIds, toggleRevisionSelection } = useBomStore()

    // 直接從 store 頂層讀取側邊欄相關設定
    const { bomSidebarWidth, isBomSidebarCollapsed, updateSettings } = useSettingsStore()
    const isCollapsed = isBomSidebarCollapsed

    // 拖曳時使用 local state，避免每次 mousemove 都觸發 IPC 儲存
    const [localWidth, setLocalWidth] = useState(bomSidebarWidth || 250)

    // 樹狀展開狀態（純 UI 狀態，不需放 Store）
    const [expandedProjects, setExpandedProjects] = useState({})

    // ========================================
    // 計算「是否所有專案都已展開」以決定切換按鈕的圖示
    // ========================================
    const allExpanded = projects.length > 0 && projects.every(p => expandedProjects[p.id])

    /**
     * 切換所有專案的展開/收合狀態。
     * 若目前所有專案均已展開，則收合全部；否則展開全部。
     */
    const toggleAllProjects = useCallback(() => {
        const next = {}
        projects.forEach(p => { next[p.id] = !allExpanded })
        setExpandedProjects(next)
    }, [projects, allExpanded])

    /**
     * 切換單一專案展開/收合狀態。
     *
     * @param {number} projectId - 專案 ID
     */
    const toggleProject = (projectId) => {
        setExpandedProjects(prev => ({ ...prev, [projectId]: !prev[projectId] }))
    }

    /**
     * 處理 BOM 點擊選取（支援 Ctrl/Cmd 多選）。
     *
     * @param {React.MouseEvent} e - 滑鼠事件
     * @param {number} bomId - BOM Revision ID
     */
    const handleBomClick = (e, bomId) => {
        e.stopPropagation()
        const isMulti = e.ctrlKey || e.metaKey
        toggleRevisionSelection(bomId, isMulti)
    }

    /**
     * 處理側邊欄寬度拖曳調整。
     *
     * 拖曳過程中只更新 local state（不觸發 IPC），
     * 放開滑鼠後才呼叫 updateSettings 持久化最終寬度。
     *
     * @param {React.MouseEvent} e - 滑鼠按下事件
     */
    const handleResize = (e) => {
        e.preventDefault()
        const startX = e.clientX
        const startWidth = localWidth

        /** 拖曳移動：即時更新 local state，不觸發 IPC。 */
        const onMouseMove = (moveEvent) => {
            const newWidth = Math.max(120, Math.min(600, startWidth + (moveEvent.clientX - startX)))
            setLocalWidth(newWidth)
        }

        /** 放開滑鼠：持久化最終寬度至設定。 */
        const onMouseUp = (upEvent) => {
            document.removeEventListener('mousemove', onMouseMove)
            document.removeEventListener('mouseup', onMouseUp)
            document.body.style.cursor = ''
            const finalWidth = Math.max(120, Math.min(600, startWidth + (upEvent.clientX - startX)))
            setLocalWidth(finalWidth)
            updateSettings({ bomSidebarWidth: finalWidth })
        }

        document.addEventListener('mousemove', onMouseMove)
        document.addEventListener('mouseup', onMouseUp)
        document.body.style.cursor = 'col-resize'
    }

    /**
     * 切換側邊欄收合/展開狀態並立即持久化。
     */
    const toggleCollapse = () => {
        updateSettings({ isBomSidebarCollapsed: !isCollapsed })
    }

    // ========================================
    // 收合狀態：僅顯示展開按鈕
    // ========================================
    if (isCollapsed) {
        return (
            <div className="h-full border-r border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-surface-900 flex flex-col items-center py-2 w-9 flex-shrink-0">
                <button
                    onClick={toggleCollapse}
                    className="p-1.5 text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-200 dark:hover:bg-slate-700 rounded transition-colors"
                    title="展開側邊欄"
                >
                    <PanelLeftOpen size={16} />
                </button>
            </div>
        )
    }

    // ========================================
    // 展開狀態：完整側邊欄
    // ========================================
    return (
        <div
            className="flex flex-col h-full border-r border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-surface-900 relative flex-shrink-0"
            style={{ width: `${localWidth}px` }}
        >
            {/* 標頭 */}
            <div className="flex items-center justify-between px-2 py-1 border-b border-slate-200 dark:border-slate-700">
                {/* 全部展開/收合切換按鈕 */}
                <button
                    onClick={toggleAllProjects}
                    className="flex items-center gap-1 px-1 py-0.5 rounded text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-200 dark:hover:bg-slate-700 transition-colors"
                    title={allExpanded ? '收合所有專案' : '展開所有專案'}
                >
                    {allExpanded
                        ? <ChevronsDownUp size={13} strokeWidth={2} />
                        : <ChevronsUpDown size={13} strokeWidth={2} />
                    }

                </button>

                {/* 收合側邊欄按鈕 */}
                <button
                    onClick={toggleCollapse}
                    className="p-1 text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-200 dark:hover:bg-slate-700 rounded transition-colors"
                    title="收合側邊欄"
                >
                    <PanelLeftClose size={14} />
                </button>
            </div>

            {/* 列表容器（緊湊） */}
            <div className="flex-1 overflow-y-auto pt-1 pb-2">
                <div className="space-y-0">
                    {projects.map(project => {
                        const boms = allBoms[project.id] || []
                        return (
                            <div key={project.id}>
                                {/* 專案列 */}
                                <div
                                    className="flex items-center gap-1.5 px-2 py-0.5 hover:bg-slate-200/60 dark:hover:bg-surface-800 transition-colors cursor-pointer select-none text-xs font-semibold text-slate-600 dark:text-slate-300"
                                    onClick={() => toggleProject(project.id)}
                                >
                                    <span className="text-slate-400 flex-shrink-0">
                                        {expandedProjects[project.id]
                                            ? <ChevronDown size={12} strokeWidth={2.5} />
                                            : <ChevronRight size={12} strokeWidth={2.5} />
                                        }
                                    </span>
                                    <FolderOpen size={14} className="text-sky-500 flex-shrink-0" />
                                    <span className="truncate">{project.project_code}</span>
                                </div>

                                {/* BOM Revision 列表（展開時顯示） */}
                                {expandedProjects[project.id] && (
                                    <div className="ml-3.5 pl-2 border-l border-slate-200/60 dark:border-slate-700/60 py-0.5 space-y-0">
                                        {boms.length > 0
                                            ? boms.map(bom => {
                                                const isSelected = selectedRevisionIds.has(bom.id)
                                                return (
                                                    <div
                                                        key={bom.id}
                                                        onClick={(e) => handleBomClick(e, bom.id)}
                                                        className={`flex items-center gap-1.5 px-2 py-0.5 rounded-sm cursor-pointer select-none text-[11px] transition-colors
                                                            ${isSelected
                                                                ? 'bg-primary-500/10 text-primary-600 dark:text-primary-400 font-medium'
                                                                : 'text-slate-500 dark:text-slate-400 hover:bg-slate-200/50 dark:hover:bg-surface-800/80 hover:text-slate-700 dark:hover:text-slate-200'
                                                            }`}
                                                    >
                                                        <FileText size={12} className={isSelected ? 'text-primary-500' : 'text-slate-400'} />
                                                        <span className="truncate">
                                                            {bom.phase_name}-{bom.version}{bom.suffix ? `-${bom.suffix}` : ''}
                                                        </span>
                                                    </div>
                                                )
                                            })
                                            : (
                                                <div className="text-[10px] text-slate-400 italic px-2 py-0.5">
                                                    尚無 BOM 版本
                                                </div>
                                            )
                                        }
                                    </div>
                                )}
                            </div>
                        )
                    })}
                </div>
            </div>

            {/* 右側拖曳調整寬度的把手 */}
            <div
                className="w-1 cursor-col-resize hover:bg-primary-500/30 absolute right-0 top-0 bottom-0 z-10 transition-colors"
                onMouseDown={handleResize}
            />
        </div>
    )
}

export default BomSidebar
