import React, { useState, useCallback } from 'react'
import {
    ChevronRight, ChevronDown, FileText, FolderOpen,
    ChevronsDownUp, ChevronsUpDown
} from 'lucide-react'
import useProjectStore from '../../stores/useProjectStore'
import useBomStore from '../../stores/useBomStore'

// ========================================
// BOM 側邊欄
// 樹狀顯示系列中所有專案與 BOM Revisions
// 支援單選/多選
//
// 寬度控制與收合狀態由父層 BomPage 的 ResizablePanelGroup 統一控管。
// 本元件為充滿父容器 (h-full w-full) 的展示型元件。
// ========================================

/**
 * BOM 側邊欄元件。
 *
 * 提供專案與 BOM Revision 的樹狀列表，支援：
 * - 點擊展開/收合專案
 * - 單選/多選 BOM 版本
 * - 標頭的「全部展開/收合」切換按鈕
 * - 資料由 useProjectStore.allBoms 統一管理
 *
 * @returns {JSX.Element} BOM 側邊欄
 */
const BomSidebar = () => {
    // ========================================
    // Store — 直接讀取 Zustand，不維護本地快取
    // ========================================
    const { projects, allBoms } = useProjectStore()
    const { selectedRevisionIds, toggleRevisionSelection } = useBomStore()

    // 樹狀展開狀態（純 UI 狀態，不需放 Store）
    const [expandedProjects, setExpandedProjects] = useState({})

    /**
     * 自動展開邏輯：
     * 當 projects 列表更新時（例如匯入新專案），
     * 預設將尚未有紀錄的專案設為展開。
     */
    React.useEffect(() => {
        if (projects.length === 0) return
        setExpandedProjects(prev => {
            const next = { ...prev }
            let changed = false
            projects.forEach(p => {
                // 如果該專案 ID 還不在 expandedProjects 中，則預設為展開（true）
                if (next[p.id] === undefined) {
                    next[p.id] = true
                    changed = true
                }
            })
            return changed ? next : prev
        })
    }, [projects])

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
        const targetValue = !allExpanded
        projects.forEach(p => { next[p.id] = targetValue })
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
     * @param {Object} bom - BOM Revision 物件
     */
    const handleBomClick = (e, bom) => {
        e.stopPropagation()
        const isMulti = e.ctrlKey || e.metaKey
        toggleRevisionSelection(bom.id, isMulti, bom)
    }

    // ========================================
    // 渲染：充滿父容器 (寬度由父層 ResizablePanel 控管)
    // ========================================
    return (
        <div
            className="flex flex-col h-full w-full border-r border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-surface-900 relative"
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
                
                {/* 註：原本的收合側邊欄按鈕已移除，改由 ResizablePanel 拖曳或自動收合控制 */}
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
                                                        onClick={(e) => handleBomClick(e, bom)}
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
        </div>
    )
}

export default BomSidebar
