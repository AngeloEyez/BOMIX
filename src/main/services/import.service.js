/**
 * @file src/main/services/import.service.js
 * @description Excel 匯入服務 (Import Service)
 * @module services/import
 */

import xlsx from 'xlsx';
import path from 'path';
import bomRevisionRepo from '../database/repositories/bom-revision.repo.js';
import partsRepo from '../database/repositories/parts.repo.js';
import secondSourceRepo from '../database/repositories/second-source.repo.js';
import projectRepo from '../database/repositories/project.repo.js';

/**
 * 解析 Excel 檔案並匯入 BOM
 *
 * @param {string} filePath - Excel 檔案路徑
 * @param {number} projectId - 專案 ID
 * @param {string} phaseName - Phase 名稱
 * @param {string} suffix - 版本後綴 (Optional)
 * @returns {Object} 匯入結果 { success: true, bomRevisionId }
 */
function importBom(filePath, projectId, phaseName, version, suffix) {
    // 1. 讀取 Excel 檔案
    const workbook = xlsx.readFile(filePath);
    const filename = path.basename(filePath);

    // 2. 解析表頭 (Header)
    // 優先順序: SMD -> PTH -> BOTTOM
    // 若 SMD 讀不到 (可能是空的或不存在)，則嘗試 PTH，依此類推。
    const potentialHeaderSheets = ['SMD', 'PTH', 'BOTTOM'];
    let headerInfo = {
        project_code: '', description: '', schematic_version: '', 
        pcb_version: '', pca_pn: '', date: ''
    };
    let headerFound = false;

    for (const sheetName of potentialHeaderSheets) {
        const sheet = workbook.Sheets[sheetName];
        if (sheet) {
            console.log(`[Debug] Trying to read header from sheet: ${sheetName}`);
            const info = parseHeader(sheet);
            
            // 檢查是否讀取成功：至少 Project Code 或 Date 或 Description 有值
            if (info.project_code || info.date || info.description || info.pca_pn) {
                console.log(`[Debug] Header found in ${sheetName}:`, info);
                headerInfo = info;
                headerFound = true;
                break;
            }
        }
    }

    if (!headerFound) {
        console.warn('[Import] Warning: Could not find valid header in SMD/PTH/BOTTOM sheets. Using default/empty values.');
    } else {
        // 驗證 Project Code (若有讀取到)
        if (headerInfo.project_code && projectId) {
            const project = projectRepo.findById(projectId);
            if (project) {
                const dbCode = project.project_code.replace(/[\s\-_]/g, '').toLowerCase();
                const headerCode = headerInfo.project_code.replace(/[\s\-_]/g, '').toLowerCase();
                
                if (dbCode !== headerCode) {
                    console.warn(`[Import] Project Code Mismatch: DB=${project.project_code}, Header=${headerInfo.project_code}`);
                    return { 
                        success: false, 
                        error: 'PROJECT_CODE_MISMATCH', 
                        parsedProjectCode: headerInfo.project_code,
                        targetProjectCode: project.project_code
                    };
                }
            }
        }
    }

    // 3. 讀取各 Sheet 的零件
    // 階段一：製程頁面
    const processSheets = ['SMD', 'PTH', 'BOTTOM'];
    const partsMap = new Map(); // Key: location, Value: Part Object
    const secondSources = [];   // List of Second Source Objects

    processSheets.forEach(type => {
        const sheet = workbook.Sheets[type];
        if (sheet) {
            parseSheet(sheet, type, 'I', partsMap, secondSources);
        }
    });

    // 階段二：狀態頁面
    const statusSheets = [
        { name: 'NI', status: 'X' },
        { name: 'PROTO', status: 'P' },
        { name: 'MP', status: 'M' }
    ];

    // 用於 Mode 判斷的集合
    const locationsInMainProcess = new Set(partsMap.keys());
    const locationsInProto = new Set();
    const locationsInMp = new Set();

    statusSheets.forEach(({ name, status }) => {
        const sheet = workbook.Sheets[name];
        if (sheet) {
            // 這裡傳入 null 作為 type，因為這些頁面不定義 type
            // 並傳入 partsMap 進行覆蓋或新增
            const sheetLocations = parseSheet(sheet, null, status, partsMap, secondSources);

            if (name === 'PROTO') {
                sheetLocations.forEach(loc => locationsInProto.add(loc));
            } else if (name === 'MP') {
                sheetLocations.forEach(loc => locationsInMp.add(loc));
            }
        }
    });

    // 4. 判斷 Mode
    const mode = determineMode(locationsInMainProcess, locationsInProto, locationsInMp);

    // 5. 儲存至資料庫
    // 建立 BOM Revision
    const revisionData = {
        project_id: projectId,
        phase_name: phaseName,
        version: version,
        description: headerInfo.description,
        schematic_version: headerInfo.schematic_version,
        pcb_version: headerInfo.pcb_version,
        pca_pn: headerInfo.pca_pn,
        bom_date: headerInfo.date,
        mode: mode,
        filename: filename,
        suffix: suffix
    };

    const bomRevision = bomRevisionRepo.create(revisionData);
    const bomRevisionId = bomRevision.id;

    // 準備批量插入資料
    const partsToInsert = Array.from(partsMap.values()).map(p => ({
        ...p,
        bom_revision_id: bomRevisionId
    }));

    // Deduplicate Second Sources based on (main_supplier, main_supplier_pn, supplier, supplier_pn)
    const uniqueSecondSourcesMap = new Map();
    secondSources.forEach(ss => {
        const key = `${ss.main_supplier}|${ss.main_supplier_pn}|${ss.supplier}|${ss.supplier_pn}`;
        if (!uniqueSecondSourcesMap.has(key)) {
            uniqueSecondSourcesMap.set(key, ss);
        }
    });

    const secondSourcesToInsert = Array.from(uniqueSecondSourcesMap.values()).map(ss => ({
        ...ss,
        bom_revision_id: bomRevisionId
    }));

    partsRepo.createMany(partsToInsert);
    secondSourceRepo.createMany(secondSourcesToInsert);

    return { success: true, bomRevisionId };
}

/**
 * 解析表頭資訊
 * @param {Object} sheet - Sheet Object
 * @returns {Object} Header Info
 */
function parseHeader(sheet) {
    const getValue = (cell) => {
        const val = sheet[cell]?.v;
        return val ? String(val).trim() : '';
    };

    const extract = (text, prefix) => {
        console.log('extract:', text, "prefix:", prefix);
        if (!text) return '';
        
        // 1. 嘗試直接比對開頭 (忽略換行與多餘空白)
        // 將 text 與 prefix 都標準化為單一空格分隔
        const normalizedText = text.replace(/[\r\n\s]+/g, ' ').trim();
        const normalizedPrefix = prefix.replace(/[\r\n\s]+/g, ' ').trim();

        if (normalizedText.startsWith(normalizedPrefix)) {
            // 但我們需要從原始 text 中切出 value
            // 由於原始 text 有換行，這裡改用 split(':') 比較安全
            const parts = text.split(':');
            if (parts.length > 1) {
                return parts.slice(1).join(':').trim();
            }
        }

        // 2. 嘗試用冒號分隔，並檢查 Label 部分是否匹配
        const parts = text.split(':');
        if (parts.length > 1) {
            const label = parts[0].replace(/[\r\n\s]+/g, ' ').trim();
            const prefixLabel = normalizedPrefix.replace(/:$/, '').trim(); // 移除 prefix 尾端冒號
            
            if (label.toLowerCase() === prefixLabel.toLowerCase()) {
                return parts.slice(1).join(':').trim();
            }
        }

        return '';
    };

    return {
        project_code: extract(getValue('B3'), 'Product Code:'),
        description: extract(getValue('B4'), 'Description:'),
        schematic_version: extract(getValue('D3'), 'Schematic Version:'),
        pcb_version: extract(getValue('F3'), 'PCB Version:'),
        pca_pn: extract(getValue('F4'), 'PCA PN:'),
        date: extract(getValue('H4'), 'Date:')
    };
}

/**
 * 解析 Sheet 中的零件
 * @param {Object} sheet - Sheet Object
 * @param {string|null} type - 製程類型 (SMD, PTH, BOTTOM)
 * @param {string} defaultStatus - 預設 BOM Status (I, X, P, M)
 * @param {Map} partsMap - 用於儲存/更新零件的 Map (Key: location)
 * @param {Array} secondSources - 用於儲存 Second Sources 的陣列
 * @returns {Array<string>} 此 Sheet 中包含的所有 locations
 */
function parseSheet(sheet, type, defaultStatus, partsMap, secondSources) {
    const range = xlsx.utils.decode_range(sheet['!ref']);
    const sheetLocations = [];

    let currentMainSource = null; // { supplier, supplier_pn }

    // 從 Row 6 (Index 5) 開始讀取
    for (let R = 5; R <= range.e.r; ++R) {
        const getCell = (C) => {
            const cell = sheet[xlsx.utils.encode_cell({ r: R, c: C })];
            return cell ? String(cell.v).trim() : '';
        };

        const item = getCell(0); // Col A
        const hhpn = getCell(1); // Col B
        // Col C, D ignored
        const description = getCell(4); // Col E
        const supplier = getCell(5); // Col F
        const supplier_pn = getCell(6); // Col G
        // Col H ignored
        const locationStr = getCell(8); // Col I
        const ccl = getCell(9); // Col J
        // Col K ignored
        const remark = getCell(11); // Col L

        // 若無供應商資料，跳過 (可能是空行)
        if (!supplier && !supplier_pn) continue;

        if (item && locationStr) {
            // === Main Source ===
            currentMainSource = { supplier, supplier_pn };

            // 處理 Location 原子化
            const locations = locationStr.split(',').map(l => l.trim()).filter(l => l);

            locations.forEach(loc => {
                sheetLocations.push(loc);

                if (partsMap.has(loc)) {
                    // 已存在 (來自其他 Sheet)，進行覆蓋
                    const existingPart = partsMap.get(loc);
                    // 規則: "以最新取得的 bom_status 覆蓋，type 不覆蓋"
                    // 若是 Phase 2 sheet (type is null), we update status.
                    // 若是 Phase 1 sheet (type is set), we update type and status?
                    // SPEC: "這三個頁面的零件 location 可能與 SMD/PTH/BOTTOM 中的零件重複... 以最新取得的 bom_status 覆蓋，type 不覆蓋"
                    // 這裡的 "這三個頁面" 指的是 Phase 2 (NI/PROTO/MP).
                    // 所以如果當前解析的是 Phase 2 Sheet (type === null)，我們只更新 status。
                    // 如果當前解析的是 Phase 1 Sheet，理論上不應該重複 (同一個零件不會同時在 SMD 和 PTH?)
                    // 但若重複，後蓋前。

                    if (type === null) {
                        // Phase 2 Sheet: Override Status only
                        existingPart.bom_status = defaultStatus;
                        // update other fields if needed? SPEC doesn't say explicitly, but usually Overlay sheets only define status.
                        // However, spec says "type 不覆蓋".
                    } else {
                        // Phase 1 Sheet logic (should not happen usually for same location)
                        // If it happens, fully overwrite?
                        existingPart.type = type;
                        existingPart.bom_status = defaultStatus;
                        existingPart.supplier = supplier;
                        existingPart.supplier_pn = supplier_pn;
                        existingPart.hhpn = hhpn;
                        existingPart.description = description;
                        existingPart.ccl = ccl;
                        existingPart.remark = remark;
                    }
                } else {
                    // 新增零件
                    partsMap.set(loc, {
                        item: parseInt(item) || null,
                        hhpn,
                        supplier,
                        supplier_pn,
                        description,
                        location: loc,
                        type: type, // 如果是 Phase 2 sheet，這裡 type 會是 null
                        bom_status: defaultStatus,
                        ccl,
                        remark
                    });
                }
            });

        } else if (currentMainSource) {
            // === Second Source ===
            // 驗證資料完整性：HHPN, Supplier, Supplier PN, Description 必須都有值
            if (hhpn && supplier && supplier_pn && description) {
                secondSources.push({
                    main_supplier: currentMainSource.supplier,
                    main_supplier_pn: currentMainSource.supplier_pn,
                    hhpn,
                    supplier,
                    supplier_pn,
                    description
                });
            }
        }
    }

    return sheetLocations;
}

/**
 * 判斷 BOM Mode (NPI/MP)
 * @param {Set} locationsInMainProcess
 * @param {Set} locationsInProto
 * @param {Set} locationsInMp
 * @returns {string} 'NPI' or 'MP'
 */
function determineMode(locationsInMainProcess, locationsInProto, locationsInMp) {
    let mode = 'NPI'; // Default

    // 檢查 MP 條件
    let hasMpIntersection = false;
    for (const loc of locationsInMp) {
        if (locationsInMainProcess.has(loc)) {
            hasMpIntersection = true;
            break;
        }
    }

    // 檢查 PROTO 條件
    let hasProtoIntersection = false;
    for (const loc of locationsInProto) {
        if (locationsInMainProcess.has(loc)) {
            hasProtoIntersection = true;
            break;
        }
    }

    if (hasMpIntersection) {
        mode = 'MP';
    }
    if (hasProtoIntersection) {
        mode = 'NPI';
    }
    return mode;
}

export default {
    importBom,
    parseHeader,
    parseSheet,
    determineMode
};
