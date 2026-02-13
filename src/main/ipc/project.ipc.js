/**
 * @file src/main/ipc/project.ipc.js
 * @description 專案管理 (Project) IPC 通道處理器
 * @module ipc/project
 */

import { createRequire } from 'module';
const require = createRequire(import.meta.url);
const projectService = require('../services/project.service.js');

/**
 * 註冊專案管理相關的 IPC 通道
 * @param {Electron.IpcMain} ipcMain - Electron IPC 主行程實例
 */
export function registerProjectIpc(ipcMain) {
    /**
     * 通用錯誤處理包裝函數
     * @param {Function} handler - 業務邏輯函數
     * @returns {Function} IPC 處理函數
     */
    const withErrorHandling = (handler) => {
        return async (event, ...args) => {
            try {
                const data = await handler(...args);
                return { success: true, data };
            } catch (error) {
                console.error(`[IPC Error] ${error.message}`);
                return { success: false, error: error.message };
            }
        };
    };

    // 註冊通道
    ipcMain.handle('project:create', withErrorHandling((projectCode, description) => {
        return projectService.createProject(projectCode, description);
    }));

    ipcMain.handle('project:getAll', withErrorHandling(() => {
        return projectService.getAllProjects();
    }));

    ipcMain.handle('project:getById', withErrorHandling((id) => {
        return projectService.getProjectById(id);
    }));

    ipcMain.handle('project:update', withErrorHandling((id, description) => {
        return projectService.updateProject(id, description);
    }));

    ipcMain.handle('project:delete', withErrorHandling((id) => {
        return projectService.deleteProject(id);
    }));
}
