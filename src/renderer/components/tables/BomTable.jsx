import { useMemo, useRef, useState, useEffect } from 'react'
import {
    useReactTable,
    getCoreRowModel,
    flexRender,
} from '@tanstack/react-table'
import { useVirtualizer } from '@tanstack/react-virtual'
import { 
    ChevronsUpDown, // All Expanded (or toggle)
    ChevronsDown,   // All Collapsed (Expand All)
    ListMinus,       // Partical
    ChevronRight, ChevronDown as ChevronDownIcon
} from 'lucide-react'

// ========================================
// 搜尋高亮輔助元件
// ========================================
const HighlightText = ({ text, term }) => {
    if (!term || !text) return text
    
    const parts = String(text).split(new RegExp(`(${term})`, 'gi'))
    return (
        <span>
            {parts.map((part, i) => 
                part.toLowerCase() === term.toLowerCase() ? (
                    <span key={i} className="bg-yellow-200 text-slate-800 dark:bg-yellow-600/50 dark:text-white rounded px-0.5">{part}</span>
                ) : (
                    part
                )
            )}
        </span>
    )
}

function BomTable({ data, isLoading, searchTerm }) {
    // 收合狀態
    const [expandedGroups, setExpandedGroups] = useState(new Set())
    const [sorting, setSorting] = useState([])

    // 取得所有具有 2nd Source 的 Group Key
    const allGroupKeys = useMemo(() => {
        if (!data) return new Set()
        const keys = new Set()
        data.forEach(item => {
            if (item.second_sources && item.second_sources.length > 0) {
                keys.add(`${item.supplier}|${item.supplier_pn}`)
            }
        })
        return keys
    }, [data])

    // 初始化：預設全部展開
    useEffect(() => {
        setExpandedGroups(allGroupKeys)
    }, [allGroupKeys])

    // 衍生狀態
    const totalGroups = allGroupKeys.size
    const expandedCount = expandedGroups.size
    
    let expandState = 'ALL_EXPANDED' // default
    if (totalGroups > 0) {
        if (expandedCount === 0) expandState = 'ALL_COLLAPSED'
        else if (expandedCount < totalGroups) expandState = 'PARTIAL'
    }

    const toggleGroup = (key) => {
        setExpandedGroups(prev => {
            const next = new Set(prev)
            if (next.has(key)) next.delete(key)
            else next.add(key)
            return next
        })
    }

    const toggleAll = () => {
        if (expandState === 'ALL_EXPANDED') {
            // Collapse All
            setExpandedGroups(new Set())
        } else {
            // Expand All (for both Collapsed and Partial)
            setExpandedGroups(allGroupKeys)
        }
    }

    // 將聚合資料展開為平面行列
    const flatRows = useMemo(() => {
        if (isLoading) return [] 
        
        let processedData = [...data]

        // 1. 執行排序 (僅針對 Main Items)
        if (sorting.length > 0) {
            const { id, desc } = sorting[0]
            processedData.sort((a, b) => {
                let valA = a[id]
                let valB = b[id]
                if (valA === null || valA === undefined) valA = ''
                if (valB === null || valB === undefined) valB = ''
                if (typeof valA === 'string') valA = valA.toLowerCase()
                if (typeof valB === 'string') valB = valB.toLowerCase()

                if (valA < valB) return desc ? 1 : -1
                if (valA > valB) return desc ? -1 : 1
                return 0
            })
        }

        // 2. 展開為平面結構
        const rows = []
        let groupIndex = 0
        processedData.forEach((mainItem) => {
            const key = `${mainItem.supplier}|${mainItem.supplier_pn}`
            const hasSecondSources = mainItem.second_sources?.length > 0
            const isExpanded = expandedGroups.has(key)
            
            // Main Item 行
            rows.push({
                ...mainItem,
                _rowType: 'main',
                _groupIndex: groupIndex,
                _key: key,
                _hasSecondSources: hasSecondSources,
                _isExpanded: isExpanded,
            })
            
            // 2nd Source 行
            if (hasSecondSources && isExpanded) {
                mainItem.second_sources.forEach((ss) => {
                    rows.push({
                        ...ss,
                        _rowType: 'second',
                        _groupIndex: groupIndex,
                        bom_status: mainItem.bom_status,
                        type: mainItem.type,
                        locations: '',
                        quantity: '',
                        ccl: '',
                        remark: '',
                    })
                })
            }
            groupIndex++
        })
        return rows
    }, [data, isLoading, sorting, expandedGroups])

    // 欄位定義
    const columns = useMemo(() => [
        {
            id: 'rowIndicator',
            header: () => {
                // Determine Icon based on expandState
                let Icon = ChevronsUpDown // Default (ALL_EXPANDED) logic

                
                if (expandState === 'ALL_EXPANDED') Icon = ChevronsUpDown
                else if (expandState === 'ALL_COLLAPSED') Icon = ChevronsDown
                else if (expandState === 'PARTIAL') Icon = ListMinus

                return (
                    <div 
                        onClick={(e) => { e.stopPropagation(); toggleAll(); }}
                        className="cursor-pointer hover:text-primary-600 select-none flex items-center justify-center w-full h-full p-1"
                        title={expandState === 'ALL_EXPANDED' ? "點擊全部收合" : "點擊全部展開"}
                    >
                        <Icon size={16} className="text-slate-500" />
                    </div>
                )
            },
            size: 60,
            cell: ({ row }) => {
                const r = row.original
                if (r._rowType === 'main') {
                    return (
                        <div 
                            className="flex items-center gap-1 cursor-pointer select-none pl-1"
                            onClick={(e) => {
                                if (r._hasSecondSources) {
                                    e.stopPropagation()
                                    toggleGroup(r._key)
                                }
                            }}
                        >
                             {/* Text: Main. No emphasis color (slate-600). */}
                            <span className="text-[10px] text-slate-600 font-bold" title="Main Source">Main</span>
                            
                            {/* Icon: Show if has 2nd sources */}
                            {r._hasSecondSources && (
                                <span className="text-slate-400">
                                    {r._isExpanded ? <ChevronDownIcon size={12}/> : <ChevronRight size={12}/>}
                                </span>
                            )}
                        </div>
                    )
                }
                return null
            },
        },
        {
            accessorKey: 'hhpn',
            header: 'HHPN',
            cell: ({ getValue }) => <HighlightText text={getValue()} term={searchTerm} />
        },
        {
            accessorKey: 'description',
            header: 'Description',
            size: 500, // Large logical size to encourage taking space
            cell: ({ getValue }) => (
                <span className="truncate block w-full" title={getValue()}>
                    <HighlightText text={getValue()} term={searchTerm} />
                </span>
            ),
        },
        {
            accessorKey: 'supplier',
            header: 'Supplier',
            cell: ({ getValue }) => <HighlightText text={getValue()} term={searchTerm} />
        },
        {
            accessorKey: 'supplier_pn',
            header: 'Supplier PN',
            cell: ({ getValue }) => <HighlightText text={getValue()} term={searchTerm} />
        },
        {
            accessorKey: 'locations',
            header: 'Location',
            size: 200, 
            cell: ({ row }) => {
                if (row.original._rowType === 'second') return ''
                const loc = row.original.locations || ''
                return (
                    <span className="truncate block" title={loc}>
                        <HighlightText text={loc} term={searchTerm} />
                    </span>
                )
            },
        },
        {
            accessorKey: 'quantity',
            header: 'Qty',
            size: 40,
            cell: ({ row }) => {
                if (row.original._rowType === 'second') return ''
                return row.original.quantity
            },
        },
        {
            accessorKey: 'bom_status',
            header: 'BOM',
            size: 40,
            cell: ({ row }) => {
                if (row.original._rowType === 'second') return ''
                const s = row.original.bom_status
                const colorMap = { I: 'text-green-600', X: 'text-red-500', P: 'text-amber-600', M: 'text-blue-600' }
                return <span className={`font-semibold ${colorMap[s] || ''}`}>{s}</span>
            },
        },
        {
            accessorKey: 'type',
            header: 'Type',
            size: 60,
            cell: ({ row }) => {
                if (row.original._rowType === 'second') return ''
                return row.original.type || ''
            },
        },
        {
            accessorKey: 'ccl',
            header: 'CCL',
            size: 35,
            cell: ({ row }) => {
                if (row.original._rowType === 'second') return ''
                const c = row.original.ccl
                if (c === 'Y') return <span className="text-red-500 font-bold">Y</span>
                return c || ''
            },
        },
        {
            accessorKey: 'remark',
            header: 'Remark',
            size: 100,
            cell: ({ row }) => {
                if (row.original._rowType === 'second') return ''
                const remark = row.original.remark || ''
                return (
                    <span className="truncate block" title={remark}>
                        <HighlightText text={remark} term={searchTerm} />
                    </span>
                )
            },
        },
    ], [expandState, toggleAll, toggleGroup, searchTerm])

    // TanStack Table 實例
    const table = useReactTable({
        data: flatRows,
        columns,
        getCoreRowModel: getCoreRowModel(),
        state: {
            sorting,
        },
        onSortingChange: setSorting,
        manualSorting: true, // We sort the data manually in useMemo to preserve grouping
    })
    
    // 虛擬捲動
    const tableContainerRef = useRef(null)
    const rowVirtualizer = useVirtualizer({
        count: table.getRowModel().rows.length,
        getScrollElement: () => tableContainerRef.current,
        estimateSize: () => 35, // 預估行高
        overscan: 10,
    })

    const virtualItems = rowVirtualizer.getVirtualItems()
    const totalSize = rowVirtualizer.getTotalSize()

    const paddingTop = virtualItems.length > 0 ? virtualItems[0].start : 0
    const paddingBottom = virtualItems.length > 0 ? totalSize - (virtualItems[virtualItems.length - 1]?.end || 0) : 0

    // ========================================
    // 渲染：無資料提示 (僅在非載入中且無資料時顯示)
    // ========================================
    if (!isLoading && data.length === 0) {
        return (
            <div className="flex items-center justify-center h-64 text-sm text-slate-400 dark:text-slate-500">
                無資料 ... 
            </div>
        )
    }

    // ========================================
    // 渲染：表格
    // ========================================
    return (
        <div ref={tableContainerRef} className="h-full overflow-auto border border-slate-200 dark:border-slate-700 rounded-lg bg-white dark:bg-surface-800 relative">
            <table className="w-full text-xs border-collapse relative">
                {/* 表頭 (Sticky) */}
                <thead className="bg-bom-header-bg sticky top-0 z-10 shadow-sm">
                    {table.getHeaderGroups().map((headerGroup) => (
                        <tr key={headerGroup.id}>
                            {headerGroup.headers.map((header) => {
                                const isSorted = header.column.getIsSorted()
                                return (
                                    <th
                                        key={header.id}
                                        onClick={header.column.getToggleSortingHandler()}
                                        className={`text-left text-[11px] font-semibold text-bom-header-text
                                            uppercase tracking-wider py-1.5 px-2 border-b border-slate-200 dark:border-slate-600
                                            whitespace-nowrap select-none bg-bom-header-bg
                                            ${header.column.getCanSort() ? 'cursor-pointer hover:text-slate-700 dark:hover:text-slate-200' : ''}`}
                                        style={{ width: header.getSize() }}
                                    >
                                        <div className="flex items-center gap-1">
                                            {flexRender(header.column.columnDef.header, header.getContext())}
                                            {{
                                                asc: ' 🔼',
                                                desc: ' 🔽',
                                            }[isSorted] ?? null}
                                        </div>
                                    </th>
                                )
                            })}
                        </tr>
                    ))}
                </thead>

                {/* 表身 */}
                <tbody>
                    {isLoading ? (
                        // 骨架屏 (Skeleton Rows)
                        Array.from({ length: 20 }).map((_, idx) => (
                            <tr key={idx} className="border-b border-slate-100 dark:border-slate-700/50">
                                {columns.map((col, cIdx) => (
                                    <td key={cIdx} className="py-2 px-2">
                                        <div className={`h-3 rounded bg-slate-200 dark:bg-surface-600 animate-pulse ${
                                            cIdx === 4 ? 'w-32' : cIdx === 6 ? 'w-24' : 'w-full'
                                        }`} />
                                    </td>
                                ))}
                            </tr>
                        ))
                    ) : (
                        // 虛擬化資料列
                        <>
                            {paddingTop > 0 && (
                                <tr>
                                    <td colSpan={columns.length} style={{ height: `${paddingTop}px` }} />
                                </tr>
                            )}
                            {virtualItems.map((virtualRow) => {
                                const row = table.getRowModel().rows[virtualRow.index]
                                const r = row.original
                                const isSecond = r._rowType === 'second'
                                const isEvenGroup = r._groupIndex % 2 === 0

                                let rowClass = ''
                                // 採用 Semantic Class (定義於 index.css & theme 檔案)
                                if (isEvenGroup) {
                                    // 偶數群組
                                    rowClass = isSecond
                                        ? 'bg-bom-row-second-even text-bom-text-second'
                                        : 'bg-bom-row-main-even text-bom-text-main'
                                } else {
                                    // 奇數群組
                                    rowClass = isSecond
                                        ? 'bg-bom-row-second-odd text-bom-text-second'
                                        : 'bg-bom-row-main-odd text-bom-text-main'
                                }

                                return (
                                    <tr
                                        key={row.id}
                                        data-index={virtualRow.index}
                                        ref={rowVirtualizer.measureElement}
                                        className={`${rowClass} border-b border-slate-100 dark:border-slate-700/50
                                            hover:bg-bom-row-hover transition-colors`}
                                    >
                                        {row.getVisibleCells().map((cell) => (
                                            <td
                                                key={cell.id}
                                                className={`py-1 px-2
                                                    ${isSecond ? 'italic' : ''}
                                                    whitespace-nowrap overflow-hidden`}
                                                style={{ maxWidth: cell.column.getSize() }}
                                            >
                                                {flexRender(cell.column.columnDef.cell, cell.getContext())}
                                            </td>
                                        ))}
                                    </tr>
                                )
                            })}
                            {paddingBottom > 0 && (
                                <tr>
                                    <td colSpan={columns.length} style={{ height: `${paddingBottom}px` }} />
                                </tr>
                            )}
                        </>
                    )}
                </tbody>
            </table>
        </div>
    )
}

export default BomTable
