/**
 * @file src/main/ipc/excel.ipc.js
 * @description Excel 匯入/匯出 (Excel Import/Export) IPC 通道處理器
 * @module ipc/excel
 */

import importService from '../services/import.service.js';
import exportService from '../services/export.service.js';

/**
 * 註冊 Excel 相關的 IPC 通道
 * @param {Electron.IpcMain} ipcMain - Electron IPC 主行程實例
 */
export function registerExcelIpc(ipcMain) {
    /**
     * 通用錯誤處理包裝函數
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

    // 匯入 Excel
    ipcMain.handle('excel:import', withErrorHandling((filePath, projectId, phaseName, version) => {
        return importService.importBom(filePath, projectId, phaseName, version);
    }));

    // 匯出 Excel
    ipcMain.handle('excel:export', withErrorHandling((bomRevisionId, outputFilePath) => {
        return exportService.exportBom(bomRevisionId, outputFilePath);
    }));
}
