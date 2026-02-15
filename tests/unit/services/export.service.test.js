
import { describe, it, expect, vi, beforeEach } from 'vitest';
import exportService from '../../../src/main/services/export.service.js';
import bomService from '../../../src/main/services/bom.service.js';
import bomRevisionRepo from '../../../src/main/database/repositories/bom-revision.repo.js';
import progressService, { TASK_STATUS } from '../../../src/main/services/progress.service.js';

// Mock Electron
vi.mock('electron', () => ({
    app: {
        isPackaged: false,
        getPath: vi.fn().mockReturnValue('/tmp')
    }
}));

// Mock fs/promises
vi.mock('fs/promises', () => ({
    default: {
        copyFile: vi.fn().mockResolvedValue(undefined),
        unlink: vi.fn().mockResolvedValue(undefined)
    },
    copyFile: vi.fn().mockResolvedValue(undefined),
    unlink: vi.fn().mockResolvedValue(undefined)
}));

// Mock services and repositories
vi.mock('../../../src/main/services/bom.service.js');
vi.mock('../../../src/main/database/repositories/bom-revision.repo.js');
vi.mock('../../../src/main/database/connection.js', () => ({
    default: {
        getDb: vi.fn()
    }
}));

// Mock template engine
vi.mock('../../../src/main/services/excel-export/template-engine.js', () => ({
    loadTemplate: vi.fn(),
    createWorkbook: vi.fn(),
    appendSheetFromTemplate: vi.fn(),
    saveWorkbook: vi.fn(),
    exportFromTemplate: vi.fn()
}));

import {
    loadTemplate,
    createWorkbook,
    appendSheetFromTemplate,
    saveWorkbook
} from '../../../src/main/services/excel-export/template-engine.js';
import fs from 'fs/promises';

describe('Export Service', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('should export multi-sheet BOM', async () => {
        const bomRevisionId = 1;
        const outputFilePath = 'output.xlsx';

        const mockRevision = {
            id: 1,
            project_id: 101,
            phase_name: 'EVT',
            version: '0.1',
            description: 'Test BOM',
            mode: 'NPI',
            project_code: 'TEST-PROJ'
        };

        const mockTemplateWb = {
            getWorksheet: vi.fn().mockReturnValue({ name: 'Sheet1' }),
            worksheets: [{ name: 'Sheet1' }]
        };
        const mockTargetWb = {};

        // Mock return values
        bomRevisionRepo.findById.mockReturnValue(mockRevision);
        loadTemplate.mockResolvedValue(mockTemplateWb);
        createWorkbook.mockReturnValue(mockTargetWb);

        // Mock executeView results
        bomService.executeView.mockReturnValue([
            { item: 1, hhpn: 'PN1', type: 'SMD', bom_status: 'I', second_sources: [] }
        ]);

        // Start export
        const { taskId } = exportService.exportBom(bomRevisionId, outputFilePath);
        expect(taskId).toBeDefined();

        // Wait for task completion
        await new Promise((resolve, reject) => {
            const check = () => {
                const task = progressService.getTask(taskId);
                if (task.status === TASK_STATUS.COMPLETED) resolve();
                else if (task.status === TASK_STATUS.FAILED) reject(task.error);
                else setTimeout(check, 10);
            };
            check();
        });

        // Verify calls
        expect(bomRevisionRepo.findById).toHaveBeenCalledWith(bomRevisionId);
        expect(loadTemplate).toHaveBeenCalledWith('ebom.xlsx');
        expect(createWorkbook).toHaveBeenCalled();
        expect(appendSheetFromTemplate).toHaveBeenCalledTimes(8);

        // Verify saveWorkbook called with temp path
        expect(saveWorkbook).toHaveBeenCalledWith(
            mockTargetWb,
            expect.stringMatching(/\/tmp\/bom_export_.*\.xlsx/)
        );

        // Verify copyFile and unlink called (instead of rename)
        expect(fs.copyFile).toHaveBeenCalledWith(
            expect.stringMatching(/\/tmp\/bom_export_.*\.xlsx/),
            outputFilePath
        );
        expect(fs.unlink).toHaveBeenCalledWith(
            expect.stringMatching(/\/tmp\/bom_export_.*\.xlsx/)
        );
    });
});
