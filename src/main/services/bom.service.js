/**
 * @file src/main/services/bom.service.js
 * @description BOM 管理 (BOM Management) 業務邏輯層
 * @module services/bom
 */

import partsRepo from '../database/repositories/parts.repo.js';
import secondSourceRepo from '../database/repositories/second-source.repo.js';
import bomRevisionRepo from '../database/repositories/bom-revision.repo.js';
import dbManager from '../database/connection.js';

// ========================================
// 內部輔助函式
// ========================================

/**
 * 將舊版 viewDefinition 物件轉換為 filters 陣列格式。
 *
 * 此函式為內部轉換工具，供 executeView 薄包裝層使用。
 * 格式說明詳見 dev/FILTER_SPEC.md。
 *
 * @param {Object} viewDefinition - 舊版 View 定義物件（已含 filter 或 filters 屬性）
 * @returns {Array<Object>} Filter 陣列
 */
function convertViewDefToFilters(viewDefinition) {
    // 若已是新格式（含 filters 陣列），直接回傳
    if (viewDefinition.filters) {
        return viewDefinition.filters;
    }

    // 向下相容：將舊格式 filter 物件轉換為 filters 陣列
    const filters = [];
    const f = viewDefinition.filter || {};

    if (f.statusLogic) {
        filters.push({ field: 'bom_status', operator: 'statusLogic', value: f.statusLogic });
    }
    if (f.bom_statuses && f.bom_statuses.length > 0) {
        filters.push({ field: 'bom_status', operator: 'in', value: f.bom_statuses });
    }
    if (f.types && f.types.length > 0) {
        filters.push({ field: 'type', operator: 'in', value: f.types });
    }
    if (f.ccl) {
        filters.push({ field: 'ccl', operator: 'eq', value: f.ccl });
    }
    return filters;
}

/**
 * 套用單一 Filter 條件。
 *
 * @param {Object} part - 零件資料
 * @param {Object} filter - Filter 條件
 * @param {string} mode - BOM 版本模式（'NPI' | 'MP'）
 * @returns {boolean} 是否通過此條件
 */
function applyFilter(part, filter, mode) {
    const { field, operator, value } = filter;

    switch (operator) {
        case 'eq':
            return part[field] === value;

        case 'neq':
            return part[field] !== value;

        case 'in':
            return Array.isArray(value) && value.includes(part[field]);

        case 'notIn':
            return Array.isArray(value) && !value.includes(part[field]);

        case 'statusLogic': {
            // 特殊邏輯：依 BOM mode 決定允許的 bom_status
            let allowedStatuses = [];
            if (value === 'ACTIVE') {
                allowedStatuses = mode === 'MP' ? ['I', 'M'] : ['I', 'P'];
            } else if (value === 'INACTIVE') {
                allowedStatuses = mode === 'MP' ? ['X', 'P'] : ['X', 'M'];
            }
            // SPECIFIC 模式：搭配同陣列中的 'in' filter 使用，此 filter 本身不過濾
            if (value === 'SPECIFIC') return true;
            return allowedStatuses.includes(part.bom_status);
        }

        default:
            // 未知 operator 視為通過（不過濾）
            return true;
    }
}

// ========================================
// 核心查詢函式
// ========================================

/**
 * 通用 BOM 資料查詢與聚合。
 *
 * 接收 BOM ID 陣列、Filter 條件陣列與選項物件，執行過濾與 In-Memory 聚合，
 * 回傳聚合後的 Main Item 列表（含關聯 Second Sources）。
 *
 * Filter 陣列格式與 Operator 說明詳見 dev/FILTER_SPEC.md。
 *
 * @param {number[]} bomIds - BOM 版本 ID 陣列（支援多 BOM Union 查詢）
 * @param {Array<Object>} [filters=[]] - Filter 條件陣列
 * @param {Object} [options={}] - 選項參數（預留，暫未實作功能）
 * @returns {Array<Object>} 聚合後的 BOM 列表
 */
export function queryBomData(bomIds, filters = [], options = {}) {
    if (!Array.isArray(bomIds) || bomIds.length === 0) return [];

    // 1. 取得 BOM Revision（以第一個的 Mode 為準）
    const firstRevision = bomRevisionRepo.findById(bomIds[0]);
    if (!firstRevision) {
        throw new Error(`找不到 ID 為 ${bomIds[0]} 的 BOM 版本`);
    }
    const mode = firstRevision.mode || 'NPI';

    // 2. 撈取所有 BOM 的零件與替代料
    let parts = partsRepo.findByBomRevisions(bomIds);
    let secondSources = [];
    for (const id of bomIds) {
        secondSources = secondSources.concat(secondSourceRepo.findByBomRevision(id));
    }

    // 3. 套用 filters 過濾零件（多個條件為 AND 關係）
    const filteredParts = parts.filter(part =>
        filters.every(filter => applyFilter(part, filter, mode))
    );

    // 4. 分組與聚合 (Group by supplier + supplier_pn)
    const groupedMap = new Map();

    for (const part of filteredParts) {
        const key = `${part.supplier}|${part.supplier_pn}`;

        if (!groupedMap.has(key)) {
            groupedMap.set(key, {
                id: part.id,
                bom_revision_id: part.bom_revision_id,
                bom_ids: new Set([part.bom_revision_id]),
                supplier: part.supplier,
                supplier_pn: part.supplier_pn,
                type: part.type,
                hhpn: part.hhpn,
                description: part.description,
                bom_status: part.bom_status,
                ccl: part.ccl,
                remark: part.remark,
                item: part.item,
                locations: [],
                quantity: 0
            });
        }

        const group = groupedMap.get(key);
        group.bom_ids.add(part.bom_revision_id);
        group.locations.push(part.location);
        group.quantity += 1;

        // 保留最小 item 作為排序基準
        if (part.item && (!group.item || part.item < group.item)) {
            group.item = part.item;
        }
    }

    // 5. 轉換為陣列並完成聚合
    const mainItems = Array.from(groupedMap.values()).map(group => {
        group.locations.sort((a, b) =>
            a.localeCompare(b, undefined, { numeric: true, sensitivity: 'base' })
        );
        return {
            ...group,
            bom_ids: Array.from(group.bom_ids),
            locations: group.locations.join(',')
        };
    });

    // 依 item 排序
    mainItems.sort((a, b) => {
        if (a.item === null) return 1;
        if (b.item === null) return -1;
        return a.item - b.item;
    });

    // 6. 關聯 Second Sources
    const ssMap = new Map();
    for (const ss of secondSources) {
        const key = `${ss.main_supplier}|${ss.main_supplier_pn}`;
        if (!ssMap.has(key)) ssMap.set(key, []);
        ssMap.get(key).push(ss);
    }

    return mainItems.map(item => ({
        ...item,
        second_sources: ssMap.get(`${item.supplier}|${item.supplier_pn}`) || []
    }));
}

/**
 * 執行 BOM View (In-Memory Aggregation)
 * 支援單一 ID 或 IDs 陣列 (Union View)
 *
 * @deprecated 此函式為向下相容層，供 export.service.js 等後端內部模組使用。
 *             前端請改用 IPC 通道 `bom:query` → `queryBomData`。
 *             當所有後端呼叫者遷移完畢後可評估移除。
 *
 * @param {number|Array<number>} bomRevisionIdOrIds
 * @param {Object} viewDefinition - View 定義物件（支援新 filters 陣列格式與舊 filter 物件格式）
 * @returns {Array<Object>} 聚合後的 BOM 列表
 */
export function executeView(bomRevisionIdOrIds, viewDefinition) {
    const ids = Array.isArray(bomRevisionIdOrIds) ? bomRevisionIdOrIds : [bomRevisionIdOrIds];
    const filters = convertViewDefToFilters(viewDefinition);
    return queryBomData(ids, filters, {});
}

/**
 * 取得 BOM 原始資料 (包含 Parts 與 Second Sources)
 * @param {number} bomRevisionId
 * @returns {Object} { parts: Array, secondSources: Array }
 */
export function getBomData(bomRevisionId) {
    const parts = partsRepo.findByBomRevision(bomRevisionId);
    const secondSources = secondSourceRepo.findByBomRevision(bomRevisionId);
    return { parts, secondSources };
}

/**
 * 取得 BOM 聚合視圖
 * 包含 Main Items (聚合後的零件) 與其關聯的 Second Sources
 *
 * @param {number} bomRevisionId - BOM 版本 ID
 * @returns {Array<Object>} BOM 列表
 * @throws {Error} 若 BOM 版本不存在
 */
export function getBomView(bomRevisionId) {
    const revision = bomRevisionRepo.findById(bomRevisionId);
    if (!revision) {
        throw new Error(`找不到 ID 為 ${bomRevisionId} 的 BOM 版本`);
    }

    // 1. 取得聚合後的 Main Items
    const mainItems = partsRepo.getAggregatedBom(bomRevisionId);

    // 2. 取得該版本所有 Second Sources
    const secondSources = secondSourceRepo.findByBomRevision(bomRevisionId);

    // 3. 將 Second Sources 關聯至對應的 Main Item
    // 建立索引以加速查找: key = `${supplier}|${supplier_pn}`
    const ssMap = new Map();
    secondSources.forEach(ss => {
        const key = `${ss.main_supplier}|${ss.main_supplier_pn}`;
        if (!ssMap.has(key)) {
            ssMap.set(key, []);
        }
        ssMap.get(key).push(ss);
    });

    // 4. 組合結果
    return mainItems.map(item => {
        const key = `${item.supplier}|${item.supplier_pn}`;
        return {
            ...item,
            second_sources: ssMap.get(key) || []
        };
    });
}

/**
 * 更新 Main Item (更新群組內所有零件)
 *
 * @param {number} bomRevisionId - BOM 版本 ID
 * @param {Object} originalKey - 原始鍵值 (用於定位群組)
 * @param {string} originalKey.supplier
 * @param {string} originalKey.supplier_pn
 * @param {string} [originalKey.type]
 * @param {Object} updates - 更新內容
 * @returns {Object} 更新結果 { success: true }
 */
export function updateMainItem(bomRevisionId, originalKey, updates) {
    const { supplier, supplier_pn } = originalKey;

    // 找出群組內所有零件
    // Note: type param is ignored by findByGroup now, but we can pass null/undefined safely.
    const parts = partsRepo.findByGroup(bomRevisionId, supplier, supplier_pn);
    if (parts.length === 0) {
        throw new Error('找不到指定的零件群組');
    }

    const db = dbManager.getDb();
    const transaction = db.transaction(() => {
        for (const part of parts) {
            partsRepo.update(part.id, updates);
        }
    });

    transaction();
    return { success: true };
}

/**
 * 刪除 Main Item (刪除群組內所有零件與關聯的 Second Sources)
 *
 * @param {number} bomRevisionId - BOM 版本 ID
 * @param {Object} key - 鍵值
 * @param {string} key.supplier
 * @param {string} key.supplier_pn
 * @param {string} [key.type]
 * @returns {Object} 刪除結果 { success: true }
 */
export function deleteMainItem(bomRevisionId, key) {
    const { supplier, supplier_pn } = key;

    // 找出群組內所有零件
    const parts = partsRepo.findByGroup(bomRevisionId, supplier, supplier_pn);

    const db = dbManager.getDb();
    const transaction = db.transaction(() => {
        // 刪除零件
        for (const part of parts) {
            partsRepo.delete(part.id);
        }

        // 檢查是否還有其他零件使用此 Supplier + PN
        // 若該 BOM 版本下沒有其他 Main Item 使用此 Supplier+PN，則刪除 Second Source。
        const allSS = secondSourceRepo.findByBomRevision(bomRevisionId);
        const relatedSS = allSS.filter(ss => ss.main_supplier === supplier && ss.main_supplier_pn === supplier_pn);

        for (const ss of relatedSS) {
             secondSourceRepo.delete(ss.id);
        }
    });

    transaction();
    return { success: true };
}

/**
 * 新增 Second Source
 *
 * @param {Object} data
 * @returns {Object} 建立的 Second Source
 */
export function addSecondSource(data) {
    return secondSourceRepo.create(data);
}

/**
 * 更新 Second Source
 *
 * @param {number} id
 * @param {Object} data
 * @returns {Object} 更新後的 Second Source
 */
export function updateSecondSource(id, data) {
    const updated = secondSourceRepo.update(id, data);
    if (!updated) {
        throw new Error(`找不到 ID 為 ${id} 的 Second Source`);
    }
    return updated;
}

/**
 * 刪除 Second Source
 *
 * @param {number} id
 * @returns {Object} { success: true }
 */
export function deleteSecondSource(id) {
    const success = secondSourceRepo.delete(id);
    if (!success) {
        throw new Error(`找不到 ID 為 ${id} 的 Second Source`);
    }
    return { success: true };
}

/**
 * 刪除整個 BOM
 * @param {number} bomRevisionId
 */
export function deleteBom(bomRevisionId) {
    return bomRevisionRepo.delete(bomRevisionId);
}

/**
 * 更新 BOM 版本資料
 * @param {number} id
 * @param {Object} updates
 * @returns {Object} 更新後的 BOM 版本物件
 */
export function updateBomRevision(id, updates) {
    const updated = bomRevisionRepo.update(id, updates);
    if (!updated) {
        throw new Error(`找不到 ID 為 ${id} 的 BOM 版本`);
    }
    return updated;
}

export default {
    queryBomData,
    getBomData,
    executeView,
    getBomView,
    updateMainItem,
    deleteMainItem,
    addSecondSource,
    updateSecondSource,
    deleteSecondSource,
    deleteBom,
    updateBomRevision
};
