import { describe, it, expect } from 'vitest';
import {
    getViewDefinition,
    getViewFilters,
    getExportDefinition,
    VIEW_IDS,
    EXPORT_IDS
} from '../../../src/main/services/bom-factory.service.js';

describe('BOM Factory Service', () => {
    describe('getViewDefinition', () => {
        it('should return correct definition for ALL view (filters array format)', () => {
            const view = getViewDefinition(VIEW_IDS.ALL);
            expect(view.id).toBe(VIEW_IDS.ALL);
            expect(view.filters).toBeDefined();
            expect(view.filters).toBeInstanceOf(Array);
            // ALL view: 僅有 statusLogic ACTIVE
            expect(view.filters).toEqual([
                { field: 'bom_status', operator: 'statusLogic', value: 'ACTIVE' }
            ]);
        });

        it('should return correct definition for SMD view (filters array format)', () => {
            const view = getViewDefinition(VIEW_IDS.SMD);
            expect(view.id).toBe(VIEW_IDS.SMD);
            expect(view.filters).toBeDefined();
            expect(view.filters).toContainEqual({ field: 'type', operator: 'in', value: ['SMD'] });
            expect(view.filters).toContainEqual({ field: 'bom_status', operator: 'statusLogic', value: 'ACTIVE' });
        });

        it('should return correct definition for NI view', () => {
            const view = getViewDefinition(VIEW_IDS.NI);
            expect(view.id).toBe(VIEW_IDS.NI);
            expect(view.filters).toContainEqual({ field: 'bom_status', operator: 'statusLogic', value: 'INACTIVE' });
        });

        it('should return correct definition for PROTO view', () => {
            const view = getViewDefinition(VIEW_IDS.PROTO);
            expect(view.id).toBe(VIEW_IDS.PROTO);
            expect(view.filters).toContainEqual({ field: 'bom_status', operator: 'statusLogic', value: 'SPECIFIC' });
            expect(view.filters).toContainEqual({ field: 'bom_status', operator: 'in', value: ['P'] });
        });

        it('should return correct definition for CCL view', () => {
            const view = getViewDefinition(VIEW_IDS.CCL);
            expect(view.id).toBe(VIEW_IDS.CCL);
            expect(view.filters).toContainEqual({ field: 'ccl', operator: 'eq', value: 'Y' });
            expect(view.filters).toContainEqual({ field: 'bom_status', operator: 'statusLogic', value: 'ACTIVE' });
        });

        it('should throw error for unknown view ID', () => {
            expect(() => getViewDefinition('unknown_view')).toThrow('Unknown View ID: unknown_view');
        });
    });

    describe('getViewFilters', () => {
        it('should return filters array for ALL view', () => {
            const filters = getViewFilters(VIEW_IDS.ALL);
            expect(filters).toBeInstanceOf(Array);
            expect(filters.length).toBeGreaterThan(0);
        });

        it('should return filters array for SMD view containing type filter', () => {
            const filters = getViewFilters(VIEW_IDS.SMD);
            expect(filters).toContainEqual({ field: 'type', operator: 'in', value: ['SMD'] });
        });

        it('should throw error for unknown view ID', () => {
            expect(() => getViewFilters('unknown_view')).toThrow('Unknown View ID: unknown_view');
        });
    });

    describe('getExportDefinition', () => {
        it('should return correct definition for EBOM export', () => {
            const def = getExportDefinition(EXPORT_IDS.EBOM);
            expect(def.id).toBe(EXPORT_IDS.EBOM);
            expect(def.sheets).toHaveLength(8);
            expect(def.sheets[0].targetSheetName).toBe('ALL');
            expect(def.sheets[0].templateFile).toBe('ebom.xlsx');
            expect(def.sheets[1].targetSheetName).toBe('SMD');
            expect(def.sheets[1].templateFile).toBe('ebom.xlsx');
        });

        it('should throw error for unknown export ID', () => {
            expect(() => getExportDefinition('unknown_export')).toThrow('Unknown Export ID: unknown_export');
        });
    });
});
