/**
 * @file src/main/services/series.service.js
 * @description 系列 (Series) 業務邏輯層
 * @module services/series
 */

import fs from 'fs';
import path from 'path';
import dbManager from '../database/connection.js';
import seriesRepo from '../database/repositories/series.repo.js';

/**
 * 建立新的系列資料庫檔案
 *
 * @param {string} filePath - 資料庫檔案路徑 (.bomix)
 * @param {string} [description] - 系列描述
 * @returns {Object} 建立的系列資訊
 * @throws {Error} 若建立失敗
 */
export function createSeries(filePath, description) {
    if (!filePath) {
        throw new Error('必須提供檔案路徑');
    }

    try {
        // 連接資料庫 (若不存在會自動建立並初始化 Schema)
        dbManager.connect(filePath);

        // 更新描述 (Schema 初始化時已插入預設資料)
        if (description) {
            return seriesRepo.updateMeta({ description });
        }

        return seriesRepo.getMeta();
    } catch (error) {
        throw new Error(`建立系列失敗: ${error.message}`);
    }
}

/**
 * 開啟現有的系列資料庫檔案
 *
 * @param {string} filePath - 資料庫檔案路徑 (.bomix)
 * @returns {Object} 系列資訊
 * @throws {Error} 若開啟失敗或檔案無效
 */
export function openSeries(filePath) {
    if (!filePath) {
        throw new Error('必須提供檔案路徑');
    }

    if (!fs.existsSync(filePath)) {
        throw new Error(`找不到檔案: ${filePath}`);
    }

    try {
        // 連接資料庫
        dbManager.connect(filePath);

        // 驗證是否為有效的 BOMIX 資料庫 (嘗試讀取 meta)
        const meta = seriesRepo.getMeta();
        if (!meta) {
            throw new Error('無效的 BOMIX 檔案: 找不到系列元資料');
        }

        return meta;
    } catch (error) {
        throw new Error(`開啟系列失敗: ${error.message}`);
    }
}

/**
 * 取得目前開啟系列的元資料
 *
 * @returns {Object} 系列資訊 (含 bomCount)
 * @throws {Error} 若尚未開啟系列
 */
export function getSeriesMeta() {
    try {
        const meta = seriesRepo.getMeta();
        if (!meta) {
            throw new Error('尚未開啟系列或系列資料損毀');
        }
        // 附加 BOM 統計資訊
        meta.bomCount = seriesRepo.getBomCount();
        return meta;
    } catch (error) {
        throw new Error(`取得系列資訊失敗: ${error.message}`);
    }
}

/**
 * 更新目前系列的描述
 *
 * @param {string} description - 新的描述
 * @returns {Object} 更新後的系列資訊
 * @throws {Error} 若更新失敗
 */
export function updateSeriesMeta(description) {
    if (description === undefined || description === null) {
        throw new Error('必須提供描述內容');
    }

    try {
        const updated = seriesRepo.updateMeta({ description });
        updated.bomCount = seriesRepo.getBomCount(); // 保持資料結構一致
        return updated;
    } catch (error) {
        throw new Error(`更新系列資訊失敗: ${error.message}`);
    }
}

/**
 * 重新命名系列 (即重新命名 .bomix 檔案)
 *
 * @param {string} newName - 新的檔案名稱 (不含路徑與副檔名)
 * @returns {Object} { success: true, newPath: string }
 * @throws {Error} 若命名失敗
 */
export function renameSeries(newName) {
    if (!newName) {
        throw new Error('必須提供新的系列名稱');
    }

    // 驗證檔名 (Windows 規則)
    // 排除 \ / : * ? " < > |
    if (/[\\/:*?"<>|]/.test(newName)) {
        throw new Error('系列名稱包含無效字元');
    }

    try {
        const currentPath = dbManager.currentPath;
        if (!currentPath) {
            throw new Error('尚未開啟任何系列');
        }

        const dir = path.dirname(currentPath);
        const ext = path.extname(currentPath);
        const newPath = path.join(dir, `${newName}${ext}`);

        if (currentPath === newPath) {
            return { success: true, newPath }; // 名稱相同，不做事
        }

        if (fs.existsSync(newPath)) {
            throw new Error(`檔案已存在: ${newName}${ext}`);
        }

        // 1. 關閉資料庫連線 Release lock
        dbManager.disconnect();

        // 2. 重新命名檔案
        fs.renameSync(currentPath, newPath);

        // 3.重新連線
        dbManager.connect(newPath);

        return { success: true, newPath };
    } catch (error) {
        // 嘗試恢復連線 (如果斷開了但重命名失敗)
        try {
            if (dbManager.currentPath && !dbManager.db) {
               dbManager.connect(dbManager.currentPath);
            }
        } catch (_e) { /* ignore */ }
        
        throw new Error(`重新命名失敗: ${error.message}`);
    }
}

export default {
    createSeries,
    openSeries,
    getSeriesMeta,
    updateSeriesMeta,
    renameSeries
};
