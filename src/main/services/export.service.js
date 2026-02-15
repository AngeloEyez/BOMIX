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

    // 4. 初始化 Target Workbook
    const targetWorkbook = createWorkbook();

    // 緩存已載入的 Template Workbooks
    const loadedTemplates = new Map();

    // 5. 處理每個 Sheet
    for (const sheetDef of exportDef.sheets) {
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
        // Fallback logic for template sheet name
        let actualSourceSheetName = sheetDef.sourceSheetName;
        const sourceSheet = templateWorkbook.getWorksheet(actualSourceSheetName);
        if (!sourceSheet) {
             if (templateWorkbook.worksheets.length > 0) {
                 actualSourceSheetName = templateWorkbook.worksheets[0].name;
             } else {
                 console.warn(`Template sheet ${sheetDef.sourceSheetName} not found in ${templateFile} and no fallback available.`);
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
