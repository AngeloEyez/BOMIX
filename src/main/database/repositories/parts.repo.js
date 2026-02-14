/**
 * @file src/main/database/repositories/parts.repo.js
 * @description 零件 (Parts) 資料存取層
 * @module database/repositories/parts
 */

import dbManager from '../connection.js';

/**
 * 建立單一零件
 * @param {Object} data - 零件資料
 * @returns {Object} 建立的零件物件
 */
function create(data) {
  const db = dbManager.getDb();
  const {
    bom_revision_id,
    item,
    hhpn,
    supplier,
    supplier_pn,
    description,
    location,
    type,
    bom_status,
    ccl,
    remark
  } = data;

  const stmt = db.prepare(`
    INSERT INTO parts (
      bom_revision_id, item, hhpn, supplier, supplier_pn,
      description, location, type, bom_status, ccl, remark
    )
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    RETURNING *
  `);

  return stmt.get(
    bom_revision_id,
    item || null,
    hhpn || null,
    supplier,
    supplier_pn,
    description || null,
    location,
    type || null,
    bom_status || 'I',
    ccl || 'N',
    remark || null
  );
}

/**
 * 批量建立零件 (使用交易)
 * @param {Array<Object>} dataArray - 零件資料陣列
 */
function createMany(dataArray) {
  const db = dbManager.getDb();
  const insert = db.prepare(`
    INSERT INTO parts (
      bom_revision_id, item, hhpn, supplier, supplier_pn,
      description, location, type, bom_status, ccl, remark
    )
    VALUES (@bom_revision_id, @item, @hhpn, @supplier, @supplier_pn,
            @description, @location, @type, @bom_status, @ccl, @remark)
  `);

  const transaction = db.transaction((parts) => {
    for (const part of parts) {
      insert.run({
        bom_revision_id: part.bom_revision_id,
        item: part.item || null,
        hhpn: part.hhpn || null,
        supplier: part.supplier,
        supplier_pn: part.supplier_pn,
        description: part.description || null,
        location: part.location,
        type: part.type || null,
        bom_status: part.bom_status || 'I',
        ccl: part.ccl || 'N',
        remark: part.remark || null
      });
    }
  });

  transaction(dataArray);
}

/**
 * 取得指定 BOM 版本的所有零件 (原子化列表)
 * @param {number} bomRevisionId - BOM 版本 ID
 * @returns {Array<Object>} 零件列表
 */
function findByBomRevision(bomRevisionId) {
  const db = dbManager.getDb();
  const stmt = db.prepare('SELECT * FROM parts WHERE bom_revision_id = ?');
  return stmt.all(bomRevisionId);
}

/**
 * 刪除指定 BOM 版本的所有零件
 * @param {number} bomRevisionId - BOM 版本 ID
 */
function deleteByBomRevision(bomRevisionId) {
  const db = dbManager.getDb();
  const stmt = db.prepare('DELETE FROM parts WHERE bom_revision_id = ?');
  stmt.run(bomRevisionId);
}

/**
 * 取得聚合後的 BOM 視圖 (BOM Main Items)
 * 根據 supplier, supplier_pn, type 進行分組
 * @param {number} bomRevisionId - BOM 版本 ID
 * @returns {Array<Object>} 聚合後的 BOM 列表
 */
function getAggregatedBom(bomRevisionId) {
  const db = dbManager.getDb();
  const stmt = db.prepare(`
    SELECT
      bom_revision_id,
      supplier,
      supplier_pn,
      type,
      hhpn,
      description,
      bom_status,
      ccl,
      remark,
      GROUP_CONCAT(location, ',') AS locations,
      COUNT(location) AS quantity,
      MIN(item) as min_item
    FROM parts
    WHERE bom_revision_id = ?
    GROUP BY supplier, supplier_pn, type
    ORDER BY min_item ASC
  `);

  return stmt.all(bomRevisionId);
}

/**
 * 根據 ID 更新零件
 * @param {number} id - 零件 ID
 * @param {Object} data - 更新資料
 * @returns {Object|undefined} 更新後的零件物件
 */
function update(id, data) {
  const db = dbManager.getDb();
  const updates = [];
  const params = [];

  const allowedFields = ['item', 'hhpn', 'supplier', 'supplier_pn', 'description', 'location', 'type', 'bom_status', 'ccl', 'remark'];

  for (const field of allowedFields) {
    if (data[field] !== undefined) {
      updates.push(`${field} = ?`);
      params.push(data[field]);
    }
  }

  if (updates.length === 0) return db.prepare('SELECT * FROM parts WHERE id = ?').get(id);

  params.push(id);
  const sql = `UPDATE parts SET ${updates.join(', ')} WHERE id = ? RETURNING *`;
  return db.prepare(sql).get(...params);
}

/**
 * 根據 ID 刪除零件
 * @param {number} id - 零件 ID
 * @returns {boolean} 是否刪除成功
 */
function deletePart(id) {
  const db = dbManager.getDb();
  const stmt = db.prepare('DELETE FROM parts WHERE id = ?');
  const result = stmt.run(id);
  return result.changes > 0;
}

/**
 * 根據群組條件查找零件
 * @param {number} bomRevisionId
 * @param {string} supplier
 * @param {string} supplier_pn
 * @param {string} type
 * @returns {Array<Object>}
 */
function findByGroup(bomRevisionId, supplier, supplier_pn, type) {
  const db = dbManager.getDb();
  let sql = `
    SELECT * FROM parts
    WHERE bom_revision_id = ?
      AND supplier = ?
      AND supplier_pn = ?
  `;
  const params = [bomRevisionId, supplier, supplier_pn];

  if (type !== undefined && type !== null) {
      sql += ` AND type = ?`;
      params.push(type);
  } else {
      sql += ` AND type IS NULL`;
  }

  return db.prepare(sql).all(...params);
}

export default {
  create,
  createMany,
  findByBomRevision,
  deleteByBomRevision,
  getAggregatedBom,
  update,
  delete: deletePart,
  findByGroup
};
