
import { describe, it, expect, vi, beforeEach } from 'vitest';
import ExcelJS from 'exceljs';
import * as templateEngine from '../../../src/main/services/excel-export/template-engine';

// Mock electron
vi.mock('electron', () => ({
    app: {
        isPackaged: false,
        getPath: vi.fn(),
    }
}));

// Mock fs and path minimally if needed, but integration with real fs/exceljs is better for this test
// as we want to test the actual merge migration logic which depends on ExcelJS behavior.

describe('Template Engine Merge Logic', () => {
    
    it('should preserve footer merges after inserting rows', async () => {
        // 1. Create a dummy template in memory
        const templateWorkbook = new ExcelJS.Workbook();
        const sheet = templateWorkbook.addWorksheet('Template');

        // Setup Header (Rows 1-2)
        sheet.getCell('A1').value = 'Header';
        sheet.getCell('A2').value = '{{M_ITEM}}'; // Main Template Row
        sheet.getCell('B2').value = '{{M_VAL}}';
        
        // Setup Footer (Row 5)
        // Gap: Row 3, 4 are empty
        sheet.getCell('A5').value = 'Footer Label';
        sheet.getCell('B5').value = '{{FOOTER_TAG}}';
        
        // Merge B5:C5
        sheet.mergeCells('B5:C5');
        
        // 2. Prepare Data
        const mockData = {
            meta: { FOOTER_TAG: 'FooterValue' },
            items: [
                { M_ITEM: 'Item 1', M_VAL: 100 },
                { M_ITEM: 'Item 2', M_VAL: 200 }, // Insert +1 row
                { M_ITEM: 'Item 3', M_VAL: 300 }  // Insert +1 row (Total +2 rows)
            ]
        };

        // 3. Execute logic
        const targetWorkbook = new ExcelJS.Workbook();
        
        // Since appendSheetFromTemplate reads from file if not passed a workbook, 
        // but here we pass workbook directly:
        // wrapper function needed? No, appendSheetFromTemplate takes (target, source, ...)
        
        templateEngine.appendSheetFromTemplate(targetWorkbook, templateWorkbook, 'Template', 'Exported', mockData);
        
        const targetSheet = targetWorkbook.getWorksheet('Exported');
        
        // 4. Verify
        // Original Footer Row 5 should move to Row 7 (5 + 2 inserted rows)
        // Merged cells B5:C5 should move to B7:C7
        
        const footerValueCell = targetSheet.getCell('B7');
        
        expect(footerValueCell.isMerged).toBe(true);
        expect(footerValueCell.master.address).toBe('B7');
        expect(footerValueCell.value).toBe('FooterValue'); // substitution happened
        
        // Verify C7 is part of merge
        const c7 = targetSheet.getCell('C7');
        expect(c7.type).toBe(ExcelJS.ValueType.Merge); 
        expect(c7.master.address).toBe('B7');

        // Verify old merge is gone (B5:C5)
        // Note: B5 might now be a data row (Item 3 is at Row 4? No.)
        // Row 2: Item 1
        // Row 3: Item 2
        // Row 4: Item 3
        // Row 5: Empty (gap)
        // Row 6: Empty (gap)
        // Row 7: Footer
        
        // B5 is empty gap row 5
        const b5 = targetSheet.getCell('B5');
        expect(b5.isMerged).toBe(false); 
        
        // Check exact merges
        // We might have other merges if header had them? No header merge here.
        // Expect only B7:C7
        const merges = targetSheet.model.merges;
        expect(merges).toContain('B7:C7');
        expect(merges).not.toContain('B5:C5');
    });

    it('should preserve footer merges in the actual ebom.xlsx template', async () => {
        // 1. Load the real template
        const templatePath = 'resources/templates/ebom.xlsx'; // relative to project root
        // If file not found in test env, skip or mock
        // But since we are running from project root, it should be fine if it exists.
        const templateWorkbook = new ExcelJS.Workbook();
        try {
            await templateWorkbook.xlsx.readFile(templatePath);
        } catch (e) {
            console.warn('Skipping ebom.xlsx test: Template file not found at', templatePath);
            return;
        }

        // 2. Prepare Mock Data (Standard BOM structure)
        const mockData = {
            meta: {
                PROJECT_CODE: 'TEST-PROJ',
                DESCRIPTION: 'Test Description',
                SCHEMATIC_VERSION: '1.0',
                PCB_VERSION: 'REV A',
                PCA_PN: '123-456',
                BOM_VERSION: '0.1',
                DATE: '2023-10-27',
                PHASE: 'EVT'
            },
            items: [
                { M_ITEM: '1', M_HHPN: 'PN-001', M_DESC: 'Resistor 10k', M_QTY: 1 },
                { M_ITEM: '2', M_HHPN: 'PN-002', M_DESC: 'Capacitor 10uF', M_QTY: 2 },
                { M_ITEM: '3', M_HHPN: 'PN-003', M_DESC: 'LED Red', M_QTY: 5 }
            ]
        };

        // 3. Render
        const targetWorkbook = new ExcelJS.Workbook();
        
        // Let's use 'SMD' sheet if available, or just the first one.
        let sourceSheet = templateWorkbook.getWorksheet('SMD');
        if (!sourceSheet) sourceSheet = templateWorkbook.worksheets[0];

        if (!sourceSheet) {
             console.warn('Skipping ebom.xlsx test: No sheets found');
             return;
        }
        
        const targetSheetName = 'TestOutput';
        
        templateEngine.appendSheetFromTemplate(targetWorkbook, templateWorkbook, sourceSheet.name, targetSheetName, mockData);

        const targetSheet = targetWorkbook.getWorksheet(targetSheetName);
        
        // 4. Verify Footer Merge (Col H & I)
        
        let foundFooterMerge = false;
        
        // Find the row with TOTAL_QTY/formula
        let formulaRowIndex = -1;
        targetSheet.eachRow((row, r) => {
             const i = row.getCell('I');
             const h = row.getCell('H');
             // The formula logic sets value to { formula: ... }
             // We check if either H or I has the formula or the tag (if logic didn't run)
             // But logic ran.
             if ((i.value && JSON.stringify(i.value).includes('SUM')) || (h.value && JSON.stringify(h.value).includes('SUM'))) {
                 formulaRowIndex = r;
                 
                 // Verify Merge
                 const cellH = row.getCell('H');
                 const cellI = row.getCell('I');
                 
                 if (cellH.isMerged && cellI.isMerged && cellH.master.address === cellI.master.address) {
                     foundFooterMerge = true;
                 }
             }
        });

        expect(formulaRowIndex).toBeGreaterThan(0);
        expect(foundFooterMerge).toBe(true, `Expected H${formulaRowIndex}:I${formulaRowIndex} to be merged`);
    });
});
