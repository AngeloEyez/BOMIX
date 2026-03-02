/**
 * @file src/main/ipc/excel.ipc.js
 * @description Excel 匯入/匯出 (Excel Import/Export) IPC 通道處理器
 *
 * 匯入與匯出皆透過 TaskManager 排程，確保 FIFO 依序執行且不阻塞 UI。
 *
 * @module ipc/excel
 */

import importService from '../services/import.service.js';
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
    // 匯入為同步操作，以 setImmediate 包裝為非同步，
    // 分三階段更新進度（解析→處理→儲存），確保不阻塞 UI。
    // ========================================
    ipcMain.handle('excel:import', withErrorHandling((filePath, projectId, phaseName, version, suffix) => {
        const taskId = taskManager.enqueue('IMPORT_BOM', {
            title: `匯入 BOM: ${filePath.split(/[/\\]/).pop()}`,
            metadata: { filePath, projectId, phaseName, version, suffix },
            executeFn: async (ctx) => {
                // 以 setImmediate 包裝同步操作，確保 UI 有機會更新
                await ctx.yield();
                const result = importService.importBom(
                    filePath, projectId, phaseName, version, suffix, ctx
                );
                return result;
            }
        });
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
