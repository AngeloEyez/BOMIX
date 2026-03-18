import React, { useState, useEffect } from 'react';
import Dialog from './Dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Plus, X, GripVertical } from 'lucide-react';
import { Sortable, SortableContent, SortableItem, SortableItemHandle } from '@/components/ui/sortable';

const DEFAULT_PHASE_ORDER = [
    'RFI',
    'RFP',
    'RFQ, RFx',
    'DB, EVT',
    'SI, DVT',
    'PV, PVT',
    'TLD, PRD',
    'MVB, MP'
];

/**
 * Phase 排序編輯對話框
 *
 * @param {Object} props
 * @param {boolean} props.isOpen - 是否開啟
 * @param {Function} props.onClose - 關閉回調
 * @param {string} props.currentPhaseOrder - JSON 格式的 phase order (series_meta.phase_order)
 * @param {Function} props.onSave - 儲存回調，接收 string[] 陣列
 * @param {string} [props.requiredPhase] - 若不為空，表示必須要包含這個 phase 才能儲存
 */
function PhaseOrderDialog({ isOpen, onClose, currentPhaseOrder, onSave, requiredPhase }) {
    const [items, setItems] = useState([]);
    const [newItemValue, setNewItemValue] = useState('');
    const [error, setError] = useState('');

    useEffect(() => {
        if (isOpen) {
            let initialOrder = DEFAULT_PHASE_ORDER;
            if (currentPhaseOrder) {
                try {
                    const parsed = JSON.parse(currentPhaseOrder);
                    if (Array.isArray(parsed) && parsed.length > 0) {
                        initialOrder = parsed;
                    }
                } catch (e) {
                    console.error('Failed to parse currentPhaseOrder', e);
                }
            }

            // 轉換為 SortableList 需要的格式 { id, value }
            setItems(initialOrder.map((val, index) => ({
                id: `phase-${index}-${Date.now()}`,
                value: val
            })));

            setError('');
            setNewItemValue('');
        }
    }, [isOpen, currentPhaseOrder]);

    const handleSave = () => {
        const orderArray = items.map(item => item.value.trim()).filter(Boolean);

        if (orderArray.length === 0) {
            setError('至少需要一個 Phase 定義');
            return;
        }

        if (requiredPhase) {
            // 檢查 requiredPhase 是否存在於任何一個 definition 中
            const normalizedReq = requiredPhase.toUpperCase();
            const exists = orderArray.some(def => {
                const parts = def.split(',').map(s => s.trim().toUpperCase());
                return parts.includes(normalizedReq);
            });

            if (!exists) {
                setError(`請將 "${requiredPhase}" 加入到列表中的任一項目內`);
                return;
            }
        }

        onSave(orderArray);
    };

    const handleAddItem = () => {
        if (!newItemValue.trim()) return;
        setItems([...items, { id: `phase-${Date.now()}`, value: newItemValue.trim() }]);
        setNewItemValue('');
    };

    const handleRemoveItem = (id) => {
        setItems(items.filter(item => item.id !== id));
    };

    const handleItemChange = (id, newValue) => {
        setItems(items.map(item => item.id === id ? { ...item, value: newValue } : item));
    };

    const handleSortEnd = (newItems) => {
        setItems(newItems);
    };

    return (
        <Dialog isOpen={isOpen} onClose={onClose} title="自訂 Phase 排序" className="max-w-md">
            <div className="space-y-4">
                <p className="text-xs text-muted-foreground leading-relaxed">
                    您可以自訂專案中 BOM Phase 的順序。這些順序將會影響「前一版」版本計算與儀表板上的排序。
                    <br />
                    請使用逗號分隔同一層級的 Phase (例如: <code>DB, EVT</code>)。
                </p>

                {requiredPhase && (
                    <div className="bg-amber-50 dark:bg-amber-950/30 text-amber-600 dark:text-amber-500 p-2 text-xs rounded-md border border-amber-200 dark:border-amber-900">
                        匯入的 BOM 包含未知的 Phase <strong>"{requiredPhase}"</strong>，請將它新增到下方的排序清單中。
                    </div>
                )}

                <div className="bg-muted/30 border border-border rounded-lg p-2 max-h-[40vh] overflow-y-auto">
                    <Sortable
                        value={items}
                        onValueChange={handleSortEnd}
                        getItemValue={(item) => item.id}
                    >
                        <SortableContent className="space-y-2">
                            {items.map((item) => (
                                <SortableItem key={item.id} value={item.id} asChild>
                                    <div className="flex items-center gap-2 bg-background border border-border p-1.5 rounded-md shadow-sm group">
                                        <SortableItemHandle asChild>
                                            <Button variant="ghost" size="icon" className="h-6 w-6 cursor-grab text-muted-foreground">
                                                <GripVertical size={14} />
                                            </Button>
                                        </SortableItemHandle>
                                        <Input
                                            value={item.value}
                                            onChange={(e) => handleItemChange(item.id, e.target.value)}
                                            className="h-7 text-xs flex-1 border-transparent focus-visible:ring-1 bg-transparent"
                                            placeholder="例如: EVT, DVT"
                                        />
                                        <Button
                                            variant="ghost"
                                            size="icon"
                                            className="h-6 w-6 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity hover:text-destructive"
                                            onClick={() => handleRemoveItem(item.id)}
                                        >
                                            <X size={14} />
                                        </Button>
                                    </div>
                                </SortableItem>
                            ))}
                        </SortableContent>
                    </Sortable>
                </div>

                <div className="flex items-center gap-2">
                    <Input
                        value={newItemValue}
                        onChange={(e) => setNewItemValue(e.target.value)}
                        placeholder="新增 Phase，如: MP"
                        className="h-8 text-xs flex-1"
                        onKeyDown={(e) => e.key === 'Enter' && handleAddItem()}
                    />
                    <Button variant="outline" size="sm" className="h-8 shrink-0" onClick={handleAddItem}>
                        <Plus size={14} className="mr-1" />
                        新增
                    </Button>
                </div>

                {error && <p className="text-xs text-destructive">{error}</p>}

                <div className="flex justify-end gap-2 pt-2 border-t border-border">
                    <Button variant="outline" size="sm" onClick={onClose}>取消</Button>
                    <Button size="sm" onClick={handleSave}>儲存設定</Button>
                </div>
            </div>
        </Dialog>
    );
}

export default PhaseOrderDialog;