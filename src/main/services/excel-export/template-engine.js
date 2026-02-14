/**
 * @file src/main/services/excel-export/template-engine.js
 * @description Excel 樣板匯出引擎
 * @module services/excel-export/template-engine
 */

import ExcelJS from 'exceljs';
import path from 'path';
import fs from 'fs';
import { app } from 'electron'; 

// 取得樣板目錄路徑
function getTemplatePath(templateName) {
    let basePath;
    if (app.isPackaged) {
        basePath = path.join(process.resourcesPath, 'templates');
    } else {
        basePath = path.join(process.cwd(), 'resources/templates');
    }
    return path.join(basePath, templateName);
}

/**
 * 建立新的 Workbook
 * @returns {ExcelJS.Workbook}
 */
export function createWorkbook() {
    return new ExcelJS.Workbook();
}

/**
 * 載入樣板 Workbook
 * @param {string} templateName
 * @returns {Promise<ExcelJS.Workbook>}
 */
export async function loadTemplate(templateName) {
    const templatePath = getTemplatePath(templateName);
    if (!fs.existsSync(templatePath)) {
        throw new Error(`找不到樣板檔案: ${templatePath}`);
    }
    const workbook = new ExcelJS.Workbook();
    await workbook.xlsx.readFile(templatePath);
    return workbook;
}

/**
 * 儲存 Workbook
 * @param {ExcelJS.Workbook} workbook
 * @param {string} outputPath
 */
export async function saveWorkbook(workbook, outputPath) {
    await workbook.xlsx.writeFile(outputPath);
}

/**
 * 從樣板複製 Sheet 並填入資料
 *
 * @param {ExcelJS.Workbook} targetWorkbook - 目標 Workbook
 * @param {ExcelJS.Workbook} templateWorkbook - 來源樣板 Workbook
 * @param {string|number} sourceSheetName - 來源 Sheet 名稱或索引
 * @param {string} targetSheetName - 目標 Sheet 名稱
 * @param {Object} data - 資料 (meta, items)
 */
export function appendSheetFromTemplate(targetWorkbook, templateWorkbook, sourceSheetName, targetSheetName, data) {
    const sourceSheet = templateWorkbook.getWorksheet(sourceSheetName);
    if (!sourceSheet) {
        throw new Error(`樣板中找不到 Sheet: ${sourceSheetName}`);
    }

    // 1. 新增 Sheet 並複製內容
    const targetSheet = targetWorkbook.addWorksheet(targetSheetName);
    copySheetContent(sourceSheet, targetSheet);

    // 2. 填入資料
    fillSheet(targetSheet, data);
}

/**
 * 執行樣板匯出 (單一 Sheet 相容模式)
 */
export async function exportFromTemplate(templateName, data, outputPath) {
    const templateWorkbook = await loadTemplate(templateName);
    const targetWorkbook = createWorkbook();

    // 預設使用第一個 Sheet
    const sourceSheet = templateWorkbook.getWorksheet(1);
    const targetSheetName = sourceSheet ? sourceSheet.name : 'Sheet1';

    appendSheetFromTemplate(targetWorkbook, templateWorkbook, 1, targetSheetName, data);

    await saveWorkbook(targetWorkbook, outputPath);
}


// ---------------- Helper Functions ----------------




function copySheetContent(source, target) {
    // 使用 Model Deep Copy 方式複製 Sheet
    // 這種方式可以完整保留所有屬性（包含 Merges, Images, DataValidations 等）
    // 且避免了手動設定 target.columns 導致 ExcelJS 誤判寫入 Header Row (Row 1) 的問題

    const targetName = target.name;
    const targetId = target.id;
    
    // console.log(`[CopySheet] Copying from ${source.name} to ${targetName}`);

    // 1. Deep Copy Model
    const modelCopy = JSON.parse(JSON.stringify(source.model));

    // 2. 修正 Model 中的 Name 與 ID
    // 必須在 assign 給 target.model 之前修正，否則可能會因為 name 重複而報錯 (ExcelJS 驗證)
    modelCopy.name = targetName;
    modelCopy.id = targetId;

    // 3. 設定給 Target Sheet
    target.model = modelCopy;

    // 4. [Fix] 手動重新套用 Merges
    // 有時 model assignment 會遺失 merge 狀態或 style，明確再做一次 mergeCells 可確保格式正確
    // source.model.merges 是字串陣列 ['A1:M1', ...]
    const merges = source.model.merges;
    if (merges && Array.isArray(merges)) {
        merges.forEach(range => {
            try {
                // mergeCells 會保留左上角儲存格的樣式
                target.mergeCells(range);
            } catch (error) {
                console.error(`[CopySheet] Failed to merge cells ${range}:`, error);
            }
        });
    }
    
    // 5. 確保 Name 沒被覆蓋 (以防萬一)
    if (target.name !== targetName) {
        // console.warn(`[CopySheet] Target name mismatch after copy. Expected: ${targetName}, Got: ${target.name}. Restoring...`);
        target.name = targetName;
    }
}




function fillSheet(sheet, data) {
    // 1. 填寫 Meta Info
    fillMeta(sheet, data.meta);

    // 2. 掃描樣板列
    const { mainTemplateRow, ssTemplateRow, startRowIndex } = findTemplateRows(sheet);
    
    if (!mainTemplateRow) {
        // 若找不到樣板列，可能只是靜態 Sheet，直接返回
        return;
    }

    // 3. 建立 Tag Mapping
    const mainMapping = createTagMapping(mainTemplateRow);
    const ssMapping = ssTemplateRow ? createTagMapping(ssTemplateRow) : {};

    // 4. 提取樣式
    const mainStyle = extractRowStyle(mainTemplateRow);
    const ssStyle = ssTemplateRow ? extractRowStyle(ssTemplateRow) : null;

    // 5. 準備插入空間
    if (ssTemplateRow) {
        sheet.spliceRows(ssTemplateRow.number, 1);
    }

    let totalDataRowsNeeded = 0;
    data.items.forEach(group => {
        totalDataRowsNeeded++;
        if (group.second_sources && group.second_sources.length > 0) {
            totalDataRowsNeeded += group.second_sources.length;
        }
    });

    const rowsToInsert = totalDataRowsNeeded - 1;
    if (rowsToInsert > 0) {
        sheet.spliceRows(startRowIndex + 1, 0, ...new Array(rowsToInsert).fill(null));
    }

    // 6. 填寫資料
    let currentRowIndex = startRowIndex;
    const COLOR_WHITE = 'FFFFFFFF';
    const COLOR_GRAY = 'D0D5DE';

    for (let i = 0; i < data.items.length; i++) {
        const group = data.items[i];
        const bgColor = i % 2 === 0 ? COLOR_WHITE : COLOR_GRAY;
        const fillStyle = {
            type: 'pattern',
            pattern: 'solid',
            fgColor: { argb: bgColor }
        };

        // Main Item
        const row = sheet.getRow(currentRowIndex);
        fillRow(row, group, mainMapping, mainStyle, fillStyle);
        currentRowIndex++;

        // Second Sources
        if (group.second_sources && group.second_sources.length > 0 && ssStyle) {
            for (const ss of group.second_sources) {
                const ssRow = sheet.getRow(currentRowIndex);
                fillRow(ssRow, ss, ssMapping, ssStyle, fillStyle);
                currentRowIndex++;
            }
        }
    }

    // 7. Footer Formulas
    const dataEndRow = currentRowIndex - 1;
    sheet.eachRow((row, rowNumber) => {
        if (rowNumber < currentRowIndex) return;

        row.eachCell((cell) => {
            if (typeof cell.value === 'string' && cell.value.includes('{{TOTAL_QTY}}')) {
                const qtyColIndex = mainMapping['M_QTY'];
                if (qtyColIndex) {
                    const colLetter = row.getCell(qtyColIndex).address.replace(/[0-9]/g, '');
                    const formula = `SUM(${colLetter}${startRowIndex}:${colLetter}${dataEndRow})`;
                    cell.value = { formula: formula };
                } else {
                    cell.value = 'ERR';
                }
            }
        });
    });
}

function fillMeta(sheet, meta) {
    sheet.eachRow((row, rowNumber) => {
        if (rowNumber > 10) return;
        row.eachCell((cell) => {
            if (typeof cell.value === 'string' && cell.value.includes('{{')) {
                let text = cell.value;
                for (const [key, value] of Object.entries(meta)) {
                    const tag = `{{${key}}}`;
                    if (text.includes(tag)) {
                        text = text.replace(tag, value || '');
                    }
                }
                cell.value = text;
            } else if (cell.value && cell.value.richText) {
                cell.value.richText.forEach(fragment => {
                    if (fragment.text && fragment.text.includes('{{')) {
                         for (const [key, value] of Object.entries(meta)) {
                            const tag = `{{${key}}}`;
                            if (fragment.text.includes(tag)) {
                                fragment.text = fragment.text.replace(tag, value || '');
                            }
                        }
                    }
                });
            }
        });
    });
}

function findTemplateRows(sheet) {
    let mainTemplateRow = null;
    let ssTemplateRow = null;
    let startRowIndex = -1;

    sheet.eachRow((row, rowNumber) => {
        if (mainTemplateRow && ssTemplateRow) return;
        const rowValues = Array.isArray(row.values) ? row.values.join('') : '';
        
        if (!mainTemplateRow && rowValues.includes('{{M_')) {
            mainTemplateRow = row;
            startRowIndex = rowNumber;
        } else if (!ssTemplateRow && rowValues.includes('{{S_')) {
            ssTemplateRow = row;
        }
    });

    return { mainTemplateRow, ssTemplateRow, startRowIndex };
}

function createTagMapping(row) {
    const mapping = {};
    row.eachCell({ includeEmpty: true }, (cell, colNumber) => {
        if (typeof cell.value === 'string' && cell.value.includes('{{')) {
            const match = cell.value.match(/{{([A-Z_]+)}}/);
            if (match) {
                mapping[match[1]] = colNumber;
            }
        }
    });
    return mapping;
}

function extractRowStyle(row) {
    const styles = [];
    row.eachCell({ includeEmpty: true }, (cell, colNumber) => {
        styles[colNumber] = JSON.parse(JSON.stringify(cell.style));
    });
    return styles;
}

function fillRow(row, dataItem, mapping, baseStyles, fillStyle) {
    for (const [tag, colNumber] of Object.entries(mapping)) {
        const value = dataItem[tag];
        if (value !== undefined) {
            row.getCell(colNumber).value = value;
        }
    }

    const maxCol = baseStyles.length - 1; 
    for (let c = 1; c <= maxCol; c++) {
        const cell = row.getCell(c);
        if (baseStyles[c]) {
            cell.style = JSON.parse(JSON.stringify(baseStyles[c]));
        }
        cell.fill = fillStyle;
        if (!cell.border && baseStyles[c] && baseStyles[c].border) {
             cell.border = baseStyles[c].border;
        } else if (!cell.border) {
             cell.border = {
                top: { style: 'thin' },
                left: { style: 'thin' },
                bottom: { style: 'thin' },
                right: { style: 'thin' }
            };
        }
    }
    row.commit();
}
