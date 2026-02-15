import { describe, it, expect, vi, beforeEach } from 'vitest';
import { importBom } from '../../../src/main/services/import.service.js';
import bomRevisionRepo from '../../../src/main/database/repositories/bom-revision.repo.js';
import partsRepo from '../../../src/main/database/repositories/parts.repo.js';
import secondSourceRepo from '../../../src/main/database/repositories/second-source.repo.js';
import projectRepo from '../../../src/main/database/repositories/project.repo.js';

// Local helper for consistency
const encode_cell = (cell) => `R${cell.r}C${cell.c}`;

// Mocks
vi.mock('xlsx', () => {
    return {
        default: {
            readFile: vi.fn(),
            utils: {
                decode_range: vi.fn(() => ({ s: { c: 0, r: 0 }, e: { c: 20, r: 100 } })),
                encode_cell: vi.fn((cell) => encode_cell(cell)),
                encode_range: vi.fn(() => 'RANGE')
            }
        }
    };
});

// Import mocked xlsx to setup return values in tests
import xlsx from 'xlsx';

vi.mock('../../../src/main/database/repositories/bom-revision.repo.js');
vi.mock('../../../src/main/database/repositories/parts.repo.js');
vi.mock('../../../src/main/database/repositories/second-source.repo.js', () => ({
    default: {
        create: vi.fn(),
        createMany: vi.fn(),
        findByBomRevision: vi.fn(),
        deleteByBomRevision: vi.fn(),
        update: vi.fn(),
        delete: vi.fn()
    }
}));
vi.mock('../../../src/main/database/repositories/project.repo.js');

describe('Import Service Logic - NPI/MP Mode & Status Updates', () => {
    
    const createMockWorkbook = (sheetsData) => {
        const Sheets = {};
        for (const [name, rows] of Object.entries(sheetsData)) {
            const sheet = {};
            sheet['!ref'] = 'RANGE'; // Mocked decode_range ignores this and returns fixed range

            rows.forEach((row, rIndex) => {
                const rowIndex = rIndex + 5; // Row 6
                const setCell = (c, v) => {
                    sheet[encode_cell({ r: rowIndex, c })] = { v };
                };

                setCell(0, row.item || '1');
                setCell(1, row.hhpn || 'PN-001');
                setCell(4, row.desc || 'Desc');
                setCell(5, row.supplier || 'Sup');
                setCell(6, row.supplier_pn || 'SPN-001');
                setCell(8, row.loc || '');
                setCell(9, row.ccl || 'N');
                setCell(11, row.remark || '');
            });
            Sheets[name] = sheet;
        }
        return { Sheets };
    };

    beforeEach(() => {
        vi.clearAllMocks();
        bomRevisionRepo.create.mockReturnValue({ id: 999 });
        projectRepo.findById.mockReturnValue({ project_code: 'TEST' });
        // Default mock implementation
        partsRepo.createMany.mockImplementation(() => {});
        secondSourceRepo.createMany.mockImplementation(() => {});
    });

    it('should handle NI sheet: Always ADD status X', () => {
        const workbook = createMockWorkbook({
            SMD: [{ loc: 'C1', item: 1 }],
            NI: [{ loc: 'C1', item: 1 }, { loc: 'C2', item: 2 }]
        });
        xlsx.readFile.mockReturnValue(workbook);

        importBom('test.xls', 1, 'DB', '0.1');

        expect(partsRepo.createMany).toHaveBeenCalled();
        const insertedParts = partsRepo.createMany.mock.calls[0][0];
        
        const c1Parts = insertedParts.filter(p => p.location === 'C1');
        // Expect 2 records: one 'I' from SMD, one 'X' from NI
        expect(c1Parts.length).toBe(2);
        expect(c1Parts.find(p => p.bom_status === 'I')).toBeTruthy();
        expect(c1Parts.find(p => p.bom_status === 'X')).toBeTruthy();

        const c2Parts = insertedParts.filter(p => p.location === 'C2');
        expect(c2Parts.length).toBe(1);
        expect(c2Parts[0].bom_status).toBe('X');
    });

    it('should handle NPI Mode: PROTO updates existing status P', () => {
        const workbook = createMockWorkbook({
            SMD: [{ loc: 'C1' }],
            PROTO: [{ loc: 'C1' }]
        });
        xlsx.readFile.mockReturnValue(workbook);

        importBom('test.xls', 1, 'DB', '0.1');

        const insertedParts = partsRepo.createMany.mock.calls[0][0];
        const c1Parts = insertedParts.filter(p => p.location === 'C1');

        expect(c1Parts.length).toBe(1);
        expect(c1Parts[0].bom_status).toBe('P');
        expect(c1Parts[0].type).toBe('SMD');
    });

    it('should handle NPI Mode: PROTO adds new status P', () => {
        const workbook = createMockWorkbook({
            SMD: [{ loc: 'C1' }],
            PROTO: [{ loc: 'C1' }, { loc: 'C2' }]
        });
        xlsx.readFile.mockReturnValue(workbook);

        importBom('test.xls', 1, 'DB', '0.1');

        const insertedParts = partsRepo.createMany.mock.calls[0][0];
        
        const c1Parts = insertedParts.filter(p => p.location === 'C1');
        expect(c1Parts.length).toBe(1);
        expect(c1Parts[0].bom_status).toBe('P');

        const c2Parts = insertedParts.filter(p => p.location === 'C2');
        expect(c2Parts.length).toBe(1);
        expect(c2Parts[0].bom_status).toBe('P');
    });

    it('should handle MP Mode: PROTO adds new status P (not overwrite)', () => {
        const workbook = createMockWorkbook({
            SMD: [{ loc: 'C1' }], // Removed C2 to avoid Proto overlap triggering NPI
            MP: [{ loc: 'C1' }], 
            PROTO: [{ loc: 'C2' }]
        });
        xlsx.readFile.mockReturnValue(workbook);

        importBom('test.xls', 1, 'DB', '0.1');
        
        // Check Mode
        expect(bomRevisionRepo.create).toHaveBeenCalledWith(expect.objectContaining({ mode: 'MP' }));

        const insertedParts = partsRepo.createMany.mock.calls[0][0];
        
        // C1: Covered by MP -> Status M
        const c1Parts = insertedParts.filter(p => p.location === 'C1');
        expect(c1Parts[0].bom_status).toBe('M');

        // C2: In PROTO but not in Phase 1 -> Add new P
        // Since it's not in Phase 1, it's just a new part.
        const c2Parts = insertedParts.filter(p => p.location === 'C2');
        expect(c2Parts.length).toBe(1);
        expect(c2Parts[0].bom_status).toBe('P');
    });

    it('should handle MP Mode: MP updates existing status M', () => {
        const workbook = createMockWorkbook({
            SMD: [{ loc: 'C1' }],
            MP: [{ loc: 'C1' }]
        });
        xlsx.readFile.mockReturnValue(workbook);

        importBom('test.xls', 1, 'DB', '0.1');

        const insertedParts = partsRepo.createMany.mock.calls[0][0];
        const c1Parts = insertedParts.filter(p => p.location === 'C1');
        expect(c1Parts.length).toBe(1);
        expect(c1Parts[0].bom_status).toBe('M');
    });

    it('should handle NPI Mode: MP adds new status M (not overwrite)', () => {
        const workbook = createMockWorkbook({
            SMD: [{ loc: 'C1' }, { loc: 'C2' }],
            PROTO: [{ loc: 'C1' }],
            MP: [{ loc: 'C2' }]
        });
        xlsx.readFile.mockReturnValue(workbook);

        importBom('test.xls', 1, 'DB', '0.1');
        
        const insertedParts = partsRepo.createMany.mock.calls[0][0];

        // C1 (PROTO in NPI) -> Status P
        const c1Parts = insertedParts.filter(p => p.location === 'C1');
        expect(c1Parts.length).toBe(1);
        expect(c1Parts[0].bom_status).toBe('P');

        // C2 (MP in NPI) -> Status M (Add)
        const c2Parts = insertedParts.filter(p => p.location === 'C2');
        expect(c2Parts.length).toBe(2);
        expect(c2Parts.find(p => p.bom_status === 'I')).toBeTruthy();
        expect(c2Parts.find(p => p.bom_status === 'M')).toBeTruthy();
    });

});
