import { describe, it, expect } from 'vitest';
import { parsePhaseName, getPhaseBaseIndex, sortBoms, DEFAULT_PHASE_ORDER } from '../src/main/utils/phase-sorter.js';

describe('phase-sorter', () => {
    describe('parsePhaseName', () => {
        it('should parse phase without number correctly', () => {
            expect(parsePhaseName('DB')).toEqual({ base: 'DB', num: -1 });
            expect(parsePhaseName('EVT')).toEqual({ base: 'EVT', num: -1 });
            expect(parsePhaseName('si')).toEqual({ base: 'SI', num: -1 }); // handles case insensitive
        });

        it('should parse phase with number correctly', () => {
            expect(parsePhaseName('DB0')).toEqual({ base: 'DB', num: 0 });
            expect(parsePhaseName('DB1')).toEqual({ base: 'DB', num: 1 });
            expect(parsePhaseName('SI2')).toEqual({ base: 'SI', num: 2 });
        });

        it('should handle unmatching gracefully', () => {
            expect(parsePhaseName('DB-Test')).toEqual({ base: 'DB-TEST', num: -1 });
        });
    });

    describe('getPhaseBaseIndex', () => {
        const customOrder = ['RFI', 'RFP', 'DB, EVT', 'SI'];
        it('should find index correctly', () => {
            expect(getPhaseBaseIndex('DB', customOrder)).toBe(2);
            expect(getPhaseBaseIndex('EVT', customOrder)).toBe(2);
            expect(getPhaseBaseIndex('SI', customOrder)).toBe(3);
        });

        it('should return Infinity if not found', () => {
            expect(getPhaseBaseIndex('UNKNOWN', customOrder)).toBe(Infinity);
        });
    });

    describe('sortBoms', () => {
        const order = DEFAULT_PHASE_ORDER;

        it('should sort by user requirement', () => {
            const boms = [
                { id: 1, phase_name: 'PV', version: '0.1', suffix: '2' }, // PV-0.1-2
                { id: 2, phase_name: 'SI', version: '0.2', suffix: null }, // SI-0.2
                { id: 3, phase_name: 'PV', version: '0.1', suffix: null }, // PV-0.1
                { id: 4, phase_name: 'DB1', version: '0.1', suffix: null }, // DB1-0.1
                { id: 5, phase_name: 'SI', version: '0.3', suffix: null }, // SI-0.3
                { id: 6, phase_name: 'PV', version: '0.1', suffix: '1' }, // PV-0.1-1
                { id: 7, phase_name: 'DB1', version: '0.2', suffix: null }  // DB1-0.2
            ];

            const sorted = sortBoms(boms, order);

            // Expected order:
            // DB1-0.1
            // DB1-0.2
            // SI-0.2
            // SI-0.3
            // PV-0.1
            // PV-0.1-1
            // PV-0.1-2
            expect(sorted.map(b => b.id)).toEqual([4, 7, 2, 5, 3, 6, 1]);
        });

        it('should sort phases without numbers before ones with numbers', () => {
             const boms = [
                 { id: 1, phase_name: 'DB1', version: '0.1', suffix: null },
                 { id: 2, phase_name: 'DB0', version: '0.1', suffix: null },
                 { id: 3, phase_name: 'DB', version: '0.1', suffix: null }
             ];

             const sorted = sortBoms(boms, order);

             // Expected: DB -> DB0 -> DB1
             expect(sorted.map(b => b.id)).toEqual([3, 2, 1]);
        });
    });
});