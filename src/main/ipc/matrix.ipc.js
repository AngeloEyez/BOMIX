/**
 * @file src/main/ipc/matrix.ipc.js
 * @description Matrix BOM IPC 通訊模組
 * @module ipc/matrix
 */

import matrixService from '../services/matrix.service.js';

/**
 * 註冊 Matrix 相關 IPC Handler
 * @param {Electron.IpcMain} ipcMainInstance
 */
export function registerMatrixIpc(ipcMainInstance) {
    /**
     * @channel matrix:createModels
     * @param {Object} event
     * @param {number} bomRevisionId
     * @param {Array<Object>} [models]
     * @returns {Object} { success, data }
     */
    ipcMainInstance.handle('matrix:createModels', async (event, bomRevisionId, models) => {
        try {
            const result = matrixService.createModels(bomRevisionId, models);
            return { success: true, data: result };
        } catch (error) {
            console.error('[IPC] matrix:createModels error:', error);
            return { success: false, error: error.message };
        }
    });

    /**
     * @channel matrix:listModels
     * @param {Object} event
     * @param {number} bomRevisionId
     * @returns {Object} { success, data }
     */
    ipcMainInstance.handle('matrix:listModels', async (event, bomRevisionId) => {
        try {
            const result = matrixService.listModels(bomRevisionId);
            return { success: true, data: result };
        } catch (error) {
            console.error('[IPC] matrix:listModels error:', error);
            return { success: false, error: error.message };
        }
    });

    /**
     * @channel matrix:updateModel
     * @param {Object} event
     * @param {number} id
     * @param {Object} updates
     * @returns {Object} { success, data }
     */
    ipcMainInstance.handle('matrix:updateModel', async (event, id, updates) => {
        try {
            const result = matrixService.updateModel(id, updates);
            return { success: true, data: result };
        } catch (error) {
            console.error('[IPC] matrix:updateModel error:', error);
            return { success: false, error: error.message };
        }
    });

    /**
     * @channel matrix:deleteModel
     * @param {Object} event
     * @param {number} id
     * @returns {Object} { success, data }
     */
    ipcMainInstance.handle('matrix:deleteModel', async (event, id) => {
        try {
            const result = matrixService.deleteModel(id);
            return { success: true, data: result };
        } catch (error) {
            console.error('[IPC] matrix:deleteModel error:', error);
            return { success: false, error: error.message };
        }
    });

    /**
     * @channel matrix:saveSelection
     * @param {Object} event
     * @param {Object} selectionData
     * @returns {Object} { success, data }
     */
    ipcMainInstance.handle('matrix:saveSelection', async (event, selectionData) => {
        try {
            const result = matrixService.saveSelection(selectionData);
            return { success: true, data: result };
        } catch (error) {
            console.error('[IPC] matrix:saveSelection error:', error);
            return { success: false, error: error.message };
        }
    });

    /**
     * @channel matrix:deleteSelection
     * @param {Object} event
     * @param {number} matrixModelId
     * @param {string} groupKey
     * @returns {Object} { success, data }
     */
    ipcMainInstance.handle('matrix:deleteSelection', async (event, matrixModelId, groupKey) => {
        try {
            const result = matrixService.deleteSelection(matrixModelId, groupKey);
            return { success: true, data: result };
        } catch (error) {
            console.error('[IPC] matrix:deleteSelection error:', error);
            return { success: false, error: error.message };
        }
    });

    /**
     * @channel matrix:getData
     * @param {Object} event
     * @param {number|Array<number>} bomRevisionIdOrIds
     * @returns {Object} { success, data }
     */
    ipcMainInstance.handle('matrix:getData', async (event, bomRevisionIdOrIds) => {
        try {
            const result = matrixService.getMatrixData(bomRevisionIdOrIds);
            return { success: true, data: result };
        } catch (error) {
            console.error('[IPC] matrix:getData error:', error);
            return { success: false, error: error.message };
        }
    });

    /**
     * @channel matrix:getSummary
     * @param {Object} event
     * @param {number} bomRevisionId
     * @returns {Object} { success, data }
     */
    ipcMainInstance.handle('matrix:getSummary', async (event, bomRevisionId) => {
        try {
            const result = matrixService.getMatrixSummary(bomRevisionId);
            return { success: true, data: result };
        } catch (error) {
            console.error('[IPC] matrix:getSummary error:', error);
            return { success: false, error: error.message };
        }
    });
}
