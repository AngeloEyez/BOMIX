/**
 * @file src/main/ipc/bom.ipc.js
 * @description BOM 管理 (BOM Management) IPC 通道處理器
 * @module ipc/bom
 */

import bomService from '../services/bom.service.js';
import bomRevisionRepo from '../database/repositories/bom-revision.repo.js';
import bomFactory, { VIEWS } from '../services/bom-factory.service.js';

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

    // 取得專案下的所有 BOM 版本
    ipcMain.handle('bom:getRevisions', withErrorHandling((projectId) => {
        return bomRevisionRepo.findByProject(projectId);
    }));
    /**
     * 通用 BOM 資料查詢
     * 接受 bomIds 陣列、filters 條件陣列與 options 物件。
     * 格式說明詳見 dev/FILTER_SPEC.md。
     */
    ipcMain.handle('bom:query', withErrorHandling((bomIds, filters, options) => {
        return bomService.queryBomData(bomIds, filters || [], options || {});
    }));

    /**
     * @deprecated 此通道保留供向下相容，前端請改用 `bom:query`。
     *             當渲染層所有呼叫點完成遷移後可評估移除。
     */
    ipcMain.handle('bom:getView', withErrorHandling((bomRevisionId, viewId) => {
        if (viewId && viewId !== 'all_view') {
             const viewDef = bomFactory.getViewDefinition(viewId);
             return bomService.executeView(bomRevisionId, viewDef);
        } else {
             if (viewId === 'all_view') {
                 const viewDef = bomFactory.VIEWS.ALL;
                 return bomService.executeView(bomRevisionId, viewDef);
             }
             // Fallback: 不帶 viewId 時回傳原始聚合視圖
             return bomService.getBomView(bomRevisionId);
        }
    }));

    // 取得所有 BOM View 定義 (Factory)，含 filters 陣列格式
    ipcMain.handle('bom:get-views', withErrorHandling(() => {
        return VIEWS;
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

    // 更新 BOM Revision (Metadata)
    ipcMain.handle('bom:updateRevision', withErrorHandling((id, updates) => {
        return bomService.updateBomRevision(id, updates);
    }));
}
