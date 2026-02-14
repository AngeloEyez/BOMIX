/**
 * @file src/main/ipc/bom.ipc.js
 * @description BOM 管理 (BOM Management) IPC 通道處理器
 * @module ipc/bom
 */

import bomService from '../services/bom.service.js';

/**
 * 註冊 BOM 管理相關的 IPC 通道
 * @param {Electron.IpcMain} ipcMain - Electron IPC 主行程實例
 */
export function registerBomIpc(ipcMain) {
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

    // 取得 BOM 視圖
    ipcMain.handle('bom:getView', withErrorHandling((bomRevisionId) => {
        return bomService.getBomView(bomRevisionId);
    }));

    // 更新 Main Item (整組)
    ipcMain.handle('bom:updateMainItem', withErrorHandling((bomRevisionId, originalKey, updates) => {
        return bomService.updateMainItem(bomRevisionId, originalKey, updates);
    }));

    // 刪除 Main Item (整組)
    ipcMain.handle('bom:deleteMainItem', withErrorHandling((bomRevisionId, key) => {
        return bomService.deleteMainItem(bomRevisionId, key);
    }));

    // 新增 Second Source
    ipcMain.handle('bom:addSecondSource', withErrorHandling((data) => {
        return bomService.addSecondSource(data);
    }));

    // 更新 Second Source
    ipcMain.handle('bom:updateSecondSource', withErrorHandling((id, data) => {
        return bomService.updateSecondSource(id, data);
    }));

    // 刪除 Second Source
    ipcMain.handle('bom:deleteSecondSource', withErrorHandling((id) => {
        return bomService.deleteSecondSource(id);
    }));

    // 刪除 BOM Revision
    ipcMain.handle('bom:delete', withErrorHandling((bomRevisionId) => {
        return bomService.deleteBom(bomRevisionId);
    }));
}
