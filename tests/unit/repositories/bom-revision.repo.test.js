import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import fs from 'fs';
import path from 'path';

const DatabaseManager = require('../../../src/main/database/connection');
const projectRepo = require('../../../src/main/database/repositories/project.repo');
const bomRevisionRepo = require('../../../src/main/database/repositories/bom-revision.repo');

describe('BOM Revision Repository', () => {
  const testDbPath = path.join(__dirname, 'bom.test.bomix');
  let projectId;

  beforeEach(() => {
    DatabaseManager.connect(testDbPath);
    // 建立一個測試專案
    const project = projectRepo.create({
      project_code: 'BOM_TEST_PROJECT',
      description: 'Test Project for BOM Revisions'
    });
    projectId = project.id;
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

  it('should create a BOM revision', () => {
    const revision = bomRevisionRepo.create({
      project_id: projectId,
      phase_name: 'EVT',
      version: '0.1',
      description: 'Initial BOM'
    });

    expect(revision).toBeDefined();
    expect(revision.project_id).toBe(projectId);
    expect(revision.phase_name).toBe('EVT');
    expect(revision.version).toBe('0.1');
  });

  it('should find revisions by project', () => {
    bomRevisionRepo.create({ project_id: projectId, phase_name: 'EVT', version: '0.1' });
    bomRevisionRepo.create({ project_id: projectId, phase_name: 'EVT', version: '0.2' });
    bomRevisionRepo.create({ project_id: projectId, phase_name: 'DVT', version: '1.0' });

    const revisions = bomRevisionRepo.findByProject(projectId);

    expect(revisions.length).toBe(3);
    // Ordered by phase_name ASC, version DESC
    // DVT 1.0
    // EVT 0.2
    // EVT 0.1
    expect(revisions[0].phase_name).toBe('DVT');
    expect(revisions[1].version).toBe('0.2');
  });

  it('should find revision by id', () => {
    const created = bomRevisionRepo.create({
      project_id: projectId,
      phase_name: 'PVT',
      version: '1.0'
    });

    const found = bomRevisionRepo.findById(created.id);
    expect(found).toBeDefined();
    expect(found.phase_name).toBe('PVT');
  });

  it('should delete revision', () => {
    const created = bomRevisionRepo.create({
      project_id: projectId,
      phase_name: 'MP',
      version: '1.0'
    });

    const success = bomRevisionRepo.delete(created.id);
    expect(success).toBe(true);

    const found = bomRevisionRepo.findById(created.id);
    expect(found).toBeUndefined();
  });

  it('should fail when creating duplicate revision for same project', () => {
    bomRevisionRepo.create({ project_id: projectId, phase_name: 'EVT', version: '0.1' });

    expect(() => {
      bomRevisionRepo.create({ project_id: projectId, phase_name: 'EVT', version: '0.1' });
    }).toThrow();
  });
});
