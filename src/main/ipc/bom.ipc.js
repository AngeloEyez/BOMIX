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
    ipcMain.handle('bom:getView', withErrorHandling((bomRevisionId, viewId) => {
        // If viewId is provided, use executeView with factory definition
        // If not (legacy or defaut), use original getBomView (which is basically ALL view Aggregation)
        // Actually, to be safe and consistent, let's switch to using factory if viewId is present.
        
        if (viewId && viewId !== 'all_view') {
             const viewDef = bomFactory.getViewDefinition(viewId);
             return bomService.executeView(bomRevisionId, viewDef);
        } else {
             // For 'all_view' or undefined, we can still use getBomView or executeView with ALL definition.
             // executeView is safer as it uses the same logic.
             // But getBomView uses partsRepo.getAggregatedBom which might be faster (SQL group by) than executeView (JS group by)?
             // Wait, executeView uses findByBomRevision and then JS group by.
             // partsRepo.getAggregatedBom uses SQL GROUP BY.
             // SQL is likely faster. 
             // However, for consistency of filtering (like excluding 'X' status for ALL view), we should use executeView OR update getAggregatedBom to match.
             // The factory definition for ALL is { statusLogic: 'ACTIVE' }.
             // getBomView (SQL) doesn't filter status! It retrieves EVERYTHING.
             // So getBomView = RAW view, not ALL view (which implies Active items).
             // If we want "ALL" to match the Button "ALL", we should use executeView with ALL definition.
             
             if (viewId === 'all_view') {
                 const viewDef = bomFactory.VIEWS.ALL;
                 return bomService.executeView(bomRevisionId, viewDef);
             }
             
             // Fallback for calls without viewId (e.g. initial load if old code) -> Return Raw
             return bomService.getBomView(bomRevisionId);
        }
    }));

    // 取得所有 BOM View 定義 (Factory)
    ipcMain.handle('bom:get-views', withErrorHandling(() => {
        // Return values of VIEWS object (or just the object itself, but frontend likely wants an array or similar)
        // Let's return the full object for flexible access, or convert to array if needed.
        // Plan said: "回傳 bom-factory.service.js 中的 VIEWS 物件"
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
