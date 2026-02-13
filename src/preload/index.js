import { contextBridge, ipcRenderer } from 'electron'
import { electronAPI } from '@electron-toolkit/preload'

// ========================================
// Preload 腳本
// 透過 contextBridge 安全地暴露 API 給渲染層
// 渲染層只能使用此處定義的 API，無法直接存取 Node.js 或 Electron 模組
// ========================================

/**
 * 自訂 API — 提供給渲染層使用的業務功能介面。
 *
 * 所有主行程功能都必須透過此 API 暴露，
 * 確保渲染層與主行程之間的安全隔離。
 */
const api = {
    // --- 應用程式資訊 ---
    getVersion: () => ipcRenderer.invoke('app:getVersion'),

    // TODO: Phase 2 — 系列管理 API
    series: {
        create: (path, description) => ipcRenderer.invoke('series:create', path, description),
        open: (path) => ipcRenderer.invoke('series:open', path),
        getMeta: () => ipcRenderer.invoke('series:getMeta'),
        updateMeta: (desc) => ipcRenderer.invoke('series:updateMeta', desc),
    },

    // TODO: Phase 3 — 專案管理 API
    project: {
        create: (projectCode, description) => ipcRenderer.invoke('project:create', projectCode, description),
        getAll: () => ipcRenderer.invoke('project:getAll'),
        getById: (id) => ipcRenderer.invoke('project:getById', id),
        update: (id, description) => ipcRenderer.invoke('project:update', id, description),
        delete: (id) => ipcRenderer.invoke('project:delete', id),
    },

    // TODO: Phase 4 — BOM 管理 API
    // bom: { ... },

    // TODO: Phase 5 — Excel 匯入匯出 API
    // excel: { ... },
}

// 在 contextBridge 可用時，安全暴露 API
if (process.contextIsolated) {
    try {
        contextBridge.exposeInMainWorld('electron', electronAPI)
        contextBridge.exposeInMainWorld('api', api)
    } catch (error) {
        console.error('contextBridge 暴露 API 失敗:', error)
    }
} else {
    // contextIsolation 未啟用時的備用方案
    window.electron = electronAPI
    window.api = api
}
