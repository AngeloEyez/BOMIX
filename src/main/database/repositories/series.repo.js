/**
 * @file src/main/database/repositories/series.repo.js
 * @description 系列元資料 (Series Meta) 資料存取層
 * @module database/repositories/series
 */

import dbManager from '../connection.js';

/**
 * 取得系列元資料
 * @returns {Object|undefined} 系列資訊物件 (含 id, description, created_at, updated_at)
 */
function getMeta() {
  const db = dbManager.getDb();
  const stmt = db.prepare('SELECT * FROM series_meta WHERE id = 1');
  return stmt.get();
}

/**
 * 更新系列元資料
 * @param {Object} data - 更新資料
 * @param {string} [data.description] - 系列描述
 * @returns {Object} 更新後的系列資訊
 */
function updateMeta(data) {
  const db = dbManager.getDb();
  const { description } = data;

  const stmt = db.prepare(`
    UPDATE series_meta
    SET description = COALESCE(?, description),
        updated_at = CURRENT_TIMESTAMP
    WHERE id = 1
    RETURNING *
  `);

  return stmt.get(description);
}

/**
 * 初始化系列元資料 (通常由 Schema 建立時自動處理，此為備用)
 */
function initMeta() {
  const db = dbManager.getDb();
  const stmt = db.prepare('INSERT OR IGNORE INTO series_meta (id, description) VALUES (1, ?)');
  stmt.run('Default Series');
}

export default {
  getMeta,
  updateMeta,
  initMeta
};
