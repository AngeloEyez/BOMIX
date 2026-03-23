import { useState, useRef, useEffect, useMemo } from 'react'
import {
    Download, Settings, X, RotateCcw, ChevronDown
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Checkbox } from "@/components/ui/checkbox"

const DEFAULT_SEARCH_FIELDS = ['hhpn', 'description', 'supplier', 'supplier_pn', 'location']
const ALL_SEARCH_FIELDS = ['hhpn', 'description', 'supplier', 'supplier_pn', 'location', 'remark']

const FIELD_LABELS = {
    hhpn: 'HHPN',
    description: 'Description',
    supplier: 'Supplier',
    supplier_pn: 'Supplier PN',
    location: 'Location',
    remark: 'Remark'
}

/**
 * 通用的 BOM 工具列
 */
export default function BomToolbar({
    headerTitle,
    selectionCount,
    bomMode,
    views,
    currentViewId,
    selectView,
    cclFilter,
    setCclFilter,
    searchTerm,
    setSearchTerm,
    searchFields,
    setSearchFields,
    isExporting,
    onExport,
    onOpenMatrixSettings,
    error,
    clearError
}) {
    const [isSearchOptionsOpen, setIsSearchOptionsOpen] = useState(false)
    const searchOptionsPanelRef = useRef(null)
    const searchToggleRef = useRef(null)

    // Handle click outside to close search options
    useEffect(() => {
        const handleClickOutside = (event) => {
            if (isSearchOptionsOpen &&
                searchOptionsPanelRef.current &&
                !searchOptionsPanelRef.current.contains(event.target) &&
                searchToggleRef.current &&
                !searchToggleRef.current.contains(event.target)) {
                setIsSearchOptionsOpen(false)
            }
        }

        document.addEventListener('mousedown', handleClickOutside)
        return () => {
            document.removeEventListener('mousedown', handleClickOutside)
        }
    }, [isSearchOptionsOpen])

    return (
        <div className="flex flex-col gap-2">
            <div className="flex items-center gap-2 flex-wrap bg-background p-2 rounded-lg shadow-sm border border-border">
                {/* 標題/狀態 */}
                <div className="flex items-center gap-2 px-2 mr-2 border-r border-border min-h-[32px]">
                    {headerTitle}
                </div>

                {/* 視圖切換 (View Switcher) */}
                {selectionCount > 0 && views && (
                    <div className="flex items-center bg-muted rounded-md p-0.5">
                        {['ALL', 'SMD', 'PTH', 'BOTTOM'].map(key => {
                            const view = views[key]
                            if (!view) return null
                            const isActive = currentViewId === view.id
                            return (
                                <button
                                    key={key}
                                    onClick={() => selectView(view.id)}
                                    className={`px-2.5 py-1 text-xs font-medium rounded transition-all
                                        ${isActive
                                            ? 'bg-background text-primary shadow-sm'
                                            : 'text-muted-foreground hover:text-foreground'
                                        }`}
                                >
                                    {key}
                                </button>
                            )
                        })}
                    </div>
                )}

                {/* Matrix 設定按鈕 (Only in Matrix Mode) */}
                {selectionCount > 0 && bomMode === 'MATRIX' && (
                    <Button
                        variant="ghost"
                        size="icon"
                        className="h-7 w-7 text-muted-foreground hover:text-indigo-500"
                        onClick={onOpenMatrixSettings}
                        title="管理 Matrix Models"
                    >
                        <Settings size={15} />
                    </Button>
                )}

                {/* CCL Filter Checkbox */}
                {selectionCount > 0 && (() => {
                    const isForced = bomMode === 'MATRIX' || currentViewId === 'ccl_view'
                    const isChecked = isForced || cclFilter

                    return (
                        <label
                            className={`flex items-center gap-1.5 px-2 py-1 text-xs rounded border transition-colors select-none
                                ${
                                    isForced
                                        ? 'bg-primary/10 text-primary border-primary/30 cursor-not-allowed opacity-70'
                                        : isChecked
                                            ? 'bg-primary/10 text-primary border-primary/30 cursor-pointer hover:bg-primary/15'
                                            : 'text-muted-foreground border-border cursor-pointer hover:text-foreground hover:bg-muted'
                                }`}
                            title={isForced ? 'CCL Filter 在此模式下強制啟用' : '切換 CCL Filter'}
                        >
                            <Checkbox
                                checked={isChecked}
                                disabled={isForced}
                                onCheckedChange={(checked) => {
                                    if (!isForced) setCclFilter(checked)
                                }}
                                className="data-[state=checked]:bg-primary data-[state=checked]:text-primary-foreground border-primary/50"
                            />
                            <span className="font-medium">CCL</span>
                        </label>
                    )
                })()}

                {/* 搜尋框 (Search) */}
                {selectionCount > 0 && (
                    <div className="relative ml-auto flex flex-col">
                        <div className="relative">
                            <input
                                type="text"
                                value={searchTerm}
                                onChange={(e) => setSearchTerm(e.target.value)}
                                onDoubleClick={() => setIsSearchOptionsOpen(prev => !prev)}
                                onFocus={() => setIsSearchOptionsOpen(false)}
                                onKeyDown={(e) => {
                                    if (e.key === 'Escape') {
                                        setSearchTerm('')
                                        setIsSearchOptionsOpen(false)
                                    }
                                }}
                                placeholder="Search..."
                                className={`pl-3 pr-14 py-1.5 text-xs h-8 transition-all
                                    bg-background border rounded-md
                                    text-foreground placeholder:text-muted-foreground
                                    focus:outline-none focus:ring-1 focus:ring-ring
                                    ${searchFields.size !== DEFAULT_SEARCH_FIELDS.length ? 'border-amber-400' : 'border-input'}
                                    ${isSearchOptionsOpen ? 'w-72' : 'focus:w-56 w-36'}`}
                            />

                            <div className="absolute right-2 top-1/2 -translate-y-1/2 flex items-center gap-1">
                                {searchTerm && (
                                    <button
                                        onClick={() => setSearchTerm('')}
                                        className="text-muted-foreground hover:text-foreground"
                                        title="Clear search"
                                    >
                                        <X size={12} />
                                    </button>
                                )}
                                {searchFields.size !== DEFAULT_SEARCH_FIELDS.length && (
                                    <button
                                        onClick={() => setSearchFields(new Set(DEFAULT_SEARCH_FIELDS))}
                                        className="text-amber-500 hover:text-amber-600"
                                        title="Reset search fields"
                                    >
                                        <RotateCcw size={12} />
                                    </button>
                                )}
                                <button
                                    ref={searchToggleRef}
                                    onClick={() => setIsSearchOptionsOpen(prev => !prev)}
                                    className={`text-muted-foreground hover:text-foreground transition-transform ${isSearchOptionsOpen ? 'rotate-180' : ''}`}
                                    title="Toggle search options"
                                >
                                    <ChevronDown size={13} />
                                </button>
                            </div>
                        </div>

                        {/* Search Options Panel */}
                        {isSearchOptionsOpen && (
                            <div
                                ref={searchOptionsPanelRef}
                                className="absolute top-full right-0 mt-1 w-72 p-2
                                    bg-background rounded-lg shadow-xl border border-border
                                    z-50 flex flex-wrap gap-1.5 animate-in fade-in slide-in-from-top-1"
                            >
                                {ALL_SEARCH_FIELDS.map(field => {
                                    const isSelected = searchFields.has(field)
                                    return (
                                        <button
                                            key={field}
                                            onClick={() => {
                                                setSearchFields(prev => {
                                                    const next = new Set(prev)
                                                    if (next.has(field)) next.delete(field)
                                                    else next.add(field)
                                                    if (next.size === 0) return prev
                                                    return next
                                                })
                                            }}
                                            onDoubleClick={(e) => {
                                                e.stopPropagation()
                                                setSearchFields(new Set([field]))
                                            }}
                                            className={`px-2 py-0.5 text-[11px] rounded border transition-colors select-none
                                                ${isSelected
                                                    ? 'bg-primary/10 text-primary border-primary/30 font-medium'
                                                    : 'bg-muted text-muted-foreground border-border hover:bg-muted/70'
                                                }`}
                                            title="Click to toggle, Double-click to select only this"
                                        >
                                            {FIELD_LABELS[field]}
                                        </button>
                                    )
                                })}
                            </div>
                        )}
                    </div>
                )}

                {/* 分隔線 */}
                <div className="h-5 w-px bg-border mx-1" />

                {/* 匯出按鈕 */}
                <Button
                    onClick={onExport}
                    disabled={selectionCount === 0 || isExporting}
                    title={isExporting ? '匯出中...' : '匯出 Excel BOM'}
                >
                    <Download size={13} className={isExporting ? 'animate-bounce' : ''} />
                    {isExporting ? '匯出中...' : '匯出'}
                </Button>
            </div>

            {/* 錯誤提示 */}
            {error && (
                <div className="flex items-center gap-2 px-3 py-1.5 bg-destructive/10
                    border border-destructive/30 rounded-lg text-xs text-destructive">
                    <span>{error}</span>
                    <button onClick={clearError} className="ml-auto hover:opacity-70"><X size={12} /></button>
                </div>
            )}
        </div>
    )
}
