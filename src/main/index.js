import { app, shell, BrowserWindow, ipcMain } from 'electron'
import { join } from 'path'
import { electronApp, optimizer, is } from '@electron-toolkit/utils'
import { registerAllIpcHandlers } from './ipc/index.js'

app.commandLine.appendSwitch('disable-gpu-sandbox')

// ========================================
// BOMIX 主行程進入點
// 負責建立視窗、註冊 IPC 通道、管理應用程式生命週期
// ========================================

/**
 * 建立主應用程式視窗。
 *
 * 設定視窗尺寸、Preload 腳本、開發/正式環境的載入邏輯。
 */
function createWindow() {
    // 建立瀏覽器視窗
    const mainWindow = new BrowserWindow({
        width: 1280,
        height: 800,
        minWidth: 960,
        minHeight: 640,
        show: false, // 等待 ready-to-show 事件再顯示，避免白屏閃爍
        autoHideMenuBar: true,
        title: 'BOMIX',
        webPreferences: {
            preload: join(__dirname, '../preload/index.js'),
            sandbox: false // better-sqlite3 需要關閉沙箱
        }
    })

    // 視窗準備好後再顯示，提供更好的使用者體驗
    mainWindow.on('ready-to-show', () => {
        mainWindow.show()
    })

    // 攔截外部連結，使用系統預設瀏覽器開啟
    mainWindow.webContents.setWindowOpenHandler((details) => {
        shell.openExternal(details.url)
        return { action: 'deny' }
    })

    // 根據環境載入不同的內容
    // 開發模式：載入 Vite Dev Server
    // 正式模式：載入編譯後的 HTML 檔案
    if (is.dev && process.env['ELECTRON_RENDERER_URL']) {
        mainWindow.loadURL(process.env['ELECTRON_RENDERER_URL'])
    } else {
        mainWindow.loadFile(join(__dirname, '../renderer/index.html'))
    }

    return mainWindow
}

// ========================================
// 應用程式生命週期管理
// ========================================

app.whenReady().then(() => {
    // 設定 app user model id（Windows 工作列群組用）
    electronApp.setAppUserModelId('com.bomix.app')

    // 開發模式下，F12 開啟 DevTools、Ctrl+R 重新載入
    app.on('browser-window-created', (_, window) => {
        optimizer.watchWindowShortcuts(window)
    })

    // 註冊所有 IPC 通道處理器
    registerAllIpcHandlers(ipcMain)

    // 建立主視窗
    createWindow()

    // macOS：點擊 dock 圖標重新建立視窗
    app.on('activate', function () {
        if (BrowserWindow.getAllWindows().length === 0) {
            createWindow()
        }
    })
})

// 所有視窗關閉時退出應用程式（macOS 除外）
app.on('window-all-closed', () => {
    if (process.platform !== 'darwin') {
        app.quit()
    }
})
