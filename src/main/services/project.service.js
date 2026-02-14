/**
 * @file src/main/services/project.service.js
 * @description 專案 (Project) 業務邏輯層
 * @module services/project
 */

import projectRepo from '../database/repositories/project.repo.js';

/**
 * 建立新專案
 *
 * @param {string} projectCode - 專案代碼 (必須唯一)
 * @param {string} [description] - 專案描述
 * @returns {Object} 建立的專案資訊
 * @throws {Error} 若專案代碼已存在或參數無效
 */
function createProject(projectCode, description) {
    if (!projectCode) {
        throw new Error('必須提供專案代碼 (Project Code)');
    }

    try {
        return projectRepo.create({ project_code: projectCode, description });
    } catch (error) {
        if (error.message.includes('UNIQUE constraint failed')) {
            throw new Error(`專案代碼 '${projectCode}' 已存在`);
        }
        throw new Error(`建立專案失敗: ${error.message}`);
    }
}

/**
 * 取得目前系列中的所有專案
 *
 * @returns {Array<Object>} 專案列表
 * @throws {Error} 若查詢失敗
 */
function getAllProjects() {
    try {
        return projectRepo.findAll();
    } catch (error) {
        throw new Error(`取得專案列表失敗: ${error.message}`);
    }
}

/**
 * 根據 ID 取得專案資訊
 *
 * @param {number} id - 專案 ID
 * @returns {Object} 專案資訊
 * @throws {Error} 若專案不存在
 */
function getProjectById(id) {
    try {
        const project = projectRepo.findById(id);
        if (!project) {
            throw new Error(`找不到 ID 為 ${id} 的專案`);
        }
        return project;
    } catch (error) {
        throw new Error(`取得專案資訊失敗: ${error.message}`);
    }
}

/**
 * 更新專案
 *
 * @param {number} id - 專案 ID
 * @param {Object} data - 更新資料
 * @param {string} [data.project_code] - 新的專案代碼
 * @param {string} [data.description] - 新的描述
 * @returns {Object} 更新後的專案資訊
 * @throws {Error} 若更新失敗
 */
function updateProject(id, data) {
    if (!data || Object.keys(data).length === 0) {
        throw new Error('未提供更新資料');
    }

    try {
        const updatedProject = projectRepo.update(id, data);
        if (!updatedProject) {
            throw new Error(`找不到 ID 為 ${id} 的專案`);
        }
        return updatedProject;
    } catch (error) {
        if (error.message.includes('UNIQUE constraint failed')) {
            throw new Error(`專案代碼 '${data.project_code}' 已存在`);
        }
        throw new Error(`更新專案失敗: ${error.message}`);
    }
}

/**
 * 刪除專案
 *
 * @param {number} id - 專案 ID
 * @returns {Object} 刪除結果 { success: true }
 * @throws {Error} 若刪除失敗
 */
function deleteProject(id) {
    try {
        const success = projectRepo.delete(id);
        if (!success) {
            throw new Error(`找不到 ID 為 ${id} 的專案或刪除失敗`);
        }
        return { success: true };
    } catch (error) {
        throw new Error(`刪除專案失敗: ${error.message}`);
    }
}

export default {
    createProject,
    getAllProjects,
    getProjectById,
    updateProject,
    deleteProject
};
