import React, { useState, useEffect, useMemo } from 'react'
import {
    ChevronRight, ChevronDown, FileText, FolderOpen, PanelLeftClose, PanelLeftOpen
} from 'lucide-react'
import useProjectStore from '../../stores/useProjectStore'
import useBomStore from '../../stores/useBomStore'
import useSettingsStore from '../../stores/useSettingsStore'

// ========================================
// BOM 側邊欄
// 樹狀顯示系列中所有專案與 BOM Revisions
// 支援單選/多選、拖曳調整寬度、收合展開
// ========================================

/**
 * BOM 側邊欄元件。
 *
 * 提供專案與 BOM Revision 的樹狀列表，支援：
 * - 拖曳右側邊界調整寬度（放開後才持久化）
 * - 點擊收合按鈕收合至最小寬度（狀態持久化）
 * - 再次點擊展開按鈕恢復至收合前的寬度
 *
 * @returns {JSX.Element} BOM 側邊欄
 */
const BomSidebar = () => {
    const { projects } = useProjectStore()
    const {
        selectedRevisionId, selectedRevisionIds, toggleRevisionSelection,
        selectProject,
    } = useBomStore()

    // 直接從 store 頂層讀取側邊欄相關設定（非巢狀 settings 物件）
    const { bomSidebarWidth, isBomSidebarCollapsed, updateSettings } = useSettingsStore()
    const isCollapsed = isBomSidebarCollapsed

    // 拖曳時使用 local state，避免每次 mousemove 都觸發 IPC 儲存
    // 初始值從 store 讀取，後續由 store 同步（如應用程式啟動後讀取持久化設定）
    const [localWidth, setLocalWidth] = useState(bomSidebarWidth || 250)

    // 當 store 中的寬度更新時（如初始化讀取持久化設定），同步 local state
    useEffect(() => {
        setLocalWidth(bomSidebarWidth || 250)
    }, [bomSidebarWidth])

    // Local state for tree expansion
    const [expandedProjects, setExpandedProjects] = useState({})
    // Local state for BOMs cache
    const [projectBoms, setProjectBoms] = useState({})

    useEffect(() => {
        // 載入所有專案的 BOM Revisions
        const loadAllRevisions = async () => {
            if (projects.length === 0) return
            const newProjectBoms = {}
            await Promise.all(projects.map(async (p) => {
                const res = await window.api.bom.getRevisions(p.id)
                if (res.success) {
                    newProjectBoms[p.id] = res.data
                }
            }))
            setProjectBoms(newProjectBoms)

            // 自動展開含有已選取 BOM 的專案
            const newExpanded = {}
            projects.forEach(p => {
                const boms = newProjectBoms[p.id] || []
                if (boms.some(b => selectedRevisionIds.has(b.id))) {
                    newExpanded[p.id] = true
                }
            })
            setExpandedProjects(prev => ({ ...prev, ...newExpanded }))
        }
        loadAllRevisions()
    }, [projects, selectedRevisionIds])

    /**
     * 切換專案展開/收合狀態。
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
        e.preventDefault() // 防止拖曳時選取文字
        const startX = e.clientX
        const startWidth = localWidth

        /**
         * 拖曳移動：即時更新 local state 以反映視覺變化，不觸發 IPC。
         * @param {MouseEvent} moveEvent
         */
        const onMouseMove = (moveEvent) => {
            const newWidth = Math.max(200, Math.min(600, startWidth + (moveEvent.clientX - startX)))
            setLocalWidth(newWidth)
        }

        /**
         * 放開滑鼠：計算最終寬度並持久化至設定。
         * @param {MouseEvent} upEvent
         */
        const onMouseUp = (upEvent) => {
            document.removeEventListener('mousemove', onMouseMove)
            document.removeEventListener('mouseup', onMouseUp)
            document.body.style.cursor = '' // 恢復游標樣式

            // 放開後才持久化最終寬度
            const finalWidth = Math.max(200, Math.min(600, startWidth + (upEvent.clientX - startX)))
            setLocalWidth(finalWidth)
            updateSettings({ bomSidebarWidth: finalWidth })
        }

        document.addEventListener('mousemove', onMouseMove)
        document.addEventListener('mouseup', onMouseUp)
        document.body.style.cursor = 'col-resize' // 拖曳期間強制顯示調整游標
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
            <div className="h-full border-r border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-surface-900 flex flex-col items-center py-2 w-10 flex-shrink-0">
                <button
                    onClick={toggleCollapse}
                    className="p-2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-200 dark:hover:bg-slate-700 rounded transition-colors"
                    title="展開側邊欄"
                >
                    <PanelLeftOpen size={18} />
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
            {/* 標頭：顯示標題與收合按鈕 */}
            <div className="flex items-center justify-between p-2 border-b border-slate-200 dark:border-slate-700">
                <span className="text-xs font-semibold text-slate-500 uppercase tracking-wider pl-2">
                    專案列表
                </span>
                <button
                    onClick={toggleCollapse}
                    className="p-1.5 text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-200 dark:hover:bg-slate-700 rounded transition-colors"
                    title="收合側邊欄"
                >
                    <PanelLeftClose size={16} />
                </button>
            </div>

            {/* 專案樹狀列表 */}
            <div className="flex-1 overflow-y-auto p-2">
                <div className="space-y-1">
                    {projects.map(project => (
                        <div key={project.id}>
                            {/* 專案列 */}
                            <div
                                className="flex items-center gap-2 px-2 py-1.5 rounded-lg cursor-pointer hover:bg-slate-100 dark:hover:bg-surface-800 transition-colors select-none text-sm font-medium text-slate-700 dark:text-slate-300"
                                onClick={() => toggleProject(project.id)}
                            >
                                {expandedProjects[project.id] ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
                                <FolderOpen size={16} className="text-sky-500" />
                                <span className="truncate">{project.project_code}</span>
                            </div>

                            {/* BOM Revision 列表（展開時顯示） */}
                            {expandedProjects[project.id] && (
                                <div className="ml-4 pl-2 border-l border-slate-200 dark:border-slate-700 mt-1 space-y-0.5">
                                    {projectBoms[project.id]?.map(bom => {
                                        const isSelected = selectedRevisionIds.has(bom.id)
                                        return (
                                            <div
                                                key={bom.id}
                                                onClick={(e) => handleBomClick(e, bom.id)}
                                                className={`flex items-center gap-2 px-2 py-1.5 rounded-md cursor-pointer select-none text-xs transition-colors
                                                    ${isSelected
                                                        ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/40 dark:text-primary-300 font-medium'
                                                        : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-surface-800'
                                                    }`}
                                            >
                                                <FileText size={14} className={isSelected ? 'text-primary-500' : 'text-slate-400'} />
                                                <span className="truncate">
                                                    {bom.phase_name} {bom.version}
                                                    {bom.suffix ? `-${bom.suffix}` : ''}
                                                </span>
                                            </div>
                                        )
                                    })}
                                </div>
                            )}
                        </div>
                    ))}
                </div>
            </div>

            {/* 右側拖曳調整寬度的把手 */}
            <div
                className="w-1 cursor-col-resize hover:bg-primary-500/50 absolute right-0 top-0 bottom-0 z-10 transition-colors"
                onMouseDown={handleResize}
            />
        </div>
    )
}

export default BomSidebar
