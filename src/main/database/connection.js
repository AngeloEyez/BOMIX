/**
 * @file src/main/database/connection.js
 * @description SQLite 資料庫連線管理模組
 * @module database/connection
 */

const Database = require('better-sqlite3');
const { createSchema } = require('./schema');

/**
 * 資料庫連線管理器 (Singleton)
 * 負責管理 SQLite 連線的生命週期與 Schema 初始化
 */
class DatabaseManager {
  constructor() {
    /**
     * @type {import('better-sqlite3').Database|null}
     * @private
     */
    this.db = null;
    /**
     * @type {string|null}
     * @private
     */
    this.currentPath = null;
  }

  /**
   * 取得 DatabaseManager 單例
   * @returns {DatabaseManager}
   */
  static getInstance() {
    if (!DatabaseManager.instance) {
      DatabaseManager.instance = new DatabaseManager();
    }
    return DatabaseManager.instance;
  }

  /**
   * 連接至指定的 .bomix 資料庫檔案
   * 若檔案不存在會自動建立，並初始化 Schema
   *
   * @param {string} filePath - 資料庫檔案路徑
   * @throws {Error} 若連線失敗或路徑無效
   * @returns {import('better-sqlite3').Database} 資料庫實例
   */
  connect(filePath) {
    if (this.db) {
      // 若已連線至同一檔案，直接回傳
      if (this.currentPath === filePath) {
        return this.db;
      }
      // 若連線至不同檔案，先關閉舊連線
      this.disconnect();
    }

    try {
      this.db = new Database(filePath, { verbose: null }); // verbose: console.log 可用於除錯 SQL
      this.db.pragma('journal_mode = WAL'); // 啟用 Write-Ahead Logging 提升效能
      this.db.pragma('foreign_keys = ON');  // 啟用外鍵約束

      // 初始化 Schema
      createSchema(this.db);

      this.currentPath = filePath;
      console.log(`[Database] 已連線至: ${filePath}`);
      return this.db;
    } catch (error) {
      console.error(`[Database] 連線失敗: ${filePath}`, error);
      this.disconnect(); // 確保清理
      throw error;
    }
  }

  /**
   * 關閉目前的資料庫連線
   */
  disconnect() {
    if (this.db) {
      try {
        this.db.close();
        console.log(`[Database] 已關閉連線: ${this.currentPath}`);
      } catch (error) {
        console.error(`[Database] 關閉連線失敗`, error);
      } finally {
        this.db = null;
        this.currentPath = null;
      }
    }
  }

  /**
   * 取得目前的資料庫實例
   * @throws {Error} 若尚未建立連線
   * @returns {import('better-sqlite3').Database}
   */
  getDb() {
    if (!this.db) {
      throw new Error('[Database] 尚未建立資料庫連線');
    }
    return this.db;
  }
}

module.exports = DatabaseManager.getInstance();
