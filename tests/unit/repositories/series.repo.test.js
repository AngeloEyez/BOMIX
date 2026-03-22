import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import fs from 'fs';
import path from 'path';

import DatabaseManager from '../../../src/main/database/connection.js';
import seriesRepo from '../../../src/main/database/repositories/series.repo.js';

describe('Series Repository', () => {
  const testDbPath = path.join(__dirname, 'series.test.bomix');

  beforeEach(() => {
    // 每個測試前連線並初始化資料庫
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

  it('should initialize with default meta', () => {
    const meta = seriesRepo.getMeta();
    expect(meta).toBeDefined();
    // Default description from schema initialization
    expect(meta.description).toBe('Default Series');
  });

  it('should update series description', () => {
    const updated = seriesRepo.updateMeta({ description: 'New Description' });
    expect(updated.description).toBe('New Description');

    const meta = seriesRepo.getMeta();
    expect(meta.description).toBe('New Description');
  });
});
