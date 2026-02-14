import { describe, it, expect, vi, beforeEach } from 'vitest';
import bomService from '../../../src/main/services/bom.service.js';
import partsRepo from '../../../src/main/database/repositories/parts.repo.js';
import secondSourceRepo from '../../../src/main/database/repositories/second-source.repo.js';
import bomRevisionRepo from '../../../src/main/database/repositories/bom-revision.repo.js';
import dbManager from '../../../src/main/database/connection.js';

vi.mock('../../../src/main/database/repositories/parts.repo.js');
vi.mock('../../../src/main/database/repositories/second-source.repo.js');
vi.mock('../../../src/main/database/repositories/bom-revision.repo.js');
vi.mock('../../../src/main/database/connection.js', () => ({
  default: {
    getDb: vi.fn()
  }
}));

describe('BOM Service', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('executeView', () => {
    it('should filter SMD parts in NPI mode (Status I, P)', () => {
      const bomRevisionId = 1;
      const mockRevision = { id: 1, mode: 'NPI' };
      const viewDef = { filter: { types: ['SMD'], statusLogic: 'ACTIVE' } };

      const parts = [
        { supplier: 'S1', supplier_pn: 'P1', type: 'SMD', bom_status: 'I', location: 'L1' },
        { supplier: 'S2', supplier_pn: 'P2', type: 'SMD', bom_status: 'P', location: 'L2' },
        { supplier: 'S3', supplier_pn: 'P3', type: 'SMD', bom_status: 'X', location: 'L3' }, // Should be filtered
        { supplier: 'S4', supplier_pn: 'P4', type: 'PTH', bom_status: 'I', location: 'L4' }  // Should be filtered
      ];

      bomRevisionRepo.findById.mockReturnValue(mockRevision);
      partsRepo.findByBomRevision.mockReturnValue(parts);
      secondSourceRepo.findByBomRevision.mockReturnValue([]);

      const result = bomService.executeView(bomRevisionId, viewDef);

      expect(result).toHaveLength(2);
      expect(result.map(r => r.supplier_pn).sort()).toEqual(['P1', 'P2']);
    });

    it('should filter SMD parts in MP mode (Status I, M)', () => {
      const bomRevisionId = 1;
      const mockRevision = { id: 1, mode: 'MP' };
      const viewDef = { filter: { types: ['SMD'], statusLogic: 'ACTIVE' } };

      const parts = [
        { supplier: 'S1', supplier_pn: 'P1', type: 'SMD', bom_status: 'I', location: 'L1' },
        { supplier: 'S2', supplier_pn: 'P2', type: 'SMD', bom_status: 'M', location: 'L2' },
        { supplier: 'S3', supplier_pn: 'P3', type: 'SMD', bom_status: 'P', location: 'L3' }, // Should be filtered (Proto part in MP)
        { supplier: 'S4', supplier_pn: 'P4', type: 'PTH', bom_status: 'I', location: 'L4' }  // Should be filtered
      ];

      bomRevisionRepo.findById.mockReturnValue(mockRevision);
      partsRepo.findByBomRevision.mockReturnValue(parts);
      secondSourceRepo.findByBomRevision.mockReturnValue([]);

      const result = bomService.executeView(bomRevisionId, viewDef);

      expect(result).toHaveLength(2);
      expect(result.map(r => r.supplier_pn).sort()).toEqual(['P1', 'P2']);
    });

    it('should aggregate parts and attach second sources', () => {
      const bomRevisionId = 1;
      const mockRevision = { id: 1, mode: 'NPI' };
      const viewDef = { filter: { statusLogic: 'ACTIVE' } };

      const parts = [
        { supplier: 'S1', supplier_pn: 'P1', type: 'SMD', bom_status: 'I', location: 'L1', item: 10 },
        { supplier: 'S1', supplier_pn: 'P1', type: 'SMD', bom_status: 'I', location: 'L2', item: 10 },
      ];

      const secondSources = [
        { main_supplier: 'S1', main_supplier_pn: 'P1', supplier: 'SS1' }
      ];

      bomRevisionRepo.findById.mockReturnValue(mockRevision);
      partsRepo.findByBomRevision.mockReturnValue(parts);
      secondSourceRepo.findByBomRevision.mockReturnValue(secondSources);

      const result = bomService.executeView(bomRevisionId, viewDef);

      expect(result).toHaveLength(1);
      expect(result[0].quantity).toBe(2);
      expect(result[0].locations).toBe('L1,L2'); // Assuming simple sort logic
      expect(result[0].second_sources).toHaveLength(1);
      expect(result[0].second_sources[0].supplier).toBe('SS1');
    });

    it('should correctly filter for NI view (Inactive)', () => {
      const bomRevisionId = 1;
      const mockRevision = { id: 1, mode: 'NPI' };
      const viewDef = { filter: { statusLogic: 'INACTIVE' } }; // NPI Inactive: X, M

      const parts = [
        { supplier: 'S1', supplier_pn: 'P1', type: 'SMD', bom_status: 'X', location: 'L1' }, // Keep
        { supplier: 'S2', supplier_pn: 'P2', type: 'SMD', bom_status: 'M', location: 'L2' }, // Keep
        { supplier: 'S3', supplier_pn: 'P3', type: 'SMD', bom_status: 'I', location: 'L3' }, // Filter
        { supplier: 'S4', supplier_pn: 'P4', type: 'SMD', bom_status: 'P', location: 'L4' }, // Filter
      ];

      bomRevisionRepo.findById.mockReturnValue(mockRevision);
      partsRepo.findByBomRevision.mockReturnValue(parts);
      secondSourceRepo.findByBomRevision.mockReturnValue([]);

      const result = bomService.executeView(bomRevisionId, viewDef);

      expect(result).toHaveLength(2);
      expect(result.map(r => r.bom_status).sort()).toEqual(['M', 'X']);
    });
  });

  describe('getBomView', () => {
    it('should return aggregated BOM view with second sources', () => {
      const bomRevisionId = 1;
      const mockRevision = { id: 1, version: '0.1' };
      const mockMainItems = [
        { supplier: 'S1', supplier_pn: 'PN1', type: 'SMD', quantity: 2, item: 1 }
      ];
      const mockSecondSources = [
        { main_supplier: 'S1', main_supplier_pn: 'PN1', supplier: 'SS1', supplier_pn: 'SSPN1' }
      ];

      bomRevisionRepo.findById.mockReturnValue(mockRevision);
      partsRepo.getAggregatedBom.mockReturnValue(mockMainItems);
      secondSourceRepo.findByBomRevision.mockReturnValue(mockSecondSources);

      const result = bomService.getBomView(bomRevisionId);

      expect(bomRevisionRepo.findById).toHaveBeenCalledWith(bomRevisionId);
      expect(partsRepo.getAggregatedBom).toHaveBeenCalledWith(bomRevisionId);
      expect(secondSourceRepo.findByBomRevision).toHaveBeenCalledWith(bomRevisionId);
      expect(result).toHaveLength(1);
      expect(result[0].second_sources).toHaveLength(1);
      expect(result[0].second_sources[0].supplier).toBe('SS1');
    });

    it('should throw error if BOM revision not found', () => {
      bomRevisionRepo.findById.mockReturnValue(null);
      expect(() => bomService.getBomView(999)).toThrow('找不到 ID 為 999 的 BOM 版本');
    });
  });

  describe('updateMainItem', () => {
    it('should update all parts in the group', () => {
      const bomRevisionId = 1;
      const key = { supplier: 'S1', supplier_pn: 'PN1', type: 'SMD' };
      const updates = { description: 'New Desc' };
      const mockParts = [{ id: 101 }, { id: 102 }];

      partsRepo.findByGroup.mockReturnValue(mockParts);

      const mockTransaction = vi.fn((cb) => cb); // Execute callback immediately
      dbManager.getDb.mockReturnValue({
        transaction: (cb) => {
             return () => cb();
        }
      });

      bomService.updateMainItem(bomRevisionId, key, updates);

      expect(partsRepo.findByGroup).toHaveBeenCalledWith(bomRevisionId, 'S1', 'PN1', 'SMD');
      expect(partsRepo.update).toHaveBeenCalledTimes(2);
      expect(partsRepo.update).toHaveBeenCalledWith(101, updates);
      expect(partsRepo.update).toHaveBeenCalledWith(102, updates);
    });

    it('should throw error if group not found', () => {
      partsRepo.findByGroup.mockReturnValue([]);
      expect(() => bomService.updateMainItem(1, { supplier: 'S', supplier_pn: 'P' }, {}))
        .toThrow('找不到指定的零件群組');
    });
  });

  describe('deleteMainItem', () => {
      it('should delete parts and related second sources', () => {
          const bomRevisionId = 1;
          const key = { supplier: 'S1', supplier_pn: 'PN1', type: 'SMD' };
          const mockParts = [{ id: 101 }];
          // Mock finding second sources
          const mockSS = [{ id: 201, main_supplier: 'S1', main_supplier_pn: 'PN1' }];

          partsRepo.findByGroup.mockReturnValue(mockParts);
          secondSourceRepo.findByBomRevision.mockReturnValue(mockSS);

          dbManager.getDb.mockReturnValue({
              transaction: (cb) => {
                  return () => cb();
              }
          });

          bomService.deleteMainItem(bomRevisionId, key);

          expect(partsRepo.delete).toHaveBeenCalledWith(101);
          expect(secondSourceRepo.delete).toHaveBeenCalledWith(201);
      });
  });

  describe('Second Source operations', () => {
      it('should add second source', () => {
          const data = { supplier: 'SS2' };
          secondSourceRepo.create.mockReturnValue(data);
          const result = bomService.addSecondSource(data);
          expect(result).toEqual(data);
          expect(secondSourceRepo.create).toHaveBeenCalledWith(data);
      });

      it('should update second source', () => {
          const id = 1;
          const data = { description: 'Updated' };
          secondSourceRepo.update.mockReturnValue({ id, ...data });

          const result = bomService.updateSecondSource(id, data);
          expect(result.description).toBe('Updated');
      });

      it('should throw error when updating non-existent second source', () => {
          secondSourceRepo.update.mockReturnValue(null);
          expect(() => bomService.updateSecondSource(999, {})).toThrow();
      });

      it('should delete second source', () => {
          secondSourceRepo.delete.mockReturnValue(true);
          const result = bomService.deleteSecondSource(1);
          expect(result.success).toBe(true);
      });
  });
});
