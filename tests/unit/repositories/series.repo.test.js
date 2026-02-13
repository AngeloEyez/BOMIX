import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import fs from 'fs';
import path from 'path';

// Use require to ensure we share the same instance as the repository which uses require
const DatabaseManager = require('../../../src/main/database/connection');
const seriesRepo = require('../../../src/main/database/repositories/series.repo');

describe('Series Repository', () => {
  const testDbPath = path.join(__dirname, 'series.test.bomix');

  beforeEach(() => {
    // 每個測試前連線並初始化資料庫
    DatabaseManager.connect(testDbPath);
  });

  afterEach(() => {
    // 每個測試後關閉連線並刪除測試檔案
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
    expect(meta.id).toBe(1);
    expect(meta.description).toBe('Default Series');
  });

  it('should update series description', () => {
    const newDescription = 'New Description';
    const updated = seriesRepo.updateMeta({ description: newDescription });

    expect(updated).toBeDefined();
    expect(updated.description).toBe(newDescription);

    const meta = seriesRepo.getMeta();
    expect(meta.description).toBe(newDescription);
  });
});
