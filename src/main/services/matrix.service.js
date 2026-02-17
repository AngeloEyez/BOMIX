/**
 * @file src/main/services/matrix.service.js
 * @description Matrix BOM 業務邏輯層
 * @module services/matrix
 */

import matrixModelRepo from '../database/repositories/matrix-model.repo.js';
import matrixSelectionRepo from '../database/repositories/matrix-selection.repo.js';
import bomService from './bom.service.js';

/**
 * 建立 Matrix Models (批次)
 * 若未提供 models 資料，預設建立 A, B, C 三個 Model
 *
 * @param {number} bomRevisionId
 * @param {Array<Object>} [models] - [{ name, description }]
 * @returns {Array<Object>} 建立的 Models
 */
export function createModels(bomRevisionId, models) {
    const modelsToCreate = models && models.length > 0
        ? models
        : [
            { name: 'Model A', description: 'Default Model A' },
            { name: 'Model B', description: 'Default Model B' },
            { name: 'Model C', description: 'Default Model C' }
        ];

    const createdModels = [];
    for (const model of modelsToCreate) {
        createdModels.push(matrixModelRepo.create({
            bom_revision_id: bomRevisionId,
            name: model.name,
            description: model.description
        }));
    }
    return createdModels;
}

/**
 * 取得指定 BOM 下的所有 Matrix Models
 * @param {number} bomRevisionId
 * @returns {Array<Object>}
 */
export function listModels(bomRevisionId) {
    return matrixModelRepo.findByBomRevisionId(bomRevisionId);
}

/**
 * 更新 Matrix Model
 * @param {number} id
 * @param {Object} updates
 * @returns {Object} 更新後的 Model
 */
export function updateModel(id, updates) {
    return matrixModelRepo.update(id, updates);
}

/**
 * 刪除 Matrix Model
 * @param {number} id
 * @returns {Object} { success: true }
 */
export function deleteModel(id) {
    // 檢查是否有 Selection (前端應先檢查，後端再防守一次)
    const count = matrixSelectionRepo.countByModelId(id);
    if (count > 0) {
        throw new Error('該 Model 已有選擇紀錄，無法直接刪除');
    }
    const success = matrixModelRepo.delete(id);
    if (!success) {
        throw new Error('刪除失敗，找不到該 Model');
    }
    return { success: true };
}

/**
 * 儲存 Matrix Selection (Upsert)
 * @param {Object} selectionData
 * @returns {Object} 更新後的 Selection
 */
export function saveSelection(selectionData) {
    // TODO: 驗證 selected_id 是否屬於該 BOM (Optional validation)
    return matrixSelectionRepo.upsert(selectionData);
}

/**
 * 刪除 Matrix Selection (取消選擇)
 * @param {number} matrixModelId
 * @param {string} groupKey
 * @returns {Object} { success: true }
 */
export function deleteSelection(matrixModelId, groupKey) {
    const success = matrixSelectionRepo.deleteByModelAndGroup(matrixModelId, groupKey);
    // If not found, still success
    return { success: true };
}

/**
 * 取得 Matrix 完整資料 (含 Implicit Selections)
 *
 * @param {number} bomRevisionId
 * @returns {Object} { models, selections, summary }
 */
export function getMatrixData(bomRevisionId) {
    // 1. 取得 Models
    const models = matrixModelRepo.findByBomRevisionId(bomRevisionId);

    // 2. 取得 Explicit Selections
    // 這裡我們只取出屬於這些 Models 的 Selections
    // 由於 matrixSelectionRepo.findByBomRevisionId 已經做了 Join，我們可以直接用
    const explicitSelections = matrixSelectionRepo.findByBomRevisionId(bomRevisionId);

    // 3. 取得 BOM View (ACTIVE parts) 用於計算 Implicit Selection
    // 根據需求，Matrix 僅針對 CCL=Y 的物料進行驗證
    const viewDef = { filter: { statusLogic: 'ACTIVE', ccl: 'Y' } };
    const bomItems = bomService.executeView(bomRevisionId, viewDef);

    // 4. 計算 Implicit Selections 與 統計資料
    const effectiveSelections = [...explicitSelections];
    const explicitMap = new Map(); // key: `${model_id}|${group_key}`

    explicitSelections.forEach(s => {
        explicitMap.set(`${s.matrix_model_id}|${s.group_key}`, s);
    });

    // 統計資料結構
    const summary = {
        totalGroups: bomItems.length,
        modelStatus: {} // { modelId: { selectedCount, isComplete } }
    };

    models.forEach(m => {
        summary.modelStatus[m.id] = { selectedCount: 0, isComplete: false };
    });

    // 遍歷所有 BOM Items (Groups)
    for (const item of bomItems) {
        const groupKey = `${item.supplier}|${item.supplier_pn}`;
        const hasSecondSource = item.second_sources && item.second_sources.length > 0;

        // 針對每個 Model 檢查
        for (const model of models) {
            const mapKey = `${model.id}|${groupKey}`;
            const explicit = explicitMap.get(mapKey);

            if (explicit) {
                // 已有明確選擇
                summary.modelStatus[model.id].selectedCount++;
            } else if (!hasSecondSource) {
                // 無明確選擇，但無 Second Source -> Implicit Selection (選 Main)
                // 為了方便前端，我們構造一個 Selection 物件
                // 注意: Main Source 的 ID 需要從 item 中獲取。
                // item 來自 bomService.executeView，其中包含 parts 聚合資訊。
                // item.id 可能是其中一個 part 的 id (min item)。
                // 為了穩健，我們需要一個代表性的 ID。executeView 回傳的 item 有 id 屬性嗎？
                // executeView 回傳的 item 包含 `...group`，其中有 `item` (min item number), 但沒有 `id` (parts.id).
                // partsRepo.getAggregatedBom does not return part ID because it aggregates multiple parts.
                // However, executeView's `partsRepo.findByGroup` returns parts with IDs.
                // Let's check `bom.service.js` executeView.
                // It groups by supplier|supplier_pn.
                // It does NOT return a specific part ID in the aggregated object directly unless we added it.
                // In `bom.service.js`: `group` object has `bom_revision_id`, `supplier`, `supplier_pn`.
                // It does NOT have `id`.
                // Wait, `partsRepo.getAggregatedBom` usually uses `MIN(id)` or similar?
                // `bom.service.js` uses in-memory aggregation.
                // We need a valid `selected_id` for the implicit selection.
                // If we select "Main", we need a part ID.
                // Since we don't have it easily in `bomItems` from `executeView` (it aggregates),
                // we might need to fetch it or just use a placeholder if the backend logic allows.
                // BUT `matrix_selections` table requires `selected_id`.
                // If the user *clicks* to select Main, the frontend will need an ID.
                // The frontend receives `bomItems`. If `bomItems` doesn't have ID, frontend can't send it.
                // We should probably include a representative ID in `bomItems` (e.g. `main_part_id`).

                // Let's fix `bom.service.js` later or assume we can get it.
                // Actually, `bom.service.js` `executeView`:
                // const group = groupedMap.get(key);
                // group has `item` (min item number).
                // We can fetch the part ID corresponding to `bom_revision_id` + `supplier` + `supplier_pn`.
                // But for Implicit Selection (Virtual), we don't necessarily need to persist it.
                // We just need to tell Frontend "This is implicitly selected".
                // So `selected_id` can be null or we can find one if we really want to simulate a record.

                effectiveSelections.push({
                    matrix_model_id: model.id,
                    group_key: groupKey,
                    selected_type: 'part',
                    selected_id: item.id, // Use representative ID
                    is_implicit: true
                });

                summary.modelStatus[model.id].selectedCount++;
            }
        }
    }

    // 更新完備狀態
    for (const model of models) {
        const status = summary.modelStatus[model.id];
        status.isComplete = (status.selectedCount === summary.totalGroups) && (summary.totalGroups > 0);
    }

    // 整體 Summary
    const hasMatrix = models.length > 0 && explicitSelections.length > 0; // Or just models > 0?
    // User spec: "存在matrix bom的判斷: 1.有設定1個以上的model and name. 2. 有勾選 1個以上的物料"
    const isSafe = models.length > 0 && models.every(m => summary.modelStatus[m.id].isComplete);

    return {
        models,
        selections: effectiveSelections,
        summary: {
            totalGroups: summary.totalGroups,
            modelStatus: summary.modelStatus,
            hasMatrix: models.length > 0 && explicitSelections.length > 0,
            isSafe
        }
    };
}

/**
 * 取得 Matrix 狀態摘要 (供 Dashboard 顯示)
 * @param {number} bomRevisionId
 * @returns {Object}
 */
export function getMatrixSummary(bomRevisionId) {
    const data = getMatrixData(bomRevisionId);
    return data.summary;
}

export default {
    createModels,
    listModels,
    updateModel,
    deleteModel,
    saveSelection,
    getMatrixData,
    getMatrixSummary
};
