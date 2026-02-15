/**
 * @file src/main/services/export.service.js
 * @description Excel 匯出服務 (Export Service)
 * @module services/export
 */

import { app } from 'electron';
import fs from 'fs/promises';
import path from 'path';
import bomService from './bom.service.js';
import bomRevisionRepo from '../database/repositories/bom-revision.repo.js';
import progressService from './progress.service.js';
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
 * 匯出 BOM 到 Excel (Async Task)
 *
 * @param {number} bomRevisionId - BOM 版本 ID
 * @param {string} [outputFilePath] - 輸出檔案路徑 (Optional)
 * @returns {Object} { taskId }
 */
export function exportBom(bomRevisionId, outputFilePath) {
    const taskId = progressService.createTask('EXPORT_BOM', { 
        title: '匯出 BOM Excel',
        metadata: { bomRevisionId, outputFilePath } 
    });

    // Start background process
    runExportTask(taskId, bomRevisionId, outputFilePath).catch(error => {
        console.error(`[Export Error] Task ${taskId} failed:`, error);
        progressService.failTask(taskId, error);
    });

    return { taskId };
}

/**
 * 執行匯出任務 (Internal)
 */
async function runExportTask(taskId, bomRevisionId, outputFilePath) {
    try {
        progressService.log(taskId, 'Initializing export process...', 'info');
        progressService.updateProgress(taskId, 5, 'Initializing...');

        // 1. 取得 BOM Revision
        const revision = bomRevisionRepo.findById(bomRevisionId);
        if (!revision) {
            throw new Error(`找不到 ID 為 ${bomRevisionId} 的 BOM 版本`);
        }
        progressService.log(taskId, `Target Revision: ${revision.project_code} - ${revision.phase_name} ${revision.version}`, 'info');

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
            progressService.updateProgress(taskId, sheetProgress, `Processing sheet: ${sheetDef.targetSheetName}`);
            // Log less frequently to avoid spam, or log every sheet start
            // progressService.log(taskId, `Starting sheet: ${sheetDef.targetSheetName}`, 'info');

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
                     progressService.log(taskId, `Template sheet ${sheetDef.sourceSheetName} not found via fallback. Skipping.`, 'warn');
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
            
            progressService.log(taskId, `Sheet "${sheetDef.targetSheetName}" generated with ${items.length} items.`, 'info');

            processedSheets++;
        }

        progressService.updateProgress(taskId, 90, 'Saving file...');
        progressService.log(taskId, 'Saving workbook to temporary file...', 'info');

        // 6. 存檔 (Temp -> Move)
        const tempDir = app.getPath('temp');
        const tempFileName = `bom_export_${taskId}_${Date.now()}.xlsx`;
        const tempFilePath = path.join(tempDir, tempFileName);

        await saveWorkbook(targetWorkbook, tempFilePath);

        let finalPath = tempFilePath;

        // 如果有指定輸出路徑，則移動檔案
        if (outputFilePath) {
            progressService.log(taskId, `Moving file to: ${outputFilePath}`, 'info');
            // 使用 copyFile + unlink 以避免跨磁碟區移動檔案時發生 EXDEV 錯誤
            await fs.copyFile(tempFilePath, outputFilePath);
            await fs.unlink(tempFilePath);
            finalPath = outputFilePath;
        }

        progressService.completeTask(taskId, { filePath: finalPath });

    } catch (error) {
        console.error(`[Export Task Error]`, error);
        progressService.failTask(taskId, error);
    }
}

export default {
    exportBom
};
