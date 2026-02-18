/**
 * @file src/main/database/repositories/matrix-model.repo.js
 * @description Matrix Model 資料存取層
 * @module database/repositories/matrix-model
 */

import dbManager from '../connection.js';

/**
 * 建立 Matrix Model
 * @param {Object} modelData
 * @param {number} modelData.bom_revision_id
 * @param {string} modelData.name
 * @param {string} [modelData.description]
 * @returns {Object} 建立的 Model 物件
 */
function create(modelData) {
    const db = dbManager.getDb();
    const stmt = db.prepare(`
        INSERT INTO matrix_models (bom_revision_id, name, description)
        VALUES (@bom_revision_id, @name, @description)
    `);

    const info = stmt.run(modelData);
    return findById(info.lastInsertRowid);
}

/**
 * 更新 Matrix Model
 * @param {number} id
 * @param {Object} updates
 * @param {string} [updates.name]
 * @param {string} [updates.description]
 * @returns {Object|null} 更新後的 Model 物件，若無變更或找不到則回傳 null/更新後物件
 */
function update(id, updates) {
    const db = dbManager.getDb();

    // 動態構建 SQL
    const fields = [];
    const params = { id };

    if (updates.name !== undefined) {
        fields.push('name = @name');
        params.name = updates.name;
    }
    if (updates.description !== undefined) {
        fields.push('description = @description');
        params.description = updates.description;
    }

    if (fields.length === 0) return findById(id);

    const sql = `UPDATE matrix_models SET ${fields.join(', ')} WHERE id = @id`;
    const stmt = db.prepare(sql);
    const info = stmt.run(params);

    if (info.changes === 0) return null;
    return findById(id);
}

/**
 * 刪除 Matrix Model
 * @param {number} id
 * @returns {boolean} 是否刪除成功
 */
function deleteModel(id) {
    const db = dbManager.getDb();
    const stmt = db.prepare('DELETE FROM matrix_models WHERE id = ?');
    const info = stmt.run(id);
    return info.changes > 0;
}

/**
 * 根據 ID 查詢 Matrix Model
 * @param {number} id
 * @returns {Object|undefined}
 */
function findById(id) {
    const db = dbManager.getDb();
    const stmt = db.prepare('SELECT * FROM matrix_models WHERE id = ?');
    return stmt.get(id);
}

/**
 * 根據 BOM Revision ID 查詢所有 Matrix Model
 * @param {number} bomRevisionId
 * @returns {Array<Object>}
 */
function findByBomRevisionId(bomRevisionId) {
    const db = dbManager.getDb();
    const stmt = db.prepare('SELECT * FROM matrix_models WHERE bom_revision_id = ? ORDER BY id ASC');
    return stmt.all(bomRevisionId);
}

/**
 * 根據多個 BOM Revision ID 查詢所有 Matrix Model
 * @param {Array<number>} bomRevisionIds
 * @returns {Array<Object>}
 */
function findByBomRevisionIds(bomRevisionIds) {
    const db = dbManager.getDb();
    if (!bomRevisionIds || bomRevisionIds.length === 0) return [];
    const placeholders = bomRevisionIds.map(() => '?').join(',');
    const stmt = db.prepare(`SELECT * FROM matrix_models WHERE bom_revision_id IN (${placeholders}) ORDER BY bom_revision_id ASC, id ASC`);
    return stmt.all(...bomRevisionIds);
}

/**
 * 計算指定 BOM Revision 下的 Matrix Model 數量
 * @param {number} bomRevisionId
 * @returns {number}
 */
function countByBomRevisionId(bomRevisionId) {
    const db = dbManager.getDb();
    const stmt = db.prepare('SELECT COUNT(*) as count FROM matrix_models WHERE bom_revision_id = ?');
    const result = stmt.get(bomRevisionId);
    return result ? result.count : 0;
}

export default {
    create,
    update,
    delete: deleteModel,
    findById,
    findByBomRevisionId,
    findByBomRevisionIds,
    countByBomRevisionId
};
