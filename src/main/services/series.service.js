/**
 * @file src/main/services/series.service.js
 * @description 系列 (Series) 業務邏輯層
 * @module services/series
 */

import fs from 'fs';
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
function createSeries(filePath, description) {
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
function openSeries(filePath) {
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
 * @returns {Object} 系列資訊
 * @throws {Error} 若尚未開啟系列
 */
function getSeriesMeta() {
    try {
        const meta = seriesRepo.getMeta();
        if (!meta) {
            throw new Error('尚未開啟系列或系列資料損毀');
        }
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
function updateSeriesMeta(description) {
    if (description === undefined || description === null) {
        throw new Error('必須提供描述內容');
    }

    try {
        return seriesRepo.updateMeta({ description });
    } catch (error) {
        throw new Error(`更新系列資訊失敗: ${error.message}`);
    }
}

export default {
    createSeries,
    openSeries,
    getSeriesMeta,
    updateSeriesMeta
};
