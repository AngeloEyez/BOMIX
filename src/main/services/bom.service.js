/**
 * @file src/main/services/bom.service.js
 * @description BOM 管理 (BOM Management) 業務邏輯層
 * @module services/bom
 */

import partsRepo from '../database/repositories/parts.repo.js';
import secondSourceRepo from '../database/repositories/second-source.repo.js';
import bomRevisionRepo from '../database/repositories/bom-revision.repo.js';
import dbManager from '../database/connection.js';

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
 * 執行 BOM View (In-Memory Aggregation)
 * 支援單一 ID 或 IDs 陣列 (Union View)
 *
 * @param {number|Array<number>} bomRevisionIdOrIds
 * @param {Object} viewDefinition - 從 bom-factory 取得的 View 定義
 * @returns {Array<Object>} 聚合後的 BOM 列表
 */
export function executeView(bomRevisionIdOrIds, viewDefinition) {
    let ids = Array.isArray(bomRevisionIdOrIds) ? bomRevisionIdOrIds : [bomRevisionIdOrIds];
    if (ids.length === 0) return [];

    // 1. 取得 BOM Revisions (確認 Mode - 優先使用第一個 BOM 的 Mode 或統一 Mode)
    // 假設 Union View 的 Mode 必須一致，或以第一個為主
    const firstRevision = bomRevisionRepo.findById(ids[0]);
    if (!firstRevision) {
        throw new Error(`找不到 ID 為 ${ids[0]} 的 BOM 版本`);
    }
    const mode = firstRevision.mode || 'NPI';

    // 2. 取得原始資料 (Multi-BOM)
    let parts = partsRepo.findByBomRevisions(ids);
    // secondSourceRepo also needs multi-id support or fetch iteratively
    // Currently secondSourceRepo doesn't have `findByBomRevisions`.
    // Let's iterate for SS or add repo method. Iterating is fine for now as SS count is usually low relative to parts.
    // Or better, update secondSourceRepo later. For now, fetch all SS for these IDs.
    // Wait, secondSourceRepo.findByBomRevision returns *all* SS for that BOM.
    let secondSources = [];
    for (const id of ids) {
        secondSources = secondSources.concat(secondSourceRepo.findByBomRevision(id));
    }

    // 3. 準備 Filter 邏輯
    const filter = viewDefinition.filter;

    // 定義狀態集合
    let allowedStatuses = [];
    if (filter.statusLogic === 'ACTIVE') {
        allowedStatuses = mode === 'MP' ? ['I', 'M'] : ['I', 'P'];
    } else if (filter.statusLogic === 'INACTIVE') {
        allowedStatuses = mode === 'MP' ? ['X', 'P'] : ['X', 'M'];
    } else if (filter.statusLogic === 'SPECIFIC') {
        allowedStatuses = filter.bom_statuses || [];
    }

    // 4. 過濾零件
    const filteredParts = parts.filter(part => {
        // Type Filter
        if (filter.types && filter.types.length > 0) {
            if (!filter.types.includes(part.type)) return false;
        }

        // Status Filter
        if (filter.statusLogic !== 'IGNORE') {
            if (!allowedStatuses.includes(part.bom_status)) return false;
        }

        // CCL Filter
        if (filter.ccl && part.ccl !== filter.ccl) {
            return false;
        }

        return true;
    });

    // 5. 分組與聚合 (Group by supplier + supplier_pn + type)
    const groupedMap = new Map();

    for (const part of filteredParts) {
        // Key logic must match `partsRepo.getAggregatedBom`
        // Key: supplier|supplier_pn
        const key = `${part.supplier}|${part.supplier_pn}`;

        if (!groupedMap.has(key)) {
            groupedMap.set(key, {
                // Main Item Representative Fields
                id: part.id, // Representative ID (Main Source ID)
                bom_revision_id: part.bom_revision_id,
                supplier: part.supplier,
                supplier_pn: part.supplier_pn,
                type: part.type, // Pick first encountered type or maybe make it null? For now, first is fine as representative.
                hhpn: part.hhpn,
                description: part.description,
                bom_status: part.bom_status, 
                ccl: part.ccl,
                remark: part.remark,
                item: part.item, // Min item usually

                // Aggregation Fields
                locations: [],
                quantity: 0
            });
        }

        const group = groupedMap.get(key);
        group.locations.push(part.location);
        group.quantity += 1;

        // Update Min Item
        if (part.item && (!group.item || part.item < group.item)) {
            group.item = part.item;
        }
    }

    // Convert Map to Array and Finalize Aggregation
    const mainItems = Array.from(groupedMap.values()).map(group => {
        // Sort Locations
        // Simple sort for now. Ideally should be alphanumeric sort (R1, R2, R10...)
        group.locations.sort((a, b) => {
             // Try to parse number from string if possible for better sort
             return a.localeCompare(b, undefined, { numeric: true, sensitivity: 'base' });
        });

        return {
            ...group,
            locations: group.locations.join(',')
        };
    });

    // Sort Main Items by Item Number (to match Repo behavior)
    mainItems.sort((a, b) => {
        if (a.item === null) return 1;
        if (b.item === null) return -1;
        return a.item - b.item;
    });

    // 6. 關聯 Second Sources
    // Optimization: Build Index for Second Sources
    // SS Key: main_supplier|main_supplier_pn
    const ssMap = new Map();
    for (const ss of secondSources) {
        const key = `${ss.main_supplier}|${ss.main_supplier_pn}`;
        if (!ssMap.has(key)) {
            ssMap.set(key, []);
        }
        ssMap.get(key).push(ss);
    }

    // Attach
    return mainItems.map(item => {
        const key = `${item.supplier}|${item.supplier_pn}`;
        return {
            ...item,
            second_sources: ssMap.get(key) || []
        };
    });
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
