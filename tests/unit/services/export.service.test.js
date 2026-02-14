import { describe, it, expect, vi, beforeEach } from 'vitest';
import exportService from '../../../src/main/services/export.service.js';
import bomService from '../../../src/main/services/bom.service.js';
import bomRevisionRepo from '../../../src/main/database/repositories/bom-revision.repo.js';

// Mock Electron to avoid 'app.isPackaged' error
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
    exportFromTemplate: vi.fn()
}));

import { exportFromTemplate } from '../../../src/main/services/excel-export/template-engine.js';

describe('Export Service', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('should export BOM to Excel using template engine', async () => {
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

        const mockBomView = [
            { item: 1, hhpn: 'PN1', type: 'SMD', bom_status: 'I', second_sources: [] },
            { item: 2, hhpn: 'PN2', type: 'PTH', bom_status: 'X', second_sources: [] }
        ];

        bomRevisionRepo.findById.mockReturnValue(mockRevision);
        bomService.getBomView.mockReturnValue(mockBomView);

        await exportService.exportBom(bomRevisionId, outputFilePath);

        expect(bomRevisionRepo.findById).toHaveBeenCalledWith(bomRevisionId);
        expect(bomService.getBomView).toHaveBeenCalledWith(bomRevisionId);

        expect(exportFromTemplate).toHaveBeenCalledTimes(1);
        expect(exportFromTemplate).toHaveBeenCalledWith(
            'ebom_template.xlsx',
            expect.objectContaining({
                meta: expect.objectContaining({
                    PROJECT_CODE: 'TEST-PROJ',
                    PHASE: 'EVT'
                }),
                items: expect.arrayContaining([
                    expect.objectContaining({ M_HHPN: 'PN1' }),
                    expect.objectContaining({ M_HHPN: 'PN2' })
                ])
            }),
            outputFilePath
        );
    });
});
