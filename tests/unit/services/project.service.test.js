import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import path from 'path';
import fs from 'fs';
import { createSeries } from '../../../src/main/services/series.service';
import { createProject, getAllProjects, getProjectById, updateProject, deleteProject } from '../../../src/main/services/project.service';
import dbManager from '../../../src/main/database/connection';

let testDbPath;

function deleteDbFiles(filePath) {
    if (!filePath) return;
    const files = [filePath, `${filePath}-wal`, `${filePath}-shm`];
    files.forEach(f => {
        if (fs.existsSync(f)) {
            try {
                fs.unlinkSync(f);
            } catch (e) {
                // console.warn(`Failed to delete ${f}: ${e.message}`);
            }
        }
    });
}

describe('Project Service', () => {
    beforeEach(() => {
        testDbPath = path.join(__dirname, `test_project_${Date.now()}_${Math.random().toString(36).substring(7)}.bomix`);
        // Initialize DB with a series
        createSeries(testDbPath, 'Test Series');
    });

    afterEach(() => {
        try {
            dbManager.disconnect();
        } catch (e) {
            console.error('Failed to disconnect:', e);
        }
        deleteDbFiles(testDbPath);
    });

    it('should create a project', () => {
        const project = createProject('PROJ-001', 'Test Project');
        expect(project).toBeDefined();
        expect(project.project_code).toBe('PROJ-001');
        expect(project.description).toBe('Test Project');
        expect(project.id).toBeDefined();
    });

    it('should fail creating duplicate project code', () => {
        createProject('PROJ-001', 'Test Project');
        expect(() => createProject('PROJ-001', 'Duplicate')).toThrow(/已存在/);
    });

    it('should get all projects', () => {
        createProject('P1', 'Project 1');
        createProject('P2', 'Project 2');

        const projects = getAllProjects();
        expect(projects.length).toBe(2);
        // Order should be descending by created_at/id
        expect(projects[0].project_code).toBe('P2');
        expect(projects[1].project_code).toBe('P1');
    });

    it('should get project by id', () => {
        const created = createProject('P1', 'Project 1');
        const retrieved = getProjectById(created.id);
        expect(retrieved).toEqual(created);
    });

    it('should update project', () => {
        const created = createProject('P1', 'Old Desc');
        const updated = updateProject(created.id, 'New Desc');
        expect(updated.description).toBe('New Desc');

        const retrieved = getProjectById(created.id);
        expect(retrieved.description).toBe('New Desc');
    });

    it('should delete project', () => {
        const created = createProject('P1', 'Project 1');
        const result = deleteProject(created.id);
        expect(result.success).toBe(true);

        const projects = getAllProjects();
        expect(projects.length).toBe(0);

        expect(() => getProjectById(created.id)).toThrow();
    });
});
