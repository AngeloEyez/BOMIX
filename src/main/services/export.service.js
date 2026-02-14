/**
 * @file src/main/services/export.service.js
 * @description Excel 匯出服務 (Export Service) - Refactored to use Template Engine
 * @module services/export
 */

import bomService from './bom.service.js';
import bomRevisionRepo from '../database/repositories/bom-revision.repo.js';
import { exportFromTemplate } from './excel-export/template-engine.js';

/**
 * 匯出 BOM 到 Excel
 *
 * @param {number} bomRevisionId - BOM 版本 ID
 * @param {string} outputFilePath - 輸出檔案路徑
 * @returns {Object} { success: true }
 */
export async function exportBom(bomRevisionId, outputFilePath) {
    // 1. 取得 BOM Data
    const bomView = bomService.getBomView(bomRevisionId);
    const revision = bomRevisionRepo.findById(bomRevisionId);

    if (!revision) {
        throw new Error(`找不到 ID 為 ${bomRevisionId} 的 BOM 版本`);
    }

    // 2. 準備 Meta Data
    const meta = {
        PROJECT_CODE: revision.project_code || revision.project_id || 'Unknown', 
        BOM_VERSION: `${revision.version}${revision.suffix ? '-' + revision.suffix : ''}`, // Renamed from VERSION
        PHASE: revision.phase_name,
        BOM_DATE: revision.bom_date || new Date().toISOString().split('T')[0], // Renamed from DATE, prefer bom_date
        SCH_VERSION: revision.schematic_version || '',
        PCB_VERSION: revision.pcb_version || '',
        PCA_PN: revision.pca_pn || '',
        DESCRIPTION: revision.description || ''
    };

    // 3. 準備 Items Data (Mapping to Tags)
    const items = bomView.map(item => {
        // 主料 Tags
        const mainTags = {
            M_ITEM: item.item,
            M_HHPN: item.hhpn,
            M_DESC: item.description,
            M_SUPP: item.supplier,
            M_SPN: item.supplier_pn,
            M_QTY: item.quantity,
            M_LOC: item.locations,
            M_REMARK: item.remark,
            M_CCL: item.ccl // New Tag
        };

        // 替代料 Tags
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

    // 4. 呼叫 Template Engine
    await exportFromTemplate('ebom_template.xlsx', { meta, items }, outputFilePath);

    return { success: true };
}

export default {
    exportBom
};
