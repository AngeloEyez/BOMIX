/**
 * @file src/main/services/excel-export/template-engine.js
 * @description Excel 樣板匯出引擎
 * @module services/excel-export/template-engine
 */

import ExcelJS from 'exceljs';
import path from 'path';
import fs from 'fs';
import { app } from 'electron'; 

// 取得樣板目錄路徑 (相容 Dev/Prod)
function getTemplatePath(templateName) {
    // 開發環境: 專案根目錄/resources/templates
    // 生產環境: resources/templates (相對於執行檔或 resourcesPath)
    
    // 這裡需要根據 electron-builder 的打包設定調整
    // 通常在 production，resources 目錄位於 process.resourcesPath
    // 在 dev，我們假設它在專案根目錄下的 resources
    
    let basePath;
    if (app.isPackaged) {
        basePath = path.join(process.resourcesPath, 'templates');
    } else {
        basePath = path.join(process.cwd(), 'resources/templates');
    }
    
    return path.join(basePath, templateName);
}

/**
 * 執行樣板匯出
 * 
 * @param {string} templateName - 樣板檔名 (e.g., 'ebom_template.xlsx')
 * @param {Object} data - 匯出資料
 * @param {Object} data.meta - Meta 資訊 key-value 對映
 * @param {Array<Object>} data.items - 資料列表 (主料 + 替代料結構)
 * @param {string} outputPath - 輸出路徑
 */
export async function exportFromTemplate(templateName, data, outputPath) {
    const templatePath = getTemplatePath(templateName);
    
    if (!fs.existsSync(templatePath)) {
        throw new Error(`找不到樣板檔案: ${templatePath}`);
    }

    const workbook = new ExcelJS.Workbook();
    await workbook.xlsx.readFile(templatePath);
    const sheet = workbook.getWorksheet(1); // 假設主要樣板在第一個 Sheet

    // 1. 填寫 Meta Info
    fillMeta(sheet, data.meta);

    // 2. 掃描樣板列 (找出含有 TAG 的列)
    // 我們假設樣板中，含有 {{M_ 開頭的是主料樣板列，含有 {{S_ 開頭的是替代料樣板列
    const { mainTemplateRow, ssTemplateRow, startRowIndex } = findTemplateRows(sheet);
    
    if (!mainTemplateRow) {
        throw new Error('樣板中找不到主料樣板列 (必須包含 {{M_ 開頭的標籤)');
    }

    // 3. 建立 Tag Mapping (標籤 -> 欄位索引)
    const mainMapping = createTagMapping(mainTemplateRow);
    const ssMapping = ssTemplateRow ? createTagMapping(ssTemplateRow) : {};

    // 4. 填寫資料
    let currentRowIndex = startRowIndex;
    
    // 定義斑馬紋顏色
    const COLOR_WHITE = 'FFFFFFFF';
    const COLOR_GRAY = 'D0D5DE';

    // 提取樣板樣式 (Deep Copy)
    const mainStyle = extractRowStyle(mainTemplateRow);
    const ssStyle = ssTemplateRow ? extractRowStyle(ssTemplateRow) : null;

    // 刪除樣板列的策略調整：
    // 我們已經提取了樣式，現在需要移除樣板列以免它殘留在最後（被推擠下去）。
    
    // 1. 如果有替代料樣板列，先將其刪除
    // 注意：刪除後，下方的 Footer 會往上替補
    if (ssTemplateRow) {
        // ssTemplateRow.number 是 1-based index
        // 使用 spliceRows 刪除 1 列
        sheet.spliceRows(ssTemplateRow.number, 1);
    }
    
    // 2. 計算總共需要多少列來放資料
    // 一個 Group 包含: 1 Main Item + N Second Sources
    let totalDataRowsNeeded = 0;
    data.items.forEach(group => {
        totalDataRowsNeeded++; // Main Item
        if (group.second_sources && group.second_sources.length > 0) {
            totalDataRowsNeeded += group.second_sources.length;
        }
    });

    // 3. 插入不足的空間
    // 我們會重用 startRowIndex 這一列 (原 Main Template Row)
    // 所以還需要插入 (totalDataRowsNeeded - 1) 列
    const rowsToInsert = totalDataRowsNeeded - 1;
    
    if (rowsToInsert > 0) {
        // 在 startRowIndex 下方插入空白列
        sheet.spliceRows(startRowIndex + 1, 0, ...new Array(rowsToInsert).fill(null));
    }
    
    // 4. 清空第一列 (startRowIndex) 的內容，準備寫入
    // 其實不需要顯式清空，因為我們會在 fillRow 覆蓋它
    // 但為了保險（例如有些 cell 沒被 mapping 到），可以先清空 values
    // sheet.getRow(startRowIndex).values = []; // 暫不執行，保留 row height 等設定
    
    // 迭代資料填寫
    for (let i = 0; i < data.items.length; i++) {
        const group = data.items[i];
        
        // 決定斑馬紋顏色 (Group Level)
        const bgColor = i % 2 === 0 ? COLOR_WHITE : COLOR_GRAY;
        const fillStyle = {
            type: 'pattern',
            pattern: 'solid',
            fgColor: { argb: bgColor }
        };

        // 4.1 填寫主料
        const row = sheet.getRow(currentRowIndex);
        // 清除可能殘留的樣板標籤 (因為是重用樣板列)
        // row.values = []; // 導致樣式遺失? 不會，因為我們會在 fillRow 重新設定
        fillRow(row, group, mainMapping, mainStyle, fillStyle);
        currentRowIndex++;

        // 4.2 填寫替代料
        if (group.second_sources && group.second_sources.length > 0 && ssStyle) {
            for (const ss of group.second_sources) {
                const ssRow = sheet.getRow(currentRowIndex);
                fillRow(ssRow, ss, ssMapping, ssStyle, fillStyle);
                currentRowIndex++;
            }
        }
    }

    // 5. 處理 Footer Tags (如 {{TOTAL_QTY}})
    // 資料寫完後，currentRowIndex 指向 Footer 的開始
    const dataEndRow = currentRowIndex - 1;
    
    sheet.eachRow((row, rowNumber) => {
        if (rowNumber < currentRowIndex) return; // 只處理 Footer 區域

        row.eachCell((cell) => {
            if (typeof cell.value === 'string' && cell.value.includes('{{TOTAL_QTY}}')) {
                // 產生 SUM 公式: SUM(H6:H10)
                // 假設 Qty 在 H 欄 (col 8) ? 
                // 我們需要知道 Qty 在哪一欄。
                // 方法 A: 硬編碼 (不建議)
                // 方法 B: 搜尋 mainMapping['M_QTY'] 得到 colNumber (建議)
                
                const qtyColIndex = mainMapping['M_QTY'];
                if (qtyColIndex) {
                    const colLetter = row.getCell(qtyColIndex).address.replace(/[0-9]/g, ''); // 取得欄位代號 (e.g. H)
                    const formula = `SUM(${colLetter}${startRowIndex}:${colLetter}${dataEndRow})`;
                    cell.value = { formula: formula };
                } else {
                    cell.value = 'ERR: No Qty Col';
                }
            }
        });
    });

    // 6. 存檔
    await workbook.xlsx.writeFile(outputPath);
}

// ---------------- Helper Functions ----------------

function fillMeta(sheet, meta) {
    // 簡單遍歷前 10 列尋找 Meta Tags
    // 效能優化：可以只找特定範圍
    sheet.eachRow((row, rowNumber) => {
        if (rowNumber > 10) return; // 假設 Meta 都在前 10 列
        row.eachCell((cell) => {
            // Case 1: 一般字串 (String)
            if (typeof cell.value === 'string' && cell.value.includes('{{')) {
                let text = cell.value;
                for (const [key, value] of Object.entries(meta)) {
                    const tag = `{{${key}}}`;
                    // Special case for typo support if user mentioned it, but let's stick to standard first.
                    // Actually, let's fix the user's typo if they have it in template "DESCRIPTIPON"
                    // But in code we use DESCRIPTION. 
                    // Let's standard replace.
                    if (text.includes(tag)) {
                        text = text.replace(tag, value || '');
                    }
                }
                cell.value = text;
            } 
            // Case 2: Rich Text (物件) - 當使用者手動編輯 Excel 且用了部分粗體/顏色時
            else if (cell.value && cell.value.richText) {
                // richText 是一個 array: [{ text: 'Project: ', font: {...} }, { text: '{{PROJECT}}', font: {...} }]
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
        if (mainTemplateRow && ssTemplateRow) return; // Found both

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
            // 假設一格只有一個 Tag，或者我們只取第一個匹配的 Tag 作為 Key
            // e.g., "{{M_PN}}" -> colNumber
            const match = cell.value.match(/{{([A-Z_]+)}}/);
            if (match) {
                mapping[match[1]] = colNumber; // Key: 'M_PN', Value: 2
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
    // 1. 根據 Mapping 填值
    // dataItem 是一個物件，例如 { M_PN: '...', M_DESC: '...' }
    // 我們需要將 dataItem 的 Key 對應到 Mapping 的 Key
    
    // 注意：傳入的 dataItem 應該已經將 Key 轉換為 Tag 名稱 (e.g. M_PN)
    // 或者我們在這裡做轉換? 
    // 為了通用性，建議 caller (export.service) 負責將資料轉為 Tag Key 格式。

    for (const [tag, colNumber] of Object.entries(mapping)) {
        const value = dataItem[tag];
        if (value !== undefined) {
            row.getCell(colNumber).value = value;
        }
    }

// 2. 套用樣式 (根據 Base Style 的範圍，確保每一格都有被處理到)
    // 解決問題：若該列後面幾個欄位沒資料 (e.g. 2nd source 的 Qty/Loc)，原本 row.eachCell 會跳過，導致沒畫到邊框
    
    // 找出 Base Style 的最大欄位數
    const maxCol = baseStyles.length - 1; 

    for (let c = 1; c <= maxCol; c++) {
        const cell = row.getCell(c);
        
        // 還原 Base Style
        if (baseStyles[c]) {
            cell.style = JSON.parse(JSON.stringify(baseStyles[c]));
        }
        
        // 疊加背景色
        cell.fill = fillStyle;

        // 補強邊框 (ExcelJS 預設若是空值有時 border 會怪怪的，確保有 border)
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
