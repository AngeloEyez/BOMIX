/**
 * @file src/main/database/repositories/bom-revision.repo.js
 * @description BOM 版本 (BOM Revision) 資料存取層
 * @module database/repositories/bom-revision
 */

import dbManager from '../connection.js';

/**
 * 建立新 BOM 版本
 * @param {Object} data - BOM 版本資料
 * @param {number} data.project_id - 所屬專案 ID
 * @param {string} data.phase_name - 階段名稱 (如 EVT, DVT)
 * @param {string} data.version - 版本號 (如 0.1, 1.0)
 * @param {string} [data.description] - 描述
 * @param {string} [data.schematic_version] - 線路圖版本
 * @param {string} [data.pcb_version] - PCB 版本
 * @param {string} [data.pca_pn] - PCA 料號
 * @param {string} [data.date] - 日期
 * @param {string} [data.note] - 備註
 * @param {string} [data.mode] - NPI/MP 模式 (預設 'NPI')
 * @returns {Object} 建立的 BOM 版本物件
 */
function create(data) {
  const db = dbManager.getDb();
  const {
    project_id,
    phase_name,
    version,
    description,
    schematic_version,
    pcb_version,
    pca_pn,
    date,
    note,
    mode
  } = data;

  const stmt = db.prepare(`
    INSERT INTO bom_revisions (
      project_id, phase_name, version, description,
      schematic_version, pcb_version, pca_pn, date, note, mode
    )
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    RETURNING *
  `);

  return stmt.get(
    project_id,
    phase_name,
    version,
    description || null,
    schematic_version || null,
    pcb_version || null,
    pca_pn || null,
    date || null,
    note || null,
    mode || 'NPI'
  );
}

/**
 * 取得專案的所有 BOM 版本
 * @param {number} projectId - 專案 ID
 * @returns {Array<Object>} BOM 版本列表，依 Phase 和 Version 排序
 */
function findByProject(projectId) {
  const db = dbManager.getDb();
  const stmt = db.prepare(`
    SELECT * FROM bom_revisions
    WHERE project_id = ?
    ORDER BY phase_name ASC, version DESC
  `);
  return stmt.all(projectId);
}

/**
 * 根據 ID 取得 BOM 版本
 * @param {number} id - BOM 版本 ID
 * @returns {Object|undefined} BOM 版本物件
 */
function findById(id) {
  const db = dbManager.getDb();
  const stmt = db.prepare('SELECT * FROM bom_revisions WHERE id = ?');
  return stmt.get(id);
}

/**
 * 刪除 BOM 版本 (會連帶刪除 parts 和 second_sources，需依賴外鍵約束)
 * @param {number} id - BOM 版本 ID
 * @returns {boolean} 是否刪除成功
 */
function deleteRevision(id) {
  const db = dbManager.getDb();
  const stmt = db.prepare('DELETE FROM bom_revisions WHERE id = ?');
  const result = stmt.run(id);
  return result.changes > 0;
}

export default {
  create,
  findByProject,
  findById,
  delete: deleteRevision
};
