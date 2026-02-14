import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import fs from 'fs';
import path from 'path';

import DatabaseManager from '../../../src/main/database/connection.js';
import projectRepo from '../../../src/main/database/repositories/project.repo.js';
import bomRevisionRepo from '../../../src/main/database/repositories/bom-revision.repo.js';
import partsRepo from '../../../src/main/database/repositories/parts.repo.js';

describe('Parts Repository', () => {
  const testDbPath = path.join(__dirname, 'parts.test.bomix');
  let bomRevisionId;

  beforeEach(() => {
    DatabaseManager.connect(testDbPath);
    // 建立一個測試專案與 BOM 版本
    const project = projectRepo.create({ project_code: 'PARTS_TEST' });
    const revision = bomRevisionRepo.create({
      project_id: project.id,
      phase_name: 'EVT',
      version: '0.1'
    });
    bomRevisionId = revision.id;
  });

  afterEach(() => {
    try {
        DatabaseManager.disconnect();
    } catch(e) {}

    if (fs.existsSync(testDbPath)) {
        try {
            fs.unlinkSync(testDbPath);
        } catch(e) {}
    }
  });

  it('should create a part', () => {
    const part = partsRepo.create({
      bom_revision_id: bomRevisionId,
      supplier: 'Murata',
      supplier_pn: 'GRM188R71H104KA93D',
      location: 'C1',
      description: 'CAP CER 0.1UF 50V X7R 0603'
    });

    expect(part).toBeDefined();
    expect(part.bom_revision_id).toBe(bomRevisionId);
    expect(part.supplier).toBe('Murata');
    expect(part.location).toBe('C1');
    expect(part.bom_status).toBe('I'); // default
    expect(part.ccl).toBe('N'); // default
  });

  it('should create many parts', () => {
    const partsData = [
      {
        bom_revision_id: bomRevisionId,
        supplier: 'Yageo',
        supplier_pn: 'RC0603FR-0710KL',
        location: 'R1'
      },
      {
        bom_revision_id: bomRevisionId,
        supplier: 'Yageo',
        supplier_pn: 'RC0603FR-0710KL',
        location: 'R2'
      }
    ];

    partsRepo.createMany(partsData);

    const parts = partsRepo.findByBomRevision(bomRevisionId);
    expect(parts.length).toBe(2);
    expect(parts[0].supplier).toBe('Yageo');
  });

  it('should get aggregated BOM', () => {
    // Create multiple parts with same supplier/pn/type but different locations
    const partsData = [
      {
        bom_revision_id: bomRevisionId,
        supplier: 'TDK',
        supplier_pn: 'C1608X7R1H104K',
        location: 'C10',
        type: 'SMD',
        item: 10
      },
      {
        bom_revision_id: bomRevisionId,
        supplier: 'TDK',
        supplier_pn: 'C1608X7R1H104K',
        location: 'C11',
        type: 'SMD',
        item: 11
      },
      {
        bom_revision_id: bomRevisionId,
        supplier: 'TDK',
        supplier_pn: 'C1608X7R1H104K',
        location: 'C12',
        type: 'SMD',
        item: 12
      }
    ];

    partsRepo.createMany(partsData);

    const aggregated = partsRepo.getAggregatedBom(bomRevisionId);
    expect(aggregated.length).toBe(1);

    const item = aggregated[0];
    expect(item.supplier).toBe('TDK');
    expect(item.quantity).toBe(3);
    // location order is not guaranteed by GROUP_CONCAT unless specified, but usually insertion order
    expect(item.locations).toContain('C10');
    expect(item.locations).toContain('C11');
    expect(item.locations).toContain('C12');
  });

  it('should delete parts by revision', () => {
    partsRepo.create({
      bom_revision_id: bomRevisionId,
      supplier: 'Test',
      supplier_pn: 'PN1',
      location: 'L1'
    });

    partsRepo.deleteByBomRevision(bomRevisionId);

    const parts = partsRepo.findByBomRevision(bomRevisionId);
    expect(parts.length).toBe(0);
  });
});
