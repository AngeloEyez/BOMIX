import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import fs from 'fs';
import path from 'path';

import DatabaseManager from '../../../src/main/database/connection.js';
import projectRepo from '../../../src/main/database/repositories/project.repo.js';
import bomRevisionRepo from '../../../src/main/database/repositories/bom-revision.repo.js';

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
      project_code: 'TEST-001',
      description: 'Test Project'
    });

    expect(project).toBeDefined();
    expect(project.id).toBeDefined();
    expect(project.project_code).toBe('TEST-001');
    expect(project.description).toBe('Test Project');
  });

  it('should list all projects', () => {
    projectRepo.create({ project_code: 'P1' });
    projectRepo.create({ project_code: 'P2' });

    const projects = projectRepo.findAll();
    expect(projects.length).toBe(2);
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

    const verify = projectRepo.findById(created.id);
    expect(verify.description).toBe('New');
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
    bomRevisionRepo.create({
      project_id: project.id,
      phase_name: 'EVT',
      version: '0.1'
    });

    projectRepo.delete(project.id);

    // Check if revisions are deleted (Assuming cascade is set up in DB schema)
    // We can check by querying revisions directly if we have a way, or trust the schema.
    // Let's verify via bomRevisionRepo
    const revisions = bomRevisionRepo.findByProject(project.id);
    expect(revisions.length).toBe(0);
  });
});
