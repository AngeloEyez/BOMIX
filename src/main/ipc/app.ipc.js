import { app, ipcMain, dialog, BrowserWindow } from 'electron'
import fs from 'fs'
import path from 'path'

// ========================================
// App & Settings IPC 模組
// 負責應用程式層級的功能：版本資訊、Changelog、設定存取、檔案對話框
// ========================================

/**
 * 取得 Changelog 內容
 * 自動判斷開發環境與生產環境的檔案路徑
 */
async function getChangelog() {
  let changelogPath
  if (app.isPackaged) {
    // 生產環境：資源目錄 (resources/CHANGELOG.md)
    changelogPath = path.join(process.resourcesPath, 'CHANGELOG.md')
  } else {
    // 開發環境：專案根目錄
    changelogPath = path.join(process.cwd(), 'CHANGELOG.md')
  }

  try {
    const content = fs.readFileSync(changelogPath, 'utf-8')
    return { success: true, data: content }
  } catch (error) {
    console.error('讀取 CHANGELOG.md 失敗:', error)
    return { success: false, error: '無法讀取更新記錄檔案' }
  }
}

/**
 * 取得使用者設定
 * 從 userData 目錄讀取 settings.json
 */
async function getSettings() {
  const settingsPath = path.join(app.getPath('userData'), 'settings.json')
  try {
    if (fs.existsSync(settingsPath)) {
      const content = fs.readFileSync(settingsPath, 'utf-8')
      return { success: true, data: JSON.parse(content) }
    }
    return { success: true, data: {} } // 預設空設定
  } catch (error) {
    console.error('讀取 settings.json 失敗:', error)
    return { success: false, error: '無法讀取設定檔' }
  }
}

/**
 * 儲存使用者設定
 * 寫入 settings.json 到 userData 目錄
 */
async function saveSettings(event, newSettings) {
  const settingsPath = path.join(app.getPath('userData'), 'settings.json')
  try {
    // 先讀取舊設定以進行合併
    let currentSettings = {}
    if (fs.existsSync(settingsPath)) {
       try {
         currentSettings = JSON.parse(fs.readFileSync(settingsPath, 'utf-8'))
       } catch (e) { /* 忽略損壞的設定檔 */ }
    }

    const updatedSettings = { ...currentSettings, ...newSettings }
    fs.writeFileSync(settingsPath, JSON.stringify(updatedSettings, null, 2), 'utf-8')
    return { success: true }
  } catch (error) {
    console.error('寫入 settings.json 失敗:', error)
    return { success: false, error: '無法儲存設定檔' }
  }
}

/**
 * 顯示開啟檔案對話框。
 *
 * @param {Electron.IpcMainInvokeEvent} event - IPC 事件
 * @param {Object} options - 對話框選項
 * @param {string} [options.title] - 對話框標題
 * @param {Array<{name: string, extensions: string[]}>} [options.filters] - 檔案篩選器
 * @returns {Promise<{success: boolean, data?: string, canceled?: boolean}>}
 */
async function showOpenDialog(event, options = {}) {
  try {
    const win = BrowserWindow.getFocusedWindow()
    const result = await dialog.showOpenDialog(win, {
      title: options.title || '開啟檔案',
      filters: options.filters || [{ name: 'BOMIX 系列檔', extensions: ['bomix'] }],
      properties: ['openFile'],
    })

    if (result.canceled) {
      return { success: true, canceled: true }
    }

    return { success: true, data: result.filePaths[0] }
  } catch (error) {
    console.error('開啟檔案對話框失敗:', error)
    return { success: false, error: error.message }
  }
}

/**
 * 顯示儲存檔案對話框。
 *
 * @param {Electron.IpcMainInvokeEvent} event - IPC 事件
 * @param {Object} options - 對話框選項
 * @param {string} [options.title] - 對話框標題
 * @param {string} [options.defaultPath] - 預設檔案名稱
 * @param {Array<{name: string, extensions: string[]}>} [options.filters] - 檔案篩選器
 * @returns {Promise<{success: boolean, data?: string, canceled?: boolean}>}
 */
async function showSaveDialog(event, options = {}) {
  try {
    const win = BrowserWindow.getFocusedWindow()
    const result = await dialog.showSaveDialog(win, {
      title: options.title || '儲存檔案',
      defaultPath: options.defaultPath || 'untitled.bomix',
      filters: options.filters || [{ name: 'BOMIX 系列檔', extensions: ['bomix'] }],
    })

    if (result.canceled) {
      return { success: true, canceled: true }
    }

    return { success: true, data: result.filePath }
  } catch (error) {
    console.error('儲存檔案對話框失敗:', error)
    return { success: false, error: error.message }
  }
}

/**
 * 註冊 App 相關的 IPC Handler
 * @param {Electron.IpcMain} ipcMain
 */
export function registerAppIpc(ipcMain) {
  ipcMain.removeHandler('app:getVersion') // 移除舊的 (如果有的話)
  ipcMain.handle('app:getVersion', () => app.getVersion())

  ipcMain.handle('app:getChangelog', getChangelog)
  ipcMain.handle('settings:get', getSettings)
  ipcMain.handle('settings:save', saveSettings)

  // 檔案對話框
  ipcMain.handle('dialog:showOpen', showOpenDialog)
  ipcMain.handle('dialog:showSave', showSaveDialog)
}
