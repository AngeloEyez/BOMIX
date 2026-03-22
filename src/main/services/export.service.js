/**
 * @file src/main/services/export.service.js
 * @description Excel 匯出服務 (Export Service)
 *
 * 負責將 BOM 資料匯出為 Excel 檔案。
 * 匯出邏輯為純業務函數，透過 taskContext 更新進度與日誌，
 * 排程由 TaskManager 統一管理。
 *
 * @module services/export
 */

import { app } from 'electron';
import fs from 'fs/promises';
import path from 'path';
import bomService from './bom.service.js';
import bomRevisionRepo from '../database/repositories/bom-revision.repo.js';
import {
    loadTemplate,
    createWorkbook,
    appendSheetFromTemplate,
    saveWorkbook
} from './excel-export/template-engine.js';
import {
    getExportDefinition,
    getViewDefinition,
    EXPORT_IDS
} from './bom-factory.service.js';

/**
 * 執行 BOM 匯出任務。
 *
 * 此函數由 TaskManager 的 executeFn 呼叫，透過 taskContext 更新進度。
 * 不再直接呼叫 progressService，改以 ctx 注入的方式與 TaskManager 溝通。
 *
 * @param {Object} ctx - TaskManager 提供的任務上下文
 * @param {Function} ctx.updateProgress - 更新進度百分比與訊息
 * @param {Function} ctx.log - 新增日誌
 * @param {Function} ctx.yield - 讓出事件迴圈
 * @param {number} bomRevisionId - BOM 版本 ID
 * @param {string} [outputFilePath] - 輸出檔案路徑
 * @returns {Promise<Object>} 匯出結果 { filePath }
 */
export async function runExport(ctx, bomRevisionId, outputFilePath) {
    ctx.log('開始匯出流程...', 'info');
    ctx.updateProgress(5, '初始化...');

    // 1. 取得 BOM Revision
    const revision = bomRevisionRepo.findById(bomRevisionId);
    if (!revision) {
        throw new Error(`找不到 ID 為 ${bomRevisionId} 的 BOM 版本`);
    }
    ctx.log(`目標版本: ${revision.project_code} - ${revision.phase_name} ${revision.version}`, 'info');

    // 2. 取得 Export Definition
    const exportDef = getExportDefinition(EXPORT_IDS.EBOM);

    // 3. 準備 Meta Data
    const meta = {
        PROJECT_CODE: revision.project_code || revision.project_id || 'Unknown',
        BOM_VERSION: `${revision.version}${revision.suffix ? '-' + revision.suffix : ''}`,
        PHASE: revision.phase_name,
        BOM_DATE: revision.bom_date || new Date().toISOString().split('T')[0],
        SCH_VERSION: revision.schematic_version || '',
        PCB_VERSION: revision.pcb_version || '',
        PCA_PN: revision.pca_pn || '',
        DESCRIPTION: revision.description || ''
    };

    // 4. 初始化 Target Workbook
    const targetWorkbook = createWorkbook();

    // 緩存已載入的 Template Workbooks
    const loadedTemplates = new Map();
    const totalSheets = exportDef.sheets.length;
    let processedSheets = 0;

    // 5. 處理每個 Sheet
    for (const sheetDef of exportDef.sheets) {
        const sheetProgress = Math.round(10 + (processedSheets / totalSheets) * 80);
        ctx.updateProgress(sheetProgress, `處理工作表: ${sheetDef.targetSheetName}`);

        // 讓出事件迴圈，確保 UI 不阻塞
        await ctx.yield();

        // 5.0 載入 Template Workbook (如果尚未載入)
        const templateFile = sheetDef.templateFile;
        if (!loadedTemplates.has(templateFile)) {
            const wb = await loadTemplate(templateFile);
            loadedTemplates.set(templateFile, wb);
        }
        const templateWorkbook = loadedTemplates.get(templateFile);

        const viewDef = getViewDefinition(sheetDef.viewId);

        // 5.1 執行 View 取得資料
        const bomView = bomService.executeView(bomRevisionId, viewDef);

        // 5.2 轉換資料結構 (Mapping to Tags)
        const items = bomView.map(item => {
            const mainTags = {
                M_ITEM: item.item,
                M_HHPN: item.hhpn,
                M_DESC: item.description,
                M_SUPP: item.supplier,
                M_SPN: item.supplier_pn,
                M_QTY: item.quantity,
                M_LOC: item.locations,
                M_REMARK: item.remark,
                M_CCL: item.ccl
            };

            const ssTags = (item.second_sources || []).map(ss => ({
                S_HHPN: ss.hhpn,
                S_DESC: ss.description,
                S_SUPP: ss.supplier,
                S_SPN: ss.supplier_pn
            }));

            return {
                ...mainTags,
                second_sources: ssTags
            };
        });

        // 5.3 寫入 Sheet
        let actualSourceSheetName = sheetDef.sourceSheetName;
        const sourceSheet = templateWorkbook.getWorksheet(actualSourceSheetName);
        if (!sourceSheet) {
             if (templateWorkbook.worksheets.length > 0) {
                 actualSourceSheetName = templateWorkbook.worksheets[0].name;
             } else {
                 ctx.log(`範本工作表 ${sheetDef.sourceSheetName} 找不到，已跳過。`, 'warn');
                 processedSheets++;
                 continue;
             }
        }

        appendSheetFromTemplate(
            targetWorkbook,
            templateWorkbook,
            actualSourceSheetName,
            sheetDef.targetSheetName,
            { meta, items }
        );
        
        ctx.log(`工作表 "${sheetDef.targetSheetName}" 已產生，共 ${items.length} 筆項目。`, 'info');

        processedSheets++;
    }

    ctx.updateProgress(90, '儲存檔案...');
    ctx.log('正在儲存工作簿至暫存檔...', 'info');

    // 6. 存檔 (Temp -> Move)
    const tempDir = app.getPath('temp');
    const tempFileName = `bom_export_${ctx.taskId}_${Date.now()}.xlsx`;
    const tempFilePath = path.join(tempDir, tempFileName);

    await saveWorkbook(targetWorkbook, tempFilePath);

    let finalPath = tempFilePath;

    // 如果有指定輸出路徑，則移動檔案
    if (outputFilePath) {
        ctx.log(`移動檔案至: ${outputFilePath}`, 'info');
        // 使用 copyFile + unlink 以避免跨磁碟區移動檔案時發生 EXDEV 錯誤
        await fs.copyFile(tempFilePath, outputFilePath);
        await fs.unlink(tempFilePath);
        finalPath = outputFilePath;
    }

    return { filePath: finalPath };
}

export default {
    runExport
};
