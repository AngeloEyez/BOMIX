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
export function importBom(filePath, projectId, phaseName, version, suffix) {
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
            //console.log(`[Debug] Trying to read header from sheet: ${sheetName}`);
            const info = parseHeader(sheet);
            
            // 檢查是否讀取成功：至少 Project Code 或 Date 或 Description 有值
            if (info.project_code || info.date || info.description || info.pca_pn) {
                //console.log(`[Debug] Header found in ${sheetName}:`, info);
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
    const additionalParts = []; // 用於存放 "新增" 的重複 location 零件 (如 NI, 或不同 Mode 的 Proto/MP 零件)
    const secondSources = [];   // List of Second Source Objects

    processSheets.forEach(type => {
        const sheet = workbook.Sheets[type];
        if (sheet) {
            parseSheet(sheet, type, 'I', partsMap, secondSources);
        }
    });

    // 階段二：狀態頁面讀取 (不直接更新 partsMap)
    // 分別讀取 NI, PROTO, MP Sheet 的內容
    const niParts = [];
    const protoParts = [];
    const mpParts = [];

    const statusSheets = [
        { name: 'NI', collection: niParts, status: 'X' },
        { name: 'PROTO', collection: protoParts, status: 'P' },
        { name: 'MP', collection: mpParts, status: 'M' }
    ];

    statusSheets.forEach(({ name, collection, status }) => {
        const sheet = workbook.Sheets[name];
        if (sheet) {
            // 這裡傳入 null 作為 type，因為這些頁面不定義 type
            // 傳入一個新的 Map 來收集該 Sheet 的零件，避免影響主 partsMap
            const sheetPartsMap = new Map();
            const sheetSecondSources = []; // 暫存，稍後合併
            parseSheet(sheet, null, status, sheetPartsMap, sheetSecondSources);
            
            // 將解析出的零件存入對應的 collection
            sheetPartsMap.forEach(part => {
                collection.push(part);
            });

            // 合併 Second Sources
            secondSources.push(...sheetSecondSources);
        }
    });

    // 建立用於 Mode 判斷的 Set (僅包含 location)
    const locationsInProto = new Set(protoParts.map(p => p.location));
    const locationsInMp = new Set(mpParts.map(p => p.location));

    // 4. 判斷 Mode
    // 判斷邏輯:
    // NPI Mode: PROTO 頁面中有任何零件出現在主製程中
    // MP Mode: MP 頁面中有任何零件出現在主製程中
    // 預設: NPI
    const locationsInMainProcess = new Set(partsMap.keys());
    const mode = determineMode(locationsInMainProcess, locationsInProto, locationsInMp);

    // 5. 根據 Mode 與 Sheet 來源更新 partsMap
    // 邏輯:
    // - NI Sheet:
    //   - 若零件已存在 partsMap (Phase 1): 新增該零件 with bom_status='X'
    //   - 若零件不存在: 新增該零件 with bom_status='X' (parseSheet 預設行為) -> 這裡我們手動處理
    
    // 處理 NI Parts
    niParts.forEach(part => {
        // NI Sheet 的零件，Spec 定義:
        // "若有重複（同一 location），以最新取得的 bom_status 覆蓋... 若無重複，則新增一筆零件紀錄。"
        // 但使用者需求變更: "如果 phase 1 partsMap 中有找到該零件, 則... 新增這顆零件 with bom_status=X"
        // 也就是說，對於 NI sheet，無論是否存在於 Phase 1，我們都要保留這個 'X' 狀態的零件。
        // 如果 Phase 1 已經有這個 location (例如 Status I)，我們不能覆蓋它，而是要新增一個 Status X 的紀錄?
        // 根據 User Request: "如果 phase 1 partsMap 中有找到該零件, 則分sheet處理: if (sheetName = 'NI') { 新增這顆零件with bom_status=X }"
        // 這意味著同一個 location 可能會有多筆紀錄 (一個 I, 一個 X)?
        // 讓我們確認 Schema: parts table primary key 是 id. unique constrain 是什麼?
        // Schema: INDEX idx_parts_location ON parts(bom_revision_id, location) -> 不是 Unique Index.
        // 所以同一個 location 可以有多筆紀錄。
        
        // 實作:
        // 在 partsMap 中，key 是 location。如果我們直接用 map.set 相同 location，會覆蓋。
        // 所以我們需要一個機制來處理 "新增"。
        // 由於最後是轉成 Array 插入 DB，我們可以將這些 "新增" 的零件直接放入 partsToInsert 陣列，而不一定要放入 partsMap。
        // 但要注意，partsMap 目前是用來去重的 (針對 Phase 1)。
        
        // 策略:
        // partsMap 保留 Phase 1 的零件 (狀態通常是 I)。
        // Phase 2 的零件，根據邏輯決定是 "修改 partsMap 中的零件" 還是 "作為新零件加入"。

        const loc = part.location;
        if (partsMap.has(loc)) {
            // Phase 1 中有此零件
            // NI Sheet: 新增這顆零件 with bom_status=X
            // 為了避免 key 衝突，我們不放入 partsMap，而是稍後直接合併到輸出陣列
            // 但為了方便，我們可以給它一個臨時的 unique key 或者直接收集到 `additionalParts`
            additionalParts.push(part); 
        } else {
            // Phase 1 中沒有此零件 -> 新增 (原邏輯)
            partsMap.set(loc, part);
        }
    });

    // 處理 PROTO Parts
    protoParts.forEach(part => {
        const loc = part.location;
        if (partsMap.has(loc)) {
            // Phase 1 中有此零件
            if (mode === 'NPI') {
                // Mode NPI: PROTO sheet內的零件覆蓋bom_Status到partsMap
                const existingPart = partsMap.get(loc);
                existingPart.bom_status = 'P';
                // 其他屬性是否要覆蓋? 根據原邏輯: "以最新取得的 bom_status 覆蓋，type 不覆蓋"
                // 這裡我們只更新 status
            } else {
                // Mode MP (或其他): PROTO sheet內零件新增 with bom_status=P
                additionalParts.push(part);
            }
        } else {
            // Phase 1 中沒有 -> 新增
            partsMap.set(loc, part);
        }
    });

    // 處理 MP Parts
    mpParts.forEach(part => {
        const loc = part.location;
        if (partsMap.has(loc)) {
            // Phase 1 中有此零件
            if (mode === 'MP') {
                // Mode MP: MP sheet內的零件覆蓋bom_status到 partsMap
                const existingPart = partsMap.get(loc);
                existingPart.bom_status = 'M';
            } else {
                // Mode NPI (或其他): MP sheet內新增這顆零件 with bom_status=M
                additionalParts.push(part);
            }
        } else {
            // Phase 1 中沒有 -> 新增
            partsMap.set(loc, part);
        }
    });

    // 6. 儲存至資料庫
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
    // 合併 partsMap 與 additionalParts
    const allParts = [
        ...Array.from(partsMap.values()),
        ...additionalParts
    ];

    const partsToInsert = allParts.map(p => ({
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
    
    if (partsToInsert.length > 0) {
        partsRepo.createMany(partsToInsert);
    }
    
    if (secondSourcesToInsert.length > 0) {
        secondSourceRepo.createMany(secondSourcesToInsert);
    }

    return { success: true, bomRevisionId };
}

/**
 * 解析表頭資訊
 * @param {Object} sheet - Sheet Object
 * @returns {Object} Header Info
 */
export function parseHeader(sheet) {
    const getValue = (cell) => {
        const val = sheet[cell]?.v;
        return val ? String(val).trim() : '';
    };

    const extract = (text, prefix) => {
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
export function parseSheet(sheet, type, defaultStatus, partsMap, secondSources) {
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
export function determineMode(locationsInMainProcess, locationsInProto, locationsInMp) {
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
