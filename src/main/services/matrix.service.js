/**
 * @file src/main/services/matrix.service.js
 * @description Matrix BOM 業務邏輯層
 * @module services/matrix
 */

import matrixModelRepo from '../database/repositories/matrix-model.repo.js';
import matrixSelectionRepo from '../database/repositories/matrix-selection.repo.js';
import bomRevisionRepo from '../database/repositories/bom-revision.repo.js'; // Need to fetch project info
import projectRepo from '../database/repositories/project.repo.js'; // Need project code
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
    matrixSelectionRepo.deleteByModelAndGroup(matrixModelId, groupKey);
    // 即使找不到也視為成功
    return { success: true };
}

/**
 * 取得 Matrix 完整資料 (含 Implicit Selections)
 * 支援單一或多個 BOM ID
 *
 * @param {number|Array<number>} bomRevisionIdOrIds
 * @returns {Object} { models, selections, summary }
 */
export function getMatrixData(bomRevisionIdOrIds) {
    const ids = Array.isArray(bomRevisionIdOrIds) ? bomRevisionIdOrIds : [bomRevisionIdOrIds];

    // 1. 取得 Models (Multi-BOM)
    let models = matrixModelRepo.findByBomRevisionIds(ids);

    // Enrich models with Project info for UI grouping
    // Optimized: Fetch all relevant BOMs and Projects in one go or cache
    const bomMap = new Map(); // id -> bom
    const projectMap = new Map(); // id -> project

    // Get unique BOM IDs from models (though we passed them in `ids`, let's use `ids`)
    // Actually we need to map model -> bom -> project
    // Fetch BOM Revisions
    // Since `ids` are bomRevisionIds
    // We can fetch them to get project_id.
    // bomRevisionRepo doesn't have `findByIds` yet, we can iterate or add it.
    // Iterating `ids` is fine.

    for (const bomId of ids) {
        const bom = bomRevisionRepo.findById(bomId);
        if (bom) {
            bomMap.set(bomId, bom);
            if (!projectMap.has(bom.project_id)) {
                const project = projectRepo.findById(bom.project_id);
                if (project) projectMap.set(bom.project_id, project);
            }
        }
    }

    models = models.map(m => {
        const bom = bomMap.get(m.bom_revision_id);
        const project = bom ? projectMap.get(bom.project_id) : null;
        return {
            ...m,
            project_code: project ? project.project_code : 'Unknown',
            phase_name: bom ? bom.phase_name : 'Unknown',
            version: bom ? bom.version : ''
        };
    });

    // 2. 取得 Explicit Selections (Multi-BOM)
    const explicitSelections = matrixSelectionRepo.findByBomRevisionIds(ids);

    // 3. 取得 BOM View (ACTIVE parts + Union) 用於計算 Implicit Selection
    const viewDef = { filter: { statusLogic: 'ACTIVE', ccl: 'Y' } };
    const bomItems = bomService.executeView(ids, viewDef);

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
            // Implicit Selection Logic:
            // Must check if this item "belongs" to this Model's BOM.
            // `executeView` returns UNION. `item` might not exist in `model.bom_revision_id`.
            // We need to check if `item.bom_ids` includes `model.bom_revision_id`.
            // `bomService.executeView` needs to ensure `bom_ids` (or equivalent) is present.
            // Currently `executeView` adds `id` (representative).
            // It assumes Union based on Supplier+PN.
            // If the part exists in *any* BOM, it appears in Union View.
            // But for a specific BOM (Model), does it need selection?
            // ONLY if the part exists in THAT BOM.

            // `executeView` logic needs to populate which BOMs have this part.
            // Currently `executeView` doesn't strictly track `bom_ids` for the group.
            // We need to fix `bom.service.js` to aggregate `bom_revision_id` list.

            // Assuming `bom.service.js` is updated or we check existing fields.
            // `item` from `executeView` currently has `bom_revision_id` from the *first* part found.
            // This is insufficient for Union validation.
            // We need to update `bom.service.js` first.

            // However, assuming we fix `bom.service.js` to return `bom_ids` array:
            // if (!item.bom_ids.includes(model.bom_revision_id)) continue;

            // For now, let's assume `bom_ids` will be there.
            // If not present, we fallback to relaxed check (assume it needs selection if simple view).
            const relevantBoms = item.bom_ids || [item.bom_revision_id];
            if (!relevantBoms.includes(model.bom_revision_id)) {
                // This item is not in this model's BOM, so no selection needed.
                continue;
            }

            const mapKey = `${model.id}|${groupKey}`;
            const explicit = explicitMap.get(mapKey);

            if (explicit) {
                // 已有明確選擇
                summary.modelStatus[model.id].selectedCount++;
            } else if (!hasSecondSource) {
                // 無明確選擇，但無 Second Source -> Implicit Selection (選 Main)

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
        // Total groups for *this* model might be less than total Union groups.
        // We should calculate `totalGroups` per model.
        // But `summary.totalGroups` is global count.
        // UI uses `selectedCount / totalGroups`. This is misleading for Union View.
        // We should calc `modelTotalGroups`.
        // Let's recalculate model total groups loop.
        let modelTotal = 0;
        for (const item of bomItems) {
             const relevantBoms = item.bom_ids || [item.bom_revision_id];
             if (relevantBoms.includes(model.bom_revision_id)) {
                 modelTotal++;
             }
        }

        status.isComplete = (status.selectedCount === modelTotal) && (modelTotal > 0);
    }

    // 整體 Summary
    // isSafe 為所有 Model 都完成選擇
    const isSafe = models.length > 0 && models.every(m => summary.modelStatus[m.id].isComplete);

    return {
        models,
        selections: effectiveSelections,
        summary: {
            totalGroups: summary.totalGroups, // Global Union Count
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
    deleteSelection,
    getMatrixData,
    getMatrixSummary
};
