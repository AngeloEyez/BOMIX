import React, { useState, useEffect, useMemo } from 'react'
import {
    ChevronRight, ChevronDown, FileText, FolderOpen
} from 'lucide-react'
import useProjectStore from '../../stores/useProjectStore'
import useBomStore from '../../stores/useBomStore'
import useSettingsStore from '../../stores/useSettingsStore'

// ========================================
// BOM 側邊欄
// 樹狀顯示系列中所有專案與 BOM Revisions
// 支援單選/多選
// ========================================

const BomSidebar = () => {
    const { projects } = useProjectStore()
    const {
        selectedRevisionId, selectedRevisionIds, toggleRevisionSelection,
        selectProject, // Used to load revisions if not loaded?
        // Actually we need to make sure we have all revisions for tree view.
        // Dashboard loads all. BomPage only loads for selected project initially.
        // We should load all if we want a full tree.
    } = useBomStore()

    const { settings, updateSettings } = useSettingsStore()
    // Safety check for settings
    const safeSettings = settings || {}
    const width = safeSettings.bomSidebarWidth || 250
    const isCollapsed = safeSettings.isBomSidebarCollapsed

    // Local state for tree expansion
    const [expandedProjects, setExpandedProjects] = useState({})
    // Local state for BOMs cache (since BomStore might only keep current project's)
    // Actually, BomStore `revisions` is for `selectedProjectId`.
    // If we want a global tree, we need `useProjectStore` to hold revisions or fetch them here.
    // Let's fetch them here for now or assume `Dashboard` logic style.
    const [projectBoms, setProjectBoms] = useState({})

    useEffect(() => {
        // Load revisions for all projects
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

            // Auto expand projects with selected BOMs
            const newExpanded = {}
            projects.forEach(p => {
                // If any BOM in this project is selected, expand it
                const boms = newProjectBoms[p.id] || []
                if (boms.some(b => selectedRevisionIds.has(b.id))) {
                    newExpanded[p.id] = true
                }
            })
            setExpandedProjects(prev => ({ ...prev, ...newExpanded }))
        }
        loadAllRevisions()
    }, [projects, selectedRevisionIds]) // Re-run when selected IDs change to auto-expand? Maybe only initially?

    const toggleProject = (projectId) => {
        setExpandedProjects(prev => ({ ...prev, [projectId]: !prev[projectId] }))
    }

    const handleBomClick = (e, bomId) => {
        e.stopPropagation()
        // Ctrl/Cmd click for multi-select
        const isMulti = e.ctrlKey || e.metaKey
        toggleRevisionSelection(bomId, isMulti)
    }

    const handleResize = (e) => {
        const startX = e.clientX
        const startWidth = width

        const onMouseMove = (moveEvent) => {
            const newWidth = Math.max(200, Math.min(600, startWidth + (moveEvent.clientX - startX)))
            updateSettings({ bomSidebarWidth: newWidth })
        }

        const onMouseUp = () => {
            document.removeEventListener('mousemove', onMouseMove)
            document.removeEventListener('mouseup', onMouseUp)
        }

        document.addEventListener('mousemove', onMouseMove)
        document.addEventListener('mouseup', onMouseUp)
    }

    if (isCollapsed) return null // Handled by parent layout usually, or we show a small strip

    return (
        <div
            className="flex h-full border-r border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-surface-900 relative flex-shrink-0"
            style={{ width: `${width}px` }}
        >
            <div className="flex-1 overflow-y-auto p-2">
                <div className="space-y-1">
                    {projects.map(project => (
                        <div key={project.id}>
                            <div
                                className="flex items-center gap-2 px-2 py-1.5 rounded-lg cursor-pointer hover:bg-slate-100 dark:hover:bg-surface-800 transition-colors select-none text-sm font-medium text-slate-700 dark:text-slate-300"
                                onClick={() => toggleProject(project.id)}
                            >
                                {expandedProjects[project.id] ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
                                <FolderOpen size={16} className="text-sky-500" />
                                <span className="truncate">{project.project_code}</span>
                            </div>

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

            {/* Resizer */}
            <div
                className="w-1 cursor-col-resize hover:bg-primary-500/50 absolute right-0 top-0 bottom-0 z-10 transition-colors"
                onMouseDown={handleResize}
            />
        </div>
    )
}

export default BomSidebar
