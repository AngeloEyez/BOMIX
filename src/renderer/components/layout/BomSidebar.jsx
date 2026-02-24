import React, { useState, useEffect } from 'react'
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
 * - 緊湊佈局：縮小空間與字體，最大化可見資訊量
 *
 * @returns {JSX.Element} BOM 側邊欄
 */
const BomSidebar = () => {
    const { projects } = useProjectStore()
    const {
        selectedRevisionIds, toggleRevisionSelection,
    } = useBomStore()

    // 直接從 store 頂層讀取側邊欄相關設定
    const { bomSidebarWidth, isBomSidebarCollapsed, updateSettings } = useSettingsStore()
    const isCollapsed = isBomSidebarCollapsed

    // 拖曳時使用 local state，避免每次 mousemove 都觸發 IPC 儲存
    // 初始化時直接讀取 store 值；拖曳放開後由 updateSettings 更新 store（不需要 Effect 反向同步）
    const [localWidth, setLocalWidth] = useState(bomSidebarWidth || 250)

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
            const newWidth = Math.max(120, Math.min(600, startWidth + (moveEvent.clientX - startX)))
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
            const finalWidth = Math.max(120, Math.min(600, startWidth + (upEvent.clientX - startX)))
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
            {/* 標頭 (緊湊) */}
            <div className="flex items-center justify-between px-2 py-1 border-b border-slate-200 dark:border-slate-700">
                <span className="text-[10px] font-bold text-slate-400 uppercase tracking-widest pl-1">
                    Projects
                </span>
                <button
                    onClick={toggleCollapse}
                    className="p-1 text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-200 dark:hover:bg-slate-700 rounded transition-colors"
                    title="收合側邊欄"
                >
                    <PanelLeftClose size={14} />
                </button>
            </div>

            {/* 列表容器 (緊湊) */}
            <div className="flex-1 overflow-y-auto pt-1 pb-2">
                <div className="space-y-0">
                    {projects.map(project => (
                        <div key={project.id}>
                            {/* 專案列 */}
                            <div
                                className="flex items-center gap-1.5 px-2 py-0.5 hover:bg-slate-200/60 dark:hover:bg-surface-800 transition-colors cursor-pointer select-none text-xs font-semibold text-slate-600 dark:text-slate-300"
                                onClick={() => toggleProject(project.id)}
                            >
                                <span className="text-slate-400 flex-shrink-0">
                                    {expandedProjects[project.id] ? <ChevronDown size={12} strokeWidth={2.5} /> : <ChevronRight size={12} strokeWidth={2.5} />}
                                </span>
                                <FolderOpen size={14} className="text-sky-500 flex-shrink-0" />
                                <span className="truncate">{project.project_code}</span>
                            </div>

                            {/* BOM Revision 列表（展開時顯示） */}
                            {expandedProjects[project.id] && (
                                <div className="ml-3.5 pl-2 border-l border-slate-200/60 dark:border-slate-700/60 py-0.5 space-y-0">
                                    {projectBoms[project.id]?.map(bom => {
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
                                    })}
                                </div>
                            )}
                        </div>
                    ))}
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
