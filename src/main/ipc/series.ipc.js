/**
 * @file src/main/ipc/series.ipc.js
 * @description 系列管理 (Series) IPC 通道處理器
 * @module ipc/series
 */

import seriesService from '../services/series.service.js';

/**
 * 註冊系列管理相關的 IPC 通道
 * @param {Electron.IpcMain} ipcMain - Electron IPC 主行程實例
 */
export function registerSeriesIpc(ipcMain) {
    /**
     * 通用錯誤處理包裝函數
     * @param {Function} handler - 業務邏輯函數
     * @returns {Function} IPC 處理函數
     */
    const withErrorHandling = (handler) => {
        return async (event, ...args) => {
            try {
                // 執行業務邏輯 (支援同步與非同步)
                const data = await handler(...args);
                return { success: true, data };
            } catch (error) {
                console.error(`[IPC Error] ${error.message}`);
                return { success: false, error: error.message };
            }
        };
    };

    // 註冊通道
    ipcMain.handle('series:create', withErrorHandling((filePath, description) => {
        return seriesService.createSeries(filePath, description);
    }));

    ipcMain.handle('series:open', withErrorHandling((filePath) => {
        return seriesService.openSeries(filePath);
    }));

    ipcMain.handle('series:getMeta', withErrorHandling(() => {
        return seriesService.getSeriesMeta();
    }));

    ipcMain.handle('series:updateMeta', withErrorHandling((description) => {
        return seriesService.updateSeriesMeta(description);
    }));

    ipcMain.handle('series:rename', withErrorHandling((newName) => {
        return seriesService.renameSeries(newName);
    }));
}
