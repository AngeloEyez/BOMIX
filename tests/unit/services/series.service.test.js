import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import path from 'path';
import fs from 'fs';
import { createSeries, openSeries, getSeriesMeta, updateSeriesMeta } from '../../../src/main/services/series.service';
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

describe('Series Service', () => {
    beforeEach(() => {
        testDbPath = path.join(__dirname, `test_series_${Date.now()}_${Math.random().toString(36).substring(7)}.bomix`);
    });

    afterEach(() => {
        try {
            dbManager.disconnect();
        } catch (e) {
            console.error('Failed to disconnect:', e);
        }
        deleteDbFiles(testDbPath);
    });

    it('should create a new series with description', () => {
        const result = createSeries(testDbPath, 'Test Series');
        expect(result).toBeDefined();
        expect(result.description).toBe('Test Series');
        expect(fs.existsSync(testDbPath)).toBe(true);
    });

    it('should create a new series without description (default)', () => {
        const result = createSeries(testDbPath);
        expect(result).toBeDefined();
        expect(result.description).toBe('Default Series');
    });

    it('should open an existing series', () => {
        createSeries(testDbPath, 'Test Series');
        dbManager.disconnect();

        const result = openSeries(testDbPath);
        expect(result).toBeDefined();
        expect(result.description).toBe('Test Series');
    });

    it('should fail to open non-existent file', () => {
        const newPath = path.join(__dirname, `non_existent_${Date.now()}.bomix`);
        expect(() => openSeries(newPath)).toThrow(/找不到檔案/);
    });

    it('should update series meta', () => {
        createSeries(testDbPath, 'Original Description');
        const updated = updateSeriesMeta('Updated Description');
        expect(updated.description).toBe('Updated Description');

        const current = getSeriesMeta();
        expect(current.description).toBe('Updated Description');
    });

    it('should get series meta', () => {
        createSeries(testDbPath, 'My Series');
        const meta = getSeriesMeta();
        expect(meta.description).toBe('My Series');
    });
});
