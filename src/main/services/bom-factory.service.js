/**
 * @file src/main/services/bom-factory.service.js
 * @description BOM View 與 Export 定義工廠
 * @module services/bom-factory
 */

/**
 * BOM View 定義集合。
 *
 * 每個 View 包含：
 *   - id: 唯一識別碼（供前端 IPC 呼叫使用）
 *   - filters: Filter 陣列（格式詳見 dev/FILTER_SPEC.md）
 */
export const VIEWS = {
    ALL: {
        id: 'all_view',
        filters: [
            { field: 'bom_status', operator: 'statusLogic', value: 'ACTIVE' }
        ]
    },
    SMD: {
        id: 'smd_view',
        filters: [
            { field: 'type', operator: 'in', value: ['SMD'] },
            { field: 'bom_status', operator: 'statusLogic', value: 'ACTIVE' }
        ]
    },
    PTH: {
        id: 'pth_view',
        filters: [
            { field: 'type', operator: 'in', value: ['PTH'] },
            { field: 'bom_status', operator: 'statusLogic', value: 'ACTIVE' }
        ]
    },
    BOTTOM: {
        id: 'bottom_view',
        filters: [
            { field: 'type', operator: 'in', value: ['BOTTOM'] },
            { field: 'bom_status', operator: 'statusLogic', value: 'ACTIVE' }
        ]
    },
    NI: {
        id: 'ni_view',
        filters: [
            { field: 'bom_status', operator: 'statusLogic', value: 'INACTIVE' }
        ]
    },
    PROTO: {
        id: 'proto_view',
        filters: [
            { field: 'bom_status', operator: 'statusLogic', value: 'SPECIFIC' },
            { field: 'bom_status', operator: 'in', value: ['P'] }
        ]
    },
    MP: {
        id: 'mp_view',
        filters: [
            { field: 'bom_status', operator: 'statusLogic', value: 'SPECIFIC' },
            { field: 'bom_status', operator: 'in', value: ['M'] }
        ]
    },
    CCL: {
        id: 'ccl_view',
        filters: [
            { field: 'ccl', operator: 'eq', value: 'Y' },
            { field: 'bom_status', operator: 'statusLogic', value: 'ACTIVE' }
        ]
    }
};

// 為了相容性與易用性，產生 VIEW_IDS 物件
export const VIEW_IDS = Object.keys(VIEWS).reduce((acc, key) => {
    acc[key] = VIEWS[key].id;
    return acc;
}, {});

// 建立 ID 查找表以優化效能 (O(1) lookup)
const ID_TO_VIEW_MAP = Object.values(VIEWS).reduce((acc, view) => {
    acc[view.id] = view;
    return acc;
}, {});

export const EXPORT_IDS = {
    EBOM: 'ebom'
};

const EXPORTS = {
    [EXPORT_IDS.EBOM]: {
        id: EXPORT_IDS.EBOM,
        sheets: [
            { targetSheetName: 'ALL', templateFile: 'ebom.xlsx', sourceSheetName: 'ALL', viewId: VIEW_IDS.ALL },
            { targetSheetName: 'SMD', templateFile: 'ebom.xlsx', sourceSheetName: 'SMD', viewId: VIEW_IDS.SMD },
            { targetSheetName: 'PTH', templateFile: 'ebom.xlsx', sourceSheetName: 'PTH', viewId: VIEW_IDS.PTH },
            { targetSheetName: 'BOTTOM', templateFile: 'ebom.xlsx', sourceSheetName: 'BOTTOM', viewId: VIEW_IDS.BOTTOM },
            { targetSheetName: 'NI', templateFile: 'ebom.xlsx', sourceSheetName: 'NI', viewId: VIEW_IDS.NI },
            { targetSheetName: 'PROTO', templateFile: 'ebom.xlsx', sourceSheetName: 'PROTO', viewId: VIEW_IDS.PROTO },
            { targetSheetName: 'MP', templateFile: 'ebom.xlsx', sourceSheetName: 'MP', viewId: VIEW_IDS.MP },
            { targetSheetName: 'CCL', templateFile: 'ebom.xlsx', sourceSheetName: 'CCL', viewId: VIEW_IDS.CCL },
        ]
    }
};

/**
 * 取得 View 定義
 * @param {string} viewId
 * @returns {Object} View 定義物件（含 filters 陣列）
 */
export function getViewDefinition(viewId) {
    const view = ID_TO_VIEW_MAP[viewId];
    if (!view) {
        throw new Error(`Unknown View ID: ${viewId}`);
    }
    return view;
}

/**
 * 取得 View 的 Filter 陣列
 *
 * 供前端透過 IPC 查詢特定 View 對應的 filters（格式詳見 dev/FILTER_SPEC.md）。
 *
 * @param {string} viewId - View ID（例如 'all_view'、'smd_view'）
 * @returns {Array<Object>} Filter 陣列
 * @throws {Error} 若 viewId 不存在
 */
export function getViewFilters(viewId) {
    const view = getViewDefinition(viewId);
    return view.filters;
}

/**
 * 取得 Export 定義
 * @param {string} exportId
 * @returns {Object} Export 定義物件
 */
export function getExportDefinition(exportId) {
    const def = EXPORTS[exportId];
    if (!def) {
        throw new Error(`Unknown Export ID: ${exportId}`);
    }
    return def;
}

export default {
    VIEWS,
    VIEW_IDS,
    EXPORT_IDS,
    getViewDefinition,
    getViewFilters,
    getExportDefinition
};
