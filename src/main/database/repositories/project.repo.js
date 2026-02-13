/**
 * @file src/main/database/repositories/project.repo.js
 * @description 專案 (Project) 資料存取層
 * @module database/repositories/project
 */

import dbManager from '../connection.js';

/**
 * 建立新專案
 * @param {Object} data - 專案資料
 * @param {string} data.project_code - 專案代碼 (Unique)
 * @param {string} [data.description] - 專案描述
 * @returns {Object} 建立的專案物件
 */
function create(data) {
  const db = dbManager.getDb();
  const { project_code, description } = data;

  const stmt = db.prepare(`
    INSERT INTO projects (project_code, description)
    VALUES (?, ?)
    RETURNING *
  `);

  return stmt.get(project_code, description || null);
}

/**
 * 取得所有專案
 * @returns {Array<Object>} 專案列表
 */
function findAll() {
  const db = dbManager.getDb();
  const stmt = db.prepare('SELECT * FROM projects ORDER BY created_at DESC, id DESC');
  return stmt.all();
}

/**
 * 根據 ID 取得專案
 * @param {number} id - 專案 ID
 * @returns {Object|undefined} 專案物件
 */
function findById(id) {
  const db = dbManager.getDb();
  const stmt = db.prepare('SELECT * FROM projects WHERE id = ?');
  return stmt.get(id);
}

/**
 * 更新專案
 * @param {number} id - 專案 ID
 * @param {Object} data - 更新資料
 * @param {string} [data.description] - 專案描述
 * @returns {Object|undefined} 更新後的專案物件
 */
function update(id, data) {
  const db = dbManager.getDb();
  const { description } = data;

  const stmt = db.prepare(`
    UPDATE projects
    SET description = COALESCE(?, description),
        updated_at = CURRENT_TIMESTAMP
    WHERE id = ?
    RETURNING *
  `);

  return stmt.get(description, id);
}

/**
 * 刪除專案 (會連帶刪除關聯的 bom_revisions，需依賴外鍵約束)
 * @param {number} id - 專案 ID
 * @returns {boolean} 是否刪除成功
 */
function deleteProject(id) {
  const db = dbManager.getDb();
  const stmt = db.prepare('DELETE FROM projects WHERE id = ?');
  const result = stmt.run(id);
  return result.changes > 0;
}

export default {
  create,
  findAll,
  findById,
  update,
  delete: deleteProject
};
