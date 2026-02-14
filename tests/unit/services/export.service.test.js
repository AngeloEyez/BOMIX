import { describe, it, expect, vi, beforeEach } from 'vitest';
import exportService from '../../../src/main/services/export.service.js';
import bomService from '../../../src/main/services/bom.service.js';
import bomRevisionRepo from '../../../src/main/database/repositories/bom-revision.repo.js';
import xlsx from 'xlsx';

vi.mock('../../../src/main/services/bom.service.js');
vi.mock('../../../src/main/database/repositories/bom-revision.repo.js');
vi.mock('../../../src/main/database/connection.js', () => ({
    default: {
        getDb: vi.fn()
    }
}));
vi.mock('xlsx', () => {
    return {
        default: {
            utils: {
                book_new: vi.fn(() => ({ SheetNames: [], Sheets: {} })),
                book_append_sheet: vi.fn(),
                aoa_to_sheet: vi.fn(() => ({}))
            },
            writeFile: vi.fn()
        }
    };
});

describe('Export Service', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('should export BOM to Excel', () => {
        const bomRevisionId = 1;
        const outputFilePath = 'output.xlsx';

        const mockRevision = {
            id: 1,
            project_id: 101,
            phase_name: 'EVT',
            version: '0.1',
            description: 'Test BOM',
            mode: 'NPI'
        };

        const mockBomView = [
            { item: 1, hhpn: 'PN1', type: 'SMD', bom_status: 'I', second_sources: [] },
            { item: 2, hhpn: 'PN2', type: 'PTH', bom_status: 'X', second_sources: [] }
        ];

        bomRevisionRepo.findById.mockReturnValue(mockRevision);
        bomService.getBomView.mockReturnValue(mockBomView);

        exportService.exportBom(bomRevisionId, outputFilePath);

        expect(bomRevisionRepo.findById).toHaveBeenCalledWith(bomRevisionId);
        expect(bomService.getBomView).toHaveBeenCalledWith(bomRevisionId);
        expect(xlsx.utils.book_new).toHaveBeenCalled();
        // Check if append_sheet was called 8 times (8 sheets)
        expect(xlsx.utils.book_append_sheet).toHaveBeenCalledTimes(8);
        expect(xlsx.writeFile).toHaveBeenCalledWith(expect.any(Object), outputFilePath);
    });
});
