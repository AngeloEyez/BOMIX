/**
 * @file src/main/ipc/excel.ipc.js
 * @description Excel 匯入/匯出 (Excel Import/Export) IPC 通道處理器
 *
 * 匯入與匯出皆透過 TaskManager 排程，確保 FIFO 依序執行且不阻塞 UI。
 *
 * @module ipc/excel
 */

import importService from '../services/import.service.js';
import importBatchService from '../services/import-batch.service.js';
import { runExport } from '../services/export.service.js';
import taskManager from '../services/task-manager.service.js';

/**
 * 註冊 Excel 相關的 IPC 通道。
 *
 * @param {Electron.IpcMain} ipcMain - Electron IPC 主行程實例
 */
export function registerExcelIpc(ipcMain) {
    /**
     * 通用錯誤處理包裝函數
     * @param {Function} handler - 實際處理函數
     * @returns {Function} 包裝後的處理函數
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

    // ========================================
    // 匯入 Excel — 透過 TaskManager 排程
    // 支援傳入多檔案路徑陣列，轉交 BatchImportService 排序與排程
    // ========================================
    ipcMain.handle('excel:import', withErrorHandling((filePaths) => {
        // 先確認是否為陣列，若是單個字串則轉為陣列以保持相容性
        const paths = Array.isArray(filePaths) ? filePaths : [filePaths];
        const taskId = importBatchService.enqueueBatchImport(paths);
        return { taskId };
    }));

    // ========================================
    // 匯出 Excel — 透過 TaskManager 排程
    // 匯出為非同步操作，原生支援進度追蹤。
    // ========================================
    ipcMain.handle('excel:export', withErrorHandling((bomRevisionId, outputFilePath) => {
        const taskId = taskManager.enqueue('EXPORT_BOM', {
            title: '匯出 BOM Excel',
            metadata: { bomRevisionId, outputFilePath },
            executeFn: async (ctx) => {
                return await runExport(ctx, bomRevisionId, outputFilePath);
            }
        });
        return { taskId };
    }));
}
