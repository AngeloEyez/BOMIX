/**
 * @file src/main/ipc/theme.ipc.js
 * @description 主題管理 IPC 介面
 */

import { ipcMain } from 'electron'
import themeService from '../services/theme.service.js'

/**
 * 註冊主題相關 IPC Handler
 * @param {Electron.IpcMain} ipc
 */
export function registerThemeIpc(ipc) {
    /**
     * 取得主題列表
     * Invoke: 'theme:get-list'
     * Return: { success: true, data: Array }
     */
    ipc.handle('theme:get-list', async () => {
        try {
            const themes = await themeService.getThemes()
            return { success: true, data: themes }
        } catch (error) {
            console.error('IPC theme:get-list error:', error)
            return { success: false, error: error.message }
        }
    })

    /**
     * 取得主題屬性
     * Invoke: 'theme:get-attributes', themeId
     * Return: { success: true, data: Object }
     */
    ipc.handle('theme:get-attributes', async (event, themeId) => {
        try {
            const attributes = await themeService.getThemeAttributes(themeId)
            
            if (!attributes) {
                 return { success: false, error: `Theme ${themeId} not found` }
            }
            
            return { success: true, data: attributes }
        } catch (error) {
            console.error(`IPC theme:get-attributes error for ${themeId}:`, error)
            return { success: false, error: error.message }
        }
    })
}
