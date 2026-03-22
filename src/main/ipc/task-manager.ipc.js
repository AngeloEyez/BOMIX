/**
 * @file src/main/ipc/task-manager.ipc.js
 * @description 任務排程管理器 IPC 通道處理器 (Task Manager IPC Handler)
 *
 * 取代舊版 progress.ipc.js，提供任務查詢、取消、移除、佇列狀態等 IPC 通道，
 * 並將任務更新與完成事件廣播至所有渲染視窗。
 *
 * @module ipc/task-manager
 */

import { BrowserWindow } from 'electron';
import taskManager from '../services/task-manager.service.js';

/**
 * 註冊任務排程管理器相關的 IPC 通道。
 *
 * @param {Electron.IpcMain} ipcMain - Electron IPC 主行程實例
 */
export function registerTaskManagerIpc(ipcMain) {
    /**
     * 通用錯誤處理包裝函數
     * @param {Function} handler - 實際處理函數
     * @returns {Function} 包裝後的處理函數
     */
    const withErrorHandling = (handler) => {
        return async (event, ...args) => {
            try {
                const result = await handler(...args);
                return { success: true, data: result };
            } catch (error) {
                console.error(`[TaskManager IPC Error] ${error.message}`);
                return { success: false, error: error.message };
            }
        };
    };

    // ========================================
    // IPC Handlers (Request-Response)
    // ========================================

    // 取得指定任務的狀態
    ipcMain.handle('task:get', withErrorHandling((taskId) => {
        const task = taskManager.getTask(taskId);
        if (!task) {
            throw new Error(`找不到 ID 為 ${taskId} 的任務`);
        }
        return task;
    }));

    // 取消指定任務（僅 QUEUED 狀態可取消）
    ipcMain.handle('task:cancel', withErrorHandling((taskId) => {
        const cancelled = taskManager.cancelTask(taskId);
        if (!cancelled) {
            throw new Error('無法取消此任務（僅排隊中的任務可取消）');
        }
        return { cancelled: true };
    }));

    // 移除任務紀錄（僅已結束的任務可移除）
    ipcMain.handle('task:remove', withErrorHandling((taskId) => {
        const removed = taskManager.removeTask(taskId);
        if (!removed) {
            throw new Error('無法移除此任務（執行中或排隊中的任務不可移除）');
        }
        return { removed: true };
    }));

    // 取得佇列狀態概覽
    ipcMain.handle('task:getQueueStatus', withErrorHandling(() => {
        return taskManager.getQueueStatus();
    }));

    // ========================================
    // Event Broadcasters (Main → Renderer)
    // ========================================

    // 任務狀態更新事件 → 廣播至所有渲染視窗
    taskManager.on('task:update', (task) => {
        broadcast('task:update', task);
    });

    // 任務完成事件 → 廣播至所有渲染視窗（用於觸發 UI callback）
    taskManager.on('task:completed', (data) => {
        broadcast('task:completed', data);
    });
}

/**
 * 廣播訊息給所有視窗。
 *
 * @param {string} channel - IPC 通道名稱
 * @param {any} data - 傳送的資料
 */
function broadcast(channel, data) {
    const windows = BrowserWindow.getAllWindows();
    windows.forEach(win => {
        if (!win.isDestroyed()) {
            win.webContents.send(channel, data);
        }
    });
}
