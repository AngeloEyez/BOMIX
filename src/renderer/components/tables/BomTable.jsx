import { useMemo, useRef } from 'react'
import {
    useReactTable,
    getCoreRowModel,
    flexRender,
} from '@tanstack/react-table'
import { useVirtualizer } from '@tanstack/react-virtual'

// ========================================
// BOM 聚合表格元件 (TanStack Table)
// 以緊湊佈局顯示 Main Items 與 Second Sources
// ========================================

/**
 * BOM 聚合表格元件。
 *
 * 將聚合後的 BOM 資料以表格呈現，Main Item 與 2nd Source
 * 透過背景色與縮排區分。表格為唯讀模式。
 *
 * @param {Object} props
 * @param {Array} props.data - 聚合 BOM 資料 (含 second_sources)
 * @param {boolean} props.isLoading - 是否正在載入
 * @returns {JSX.Element}
 */
function BomTable({ data, isLoading }) {
    // 將聚合資料展開為平面行列 (Main + 2nd Source 交錯)
    const flatRows = useMemo(() => {
        if (isLoading) return [] // 載入中不處理資料
        const rows = []
        let groupIndex = 0
        data.forEach((mainItem) => {
            // Main Item 行
            rows.push({
                ...mainItem,
                _rowType: 'main',
                _groupIndex: groupIndex,
            })
            // 2nd Source 行
            if (mainItem.second_sources?.length > 0) {
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
    }, [data, isLoading])

    // 欄位定義
    const columns = useMemo(() => [
        {
            id: 'rowIndicator',
            header: '',
            size: 28,
            cell: ({ row }) => {
                const r = row.original
                if (r._rowType === 'second') {
                    return (
                        <span className="text-[10px] text-slate-400 pl-1" title="2nd Source">
                            2nd
                        </span>
                    )
                }
                return null
            },
        },
        {
            accessorKey: 'hhpn',
            header: 'HHPN',
            size: 140,
        },
        {
            accessorKey: 'supplier',
            header: 'Supplier',
            size: 100,
        },
        {
            accessorKey: 'supplier_pn',
            header: 'Supplier PN',
            size: 160,
        },
        {
            accessorKey: 'description',
            header: 'Description',
            size: 260,
            cell: ({ getValue }) => (
                <span className="truncate block" title={getValue()}>
                    {getValue()}
                </span>
            ),
        },
        {
            accessorKey: 'quantity',
            header: 'Qty',
            size: 45,
            cell: ({ row }) => {
                if (row.original._rowType === 'second') return ''
                return row.original.quantity
            },
        },
        {
            accessorKey: 'locations',
            header: 'Location',
            size: 180,
            cell: ({ row }) => {
                if (row.original._rowType === 'second') return ''
                const loc = row.original.locations || ''
                return (
                    <span className="truncate block" title={loc}>
                        {loc}
                    </span>
                )
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
            size: 65,
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
            size: 120,
            cell: ({ row }) => {
                if (row.original._rowType === 'second') return ''
                const remark = row.original.remark || ''
                return (
                    <span className="truncate block" title={remark}>
                        {remark}
                    </span>
                )
            },
        },
    ], [])

    // TanStack Table 實例
    const table = useReactTable({
        data: flatRows,
        columns,
        getCoreRowModel: getCoreRowModel(),
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
                尚無 BOM 資料。請選擇 BOM 版本，或匯入 Excel。
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
                            {headerGroup.headers.map((header) => (
                                <th
                                    key={header.id}
                                    className="text-left text-[11px] font-semibold text-bom-header-text
                                        uppercase tracking-wider py-1.5 px-2 border-b border-slate-200 dark:border-slate-600
                                        whitespace-nowrap select-none bg-bom-header-bg"
                                    style={{ width: header.getSize() }}
                                >
                                    {flexRender(header.column.columnDef.header, header.getContext())}
                                </th>
                            ))}
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
