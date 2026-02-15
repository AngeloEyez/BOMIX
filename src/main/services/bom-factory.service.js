/**
 * @file src/main/services/bom-factory.service.js
 * @description BOM View 與 Export 定義工廠
 * @module services/bom-factory
 */

const VIEWS = {
    ALL: {
        id: 'all_view',
        filter: { statusLogic: 'ACTIVE' }
    },
    SMD: {
        id: 'smd_view',
        filter: { types: ['SMD'], statusLogic: 'ACTIVE' }
    },
    PTH: {
        id: 'pth_view',
        filter: { types: ['PTH'], statusLogic: 'ACTIVE' }
    },
    BOTTOM: {
        id: 'bottom_view',
        filter: { types: ['BOTTOM'], statusLogic: 'ACTIVE' }
    },
    NI: {
        id: 'ni_view',
        filter: { bom_statuses: ['X'], statusLogic: 'SPECIFIC' }
        //filter: { statusLogic: 'INACTIVE' }
    },
    PROTO: {
        id: 'proto_view',
        filter: { bom_statuses: ['P'], statusLogic: 'SPECIFIC' }
    },
    MP: {
        id: 'mp_view',
        filter: { bom_statuses: ['M'], statusLogic: 'SPECIFIC' }
    },
    CCL: {
        id: 'ccl_view',
        filter: { ccl: 'Y', statusLogic: 'ACTIVE' }
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
 * @returns {Object} View 定義物件
 */
export function getViewDefinition(viewId) {
    const view = ID_TO_VIEW_MAP[viewId];
    if (!view) {
        throw new Error(`Unknown View ID: ${viewId}`);
    }
    return view;
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
    VIEW_IDS,
    EXPORT_IDS,
    getViewDefinition,
    getExportDefinition
};
