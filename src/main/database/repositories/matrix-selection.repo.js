/**
 * @file src/main/database/repositories/matrix-selection.repo.js
 * @description Matrix Selection 資料存取層
 * @module database/repositories/matrix-selection
 */

import dbManager from '../connection.js';

/**
 * 建立或更新 Matrix Selection (Upsert)
 * @param {Object} selectionData
 * @param {number} selectionData.matrix_model_id
 * @param {string} selectionData.group_key
 * @param {string} selectionData.selected_type
 * @param {number} selectionData.selected_id
 * @returns {Object} 建立或更新的 Selection 物件
 */
function upsert(selectionData) {
    const db = dbManager.getDb();

    // SQLite upsert syntax: INSERT INTO ... ON CONFLICT(matrix_model_id, group_key) DO UPDATE SET ...
    const sql = `
        INSERT INTO matrix_selections (matrix_model_id, group_key, selected_type, selected_id)
        VALUES (@matrix_model_id, @group_key, @selected_type, @selected_id)
        ON CONFLICT(matrix_model_id, group_key)
        DO UPDATE SET
            selected_type = excluded.selected_type,
            selected_id = excluded.selected_id
    `;

    const stmt = db.prepare(sql);
    stmt.run(selectionData);

    // Return the updated record
    return findByModelAndGroup(selectionData.matrix_model_id, selectionData.group_key);
}

/**
 * 根據 Model ID 與 Group Key 查詢 Selection
 * @param {number} matrixModelId
 * @param {string} groupKey
 * @returns {Object|undefined}
 */
function findByModelAndGroup(matrixModelId, groupKey) {
    const db = dbManager.getDb();
    const stmt = db.prepare('SELECT * FROM matrix_selections WHERE matrix_model_id = ? AND group_key = ?');
    return stmt.get(matrixModelId, groupKey);
}

/**
 * 根據 Matrix Model ID 查詢所有 Selection
 * @param {number} matrixModelId
 * @returns {Array<Object>}
 */
function findByModelId(matrixModelId) {
    const db = dbManager.getDb();
    const stmt = db.prepare('SELECT * FROM matrix_selections WHERE matrix_model_id = ?');
    return stmt.all(matrixModelId);
}

/**
 * 根據 BOM Revision ID 查詢所有相關 Selection (跨 Model)
 * @param {number} bomRevisionId
 * @returns {Array<Object>} Returns selections with model_id info
 */
function findByBomRevisionId(bomRevisionId) {
    const db = dbManager.getDb();
    const sql = `
        SELECT ms.*
        FROM matrix_selections ms
        JOIN matrix_models mm ON ms.matrix_model_id = mm.id
        WHERE mm.bom_revision_id = ?
    `;
    const stmt = db.prepare(sql);
    return stmt.all(bomRevisionId);
}

/**
 * 根據多個 BOM Revision ID 查詢所有相關 Selection
 * @param {Array<number>} bomRevisionIds
 * @returns {Array<Object>} Returns selections with model_id info
 */
function findByBomRevisionIds(bomRevisionIds) {
    const db = dbManager.getDb();
    if (!bomRevisionIds || bomRevisionIds.length === 0) return [];

    const placeholders = bomRevisionIds.map(() => '?').join(',');
    const sql = `
        SELECT ms.*
        FROM matrix_selections ms
        JOIN matrix_models mm ON ms.matrix_model_id = mm.id
        WHERE mm.bom_revision_id IN (${placeholders})
    `;
    const stmt = db.prepare(sql);
    return stmt.all(...bomRevisionIds);
}

/**
 * 計算指定 Model 下的 Selection 數量
 * @param {number} matrixModelId
 * @returns {number}
 */
function countByModelId(matrixModelId) {
    const db = dbManager.getDb();
    const stmt = db.prepare('SELECT COUNT(*) as count FROM matrix_selections WHERE matrix_model_id = ?');
    const result = stmt.get(matrixModelId);
    return result ? result.count : 0;
}

/**
 * 刪除指定 Model 的所有 Selection
 * @param {number} matrixModelId
 * @returns {boolean}
 */
function deleteByModelId(matrixModelId) {
    const db = dbManager.getDb();
    const stmt = db.prepare('DELETE FROM matrix_selections WHERE matrix_model_id = ?');
    const info = stmt.run(matrixModelId);
    return info.changes > 0;
}

/**
 * 刪除指定 Selection (依據 Model ID 與 Group Key)
 * @param {number} matrixModelId
 * @param {string} groupKey
 * @returns {boolean}
 */
function deleteByModelAndGroup(matrixModelId, groupKey) {
    const db = dbManager.getDb();
    const stmt = db.prepare('DELETE FROM matrix_selections WHERE matrix_model_id = ? AND group_key = ?');
    const info = stmt.run(matrixModelId, groupKey);
    return info.changes > 0;
}

export default {
    upsert,
    findByModelId,
    findByModelAndGroup,
    findByBomRevisionId,
    findByBomRevisionIds,
    countByModelId,
    deleteByModelId,
    deleteByModelAndGroup
};
