import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import fs from 'fs';
import path from 'path';

const DatabaseManager = require('../../../src/main/database/connection');
const projectRepo = require('../../../src/main/database/repositories/project.repo');
const bomRevisionRepo = require('../../../src/main/database/repositories/bom-revision.repo');

describe('Project Repository', () => {
  const testDbPath = path.join(__dirname, 'project.test.bomix');

  beforeEach(() => {
    DatabaseManager.connect(testDbPath);
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

  it('should create a project', () => {
    const project = projectRepo.create({
      project_code: 'TEST_PROJECT',
      description: 'Test Description'
    });

    expect(project).toBeDefined();
    expect(project.id).toBeDefined();
    expect(project.project_code).toBe('TEST_PROJECT');
    expect(project.description).toBe('Test Description');
  });

  it('should list all projects', () => {
    projectRepo.create({ project_code: 'P1' });
    projectRepo.create({ project_code: 'P2' });

    const projects = projectRepo.findAll();
    expect(projects.length).toBe(2);
    expect(projects[0].project_code).toBe('P2'); // Ordered by created_at DESC
  });

  it('should find project by id', () => {
    const created = projectRepo.create({ project_code: 'FIND_ME' });
    const found = projectRepo.findById(created.id);

    expect(found).toBeDefined();
    expect(found.project_code).toBe('FIND_ME');
  });

  it('should update project', () => {
    const created = projectRepo.create({ project_code: 'UPDATE_ME', description: 'Old' });
    const updated = projectRepo.update(created.id, { description: 'New' });

    expect(updated.description).toBe('New');

    const check = projectRepo.findById(created.id);
    expect(check.description).toBe('New');
  });

  it('should delete project', () => {
    const created = projectRepo.create({ project_code: 'DELETE_ME' });
    const success = projectRepo.delete(created.id);

    expect(success).toBe(true);

    const found = projectRepo.findById(created.id);
    expect(found).toBeUndefined();
  });

  it('should fail when creating project with duplicate code', () => {
    projectRepo.create({ project_code: 'DUPLICATE' });
    expect(() => {
      projectRepo.create({ project_code: 'DUPLICATE' });
    }).toThrow();
  });

  it('should cascade delete related bom revisions', () => {
    const project = projectRepo.create({ project_code: 'CASCADE_TEST' });
    const revision = bomRevisionRepo.create({
      project_id: project.id,
      phase_name: 'EVT',
      version: '0.1'
    });

    // Verify revision exists
    const foundRevision = bomRevisionRepo.findById(revision.id);
    expect(foundRevision).toBeDefined();

    // Delete project
    projectRepo.delete(project.id);

    // Verify revision is gone
    const deletedRevision = bomRevisionRepo.findById(revision.id);
    expect(deletedRevision).toBeUndefined();
  });
});
