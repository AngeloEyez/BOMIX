/**
 * @file src/main/services/import-batch.service.js
 * @description 批次匯入服務 (Batch Import Service)
 *
 * 負責處理多檔案匯入時的前置作業：
 * 1. 檔名格式解析與驗證
 * 2. 根據 DB 或同批次狀態驗證 BOM Type 相依性
 * 3. 排列合法檔案的匯入順序並Enqueue
 *
 * @module services/import-batch
 */

import path from 'path'
import bomRevisionRepo from '../database/repositories/bom-revision.repo.js'
import projectRepo from '../database/repositories/project.repo.js'
import taskManager from './task-manager.service.js'
import importService from './import.service.js'

/**
 * 檔名解析的正則表達式
 * 格式範例: {project name}_EZBOM_{phase}_{version}_{bomtype}_20241008_WithProtoPart.xls
 */
const FILENAME_REGEX = /^(.+?)_EZBOM_(.+?)_(.+?)_(BOM|MatrixBOM)_.*?\.(xlsx?)$/i

/**
 * 解析檔名取得 Metadata
 * @param {string} filePath - 檔案路徑
 * @returns {Object|null} 解析結果，若不符格式則回傳 null
 */
export function parseImportFilename(filePath) {
    const filename = path.basename(filePath)
    const match = filename.match(FILENAME_REGEX)
    
    if (!match) return null

    return {
        projectName: match[1],
        phase: match[2],
        version: match[3],
        bomType: match[4].toUpperCase(), // 'BOM' 或 'MATRIXBOM'
        extension: `.${match[5].toLowerCase()}`,
        originalPath: filePath,
        originalName: filename
    }
}

/**
 * 分析傳入的檔案清單，進行驗證與相依性檢查
 * 
 * 規則:
 * 1. 必須是 .xls 或 .xlsx
 * 2. 必須符合檔名格式
 * 3. 若為非 BOM (e.g., MatrixBOM)，則必須確認 DB 中已存在對應的 BOM，或此批次中包含對應的 BOM。
 *
 * @param {Array<string>} filePaths - 要匯入的檔案路徑陣列
 * @param {Object} ctx - TaskManager Context 用於 log 與 progress (Optional)
 * @returns {{ validFiles: Array<Object>, errors: Array<Object> }}
 */
export async function analyzeFiles(filePaths, ctx = null) {
    const log = (msg, level = 'info') => ctx?.log?.(msg, level)
    const validFiles = []
    const errors = []

    // 1. 初步解析與檔名過濾
    const parsedFiles = []
    
    for (const filePath of filePaths) {
        const ext = path.extname(filePath).toLowerCase()
        if (ext !== '.xls' && ext !== '.xlsx') {
            const err = `略過非 Excel 檔案: ${path.basename(filePath)}`
            log(err, 'warn')
            errors.push({ file: filePath, error: err })
            continue
        }

        const parsed = parseImportFilename(filePath)
        if (!parsed) {
            const err = `檔名不符合匯入規則: ${path.basename(filePath)}`
            log(err, 'warn')
            errors.push({ file: filePath, error: err })
            continue
        }

        parsedFiles.push(parsed)
    }

    // 將同批次的 BOM 收集起來方便快速查表 (Project_Phase_Version)
    const batchBomKeys = new Set(
        parsedFiles
            .filter(f => f.bomType === 'BOM')
            .map(f => `${f.projectName}_${f.phase}_${f.version}`.toLowerCase())
    )

    // 2. 檢查相依性與順序建構
    // 同批次中：一般 BOM 優先執行
    const bomFiles = []
    const matrixBomFiles = []

    for (const file of parsedFiles) {
        const key = `${file.projectName}_${file.phase}_${file.version}`.toLowerCase()

        if (file.bomType === 'BOM') {
            bomFiles.push(file)
            continue
        }

        if (file.bomType === 'MATRIXBOM') {
            // 檢查相依性：這批檔案內有無對應的主 BOM？
            if (batchBomKeys.has(key)) {
                matrixBomFiles.push(file)
                continue
            }

            // 若這批沒有，檢查資料庫是否有？
            const existProject = projectRepo.findByCode(file.projectName)
            let hasDbBom = false

            if (existProject) {
                const revisions = bomRevisionRepo.findByProject(existProject.id)
                hasDbBom = revisions.some(r => 
                    r.phase_name.toLowerCase() === file.phase.toLowerCase() && 
                    r.version.toLowerCase() === file.version.toLowerCase()
                )
            }

            if (hasDbBom) {
                matrixBomFiles.push(file)
            } else {
                const err = `缺少相依的主 BOM 紀錄 (專案:${file.projectName}, Phase:${file.phase}, Ver:${file.version})，已略過: ${file.originalName}`
                log(err, 'error')
                errors.push({ file: file.originalPath, error: err })
            }
        }
    }

    // 排序完成：先處理 BOM 再處理 MatrixBOM
    validFiles.push(...bomFiles, ...matrixBomFiles)

    return { validFiles, errors }
}

/**
 * 建立匯入批次任務，並在批次開始時解析然後依序派發子任務
 * 
 * @param {Array<string>} filePaths 
 * @returns {string} Batch Task ID
 */
export function enqueueBatchImport(filePaths) {
    return taskManager.enqueue('BATCH_IMPORT', {
        title: `批次匯入 (${filePaths.length} 個檔案)`,
        metadata: { filePaths },
        executeFn: async (ctx) => {
            ctx.updateProgress(10, '解析檔名與分析相依性...')
            await ctx.yield()

            const { validFiles, errors } = await analyzeFiles(filePaths, ctx)

            // 將解析錯誤的顯示在最終結果中
            if (errors.length > 0) {
                ctx.log(`共 ${errors.length} 個檔案解析或相依驗證失敗`, 'warn')
            }

            if (validFiles.length === 0) {
                throw new Error('沒有可用於匯入的合法 Excel 檔案')
            }

            ctx.updateProgress(30, '準備分配子任務...')
            
            // 替合格的每個檔案建立一個獨立的 Task
            let enqueuedCount = 0
            for (const file of validFiles) {
                const taskTitle = `匯入 ${file.bomType}: ${file.originalName}`
                // 如果是第一版，可以直接叫用 importBom 並透過 task manager
                taskManager.enqueue('IMPORT_BOM', {
                    title: taskTitle,
                    metadata: { 
                        filePath: file.originalPath, 
                        projectName: file.projectName, 
                        phaseName: file.phase, 
                        version: file.version,
                        bomType: file.bomType // 保留未來支援不同型態的空間
                    },
                    executeFn: async (childCtx) => {
                        await childCtx.yield()
                        return importService.importBom(
                            file.projectName, 
                            file.phase, 
                            file.version, 
                            file.bomType,
                            file.originalPath, 
                            childCtx
                        )
                    }
                })
                enqueuedCount++
            }

            ctx.updateProgress(100, `已將 ${enqueuedCount} 個檔案排入獨立的匯入隊列`)
            return {
                message: `成功排入 ${enqueuedCount} 個檔案`,
                errors
            }
        }
    })
}

export default {
    parseImportFilename,
    analyzeFiles,
    enqueueBatchImport
}
