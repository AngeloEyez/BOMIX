// ========================================
// 設定系統的單一真實來源 (Single Source of Truth)
// 所有設定的分類、選項、預設值、標題與說明均在此定義
// ========================================

import { Settings2, Monitor } from 'lucide-react'

// ========================================
// 分類定義 — 所有大分類集中維護
// icon: Lucide React 元件
// ========================================

/**
 * 設定大分類定義（左側目錄的第一層）
 * 
 * @type {Array<{id: string, title: string, icon: React.ReactElement}>}
 */
export const SETTINGS_CATEGORIES = [
    {
        id: 'general',
        title: '一般',
        icon: Settings2,
    },
    {
        id: 'appearance',
        title: '外觀',
        icon: Monitor,
    },
]

// ========================================
// 子分類定義 — 集中維護，各設定項目透過 subCategoryId 引用
// ========================================

/**
 * 設定子分類定義（左側目錄的第二層）
 * 
 * @type {Array<{id: string, categoryId: string, title: string}>}
 */
export const SETTINGS_SUB_CATEGORIES = [
    { id: 'general.startup',   categoryId: 'general',    title: '啟動行為' },
    { id: 'appearance.theme',  categoryId: 'appearance', title: '佈景主題' },
    //{ id: 'appearance.colors', categoryId: 'appearance', title: '主題色彩' },
]

// ========================================
// 設定項目定義
// 
// 每個設定項目的欄位說明:
//   key           {string}   Store 中的狀態鍵名（必須與 useSettingsStore 的欄位一致）
//   default       {any}      預設值
//   title         {string}   UI 顯示的設定標題
//   description   {string}   設定項目的說明文字
//   subCategoryId {string}   歸屬的子分類 ID（對應 SETTINGS_SUB_CATEGORIES.id）
//   type          {string}   UI 控制元件類型：'toggle' | 'select' | 'number' | 'readonly'
//   options       {Array|null} select 類型的靜態選項陣列 [{value, label}]，動態選項傳 null
//   persist       {boolean}  是否需要持久化儲存到後端
// ========================================

/**
 * 所有設定項目的完整定義陣列。
 * 
 * @type {Array<{
 *   key: string,
 *   default: any,
 *   title: string,
 *   description: string,
 *   subCategoryId: string,
 *   type: 'toggle'|'select'|'number'|'readonly',
 *   options: Array<{value:any, label:string}>|null,
 *   persist: boolean
 * }>}
 */
export const SETTINGS_CONFIG = [
    // ---- 一般 / 啟動行為 ----
    {
        key: 'startupPage',
        default: 'dashboard',
        title: '啟動時開啟的頁面',
        description: '控制啟動 BOMIX 時預設導航的頁面。',
        subCategoryId: 'general.startup',
        type: 'select',
        options: [
            { value: 'dashboard', label: 'Dashboard (預設)' },
            { value: 'bom', label: 'BOM 編輯器' },
        ],
        persist: true,
    },

    // ---- 外觀 / 佈景主題 ----
    {
        key: 'theme',
        default: 'light',
        title: '深色模式',
        description: '切換應用程式整體的明暗佈景主題。',
        subCategoryId: 'appearance.theme',
        type: 'toggle',
        options: null,
        persist: true,
    },

    // ---- 外觀 / 主題色彩 ----
    {
        key: 'activeThemeId',
        default: 'default',
        title: '主題色彩',
        description: '選擇應用程式的品牌顏色風格。',
        subCategoryId: 'appearance.theme',
        // options 為 null 表示動態選項（需從 Store availableThemes 取得）
        type: 'select',
        options: null,
        persist: true,
    },
]

// ========================================
// 工具函數
// ========================================

/**
 * 從 SETTINGS_CONFIG 彙整所有設定的預設值。
 * 供 useSettingsStore 建立初始狀態使用。
 * 
 * @returns {Object} key-value 對映，例如 { theme: 'light', activeThemeId: 'default', ... }
 */
export function getDefaults() {
    return SETTINGS_CONFIG.reduce((acc, item) => {
        acc[item.key] = item.default
        return acc
    }, {})
}

/**
 * 根據分類/子分類定義與各設定項目，自動組合左側目錄樹。
 * 回傳結構供 SettingsPage 的左側 Collapsible 目錄直接使用。
 * 
 * @returns {Array<{
 *   id: string,
 *   title: string,
 *   icon: React.ComponentType,
 *   items: Array<{id: string, title: string, settings: Array}>
 * }>}
 */
export function buildSettingsTree() {
    return SETTINGS_CATEGORIES.map(category => {
        // 找出隸屬於此大分類的子分類
        const subCats = SETTINGS_SUB_CATEGORIES.filter(sc => sc.categoryId === category.id)

        const items = subCats.map(subCat => {
            // 找出歸屬於此子分類的設定項目（按陣列順序）
            const settings = SETTINGS_CONFIG.filter(cfg => cfg.subCategoryId === subCat.id)
            return {
                id: subCat.id,
                title: subCat.title,
                settings,
            }
        })

        return {
            id: category.id,
            title: category.title,
            icon: category.icon,
            items,
        }
    })
}

/**
 * 取得所有需要持久化的設定項目 key 清單。
 * 供 saveSettings 使用，避免硬編碼需儲存的欄位列表。
 * 
 * @returns {string[]} 需持久化的 key 陣列
 */
export function getPersistKeys() {
    return SETTINGS_CONFIG.filter(item => item.persist).map(item => item.key)
}
