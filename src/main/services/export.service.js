/**
 * @file src/main/services/export.service.js
 * @description Excel 匯出服務 (Export Service)
 * @module services/export
 */

import xlsx from 'xlsx';
import bomService from './bom.service.js';
import bomRevisionRepo from '../database/repositories/bom-revision.repo.js';

/**
 * 匯出 BOM 到 Excel
 *
 * @param {number} bomRevisionId - BOM 版本 ID
 * @param {string} outputFilePath - 輸出檔案路徑
 * @returns {Object} { success: true }
 */
function exportBom(bomRevisionId, outputFilePath) {
    // 1. 取得 BOM Data
    const bomView = bomService.getBomView(bomRevisionId);
    const revision = bomRevisionRepo.findById(bomRevisionId);

    if (!revision) {
        throw new Error(`找不到 ID 為 ${bomRevisionId} 的 BOM 版本`);
    }

    const mode = revision.mode || 'NPI';
    const workbook = xlsx.utils.book_new();

    // 2. 定義 Sheet 與 Filter 邏輯
    const sheets = [
        { name: 'ALL', filter: (item) => isInstalled(item, mode) },
        { name: 'SMD', filter: (item) => isInstalled(item, mode) && item.type === 'SMD' },
        { name: 'PTH', filter: (item) => isInstalled(item, mode) && item.type === 'PTH' },
        { name: 'BOTTOM', filter: (item) => isInstalled(item, mode) && item.type === 'BOTTOM' },
        { name: 'NI', filter: (item) => isNotInstalled(item, mode) },
        { name: 'PROTO', filter: (item) => item.bom_status === 'P' },
        { name: 'MP', filter: (item) => item.bom_status === 'M' },
        { name: 'CCL', filter: (item) => item.ccl === 'Y' }
    ];

    // 3. 產生各 Sheet
    sheets.forEach(sheetDef => {
        const filteredItems = bomView.filter(sheetDef.filter);
        const worksheet = generateSheet(filteredItems, revision, sheetDef.name);
        xlsx.utils.book_append_sheet(workbook, worksheet, sheetDef.name);
    });

    // 4. 寫入檔案
    xlsx.writeFile(workbook, outputFilePath);

    return { success: true };
}

/**
 * 判斷零件是否為 "Installed"
 */
function isInstalled(item, mode) {
    if (mode === 'NPI') {
        return item.bom_status === 'I' || item.bom_status === 'P';
    } else { // MP
        return item.bom_status === 'I' || item.bom_status === 'M';
    }
}

/**
 * 判斷零件是否為 "Not Installed"
 */
function isNotInstalled(item, mode) {
    if (mode === 'NPI') {
        return item.bom_status === 'X' || item.bom_status === 'M'; // M in NPI is NI
    } else { // MP
        return item.bom_status === 'X' || item.bom_status === 'P'; // P in MP is NI
    }
}

/**
 * 產生單一 Worksheet
 */
function generateSheet(items, revision, sheetName) {
    // Header Data
    const headerRows = [
        ['FUJIN PRECISION INDUSTRY(SHENZHEN) CO.,LTD'], // A1
        ['BILL OF MATERIAL'], // A2
        ['', `Product Code: ${revision.project_id}`, '', `Schematic Version: ${revision.schematic_version || ''}`, '', `PCB Version: ${revision.pcb_version || ''}`, '', `BOM Version: ${revision.version || ''}`, '', `Phase: ${revision.phase_name || ''}`], // Row 3
        ['', `Description: ${revision.description || ''}`, '', '', '', `PCA PN: ${revision.pca_pn || ''}`, '', `Date: ${revision.date || ''}`] // Row 4
    ];

    // 欄位名稱 Row 5
    const columns = ['Item', 'HH PN', 'STD PN', 'GRP PN', 'Description', 'Supplier', 'Supplier PN', 'Qty', 'Location', 'CCL', 'Lead Time', 'Remark', 'Comp Approval'];
    headerRows.push(columns);

    const dataRows = [];

    // Data Rows
    items.forEach(item => {
        // Main Item
        dataRows.push([
            item.item || '',
            item.hhpn || '',
            '', // STD PN
            '', // GRP PN
            item.description || '',
            item.supplier || '',
            item.supplier_pn || '',
            item.quantity || '',
            item.locations || '', // aggregated locations string
            item.ccl || '',
            '', // Lead Time
            item.remark || '',
            '' // Comp Approval
        ]);

        // Second Sources
        if (item.second_sources && item.second_sources.length > 0) {
            item.second_sources.forEach(ss => {
                dataRows.push([
                    '', // Item (Empty for 2nd source)
                    ss.hhpn || '',
                    '',
                    '',
                    ss.description || '',
                    ss.supplier || '',
                    ss.supplier_pn || '',
                    '', // Qty
                    '', // Location
                    '', // CCL
                    '',
                    '',
                    ''
                ]);
            });
        }
    });

    const wsData = [...headerRows, ...dataRows];
    const ws = xlsx.utils.aoa_to_sheet(wsData);

    // Merges
    ws['!merges'] = [
        { s: { r: 0, c: 0 }, e: { r: 0, c: 12 } }, // A1:M1
        { s: { r: 1, c: 0 }, e: { r: 1, c: 12 } }  // A2:M2
    ];

    // Col Widths
    ws['!cols'] = [
        { wch: 5 },  // Item
        { wch: 15 }, // HH PN
        { wch: 10 }, // STD PN
        { wch: 10 }, // GRP PN
        { wch: 30 }, // Description
        { wch: 15 }, // Supplier
        { wch: 20 }, // Supplier PN
        { wch: 5 },  // Qty
        { wch: 20 }, // Location
        { wch: 5 },  // CCL
        { wch: 10 }, // Lead Time
        { wch: 10 }, // Remark
        { wch: 10 }  // Comp Approval
    ];

    return ws;
}

export default {
    exportBom
};
