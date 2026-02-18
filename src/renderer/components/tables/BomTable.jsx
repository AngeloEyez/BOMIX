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
    ChevronRight, ChevronDown as ChevronDownIcon,
    AlertTriangle, CheckCircle2
} from 'lucide-react'
import useMatrixStore from '../../stores/useMatrixStore'
import useBomStore from '../../stores/useBomStore'

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


function BomTable(props) {
    const { data, isLoading, searchTerm, searchFields, mode, viewContextIds } = props;
    // --- Context / Store ---
    const {
        matrixData,
        saveSelection,
        deleteSelection,
        fetchMatrixData
    } = useMatrixStore()

    // Matrix Data Lookup
    // Since `matrixData` is stored by key (single ID or 'multi'), we need to access it correctly.
    // If mode is MATRIX and we have data, we assume matrixData has been fetched for current selection.
    // We'll iterate over all keys in matrixData or rely on a "current" pointer?
    // Actually, `useMatrixStore` stores data by `bomRevisionId`.
    // If multiple BOMs are selected, `matrixData` might have 'multi' key.
    // Let's assume the page triggers fetch and stores in a predictable key.
    // However, `matrixData` structure is `{ [key]: { models, selections } }`.
    // If we have dynamic columns from potentially multiple BOMs, we need to merge them.
    // For simplicity, let's assume `useMatrixStore` exposes a `currentMatrixData` selector or we use the `matrixData` directly.

    // Derived Matrix Data
    const matrixInfo = useMemo(() => {
        if (mode !== 'MATRIX') return null;

        // Aggregate models and selections from all loaded matrix data that matches current view rows?
        // Or simpler: `useMatrixStore` should have been called with the current selection IDs.
        // If single BOM: key is ID. If multi: key is 'multi' (as per my update).
        // Let's try to find the data.
        // Since I don't have the key here easily without passing it, I might iterate `matrixData` values?
        // No, `fetchMatrixData` was called with specific IDs.
        // Let's assume the parent passes `matrixKey` or `matrixData` directly?
        // Passing `matrixData` directly from Store to Component might be cleaner.
        // But `useMatrixStore` is global.

        // Hack: Check if 'multi' exists, else check single ID if data length > 0
        // Ideally `useBomStore` knows the selected IDs.
        // Let's try to grab 'multi' if it exists, else look for single.

        let data = matrixData['multi'];
        if (!data) {
            // Find any single key
            const keys = Object.keys(matrixData).filter(k => k !== 'multi');
            if (keys.length === 1) {
                data = matrixData[keys[0]];
            }
        }
        return data;
    }, [matrixData, mode]);

    const models = matrixInfo?.models || [];
    const selections = matrixInfo?.selections || [];
    const summary = matrixInfo?.summary || {};

    // Selection Map
    const selectionMap = useMemo(() => {
        const map = new Map();
        selections.forEach(s => {
            map.set(`${s.matrix_model_id}|${s.group_key}`, s);
        });
        return map;
    }, [selections]);


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

    // Matrix Handler
    const handleMatrixSelection = async (modelId, groupKey, type, id, isCurrentlySelected) => {
        // Use current view context IDs for refresh
        // Get selected IDs from BomStore? Or pass as prop?
        // Let's pass `viewContextIds` as prop or use `useBomStore`?
        // `BomTable` doesn't know about `selectedRevisionIds` unless passed.
        // Let's add `viewContextIds` prop to `BomTable`.
        const contextIds = props.viewContextIds;

        if (isCurrentlySelected) {
            await deleteSelection(contextIds, modelId, groupKey);
        } else {
            await saveSelection(contextIds, {
                matrix_model_id: modelId,
                group_key: groupKey,
                selected_type: type,
                selected_id: id
            });
        }
    };

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
                location: mainItem.locations,
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
                        location: '',
                        _rowType: 'second',
                        _groupIndex: groupIndex,
                        _key: key, // Keep key for Matrix selection mapping
                        bom_status: mainItem.bom_status,
                        type: mainItem.type,
                        locations: undefined,
                        quantity: '',
                        ccl: '',
                        remark: '',
                        // Needed for Matrix selection to identify row correctly
                        bom_revision_id: ss.bom_revision_id || mainItem.bom_revision_id,
                    })
                })
            }
            groupIndex++
        })
        return rows
    }, [data, isLoading, sorting, expandedGroups])

    // 欄位定義
    const columns = useMemo(() => {
        const baseCols = [
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
                cell: ({ getValue }) => <HighlightText text={getValue()} term={searchFields?.has('hhpn') ? searchTerm : ''} />
            },
            {
                accessorKey: 'description',
                header: 'Description',
                size: 500, // Large logical size to encourage taking space
                cell: ({ getValue }) => (
                    <span className="truncate block w-full" title={getValue()}>
                        <HighlightText text={getValue()} term={searchFields?.has('description') ? searchTerm : ''} />
                    </span>
                ),
            },
            {
                accessorKey: 'supplier',
                header: 'Supplier',
                cell: ({ getValue }) => <HighlightText text={getValue()} term={searchFields?.has('supplier') ? searchTerm : ''} />
            },
            {
                accessorKey: 'supplier_pn',
                header: 'Supplier PN',
                cell: ({ getValue }) => <HighlightText text={getValue()} term={searchFields?.has('supplier_pn') ? searchTerm : ''} />
            },
        ];

        // Standard BOM Columns
        const standardCols = [
            {
                accessorKey: 'location',
                header: 'Location',
                size: 200,
                cell: ({ row }) => {
                    if (row.original._rowType === 'second') return ''
                    const loc = row.original.location || ''
                    return (
                        <span className="truncate block" title={loc}>
                            <HighlightText text={loc} term={searchFields?.has('location') ? searchTerm : ''} />
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
                            <HighlightText text={remark} term={searchFields?.has('remark') ? searchTerm : ''} />
                        </span>
                    )
                },
            },
        ];

        // Matrix Columns (Grouped by Project)
        // Note: models now contain `bom_revision_id`. We need to group them by project.
        // But `matrixData` comes from backend and `models` is flat list.
        // We need Project Code info. Backend `getMatrixData` should probably return project info or we infer it.
        // Current `models` structure: { id, bom_revision_id, name, description }
        // We need map `bom_revision_id` -> `Project Code`.
        // `useBomStore` has `revisions`? Or `BomSidebar` loads them.
        // Let's assume we can group by `bom_revision_id` and use `bom_revision_id` as header group?
        // TanStack Table supports header grouping.
        // Structure:
        // Project A (Header Group) -> [Model A, Model B] (Columns)
        // Project B (Header Group) -> [Model A] (Columns)

        // 1. Group models by BOM Revision ID (which acts as proxy for Project Phase/Version)
        // Better: Group by Project Code. But we might not have project code in `models`.
        // Let's iterate models and build columns.

        const matrixCols = [];
        if (mode === 'MATRIX') {
            // Group models by bom_revision_id
            const groupedModels = {};
            models.forEach(m => {
                if (!groupedModels[m.bom_revision_id]) groupedModels[m.bom_revision_id] = [];
                groupedModels[m.bom_revision_id].push(m);
            });

            // Need to fetch BOM info to display Project Name?
            // We don't have it here easily.
            // Workaround: Use `bom_revision_id` temporarily or assume `models` has extra info.
            // Ideally Backend should enrich `models` with `project_code` or `phase_name`.
            // For now, let's just list them flattened but grouped if TanStack supports dynamic groups easily.
            // Dynamic grouping in TanStack Table requires nested columns structure.

            // Let's try flat columns first with header indicating project?
            // "Project A - Model A"
            // Or better: Update backend to return project info in models.
            // Assuming backend update will happen in next step (as per plan).
            // I will implement Grouped Columns logic assuming `model.project_code` exists or similar.

            // If backend hasn't updated yet, we use placeholder.
            Object.entries(groupedModels).forEach(([bomId, bomModels]) => {
                // Find project info?
                // Let's assume models have `project_code` and `phase_name` (added in next backend step).
                const firstModel = bomModels[0];
                const headerTitle = firstModel.project_code
                    ? `${firstModel.project_code} (${firstModel.phase_name})`
                    : `BOM ${bomId}`;

                const groupCol = {
                    id: `bom_group_${bomId}`,
                    meta: { isFirstOfGroup: true }, // 頂層分組也標記為第一欄，以顯示分隔線
                    header: () => (
                        <div className="w-full text-center">
                            {headerTitle}
                        </div>
                    ),
                    columns: bomModels.map((model, idx) => {
                        const status = summary.modelStatus?.[model.id] || {};
                        const isComplete = status.isComplete;
                        
                        return {
                            id: `model_${model.id}`,
                            meta: { 
                                isComplete, 
                                isFirstOfGroup: idx === 0 // 標記是否為專案分組的第一欄
                            },
                            header: () => {
                                return (
                                    <div className="flex flex-col items-center justify-center py-0.5 text-center px-1" title={model.description}>
                                        <div className="flex flex-col items-center gap-0.5">
                                            <span className="truncate max-w-[90px]">{model.name}</span>
                                            {isComplete && (
                                                <CheckCircle2 size={11} className="text-green-500 flex-shrink-0" />
                                            )}
                                        </div>
                                    </div>
                                );
                            },
                        size: 100,
                        cell: ({ row }) => {
                            const r = row.original;
                            const groupKey = r._key;
                            const rowType = r._rowType === 'main' ? 'part' : 'second_source';
                            const rowId = r.id;
                            const rowBomId = r.bom_revision_id;

                            // Only render checkbox if this row belongs to this BOM (for Union View)
                            // or if it's a "Union Row" (exists in multiple).
                            // Current `executeView` returns UNION of parts.
                            // If a part exists in BOM A but not BOM B, should BOM B column show checkbox?
                            // Requirement: "不屬於該專案的物料, 儲存格以disable的效果呈現 (讓user知道這專案沒有用到這物料)"
                            // So we need to know if this row "exists" in this BOM.
                            // `row` object from `executeView` (Union) should have info about which BOMs it belongs to?
                            // Currently `executeView` aggregates by Supplier+PN.
                            // If we have multiple BOMs, `quantity` etc might be ambiguous.
                            // The `executeView` union logic needs to provide `existsInBomIds` array?
                            // Or we check `bom_revision_id`?
                            // If `executeView` returns one row per unique part, `bom_revision_id` is ambiguous (it picks first).
                            // We need `bom_ids` array in the row data!

                            // Assuming backend updates `executeView` to include `bom_ids` array.
                            const existsInBom = r.bom_ids?.includes(model.bom_revision_id);

                            if (!existsInBom && r.bom_ids) {
                                return <div className="h-full w-full bg-slate-100 dark:bg-slate-800/50" title="此專案無此物料" />;
                            }

                            // Selection check
                            const selection = selectionMap.get(`${model.id}|${groupKey}`);
                            const isSelected = selection &&
                                               selection.selected_type === rowType &&
                                               selection.selected_id === rowId;

                            const isImplicit = isSelected && selection.is_implicit;
                            const isExplicit = isSelected && !selection.is_implicit;
                            const isGroupComplete = !!selection;
                            const showWarning = !isGroupComplete && r._rowType === 'main';

                            return (
                                <div className={`flex items-center justify-center h-full w-full ${showWarning ? 'bg-amber-50 dark:bg-amber-900/20 box-border border-2 border-amber-300 dark:border-amber-600' : ''}`}>
                                    <input
                                        type="checkbox"
                                        checked={!!isSelected}
                                        disabled={isImplicit}
                                        onChange={() => !isImplicit && handleMatrixSelection(model.id, groupKey, rowType, rowId, isExplicit)}
                                        className={`w-4 h-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 cursor-pointer disabled:opacity-50 ${isImplicit ? 'cursor-not-allowed' : ''}`}
                                        title={isImplicit ? "自動選中 (唯一選項)" : ""}
                                    />
                                </div>
                            );
                        }
                    };
                })
                };
                matrixCols.push(groupCol);
            });
        }

        // In Matrix Mode, we replace Standard Cols?
        // User said: "supplier PN 之前的欄位都維持不變, 後面接著是專案的model selection欄位"
        // So we append matrixCols AFTER Supplier PN.
        // Base Cols ends at Supplier PN.
        // So we split standardCols?
        // Wait, current baseCols includes HHPN, Desc, Supplier, Supplier PN.
        // standardCols has Location, Qty, BOM, Type, CCL, Remark.

        if (mode === 'MATRIX') {
            return [...baseCols, ...matrixCols];
        } else {
            return [...baseCols, ...standardCols];
        }

    }, [expandState, toggleAll, toggleGroup, searchTerm, searchFields, mode, models, summary, selectionMap])

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
                <thead className="sticky top-0 z-10 shadow-sm">
                    {table.getHeaderGroups().map((headerGroup, index) => (
                        <tr key={headerGroup.id}>
                            {headerGroup.headers.map((header) => {
                                // 在 Matrix 模式下，處理基礎欄位的合併 (rowSpan)
                                // 如果是第一層 (index 0) 且該欄位沒有子欄位 (isPlaceholder 為假)，則讓他跨兩行
                                const isMatrix = mode === 'MATRIX'
                                const isFirstGroup = index === 0
                                const isLeaf = !header.column.columnDef.columns
                                
                                // 如果在第二層以後：
                                // 1. 如果是佔位符 (代表他在上面的層級已經被 rowSpan 處理過)，則不渲染
                                // 2. 在 Matrix 模式下，第二層只應該顯示屬於分組內 (有 parent) 的欄位
                                if (isMatrix && !isFirstGroup) {
                                    if (header.isPlaceholder || !header.column.parent) {
                                        return null
                                    }
                                }

                                const rowSpan = (isMatrix && isFirstGroup && isLeaf) ? 2 : 1
                                const colSpan = header.colSpan
                                const isSorted = header.column.getIsSorted()

                                // 警告樣式邏輯
                                const meta = header.column.columnDef.meta
                                const isUnfinishedModel = isMatrix && !isFirstGroup && meta && meta.isComplete === false
                                
                                // 分隔線邏輯：Matrix 模式下，每個分組的第一個欄位左側加深框線
                                const hasLeftDivider = isMatrix && meta?.isFirstOfGroup

                                return (
                                    <th
                                        key={header.id}
                                        colSpan={colSpan}
                                        rowSpan={rowSpan}
                                        onClick={header.column.getToggleSortingHandler()}
                                        className={`text-left text-[11px] font-semibold
                                            py-1.5 px-2 border-b border-slate-200 dark:border-slate-600
                                            whitespace-nowrap select-none bg-clip-padding
                                            ${isUnfinishedModel 
                                                ? 'bg-slate-50 dark:bg-surface-800 text-amber-600 dark:text-amber-400' 
                                                : 'bg-slate-50 dark:bg-surface-800 text-bom-header-text'}
                                            ${hasLeftDivider ? 'border-l-2 border-slate-300 dark:border-slate-500' : ''}
                                            ${header.column.getCanSort() ? 'cursor-pointer hover:text-slate-700 dark:hover:text-slate-200' : ''}`}
                                        style={{ width: header.getSize() }}
                                    >
                                        <div className={`flex items-center gap-1 ${!isFirstGroup && isMatrix ? 'justify-center' : ''}`}>
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
                                        {row.getVisibleCells().map((cell) => {
                                            const cellMeta = cell.column.columnDef.meta;
                                            const hasLeftDivider = mode === 'MATRIX' && cellMeta?.isFirstOfGroup;
                                            
                                            return (
                                                <td
                                                    key={cell.id}
                                                    className={`py-1 px-2
                                                        ${isSecond ? 'italic' : ''}
                                                        ${hasLeftDivider ? 'border-l-2 border-slate-200 dark:border-slate-700' : ''}
                                                        whitespace-nowrap overflow-hidden`}
                                                    style={{ maxWidth: cell.column.getSize() }}
                                                >
                                                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                                                </td>
                                            );
                                        })}
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
