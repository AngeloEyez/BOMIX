/**
 * @file src/main/services/theme.service.js
 * @description UI 主題管理服務 (Theme Management Service)
 * @module services/theme
 */

import { app } from 'electron'
import path from 'path'
import fs from 'fs/promises'

class ThemeService {
    constructor() {
        this.themesPath = this._getThemesPath()
        this.cache = new Map() // Cache content to avoid repeated IO
    }

    /**
     * 取得 themes 資料夾路徑
     * 开发环境: project/resources/themes
     * 生产环境: resource/themes (in ASAR or adjacent)
     */
    _getThemesPath() {
        if (app.isPackaged) {
            return path.join(process.resourcesPath, 'themes')
        } else {
            return path.join(app.getAppPath(), 'resources', 'themes')
        }
    }

    /**
     * 掃描並回傳所有可用主題列表
     * @returns {Promise<Array<{id: string, name: string, author: string}>>}
     */
    async getThemes() {
        try {
            const files = await fs.readdir(this.themesPath)
            const themes = []

            for (const file of files) {
                if (file.endsWith('.theme')) {
                    const themeId = path.basename(file, '.theme')
                    try {
                        const content = await this._readThemeFile(file)
                        themes.push({
                            id: themeId,
                            name: content.name || themeId,
                            author: content.author || 'Unknown'
                        })
                    } catch (err) {
                        console.error(`[ThemeService] Failed to load theme ${file}:`, err)
                    }
                }
            }

            // Ensure default exists in list if file is missing (failsafe)
            if (themes.length === 0) {
                return [{ id: 'default', name: 'Default Blue', author: 'System' }]
            }

            return themes
        } catch (error) {
            console.error('[ThemeService] Error listing themes:', error)
            return []
        }
    }

    /**
     * 讀取指定主題的完整屬性 (CSS Variables)
     * @param {string} themeId
     * @returns {Promise<Object|null>}
     */
    async getThemeAttributes(themeId) {
        try {
            const filename = `${themeId}.theme`
            const content = await this._readThemeFile(filename)
            return content
        } catch (error) {
            console.error(`[ThemeService] Error getting attributes for ${themeId}:`, error)
            return null
        }
    }

    /**
     * 內部方法：讀取並解析 Theme 檔案
     * @param {string} filename
     */
    async _readThemeFile(filename) {
        // Enforce no cache in development for easier theme editing
        if (app.isPackaged && this.cache.has(filename)) {
            return this.cache.get(filename)
        }

        const filePath = path.join(this.themesPath, filename)
        const raw = await fs.readFile(filePath, 'utf-8')
        
        // Remove comments (//... and /*...*/) to allow "JSON with Comments"
        const jsonContent = raw.replace(/\/\/.*$/gm, '').replace(/\/\*[\s\S]*?\*\//g, '')
        
        const content = JSON.parse(jsonContent)
        
        // Only cache in production
        if (app.isPackaged) {
            this.cache.set(filename, content)
        }
        return content
    }
}

const themeService = new ThemeService()
export default themeService
