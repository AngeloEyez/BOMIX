/**
 * @file src/main/database/repositories/second-source.repo.js
 * @description 替代料 (Second Source) 資料存取層
 * @module database/repositories/second-source
 */

import dbManager from '../connection.js';

/**
 * 建立替代料
 * @param {Object} data - 替代料資料
 * @returns {Object} 建立的替代料物件
 */
function create(data) {
  const db = dbManager.getDb();
  const {
    bom_revision_id,
    main_supplier,
    main_supplier_pn,
    hhpn,
    supplier,
    supplier_pn,
    description
  } = data;

  const stmt = db.prepare(`
    INSERT INTO second_sources (
      bom_revision_id, main_supplier, main_supplier_pn,
      hhpn, supplier, supplier_pn, description
    )
    VALUES (?, ?, ?, ?, ?, ?, ?)
    RETURNING *
  `);

  return stmt.get(
    bom_revision_id,
    main_supplier,
    main_supplier_pn,
    hhpn || null,
    supplier,
    supplier_pn,
    description || null
  );
}

/**
 * 批量建立替代料
 * @param {Array<Object>} dataArray - 替代料資料陣列
 */
function createMany(dataArray) {
  const db = dbManager.getDb();
  const insert = db.prepare(`
    INSERT INTO second_sources (
      bom_revision_id, main_supplier, main_supplier_pn,
      hhpn, supplier, supplier_pn, description
    )
    VALUES (@bom_revision_id, @main_supplier, @main_supplier_pn,
            @hhpn, @supplier, @supplier_pn, @description)
  `);

  const transaction = db.transaction((items) => {
    for (const item of items) {
      insert.run({
        bom_revision_id: item.bom_revision_id,
        main_supplier: item.main_supplier,
        main_supplier_pn: item.main_supplier_pn,
        hhpn: item.hhpn || null,
        supplier: item.supplier,
        supplier_pn: item.supplier_pn,
        description: item.description || null
      });
    }
  });

  transaction(dataArray);
}

/**
 * 取得指定 BOM 版本的所有替代料
 * @param {number} bomRevisionId - BOM 版本 ID
 * @returns {Array<Object>} 替代料列表
 */
function findByBomRevision(bomRevisionId) {
  const db = dbManager.getDb();
  const stmt = db.prepare('SELECT * FROM second_sources WHERE bom_revision_id = ?');
  return stmt.all(bomRevisionId);
}

/**
 * 刪除指定 BOM 版本的所有替代料
 * @param {number} bomRevisionId - BOM 版本 ID
 */
function deleteByBomRevision(bomRevisionId) {
  const db = dbManager.getDb();
  const stmt = db.prepare('DELETE FROM second_sources WHERE bom_revision_id = ?');
  stmt.run(bomRevisionId);
}

/**
 * 更新替代料
 * @param {number} id - 替代料 ID
 * @param {Object} data - 更新資料
 * @returns {Object|undefined} 更新後的替代料物件
 */
function update(id, data) {
  const db = dbManager.getDb();
  const updates = [];
  const params = [];
  const allowedFields = ['hhpn', 'supplier', 'supplier_pn', 'description', 'main_supplier', 'main_supplier_pn'];

  for (const field of allowedFields) {
    if (data[field] !== undefined) {
      updates.push(`${field} = ?`);
      params.push(data[field]);
    }
  }

  if (updates.length === 0) {
    return db.prepare('SELECT * FROM second_sources WHERE id = ?').get(id);
  }

  params.push(id);
  const sql = `UPDATE second_sources SET ${updates.join(', ')} WHERE id = ? RETURNING *`;
  return db.prepare(sql).get(...params);
}

/**
 * 刪除替代料
 * @param {number} id - 替代料 ID
 * @returns {boolean} 是否刪除成功
 */
function deleteSecondSource(id) {
  const db = dbManager.getDb();
  const stmt = db.prepare('DELETE FROM second_sources WHERE id = ?');
  const result = stmt.run(id);
  return result.changes > 0;
}

export default {
  create,
  createMany,
  findByBomRevision,
  deleteByBomRevision,
  update,
  delete: deleteSecondSource
};
