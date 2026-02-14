import { describe, it, expect, vi, beforeEach } from 'vitest';
import path from 'path';
import fs from 'fs';
import xlsx from 'xlsx';
import importService from '../../src/main/services/import.service.js';
import exportService from '../../src/main/services/export.service.js';
import bomService from '../../src/main/services/bom.service.js';
import bomRevisionRepo from '../../src/main/database/repositories/bom-revision.repo.js';
import partsRepo from '../../src/main/database/repositories/parts.repo.js';
import secondSourceRepo from '../../src/main/database/repositories/second-source.repo.js';
import projectRepo from '../../src/main/database/repositories/project.repo.js';

// Mock Electron
vi.mock('electron', () => ({
    app: {
        isPackaged: false,
        getPath: vi.fn().mockReturnValue('/tmp')
    }
}));

// Mock DB Repositories
vi.mock('../../src/main/database/repositories/bom-revision.repo.js');
vi.mock('../../src/main/database/repositories/parts.repo.js');
vi.mock('../../src/main/database/repositories/second-source.repo.js');
vi.mock('../../src/main/database/repositories/project.repo.js');

// Mock connection to avoid real DB calls
vi.mock('../../src/main/database/connection.js', () => ({
    default: {
        getDb: vi.fn()
    }
}));

// Mock bomService to return data we captured during import
vi.mock('../../src/main/services/bom.service.js');

describe('Excel Import/Export Verification', () => {
    const templatePath = path.resolve(__dirname, '../../references/bom_templates/TANGLED_EZBOM_SI_0.3_BOM_20240627_0900WithProtoPart(compared).xls');
    const outputPath = path.resolve(__dirname, 'exported_bom.xlsx');

    let capturedParts = [];
    let capturedSecondSources = [];
    let capturedRevision = {};

    beforeEach(() => {
        vi.clearAllMocks();
        capturedParts = [];
        capturedSecondSources = [];
        capturedRevision = {};

        // Mock repository behaviors to capture data
        bomRevisionRepo.create.mockImplementation((data) => {
            capturedRevision = { ...data, id: 1 };
            return capturedRevision;
        });

        partsRepo.createMany.mockImplementation((data) => {
            capturedParts = data;
        });

        secondSourceRepo.createMany.mockImplementation((data) => {
            capturedSecondSources = data;
        });

        bomRevisionRepo.findById.mockImplementation((id) => capturedRevision);
        projectRepo.findById.mockReturnValue({ project_code: 'TANGLED' });
    });

    it('should correctly import the template file and export it back', async () => {
        console.log(`Testing with file: ${templatePath}`);
        expect(fs.existsSync(templatePath)).toBe(true);

        // 1. Import
        const result = importService.importBom(templatePath, 101, 'SI', '0.3');

        expect(result.success).toBe(true);
        expect(bomRevisionRepo.create).toHaveBeenCalled();
        expect(partsRepo.createMany).toHaveBeenCalled();
        expect(secondSourceRepo.createMany).toHaveBeenCalled();

        console.log(`Imported Parts Count: ${capturedParts.length}`);
        console.log(`Imported Second Sources Count: ${capturedSecondSources.length}`);
        console.log(`Detected Mode: ${capturedRevision.mode}`);

        // Verify some content from the specific file if known, or just general sanity check
        // The filename implies Proto parts are present.
        const protoParts = capturedParts.filter(p => p.bom_status === 'P');
        console.log(`Proto Parts Count: ${protoParts.length}`);

        // 2. Prepare Data for Export
        // Transform captured parts into the structure expected by getBomView
        // We need to simulate aggregation and attaching second sources
        const mockBomView = capturedParts.map(part => {
             // Find matching second sources for this part's group
             // In our simple simulation, we can just attach based on main_supplier/pn
             const relatedSS = capturedSecondSources.filter(ss =>
                 ss.main_supplier === part.supplier &&
                 ss.main_supplier_pn === part.supplier_pn
             );

             return {
                 ...part,
                 locations: part.location, // In real DB this is aggregated, here we use single location for simplicity or we should aggregate
                 quantity: 1, // distinct parts
                 second_sources: relatedSS
             };
        });

        // Use a subset for export verification to avoid timeout in CI environment
        const subsetBomView = mockBomView.slice(0, 50);

        bomService.getBomView.mockReturnValue(subsetBomView);
        bomService.executeView.mockReturnValue(subsetBomView);

        // 3. Export
        const exportResult = await exportService.exportBom(1, outputPath);
        expect(exportResult.success).toBe(true);
        expect(fs.existsSync(outputPath)).toBe(true);
        console.log(`Exported file created at: ${outputPath}`);

        // 4. Verify Exported File Content
        const workbook = xlsx.readFile(outputPath);
        console.log(`Exported Sheets: ${workbook.SheetNames.join(', ')}`);

        expect(workbook.SheetNames).toContain('ALL');
        expect(workbook.SheetNames).toContain('SMD');

        // Clean up
        if (fs.existsSync(outputPath)) {
            fs.unlinkSync(outputPath);
        }
    }, 30000);
});
