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
    date TEXT,
    note TEXT,
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

        // 確保 series_meta 至少有一筆資料 (id=1)
        const stmt = db.prepare('INSERT OR IGNORE INTO series_meta (id, description) VALUES (1, ?)');
        stmt.run('Default Series');
    });

    transaction();
}

export {
    createSchema
};
