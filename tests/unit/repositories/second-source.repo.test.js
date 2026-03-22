import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import fs from 'fs';
import path from 'path';

import DatabaseManager from '../../../src/main/database/connection.js';
import projectRepo from '../../../src/main/database/repositories/project.repo.js';
import bomRevisionRepo from '../../../src/main/database/repositories/bom-revision.repo.js';
import secondSourceRepo from '../../../src/main/database/repositories/second-source.repo.js';

describe('Second Source Repository', () => {
  const testDbPath = path.join(__dirname, 'second_source.test.bomix');
  let bomRevisionId;

  beforeEach(() => {
    DatabaseManager.connect(testDbPath);
    const project = projectRepo.create({ project_code: 'SS_TEST' });
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

  it('should create a second source', () => {
    const ss = secondSourceRepo.create({
      bom_revision_id: bomRevisionId,
      main_supplier: 'MainSupp',
      main_supplier_pn: 'MainPN',
      supplier: 'SecondSupp',
      supplier_pn: 'SecondPN',
      hhpn: '123-456',
      description: 'Resistor'
    });

    expect(ss).toBeDefined();
    expect(ss.supplier).toBe('SecondSupp');
    expect(ss.main_supplier).toBe('MainSupp');
  });

  it('should create many second sources', () => {
    const ssData = [
      {
        bom_revision_id: bomRevisionId,
        main_supplier: 'Main1',
        main_supplier_pn: 'PN1',
        supplier: 'Sub1',
        supplier_pn: 'SubPN1',
        description: 'D1'
      },
      {
        bom_revision_id: bomRevisionId,
        main_supplier: 'Main1',
        main_supplier_pn: 'PN1',
        supplier: 'Sub2',
        supplier_pn: 'SubPN2',
        description: 'D2'
      }
    ];

    secondSourceRepo.createMany(ssData);

    const results = secondSourceRepo.findByBomRevision(bomRevisionId);
    expect(results.length).toBe(2);
  });

  it('should delete second sources by revision', () => {
    secondSourceRepo.create({
      bom_revision_id: bomRevisionId,
      main_supplier: 'A',
      main_supplier_pn: 'B',
      supplier: 'C',
      supplier_pn: 'D',
      description: 'E'
    });

    secondSourceRepo.deleteByBomRevision(bomRevisionId);

    const results = secondSourceRepo.findByBomRevision(bomRevisionId);
    expect(results.length).toBe(0);
  });
});
