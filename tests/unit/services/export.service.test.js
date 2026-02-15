import { describe, it, expect, vi, beforeEach } from 'vitest';
import exportService from '../../../src/main/services/export.service.js';
import bomService from '../../../src/main/services/bom.service.js';
import bomRevisionRepo from '../../../src/main/database/repositories/bom-revision.repo.js';

// Mock Electron
vi.mock('electron', () => ({
    app: {
        isPackaged: false,
        getPath: vi.fn().mockReturnValue('/tmp')
    }
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

        await exportService.exportBom(bomRevisionId, outputFilePath);

        // Verify calls
        expect(bomRevisionRepo.findById).toHaveBeenCalledWith(bomRevisionId);
        // loadTemplate should be called for each unique template file
        // In default config, all 8 sheets use 'ebom.xlsx', so it should be called (at least) once.
        // Due to caching implementation, it might be called just once if sequential.
        expect(loadTemplate).toHaveBeenCalledWith('ebom.xlsx');

        expect(createWorkbook).toHaveBeenCalled();

        // Should call appendSheetFromTemplate 8 times (for 8 sheets in export definition)
        expect(appendSheetFromTemplate).toHaveBeenCalledTimes(8);

        // Verify first call (ALL sheet)
        // Since we mocked getWorksheet to return a sheet, the logic uses the requested source name 'ALL'
        expect(appendSheetFromTemplate).toHaveBeenNthCalledWith(
            1,
            mockTargetWb,
            mockTemplateWb,
            'ALL', // source
            'ALL', // target
            expect.objectContaining({
                meta: expect.objectContaining({ PROJECT_CODE: 'TEST-PROJ' }),
                items: expect.any(Array)
            })
        );

        expect(saveWorkbook).toHaveBeenCalledWith(mockTargetWb, outputFilePath);
    });
});
