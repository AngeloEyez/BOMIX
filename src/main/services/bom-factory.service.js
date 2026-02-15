/**
 * @file src/main/services/bom-factory.service.js
 * @description BOM View 與 Export 定義工廠
 * @module services/bom-factory
 */

export const VIEW_IDS = {
    ALL: 'all_view',
    SMD: 'smd_view',
    PTH: 'pth_view',
    BOTTOM: 'bottom_view',
    NI: 'ni_view',
    PROTO: 'proto_view',
    MP: 'mp_view',
    CCL: 'ccl_view'
};

const VIEWS = {
    [VIEW_IDS.ALL]: {
        id: VIEW_IDS.ALL,
        filter: { statusLogic: 'ACTIVE' }
    },
    [VIEW_IDS.SMD]: {
        id: VIEW_IDS.SMD,
        filter: { types: ['SMD'], statusLogic: 'ACTIVE' }
    },
    [VIEW_IDS.PTH]: {
        id: VIEW_IDS.PTH,
        filter: { types: ['PTH'], statusLogic: 'ACTIVE' }
    },
    [VIEW_IDS.BOTTOM]: {
        id: VIEW_IDS.BOTTOM,
        filter: { types: ['BOTTOM'], statusLogic: 'ACTIVE' }
    },
    [VIEW_IDS.NI]: {
        id: VIEW_IDS.NI,
        filter: { statusLogic: 'INACTIVE' }
    },
    [VIEW_IDS.PROTO]: {
        id: VIEW_IDS.PROTO,
        filter: { bom_statuses: ['P'], statusLogic: 'SPECIFIC' }
    },
    [VIEW_IDS.MP]: {
        id: VIEW_IDS.MP,
        filter: { bom_statuses: ['M'], statusLogic: 'SPECIFIC' }
    },
    [VIEW_IDS.CCL]: {
        id: VIEW_IDS.CCL,
        filter: { ccl: 'Y', statusLogic: 'IGNORE' }
    }
};

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
    const view = VIEWS[viewId];
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
