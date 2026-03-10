import { describe, it, expect, vi, beforeEach } from 'vitest';
import bomService, { queryBomData } from '../../../src/main/services/bom.service.js';
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

// ========================================
// queryBomData — 核心通用查詢函式測試
// ========================================
describe('BOM Service - queryBomData', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('應過濾 SMD 零件 (NPI 模式：允許 I, P)', () => {
    bomRevisionRepo.findById.mockReturnValue({ id: 1, mode: 'NPI' });
    partsRepo.findByBomRevisions.mockReturnValue([
      { supplier: 'S1', supplier_pn: 'P1', type: 'SMD', bom_status: 'I', location: 'L1', bom_revision_id: 1 },
      { supplier: 'S2', supplier_pn: 'P2', type: 'SMD', bom_status: 'P', location: 'L2', bom_revision_id: 1 },
      { supplier: 'S3', supplier_pn: 'P3', type: 'SMD', bom_status: 'X', location: 'L3', bom_revision_id: 1 }, // 應被過濾
      { supplier: 'S4', supplier_pn: 'P4', type: 'PTH', bom_status: 'I', location: 'L4', bom_revision_id: 1 }  // 應被過濾
    ]);
    secondSourceRepo.findByBomRevision.mockReturnValue([]);

    const filters = [
      { field: 'type', operator: 'in', value: ['SMD'] },
      { field: 'bom_status', operator: 'statusLogic', value: 'ACTIVE' }
    ];
    const result = queryBomData([1], filters, {});

    expect(result).toHaveLength(2);
    expect(result.map(r => r.supplier_pn).sort()).toEqual(['P1', 'P2']);
  });

  it('應過濾 SMD 零件 (MP 模式：允許 I, M)', () => {
    bomRevisionRepo.findById.mockReturnValue({ id: 1, mode: 'MP' });
    partsRepo.findByBomRevisions.mockReturnValue([
      { supplier: 'S1', supplier_pn: 'P1', type: 'SMD', bom_status: 'I', location: 'L1', bom_revision_id: 1 },
      { supplier: 'S2', supplier_pn: 'P2', type: 'SMD', bom_status: 'M', location: 'L2', bom_revision_id: 1 },
      { supplier: 'S3', supplier_pn: 'P3', type: 'SMD', bom_status: 'P', location: 'L3', bom_revision_id: 1 }, // 應被過濾
      { supplier: 'S4', supplier_pn: 'P4', type: 'PTH', bom_status: 'I', location: 'L4', bom_revision_id: 1 }  // 應被過濾
    ]);
    secondSourceRepo.findByBomRevision.mockReturnValue([]);

    const filters = [
      { field: 'type', operator: 'in', value: ['SMD'] },
      { field: 'bom_status', operator: 'statusLogic', value: 'ACTIVE' }
    ];
    const result = queryBomData([1], filters, {});

    expect(result).toHaveLength(2);
    expect(result.map(r => r.supplier_pn).sort()).toEqual(['P1', 'P2']);
  });

  it('應正確套用 eq operator 過濾 CCL=Y', () => {
    bomRevisionRepo.findById.mockReturnValue({ id: 1, mode: 'NPI' });
    partsRepo.findByBomRevisions.mockReturnValue([
      { supplier: 'S1', supplier_pn: 'P1', bom_status: 'I', ccl: 'Y', location: 'L1', bom_revision_id: 1 },
      { supplier: 'S2', supplier_pn: 'P2', bom_status: 'I', ccl: 'N', location: 'L2', bom_revision_id: 1 }
    ]);
    secondSourceRepo.findByBomRevision.mockReturnValue([]);

    const filters = [{ field: 'ccl', operator: 'eq', value: 'Y' }];
    const result = queryBomData([1], filters, {});

    expect(result).toHaveLength(1);
    expect(result[0].supplier_pn).toBe('P1');
  });

  it('應正確套用 neq operator', () => {
    bomRevisionRepo.findById.mockReturnValue({ id: 1, mode: 'NPI' });
    partsRepo.findByBomRevisions.mockReturnValue([
      { supplier: 'S1', supplier_pn: 'P1', bom_status: 'I', ccl: 'Y', location: 'L1', bom_revision_id: 1 },
      { supplier: 'S2', supplier_pn: 'P2', bom_status: 'I', ccl: 'N', location: 'L2', bom_revision_id: 1 }
    ]);
    secondSourceRepo.findByBomRevision.mockReturnValue([]);

    const filters = [{ field: 'ccl', operator: 'neq', value: 'Y' }];
    const result = queryBomData([1], filters, {});

    expect(result).toHaveLength(1);
    expect(result[0].supplier_pn).toBe('P2');
  });

  it('應正確套用 notIn operator', () => {
    bomRevisionRepo.findById.mockReturnValue({ id: 1, mode: 'NPI' });
    partsRepo.findByBomRevisions.mockReturnValue([
      { supplier: 'S1', supplier_pn: 'P1', type: 'SMD', bom_status: 'I', location: 'L1', bom_revision_id: 1 },
      { supplier: 'S2', supplier_pn: 'P2', type: 'PTH', bom_status: 'I', location: 'L2', bom_revision_id: 1 }
    ]);
    secondSourceRepo.findByBomRevision.mockReturnValue([]);

    const filters = [{ field: 'type', operator: 'notIn', value: ['PTH'] }];
    const result = queryBomData([1], filters, {});

    expect(result).toHaveLength(1);
    expect(result[0].supplier_pn).toBe('P1');
  });

  it('空 filters 應回傳所有零件', () => {
    bomRevisionRepo.findById.mockReturnValue({ id: 1, mode: 'NPI' });
    partsRepo.findByBomRevisions.mockReturnValue([
      { supplier: 'S1', supplier_pn: 'P1', bom_status: 'I', location: 'L1', bom_revision_id: 1 },
      { supplier: 'S2', supplier_pn: 'P2', bom_status: 'X', location: 'L2', bom_revision_id: 1 }
    ]);
    secondSourceRepo.findByBomRevision.mockReturnValue([]);

    const result = queryBomData([1], [], {});

    expect(result).toHaveLength(2);
  });

  it('statusLogic INACTIVE (NPI 模式：允許 X, M)', () => {
    bomRevisionRepo.findById.mockReturnValue({ id: 1, mode: 'NPI' });
    partsRepo.findByBomRevisions.mockReturnValue([
      { supplier: 'S1', supplier_pn: 'P1', bom_status: 'X', location: 'L1', bom_revision_id: 1 }, // 保留
      { supplier: 'S2', supplier_pn: 'P2', bom_status: 'M', location: 'L2', bom_revision_id: 1 }, // 保留
      { supplier: 'S3', supplier_pn: 'P3', bom_status: 'I', location: 'L3', bom_revision_id: 1 }, // 過濾
    ]);
    secondSourceRepo.findByBomRevision.mockReturnValue([]);

    const filters = [{ field: 'bom_status', operator: 'statusLogic', value: 'INACTIVE' }];
    const result = queryBomData([1], filters, {});

    expect(result).toHaveLength(2);
    expect(result.map(r => r.bom_status).sort()).toEqual(['M', 'X']);
  });

  it('statusLogic SPECIFIC 搭配 in operator', () => {
    bomRevisionRepo.findById.mockReturnValue({ id: 1, mode: 'NPI' });
    partsRepo.findByBomRevisions.mockReturnValue([
      { supplier: 'S1', supplier_pn: 'P1', bom_status: 'P', location: 'L1', bom_revision_id: 1 }, // 保留
      { supplier: 'S2', supplier_pn: 'P2', bom_status: 'I', location: 'L2', bom_revision_id: 1 }, // 過濾（in 只允許 P）
    ]);
    secondSourceRepo.findByBomRevision.mockReturnValue([]);

    const filters = [
      { field: 'bom_status', operator: 'statusLogic', value: 'SPECIFIC' },
      { field: 'bom_status', operator: 'in', value: ['P'] }
    ];
    const result = queryBomData([1], filters, {});

    expect(result).toHaveLength(1);
    expect(result[0].bom_status).toBe('P');
  });

  it('多 filters AND 組合：ACTIVE + CCL=Y', () => {
    bomRevisionRepo.findById.mockReturnValue({ id: 1, mode: 'NPI' });
    partsRepo.findByBomRevisions.mockReturnValue([
      { supplier: 'S1', supplier_pn: 'P1', bom_status: 'I', ccl: 'Y', location: 'L1', bom_revision_id: 1 }, // 保留
      { supplier: 'S2', supplier_pn: 'P2', bom_status: 'I', ccl: 'N', location: 'L2', bom_revision_id: 1 }, // ccl 不符
      { supplier: 'S3', supplier_pn: 'P3', bom_status: 'X', ccl: 'Y', location: 'L3', bom_revision_id: 1 }, // status 不符
    ]);
    secondSourceRepo.findByBomRevision.mockReturnValue([]);

    const filters = [
      { field: 'bom_status', operator: 'statusLogic', value: 'ACTIVE' },
      { field: 'ccl', operator: 'eq', value: 'Y' }
    ];
    const result = queryBomData([1], filters, {});

    expect(result).toHaveLength(1);
    expect(result[0].supplier_pn).toBe('P1');
  });

  it('應正確聚合零件並關聯 Second Sources', () => {
    bomRevisionRepo.findById.mockReturnValue({ id: 1, mode: 'NPI' });
    partsRepo.findByBomRevisions.mockReturnValue([
      { supplier: 'S1', supplier_pn: 'P1', bom_status: 'I', location: 'L1', item: 10, bom_revision_id: 1 },
      { supplier: 'S1', supplier_pn: 'P1', bom_status: 'I', location: 'L2', item: 10, bom_revision_id: 1 },
    ]);
    secondSourceRepo.findByBomRevision.mockReturnValue([
      { main_supplier: 'S1', main_supplier_pn: 'P1', supplier: 'SS1' }
    ]);

    const result = queryBomData([1], [], {});

    expect(result).toHaveLength(1);
    expect(result[0].quantity).toBe(2);
    expect(result[0].locations).toBe('L1,L2');
    expect(result[0].second_sources).toHaveLength(1);
    expect(result[0].second_sources[0].supplier).toBe('SS1');
  });

  it('空 bomIds 應回傳空陣列', () => {
    const result = queryBomData([], [], {});
    expect(result).toEqual([]);
  });
});

// ========================================
// executeView — 向下相容薄包裝層測試（仍應正常運作）
// ========================================
describe('BOM Service - executeView (deprecated wrapper)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('應透過舊版 filter 物件格式正確過濾（向下相容）', () => {
    bomRevisionRepo.findById.mockReturnValue({ id: 1, mode: 'NPI' });
    partsRepo.findByBomRevisions.mockReturnValue([
      { supplier: 'S1', supplier_pn: 'P1', type: 'SMD', bom_status: 'I', location: 'L1', bom_revision_id: 1 },
      { supplier: 'S2', supplier_pn: 'P2', type: 'PTH', bom_status: 'I', location: 'L2', bom_revision_id: 1 }
    ]);
    secondSourceRepo.findByBomRevision.mockReturnValue([]);

    // 使用舊版 filter 物件格式
    const viewDef = { filter: { types: ['SMD'], statusLogic: 'ACTIVE' } };
    const result = bomService.executeView(1, viewDef);

    expect(result).toHaveLength(1);
    expect(result[0].supplier_pn).toBe('P1');
  });

  it('應透過新版 filters 陣列格式正確過濾', () => {
    bomRevisionRepo.findById.mockReturnValue({ id: 1, mode: 'NPI' });
    partsRepo.findByBomRevisions.mockReturnValue([
      { supplier: 'S1', supplier_pn: 'P1', type: 'SMD', bom_status: 'I', location: 'L1', bom_revision_id: 1 },
      { supplier: 'S2', supplier_pn: 'P2', type: 'PTH', bom_status: 'I', location: 'L2', bom_revision_id: 1 }
    ]);
    secondSourceRepo.findByBomRevision.mockReturnValue([]);

    // 使用新版 filters 陣列格式
    const viewDef = { filters: [{ field: 'type', operator: 'in', value: ['SMD'] }] };
    const result = bomService.executeView(1, viewDef);

    expect(result).toHaveLength(1);
    expect(result[0].supplier_pn).toBe('P1');
  });
});

// ========================================
// getBomView — 原始聚合視圖測試（不變）
// ========================================
describe('BOM Service - getBomView', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

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

// ========================================
// updateMainItem — 不變
// ========================================
describe('BOM Service - updateMainItem', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should update all parts in the group', () => {
    const bomRevisionId = 1;
    const key = { supplier: 'S1', supplier_pn: 'PN1', type: 'SMD' };
    const updates = { description: 'New Desc' };
    const mockParts = [{ id: 101 }, { id: 102 }];

    partsRepo.findByGroup.mockReturnValue(mockParts);

    dbManager.getDb.mockReturnValue({
      transaction: (cb) => {
           return () => cb();
      }
    });

    bomService.updateMainItem(bomRevisionId, key, updates);

    expect(partsRepo.findByGroup).toHaveBeenCalledWith(bomRevisionId, 'S1', 'PN1');
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

// ========================================
// deleteMainItem — 不變
// ========================================
describe('BOM Service - deleteMainItem', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should delete parts and related second sources', () => {
    const bomRevisionId = 1;
    const key = { supplier: 'S1', supplier_pn: 'PN1', type: 'SMD' };
    const mockParts = [{ id: 101 }];
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

// ========================================
// Second Source operations — 不變
// ========================================
describe('BOM Service - Second Source operations', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

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
