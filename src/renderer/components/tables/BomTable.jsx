import { useMemo } from 'react'
import {
    useReactTable,
    getCoreRowModel,
    flexRender,
} from '@tanstack/react-table'

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
 * @returns {JSX.Element}
 */
function BomTable({ data }) {
    // 將聚合資料展開為平面行列 (Main + 2nd Source 交錯)
    const flatRows = useMemo(() => {
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
                        // 2nd Source 繼承 Main 的部分欄位
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
    }, [data])

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

    // ========================================
    // 渲染：無資料提示
    // ========================================
    if (data.length === 0) {
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
        <div className="overflow-auto border border-slate-200 dark:border-slate-700 rounded-lg">
            <table className="w-full text-xs border-collapse">
                {/* 表頭 */}
                <thead className="bg-slate-100 dark:bg-surface-700 sticky top-0 z-10">
                    {table.getHeaderGroups().map((headerGroup) => (
                        <tr key={headerGroup.id}>
                            {headerGroup.headers.map((header) => (
                                <th
                                    key={header.id}
                                    className="text-left text-[11px] font-semibold text-slate-500 dark:text-slate-400
                                        uppercase tracking-wider py-1.5 px-2 border-b border-slate-200 dark:border-slate-600
                                        whitespace-nowrap select-none"
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
                    {table.getRowModel().rows.map((row) => {
                        const r = row.original
                        const isSecond = r._rowType === 'second'
                        // 條紋底色 (依群組交替)
                        const isEvenGroup = r._groupIndex % 2 === 0

                        let rowClass = ''
                        if (isSecond) {
                            // 2nd Source: 更淡的背景
                            rowClass = isEvenGroup
                                ? 'bg-sky-50/40 dark:bg-sky-900/10'
                                : 'bg-indigo-50/40 dark:bg-indigo-900/10'
                        } else {
                            // Main Item: 條紋
                            rowClass = isEvenGroup
                                ? 'bg-white dark:bg-surface-800'
                                : 'bg-slate-50/70 dark:bg-surface-800/60'
                        }

                        return (
                            <tr
                                key={row.id}
                                className={`${rowClass} border-b border-slate-100 dark:border-slate-700/50
                                    hover:bg-primary-50/50 dark:hover:bg-primary-900/10 transition-colors`}
                            >
                                {row.getVisibleCells().map((cell) => (
                                    <td
                                        key={cell.id}
                                        className={`py-1 px-2 text-slate-700 dark:text-slate-300
                                            ${isSecond ? 'text-slate-500 dark:text-slate-400 italic' : ''}
                                            whitespace-nowrap overflow-hidden`}
                                        style={{ maxWidth: cell.column.getSize() }}
                                    >
                                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                                    </td>
                                ))}
                            </tr>
                        )
                    })}
                </tbody>
            </table>
        </div>
    )
}

export default BomTable
