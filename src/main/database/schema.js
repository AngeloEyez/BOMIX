/**
 * @file src/main/database/schema.js
 * @description 定義 BOMIX 資料庫 Schema 與初始化邏輯
 * @module database/schema
 */

/**
 * 建立系列元資料表 SQL
 * @type {string}
 */
const CREATE_SERIES_META = `
CREATE TABLE IF NOT EXISTS series_meta (
    id INTEGER PRIMARY KEY DEFAULT 1,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`;

/**
 * 建立專案表 SQL
 * @type {string}
 */
const CREATE_PROJECTS = `
CREATE TABLE IF NOT EXISTS projects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_code TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`;

/**
 * 建立 BOM 版本表 SQL
 * @type {string}
 */
const CREATE_BOM_REVISIONS = `
CREATE TABLE IF NOT EXISTS bom_revisions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    phase_name TEXT NOT NULL,
    version TEXT NOT NULL,
    description TEXT,
    schematic_version TEXT,
    pcb_version TEXT,
    pca_pn TEXT,
    bom_date TEXT,
    note TEXT,
    mode TEXT DEFAULT 'NPI',
    filename TEXT,
    suffix TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    UNIQUE(project_id, phase_name, version)
);
`;

/**
 * 建立零件表 SQL
 * @type {string}
 */
const CREATE_PARTS = `
CREATE TABLE IF NOT EXISTS parts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    bom_revision_id INTEGER NOT NULL,
    item INTEGER,
    hhpn TEXT,
    supplier TEXT NOT NULL,
    supplier_pn TEXT NOT NULL,
    description TEXT,
    location TEXT NOT NULL,
    type TEXT,
    bom_status TEXT NOT NULL DEFAULT 'I',
    ccl TEXT DEFAULT 'N',
    remark TEXT,
    FOREIGN KEY (bom_revision_id) REFERENCES bom_revisions(id) ON DELETE CASCADE
);
`;

/**
 * 建立零件表索引 SQL
 * @type {string[]}
 */
const CREATE_PARTS_INDICES = [
    `CREATE INDEX IF NOT EXISTS idx_parts_group ON parts(bom_revision_id, supplier, supplier_pn, type);`,
    `CREATE INDEX IF NOT EXISTS idx_parts_location ON parts(bom_revision_id, location);`
];

/**
 * 建立替代料表 SQL
 * @type {string}
 */
const CREATE_SECOND_SOURCES = `
CREATE TABLE IF NOT EXISTS second_sources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    bom_revision_id INTEGER NOT NULL,
    main_supplier TEXT NOT NULL,
    main_supplier_pn TEXT NOT NULL,
    hhpn TEXT,
    supplier TEXT NOT NULL,
    supplier_pn TEXT NOT NULL,
    description TEXT,
    FOREIGN KEY (bom_revision_id) REFERENCES bom_revisions(id) ON DELETE CASCADE
);
`;

/**
 * 建立替代料表索引 SQL
 * @type {string[]}
 */
const CREATE_SECOND_SOURCES_INDICES = [
    `CREATE INDEX IF NOT EXISTS idx_ss_main ON second_sources(bom_revision_id, main_supplier, main_supplier_pn);`
];

/**
 * 建立 Matrix Models 表 SQL
 * @type {string}
 */
const CREATE_MATRIX_MODELS = `
CREATE TABLE IF NOT EXISTS matrix_models (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    bom_revision_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (bom_revision_id) REFERENCES bom_revisions(id) ON DELETE CASCADE
);
`;

/**
 * 建立 Matrix Selections 表 SQL
 * @type {string}
 */
const CREATE_MATRIX_SELECTIONS = `
CREATE TABLE IF NOT EXISTS matrix_selections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    matrix_model_id INTEGER NOT NULL,
    group_key TEXT NOT NULL,
    selected_type TEXT NOT NULL,
    selected_id INTEGER NOT NULL,
    FOREIGN KEY (matrix_model_id) REFERENCES matrix_models(id) ON DELETE CASCADE,
    UNIQUE(matrix_model_id, group_key)
);
`;

/**
 * 初始化資料庫 Schema
 * @param {import('better-sqlite3').Database} db - 資料庫實例
 */
function createSchema(db) {
    // 開啟交易確保原子性
    const transaction = db.transaction(() => {
        db.exec(CREATE_SERIES_META);
        db.exec(CREATE_PROJECTS);
        db.exec(CREATE_BOM_REVISIONS);
        db.exec(CREATE_PARTS);
        CREATE_PARTS_INDICES.forEach(sql => db.exec(sql));
        db.exec(CREATE_SECOND_SOURCES);
        CREATE_SECOND_SOURCES_INDICES.forEach(sql => db.exec(sql));
        db.exec(CREATE_MATRIX_MODELS);
        db.exec(CREATE_MATRIX_SELECTIONS);

        // 確保 series_meta 至少有一筆資料 (id=1)
        const stmt = db.prepare('INSERT OR IGNORE INTO series_meta (id, description) VALUES (1, ?)');
        stmt.run('Default Series');
    });

    transaction();
}

/**
 * 遷移資料庫 Schema (處理舊版本資料庫升級)
 * @param {import('better-sqlite3').Database} db - 資料庫實例
 */
function migrateSchema(db) {
    const tableInfo = db.pragma('table_info(bom_revisions)');
    const columns = new Set(tableInfo.map(col => col.name));

    // 1. Phase 4 -> Phase 5: mode
    if (!columns.has('mode')) {
        console.log('[Schema] 正在遷移: 新增 mode 欄位至 bom_revisions');
        db.exec("ALTER TABLE bom_revisions ADD COLUMN mode TEXT DEFAULT 'NPI'");
    }

    // 2. User Request: Rename date -> bom_date, Add filename, suffix
    if (columns.has('date') && !columns.has('bom_date')) {
        console.log('[Schema] 正在遷移: 重新命名 bom_revisions.date -> bom_date');
        try {
            db.exec('ALTER TABLE bom_revisions RENAME COLUMN date TO bom_date');
        } catch (e) {
            console.error('[Schema] 重命名失敗 (可能是 SQLite 版本過舊):', e);
            // Fallback: Add new column and copy data? Or just ignore if empty.
            // Assuming SQLite 3.25+ is available with better-sqlite3.
        }
    }

    if (!columns.has('filename')) {
        console.log('[Schema] 正在遷移: 新增 filename 欄位至 bom_revisions');
        db.exec("ALTER TABLE bom_revisions ADD COLUMN filename TEXT");
    }

    if (!columns.has('suffix')) {
        console.log('[Schema] 正在遷移: 新增 suffix 欄位至 bom_revisions');
        db.exec("ALTER TABLE bom_revisions ADD COLUMN suffix TEXT");
    }

    // 3. Phase 7: Matrix BOM
    // 檢查 matrix_models 表是否存在
    const tables = db.prepare("SELECT name FROM sqlite_master WHERE type='table' AND name='matrix_models'").get();
    if (!tables) {
        console.log('[Schema] 正在遷移: 建立 Matrix BOM 相關資料表');
        db.exec(CREATE_MATRIX_MODELS);
        db.exec(CREATE_MATRIX_SELECTIONS);
    }
}

export {
    createSchema,
    migrateSchema
};
