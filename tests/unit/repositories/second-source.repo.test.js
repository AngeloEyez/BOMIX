import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import fs from 'fs';
import path from 'path';

const DatabaseManager = require('../../../src/main/database/connection');
const projectRepo = require('../../../src/main/database/repositories/project.repo');
const bomRevisionRepo = require('../../../src/main/database/repositories/bom-revision.repo');
const secondSourceRepo = require('../../../src/main/database/repositories/second-source.repo');

describe('Second Source Repository', () => {
  const testDbPath = path.join(__dirname, 'second-source.test.bomix');
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
      main_supplier: 'Murata',
      main_supplier_pn: 'GRM188',
      supplier: 'Yageo',
      supplier_pn: 'RC0603'
    });

    expect(ss).toBeDefined();
    expect(ss.main_supplier).toBe('Murata');
    expect(ss.supplier).toBe('Yageo');
  });

  it('should create many second sources', () => {
    const data = [
      {
        bom_revision_id: bomRevisionId,
        main_supplier: 'Murata',
        main_supplier_pn: 'GRM188',
        supplier: 'Yageo',
        supplier_pn: 'RC0603'
      },
      {
        bom_revision_id: bomRevisionId,
        main_supplier: 'Murata',
        main_supplier_pn: 'GRM188',
        supplier: 'Samsung',
        supplier_pn: 'CL10'
      }
    ];

    secondSourceRepo.createMany(data);

    const items = secondSourceRepo.findByBomRevision(bomRevisionId);
    expect(items.length).toBe(2);
  });

  it('should delete second sources by revision', () => {
    secondSourceRepo.create({
      bom_revision_id: bomRevisionId,
      main_supplier: 'A',
      main_supplier_pn: 'A1',
      supplier: 'B',
      supplier_pn: 'B1'
    });

    secondSourceRepo.deleteByBomRevision(bomRevisionId);

    const items = secondSourceRepo.findByBomRevision(bomRevisionId);
    expect(items.length).toBe(0);
  });
});
