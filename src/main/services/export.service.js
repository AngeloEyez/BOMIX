/**
 * @file src/main/services/export.service.js
 * @description Excel 匯出服務 (Export Service)
 * @module services/export
 */

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
 * 匯出 BOM 到 Excel
 *
 * @param {number} bomRevisionId - BOM 版本 ID
 * @param {string} outputFilePath - 輸出檔案路徑
 * @returns {Object} { success: true }
 */
export async function exportBom(bomRevisionId, outputFilePath) {
    // 1. 取得 BOM Revision
    const revision = bomRevisionRepo.findById(bomRevisionId);
    if (!revision) {
        throw new Error(`找不到 ID 為 ${bomRevisionId} 的 BOM 版本`);
    }

    // 2. 取得 Export Definition
    // 目前固定使用 EBOM，未來可從參數傳入
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

    // 4. 初始化 Workbook
    const templateWorkbook = await loadTemplate(exportDef.templateFile);
    const targetWorkbook = createWorkbook();

    // 5. 處理每個 Sheet
    for (const sheetDef of exportDef.sheets) {
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
        // 這裡需要注意：sourceSheetName 在 templateWorkbook 中必須存在
        // 若 template 只有一個 Sheet (如 'Sheet1')，但 sheetDef 指定了 'SMD'
        // 我們需要確認 template 結構。
        // 根據假設：若 template 是單 Sheet 結構，則 sheetDef.sourceSheetName 應該都指向該 Sheet。
        // 但目前 EXPORTS 定義中，sourceSheetName 分別為 'ALL', 'SMD' 等。
        // 如果實際 Template 只有一個 Sheet，這裡會報錯。
        // 為了相容現有 ebom_template.xlsx (通常只有一個 Sheet)，
        // 我們可以做個 fallback: 若找不到指定的 sourceSheetName，且 Template 只有一個 Sheet，則使用第一個 Sheet。

        let actualSourceSheetName = sheetDef.sourceSheetName;
        const sourceSheet = templateWorkbook.getWorksheet(actualSourceSheetName);
        if (!sourceSheet) {
             // Fallback logic
             if (templateWorkbook.worksheets.length > 0) {
                 actualSourceSheetName = templateWorkbook.worksheets[0].name;
             } else {
                 console.warn(`Template sheet ${sheetDef.sourceSheetName} not found and no fallback available.`);
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
    }

    // 6. 存檔
    await saveWorkbook(targetWorkbook, outputFilePath);

    return { success: true };
}

export default {
    exportBom
};
