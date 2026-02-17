import React, { useMemo, useRef, useState, useEffect } from 'react';
import {
    useReactTable,
    getCoreRowModel,
    flexRender,
} from '@tanstack/react-table';
import { useVirtualizer } from '@tanstack/react-virtual';
import {
    ChevronsUpDown,
    ChevronsDown,
    ListMinus,
    ChevronRight,
    ChevronDown as ChevronDownIcon,
    AlertTriangle,
    CheckCircle2
} from 'lucide-react';
import useMatrixStore from '../../stores/useMatrixStore';

// Helper for highlighting text
const HighlightText = ({ text, term }) => {
    if (!term || !text) return text;
    const parts = String(text).split(new RegExp(`(${term})`, 'gi'));
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
    );
};

const MatrixTable = ({ data, bomRevisionId, searchTerm }) => {
    const { matrixData, saveSelection, deleteSelection } = useMatrixStore();
    const currentMatrix = matrixData[bomRevisionId];
    const models = currentMatrix?.models || [];
    const selections = currentMatrix?.selections || [];
    const summary = currentMatrix?.summary || {};

    // Build Selection Map for O(1) lookup
    // Key: `${matrix_model_id}|${group_key}` -> Selection Object
    const selectionMap = useMemo(() => {
        const map = new Map();
        selections.forEach(s => {
            map.set(`${s.matrix_model_id}|${s.group_key}`, s);
        });
        return map;
    }, [selections]);

    // Flatten Data Logic (Similar to BomTable)
    const [expandedGroups, setExpandedGroups] = useState(new Set());

    // Auto-expand all groups with 2nd sources initially
    useEffect(() => {
        if (data) {
            const keys = new Set();
            data.forEach(item => {
                if (item.second_sources && item.second_sources.length > 0) {
                    keys.add(`${item.supplier}|${item.supplier_pn}`);
                }
            });
            setExpandedGroups(keys);
        }
    }, [data]);

    const toggleGroup = (key) => {
        setExpandedGroups(prev => {
            const next = new Set(prev);
            if (next.has(key)) next.delete(key);
            else next.add(key);
            return next;
        });
    };

    const flatRows = useMemo(() => {
        if (!data) return [];
        const rows = [];
        let groupIndex = 0;
        data.forEach((mainItem) => {
            const key = `${mainItem.supplier}|${mainItem.supplier_pn}`;
            const hasSecondSources = mainItem.second_sources?.length > 0;
            const isExpanded = expandedGroups.has(key);

            // Main Item
            rows.push({
                ...mainItem,
                _rowType: 'main',
                _groupIndex: groupIndex,
                _key: key,
                _hasSecondSources: hasSecondSources,
                _isExpanded: isExpanded,
            });

            // Second Sources
            if (hasSecondSources && isExpanded) {
                mainItem.second_sources.forEach((ss) => {
                    rows.push({
                        ...ss,
                        _rowType: 'second',
                        _groupIndex: groupIndex,
                        _key: key, // Same group key
                        // inherit some fields for context if needed
                    });
                });
            }
            groupIndex++;
        });
        return rows;
    }, [data, expandedGroups]);

    const handleSelection = async (modelId, groupKey, type, id, isCurrentlySelected) => {
        if (isCurrentlySelected) {
            await deleteSelection(bomRevisionId, modelId, groupKey);
        } else {
            await saveSelection(bomRevisionId, {
                matrix_model_id: modelId,
                group_key: groupKey,
                selected_type: type,
                selected_id: id
            });
        }
    };

    const columns = useMemo(() => {
        const baseColumns = [
            {
                id: 'rowIndicator',
                header: '#',
                size: 50,
                cell: ({ row }) => {
                    const r = row.original;
                    if (r._rowType === 'main') {
                        return (
                            <div
                                className="flex items-center gap-1 cursor-pointer select-none pl-1"
                                onClick={(e) => {
                                    if (r._hasSecondSources) {
                                        e.stopPropagation();
                                        toggleGroup(r._key);
                                    }
                                }}
                            >
                                <span className="text-[10px] text-slate-600 font-bold">Main</span>
                                {r._hasSecondSources && (
                                    <span className="text-slate-400">
                                        {r._isExpanded ? <ChevronDownIcon size={12}/> : <ChevronRight size={12}/>}
                                    </span>
                                )}
                            </div>
                        );
                    }
                    return <span className="text-[10px] text-slate-400 italic pl-1">2nd</span>;
                }
            },
            {
                accessorKey: 'hhpn',
                header: 'HHPN',
                size: 120,
                cell: ({ getValue }) => <HighlightText text={getValue()} term={searchTerm} />
            },
            {
                accessorKey: 'supplier',
                header: 'Supplier',
                size: 100,
                cell: ({ getValue }) => <HighlightText text={getValue()} term={searchTerm} />
            },
            {
                accessorKey: 'supplier_pn',
                header: 'Supplier PN',
                size: 120,
                cell: ({ getValue }) => <HighlightText text={getValue()} term={searchTerm} />
            },
             {
                accessorKey: 'description',
                header: 'Description',
                size: 200,
                cell: ({ getValue }) => (
                    <span className="truncate block" title={getValue()}>
                        <HighlightText text={getValue()} term={searchTerm} />
                    </span>
                )
            },
        ];

        // Dynamic Matrix Columns
        const matrixCols = models.map(model => ({
            id: `model_${model.id}`,
            header: () => {
                // Header with Lamp
                const status = summary.modelStatus?.[model.id] || {};
                const isComplete = status.isComplete;
                return (
                    <div className="flex flex-col items-center justify-center gap-1" title={model.description}>
                        <div className="flex items-center gap-1">
                            <span>{model.name}</span>
                            {isComplete ? (
                                <CheckCircle2 size={14} className="text-green-500" />
                            ) : (
                                <AlertTriangle size={14} className="text-amber-500" />
                            )}
                        </div>
                        <span className="text-[10px] font-normal text-slate-400">
                            {status.selectedCount || 0}/{summary.totalGroups || 0}
                        </span>
                    </div>
                );
            },
            size: 100,
            cell: ({ row }) => {
                const r = row.original;
                const groupKey = r._key;
                const rowType = r._rowType === 'main' ? 'part' : 'second_source';
                const rowId = r.id;

                // Check Selection
                const selection = selectionMap.get(`${model.id}|${groupKey}`);
                const isSelected = selection &&
                                   selection.selected_type === rowType &&
                                   selection.selected_id === rowId;

                const isImplicit = isSelected && selection.is_implicit;
                const isExplicit = isSelected && !selection.is_implicit;

                // Determine if this GROUP is incomplete for this model
                // If NO selection exists for this group in this model -> Show warning border?
                // But we only show warning on CELL if it SHOULD be selected but isn't?
                // Actually, if the group is incomplete, ALL cells in that group for this model column could have a warning border?
                // Or just the Main cell?
                const isGroupComplete = !!selection;
                const showWarning = !isGroupComplete && r._rowType === 'main'; // Only warn on Main row to avoid clutter

                return (
                    <div className={`flex items-center justify-center h-full w-full ${showWarning ? 'bg-amber-50 dark:bg-amber-900/20 box-border border-2 border-amber-300 dark:border-amber-600' : ''}`}>
                        <input
                            type="checkbox"
                            checked={!!isSelected}
                            disabled={isImplicit} // Implicit selection is read-only (or force user to explicitly select another if available)
                            onChange={() => !isImplicit && handleSelection(model.id, groupKey, rowType, rowId, isExplicit)}
                            className={`w-4 h-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 cursor-pointer disabled:opacity-50 ${isImplicit ? 'cursor-not-allowed' : ''}`}
                            title={isImplicit ? "自動選中 (唯一選項)" : ""}
                        />
                    </div>
                );
            }
        }));

        return [...baseColumns, ...matrixCols];
    }, [models, summary, selectionMap, searchTerm, bomRevisionId]); // Added dependencies

    const table = useReactTable({
        data: flatRows,
        columns,
        getCoreRowModel: getCoreRowModel(),
    });

    const parentRef = useRef(null);
    const rowVirtualizer = useVirtualizer({
        count: table.getRowModel().rows.length,
        getScrollElement: () => parentRef.current,
        estimateSize: () => 35,
        overscan: 10,
    });

    const virtualItems = rowVirtualizer.getVirtualItems();
    const totalSize = rowVirtualizer.getTotalSize();
    const paddingTop = virtualItems.length > 0 ? virtualItems[0].start : 0;
    const paddingBottom = virtualItems.length > 0 ? totalSize - (virtualItems[virtualItems.length - 1]?.end || 0) : 0;

    return (
        <div ref={parentRef} className="h-full overflow-auto border border-slate-200 dark:border-slate-700 rounded-lg bg-white dark:bg-surface-800 relative">
            <table className="w-full text-xs border-collapse relative">
                <thead className="bg-bom-header-bg sticky top-0 z-10 shadow-sm">
                    {table.getHeaderGroups().map(headerGroup => (
                        <tr key={headerGroup.id}>
                            {headerGroup.headers.map(header => (
                                <th key={header.id} className="text-center text-[11px] font-semibold text-bom-header-text py-2 px-2 border-b border-r border-slate-200 dark:border-slate-600 whitespace-nowrap bg-bom-header-bg" style={{ width: header.getSize() }}>
                                    {flexRender(header.column.columnDef.header, header.getContext())}
                                </th>
                            ))}
                        </tr>
                    ))}
                </thead>
                <tbody>
                    {paddingTop > 0 && <tr><td colSpan={columns.length} style={{ height: `${paddingTop}px` }} /></tr>}
                    {virtualItems.map(virtualRow => {
                        const row = table.getRowModel().rows[virtualRow.index];
                        const r = row.original;
                        const isSecond = r._rowType === 'second';
                        const isEvenGroup = r._groupIndex % 2 === 0;
                        const rowClass = isEvenGroup
                            ? (isSecond ? 'bg-bom-row-second-even text-bom-text-second' : 'bg-bom-row-main-even text-bom-text-main')
                            : (isSecond ? 'bg-bom-row-second-odd text-bom-text-second' : 'bg-bom-row-main-odd text-bom-text-main');

                        return (
                            <tr key={row.id} className={`${rowClass} border-b border-slate-100 dark:border-slate-700/50 hover:bg-bom-row-hover transition-colors`}>
                                {row.getVisibleCells().map(cell => (
                                    <td key={cell.id} className="py-1 px-2 border-r border-slate-100 dark:border-slate-700/30 whitespace-nowrap overflow-hidden" style={{ maxWidth: cell.column.getSize() }}>
                                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                                    </td>
                                ))}
                            </tr>
                        );
                    })}
                    {paddingBottom > 0 && <tr><td colSpan={columns.length} style={{ height: `${paddingBottom}px` }} /></tr>}
                </tbody>
            </table>
        </div>
    );
};

export default MatrixTable;
