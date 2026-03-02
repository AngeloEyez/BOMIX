import { describe, it, expect, vi, beforeEach } from 'vitest'
import path from 'path'

// Mock dependencies
import bomRevisionRepo from '../../src/main/database/repositories/bom-revision.repo.js'
import projectRepo from '../../src/main/database/repositories/project.repo.js'
import taskManager from '../../src/main/services/task-manager.service.js'
import importService from '../../src/main/services/import.service.js'

// 必須 mock 所有 dependencies 之後才 import 實際負責驗證的函式
vi.mock('../../src/main/database/repositories/bom-revision.repo.js', () => {
    return {
        default: {
            findByProject: vi.fn(),
            create: vi.fn(),
            delete: vi.fn()
        }
    }
})

vi.mock('../../src/main/database/repositories/project.repo.js', () => {
    return {
        default: {
            findByCode: vi.fn(),
            findById: vi.fn(),
            create: vi.fn()
        }
    }
})

vi.mock('../../src/main/services/task-manager.service.js', () => {
    return {
        default: {
            enqueue: vi.fn()
        }
    }
})

vi.mock('../../src/main/services/import.service.js', () => {
    return {
        default: {
            importBom: vi.fn()
        }
    }
})

import { parseImportFilename, analyzeFiles, enqueueBatchImport } from '../../src/main/services/import-batch.service.js'

describe('import-batch.service 測試', () => {

    beforeEach(() => {
        vi.clearAllMocks()
    })

    describe('parseImportFilename 檔名解析測試', () => {
        it('應該成功解析合法的 BOM 檔名', () => {
            const result = parseImportFilename('C:\\TEMP\\ProjectX_EZBOM_EVT_0.1_BOM_20241008.xlsx')
            expect(result).not.toBeNull()
            expect(result.projectName).toBe('ProjectX')
            expect(result.phase).toBe('EVT')
            expect(result.version).toBe('0.1')
            expect(result.bomType).toBe('BOM')
            expect(result.extension).toBe('.xlsx')
        })

        it('應該成功解析合法的 MatrixBOM 檔名 (大小寫相容)', () => {
            const result = parseImportFilename('/usr/local/Test_ProJ_EZBOM_SI_1.0_MatrixBom_2023.xls')
            expect(result).not.toBeNull()
            expect(result.projectName).toBe('Test_ProJ')
            expect(result.phase).toBe('SI')
            expect(result.version).toBe('1.0')
            expect(result.bomType).toBe('MATRIXBOM')
        })

        it('應該拒絕不符合格式的檔名', () => {
            const result1 = parseImportFilename('ProjectX_BOM_0.1.xls') // 缺少 EZBOM / Phase
            expect(result1).toBeNull()

            const result2 = parseImportFilename('ProjectX_EZBOM_SI_1.0_UnknownType_2024.xls') // 不支援的 BOM Type
            expect(result2).toBeNull()
        })
    })

    describe('analyzeFiles 批次分析測試', () => {
        it('應該過濾非 Excel 檔案與不合格式檔名', async () => {
            const files = ['ProjectX_EZBOM_P1_V1_BOM_2024.xlsx', 'invalid_file.txt', 'fake_EZBOM.pdf']
            const { validFiles, errors } = await analyzeFiles(files)

            expect(validFiles.length).toBe(1)
            expect(errors.length).toBe(2)
            expect(errors[0].error).toContain('略過非 Excel 檔案')
        })

        it('排序規則：必須讓一般 BOM 排在前面', async () => {
            const files = [
                'ProjA_EZBOM_EVT_0.1_MatrixBOM_date.xlsx',
                'ProjA_EZBOM_EVT_0.1_BOM_date.xlsx'
            ]
            const { validFiles } = await analyzeFiles(files)

            expect(validFiles.length).toBe(2)
            expect(validFiles[0].bomType).toBe('BOM') // 第二個檔被排到前面了
            expect(validFiles[1].bomType).toBe('MATRIXBOM')
        })

        it('當 MatrixBOM 的主 BOM 不在本批次亦不在資料庫時，應該排除該 MatrixBOM', async () => {
            // Mock: DB 中找不到該專案與對應的 BOM
            projectRepo.findByCode.mockReturnValue(null)

            const files = ['ProjMissing_EZBOM_PVT_2.0_MatrixBOM_2024.xlsx']
            const { validFiles, errors } = await analyzeFiles(files)

            expect(validFiles.length).toBe(0)
            expect(errors.length).toBe(1)
            expect(errors[0].error).toContain('缺少相依的主 BOM 紀錄')
        })

        it('當 MatrixBOM 的主 BOM 在資料庫有紀錄時，應該允許匯入', async () => {
            // Mock DB
            projectRepo.findByCode.mockReturnValue({ id: 1, project_code: 'ProjDB' })
            bomRevisionRepo.findByProject.mockReturnValue([
                { phase_name: 'DVT', version: '1.5' } // 有 DVT 1.5
            ])

            const files = ['ProjDB_EZBOM_DVT_1.5_MatrixBOM_2024.xlsx']
            const { validFiles, errors } = await analyzeFiles(files)

            expect(errors.length).toBe(0)
            expect(validFiles.length).toBe(1)
        })
    })

    describe('enqueueBatchImport 佇列建立測試', () => {
        it('應該將符合條件的檔案丟入 TaskManager', async () => {
            taskManager.enqueue.mockReturnValue('BATCH_TASK_ID')

            const files = ['ProjX_EZBOM_SI_0.1_BOM_2024.xlsx']
            const result = await enqueueBatchImport(files)

            expect(result).toBe('BATCH_TASK_ID')
            expect(taskManager.enqueue).toHaveBeenCalledTimes(1)
            expect(taskManager.enqueue.mock.calls[0][0]).toBe('BATCH_IMPORT')
        })
    })
})
