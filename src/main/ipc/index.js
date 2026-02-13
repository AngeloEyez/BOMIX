// ========================================
// IPC Handler 統一註冊模組
// 集中管理所有 IPC 通道，方便維護與擴展
// ========================================

// TODO: Phase 2 將在此引入各模組的 IPC handler
// import { registerSeriesIpc } from './series.ipc.js'
// import { registerProjectIpc } from './project.ipc.js'
// import { registerBomIpc } from './bom.ipc.js'
// import { registerExcelIpc } from './excel.ipc.js'

/**
 * 註冊所有 IPC 通道處理器。
 *
 * 此函數為 IPC 通訊的統一入口，由主行程啟動時呼叫。
 * 各模組的 IPC handler 將在對應的開發階段逐步加入。
 *
 * @param {Electron.IpcMain} ipcMain - Electron IPC 主行程實例
 */
export function registerAllIpcHandlers(ipcMain) {
    // --- 應用程式基本資訊 ---
    ipcMain.handle('app:getVersion', () => {
        const { app } = require('electron')
        return app.getVersion()
    })

    // TODO: Phase 2 — 系列管理 IPC
    // registerSeriesIpc(ipcMain)

    // TODO: Phase 3 — 專案管理 IPC
    // registerProjectIpc(ipcMain)

    // TODO: Phase 4 — BOM 管理 IPC
    // registerBomIpc(ipcMain)

    // TODO: Phase 5 — Excel 匯入匯出 IPC
    // registerExcelIpc(ipcMain)
}
