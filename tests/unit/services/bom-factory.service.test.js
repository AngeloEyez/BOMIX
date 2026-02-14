import { describe, it, expect } from 'vitest';
import {
    getViewDefinition,
    getExportDefinition,
    VIEW_IDS,
    EXPORT_IDS
} from '../../../src/main/services/bom-factory.service.js';

describe('BOM Factory Service', () => {
    describe('getViewDefinition', () => {
        it('should return correct definition for ALL view', () => {
            const view = getViewDefinition(VIEW_IDS.ALL);
            expect(view).toEqual({
                id: VIEW_IDS.ALL,
                filter: { statusLogic: 'ACTIVE' }
            });
        });

        it('should return correct definition for SMD view', () => {
            const view = getViewDefinition(VIEW_IDS.SMD);
            expect(view).toEqual({
                id: VIEW_IDS.SMD,
                filter: { types: ['SMD'], statusLogic: 'ACTIVE' }
            });
        });

        it('should return correct definition for NI view', () => {
            const view = getViewDefinition(VIEW_IDS.NI);
            expect(view).toEqual({
                id: VIEW_IDS.NI,
                filter: { statusLogic: 'INACTIVE' }
            });
        });

        it('should return correct definition for PROTO view', () => {
            const view = getViewDefinition(VIEW_IDS.PROTO);
            expect(view).toEqual({
                id: VIEW_IDS.PROTO,
                filter: { bom_statuses: ['P'], statusLogic: 'SPECIFIC' }
            });
        });

        it('should throw error for unknown view ID', () => {
            expect(() => getViewDefinition('unknown_view')).toThrow('Unknown View ID: unknown_view');
        });
    });

    describe('getExportDefinition', () => {
        it('should return correct definition for EBOM export', () => {
            const def = getExportDefinition(EXPORT_IDS.EBOM);
            expect(def.id).toBe(EXPORT_IDS.EBOM);
            expect(def.templateFile).toBe('ebom.xlsx');
            expect(def.sheets).toHaveLength(8);
            expect(def.sheets[0].targetSheetName).toBe('ALL');
            expect(def.sheets[1].targetSheetName).toBe('SMD');
        });

        it('should throw error for unknown export ID', () => {
            expect(() => getExportDefinition('unknown_export')).toThrow('Unknown Export ID: unknown_export');
        });
    });
});
