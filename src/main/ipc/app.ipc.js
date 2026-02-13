import { app, ipcMain } from 'electron'
import fs from 'fs'
import path from 'path'

// ========================================
// App & Settings IPC 模組
// 負責應用程式層級的功能：版本資訊、Changelog、設定存取
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
    // 確保目錄存在 (通常 userData 目錄已存在，但保險起見)
    // fs.mkdirSync(app.getPath('userData'), { recursive: true }) 
    
    // 先讀取舊設定以進行合併 (Optional,視需求，這裡直接覆蓋或合併)
    // 簡單起見，這裡假設前端傳來的是完整需儲存的片段，我們將其與現有檔案合併
    let currentSettings = {}
    if (fs.existsSync(settingsPath)) {
       try {
         currentSettings = JSON.parse(fs.readFileSync(settingsPath, 'utf-8'))
       } catch (e) { /* ignore corruption */ }
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
 * 註冊 App 相關的 IPC Handler
 * @param {Electron.IpcMain} ipcMain 
 */
export function registerAppIpc(ipcMain) {
  // 覆蓋 index.js 原本的簡單實作，改用統一管理
  // 注意：index.js 如果已經註冊過 app:getVersion，這裡會重複註冊導致錯誤
  // 策略：index.js 裡面的 app:getVersion 應該移除，統一在這裡註冊
  // 或者保留 index.js 的通用性，這裡只註冊特有的。
  // 根據 Plan，我們將 app:getVersion 移動到這裡。
  
  ipcMain.removeHandler('app:getVersion') // 移除舊的 (如果有的話)
  ipcMain.handle('app:getVersion', () => app.getVersion())

  ipcMain.handle('app:getChangelog', getChangelog)
  ipcMain.handle('settings:get', getSettings)
  ipcMain.handle('settings:save', saveSettings)
}
