import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import DatabaseManager from '../../../src/main/database/connection';
import fs from 'fs';
import path from 'path';

describe('Database Connection & Schema', () => {
    const testDbPath = path.join(__dirname, 'test.bomix');

    afterEach(() => {
        try {
            DatabaseManager.disconnect();
        } catch (e) {
            console.error(e);
        }

        if (fs.existsSync(testDbPath)) {
            try {
                fs.unlinkSync(testDbPath);
            } catch (e) {
                console.error('Failed to delete test db file', e);
            }
        }
    });

    it('should connect to a new database file and create schema', () => {
        const db = DatabaseManager.connect(testDbPath);
        expect(db).toBeDefined();
        expect(fs.existsSync(testDbPath)).toBe(true);

        // Check if tables exist
        const tables = db.prepare("SELECT name FROM sqlite_master WHERE type='table'").all();
        const tableNames = tables.map(t => t.name);

        expect(tableNames).toContain('series_meta');
        expect(tableNames).toContain('projects');
        expect(tableNames).toContain('bom_revisions');
        expect(tableNames).toContain('parts');
        expect(tableNames).toContain('second_sources');
    });

    it('should initialize series_meta with default record', () => {
        const db = DatabaseManager.connect(testDbPath);
        const meta = db.prepare('SELECT * FROM series_meta WHERE id = 1').get();
        expect(meta).toBeDefined();
        expect(meta.description).toBe('Default Series');
    });
});
