/**
 * @file src/main/ipc/progress.ipc.js
 * @description 進度追蹤 IPC 通道處理器 (Progress IPC Handler)
 * @module ipc/progress
 */

import { BrowserWindow } from 'electron';
import progressService, { TASK_STATUS } from '../services/progress.service.js';

/**
 * 註冊進度追蹤相關的 IPC 通道
 * @param {Electron.IpcMain} ipcMain - Electron IPC 主行程實例
 */
export function registerProgressIpc(ipcMain) {
    /**
     * 通用錯誤處理包裝函數
     */
    const withErrorHandling = (handler) => {
        return async (event, ...args) => {
            try {
                const result = await handler(...args);
                return { success: true, data: result };
            } catch (error) {
                console.error(`[Progress IPC Error] ${error.message}`);
                return { success: false, error: error.message };
            }
        };
    };

    // === Handlers ===

    // 取得指定任務的狀態
    ipcMain.handle('progress:get', withErrorHandling((taskId) => {
        const task = progressService.getTask(taskId);
        if (!task) {
            throw new Error(`找不到 ID 為 ${taskId} 的任務`);
        }
        return task;
    }));

    // 取消指定任務
    ipcMain.handle('progress:cancel', withErrorHandling((taskId) => {
        progressService.cancelTask(taskId);
        return { cancelled: true };
    }));


    // === Event Broadcasters ===

    // 當任務進度更新時，通知所有渲染視窗
    progressService.on('task:update', (task) => {
        broadcast('progress:update', task);
    });

    // 當任務完成時，通知所有渲染視窗 (Optional: 如果 'task:update' 已包含完成狀態，這可能多餘，但可作為明確訊號)
    // progressService.on('task:complete', (task) => {
    //     broadcast('progress:complete', task);
    // });

    // 當任務錯誤時
    // progressService.on('task:error', (task) => {
    //     broadcast('progress:error', task);
    // });
}

/**
 * 廣播訊息給所有視窗
 * @param {string} channel
 * @param {any} data
 */
function broadcast(channel, data) {
    const windows = BrowserWindow.getAllWindows();
    windows.forEach(win => {
        if (!win.isDestroyed()) {
            win.webContents.send(channel, data);
        }
    });
}
