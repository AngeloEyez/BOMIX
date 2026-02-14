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
    // 複製屬性
    if (source.properties) {
        target.properties = JSON.parse(JSON.stringify(source.properties));
        // Reset tabColor or other distinct properties if needed? Keep for now.
    }

    // 複製頁面設定
    if (source.pageSetup) {
        target.pageSetup = JSON.parse(JSON.stringify(source.pageSetup));
    }

    // 複製視圖 (Frozen panes etc)
    if (source.views) {
        target.views = JSON.parse(JSON.stringify(source.views));
    }

    // 複製欄寬與樣式
    // Note: iterating columns directly might miss some if not defined?
    // ExcelJS `columnCount` is reliable.
    if (source.columns) {
         // Extract column definitions manually to avoid object reference issues
         const cols = [];
         source.columns.forEach((col, index) => {
             cols.push({
                 header: col.header,
                 key: col.key,
                 width: col.width,
                 style: col.style ? JSON.parse(JSON.stringify(col.style)) : undefined,
                 hidden: col.hidden,
                 outlineLevel: col.outlineLevel
             });
         });
         target.columns = cols;
    }

    // 複製列資料與樣式
    source.eachRow({ includeEmpty: true }, (row, rowNumber) => {
        const targetRow = target.getRow(rowNumber);

        // Copy Values
        // Use JSON parse/stringify to break references for rich text or objects
        if (row.values) {
             targetRow.values = JSON.parse(JSON.stringify(row.values));
        }

        // Copy Row Properties
        targetRow.height = row.height;
        targetRow.hidden = row.hidden;
        targetRow.outlineLevel = row.outlineLevel;

        // Copy Cell Styles
        row.eachCell({ includeEmpty: true }, (cell, colNumber) => {
             const targetCell = targetRow.getCell(colNumber);
             if (cell.style) {
                 targetCell.style = JSON.parse(JSON.stringify(cell.style));
             }
             if (cell.value && typeof cell.value === 'object' && cell.value.formula) {
                 targetCell.value = {
                     formula: cell.value.formula,
                     result: cell.value.result
                 };
             } else if (cell.value !== undefined) {
                 targetCell.value = JSON.parse(JSON.stringify(cell.value));
             }
        });

        targetRow.commit();
    });

    // 複製合併儲存格
    const merges = source.model.merges || []; // Direct access to model merges often safer
    merges.forEach(merge => {
        target.mergeCells(merge);
    });

    // 複製圖片 (ExcelJS 支援度有限，這裡嘗試複製 Image Embeddings)
    // If template has images, we might need:
    // const images = source.getImages();
    // images.forEach(img => { ... })
    // Skipping images for now as per plan.
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
