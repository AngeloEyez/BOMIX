/**
 * @file src/main/services/task-manager.service.js
 * @description 任務排程管理器 (Task Manager Service)
 *
 * 提供 FIFO 佇列排程機制，統一管理所有非同步任務（匯入、匯出、資料庫維護等）。
 * 取代舊版 progress.service.js，新增佇列排程、任務生命週期管理與 callback 機制。
 *
 * @module services/task-manager
 */

import { EventEmitter } from 'events';
import { randomUUID } from 'crypto';

// ========================================
// 任務狀態常數
// ========================================

export const TASK_STATUS = {
    QUEUED: 'QUEUED',         // 已加入佇列，等待排程
    RUNNING: 'RUNNING',       // 正在執行
    COMPLETED: 'COMPLETED',   // 執行完成
    FAILED: 'FAILED',         // 執行失敗
    CANCELLED: 'CANCELLED'    // 被使用者取消（僅 QUEUED 狀態可取消）
};

// ========================================
// TaskManager 類別
// ========================================

class TaskManager extends EventEmitter {
    constructor() {
        super();
        /** @type {Map<string, Object>} 所有任務（含歷史紀錄） */
        this.tasks = new Map();
        /** @type {Array<string>} 待執行任務 ID 佇列 (FIFO) */
        this.queue = [];
        /** @type {string|null} 正在執行的任務 ID */
        this.currentTaskId = null;
        /** @type {boolean} 佇列處理器是否正在運行 */
        this.processing = false;
    }

    // ========================================
    // 公開 API
    // ========================================

    /**
     * 將任務加入排程佇列。
     *
     * 建立新任務並加入 FIFO 佇列，若佇列處理器未運行則自動啟動。
     *
     * @param {string} type - 任務類型 (e.g., 'EXPORT_BOM', 'IMPORT_BOM')
     * @param {Object} options - 任務選項
     * @param {string} [options.title] - 任務顯示名稱
     * @param {Object} [options.metadata={}] - 任務相關元資料
     * @param {Function} options.executeFn - 實際執行函數，接收 taskContext 參數，回傳 Promise
     * @returns {string} taskId
     */
    enqueue(type, { title, metadata = {}, executeFn } = {}) {
        if (typeof executeFn !== 'function') {
            throw new Error('executeFn 必須是一個函數');
        }

        const taskId = randomUUID();
        const task = {
            id: taskId,
            type,
            title: title || type,
            status: TASK_STATUS.QUEUED,
            progress: 0,
            message: '等待排程中...',
            logs: [],
            result: null,
            error: null,
            metadata,
            // executeFn 不會透過 IPC 傳送到渲染層（存於 _internal）
            _internal: { executeFn },
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString()
        };

        this.tasks.set(taskId, task);

        // 加入 FIFO 佇列
        this.queue.push(taskId);

        // 廣播任務建立事件
        this._broadcastUpdate(task);

        // 自動啟動佇列處理器
        this._processQueue();

        return taskId;
    }

    /**
     * 更新任務進度。
     *
     * @param {string} taskId - 任務 ID
     * @param {number} progress - 進度百分比 (0-100)
     * @param {string} [message] - 進度訊息
     */
    updateProgress(taskId, progress, message = null) {
        const task = this.tasks.get(taskId);
        if (!task) return;

        // 已結束的任務不再更新
        if ([TASK_STATUS.COMPLETED, TASK_STATUS.FAILED, TASK_STATUS.CANCELLED].includes(task.status)) {
            return;
        }

        task.progress = Math.min(100, Math.max(0, progress));
        if (message) {
            task.message = message;
        }
        task.updatedAt = new Date().toISOString();

        this._broadcastUpdate(task);
    }

    /**
     * 新增任務日誌。
     *
     * @param {string} taskId - 任務 ID
     * @param {string} message - 日誌訊息
     * @param {string} [level='info'] - 日誌等級 ('info' | 'warn' | 'error')
     */
    log(taskId, message, level = 'info') {
        const task = this.tasks.get(taskId);
        if (!task) return;

        const logEntry = {
            timestamp: new Date().toISOString(),
            message,
            level
        };

        task.logs.push(logEntry);
        task.updatedAt = new Date().toISOString();

        // info 等級的 log 同時更新 message（用於 StatusBar 顯示）
        if (level === 'info') {
            task.message = message;
        }

        this._broadcastUpdate(task);
    }

    /**
     * 標記任務完成。
     *
     * @param {string} taskId - 任務 ID
     * @param {any} [result=null] - 任務執行結果
     */
    completeTask(taskId, result = null) {
        const task = this.tasks.get(taskId);
        if (!task) return;

        task.status = TASK_STATUS.COMPLETED;
        task.progress = 100;
        task.message = '已完成';
        task.result = result;
        task.updatedAt = new Date().toISOString();

        this.log(taskId, '任務執行完成。', 'info');

        this._broadcastUpdate(task);

        // 發送任務完成事件（用於 UI callback，帶 type 與 result）
        this.emit('task:completed', {
            id: task.id,
            type: task.type,
            result: task.result,
            metadata: task.metadata
        });
    }

    /**
     * 標記任務失敗。
     *
     * @param {string} taskId - 任務 ID
     * @param {string|Error} error - 錯誤訊息或物件
     */
    failTask(taskId, error) {
        const task = this.tasks.get(taskId);
        if (!task) return;

        const errorMessage = error instanceof Error ? error.message : String(error);

        task.status = TASK_STATUS.FAILED;
        task.error = errorMessage;
        task.updatedAt = new Date().toISOString();

        this.log(taskId, `任務失敗: ${errorMessage}`, 'error');

        this._broadcastUpdate(task);
    }

    /**
     * 取消任務（僅 QUEUED 狀態可取消）。
     *
     * 已開始執行的任務不支援中途取消。
     *
     * @param {string} taskId - 任務 ID
     * @returns {boolean} 是否成功取消
     */
    cancelTask(taskId) {
        const task = this.tasks.get(taskId);
        if (!task) return false;

        // 只能取消排隊中的任務
        if (task.status !== TASK_STATUS.QUEUED) {
            return false;
        }

        task.status = TASK_STATUS.CANCELLED;
        task.message = '已取消';
        task.updatedAt = new Date().toISOString();

        // 從佇列中移除
        this.queue = this.queue.filter(id => id !== taskId);

        this.log(taskId, '任務已被使用者取消。', 'warn');

        this._broadcastUpdate(task);
        return true;
    }

    /**
     * 取得任務資訊。
     *
     * @param {string} taskId - 任務 ID
     * @returns {Object|null} 任務物件（已去除 _internal）
     */
    getTask(taskId) {
        const task = this.tasks.get(taskId);
        if (!task) return null;
        return this._sanitizeTask(task);
    }

    /**
     * 移除任務紀錄（僅限已結束的任務）。
     *
     * @param {string} taskId - 任務 ID
     * @returns {boolean} 是否成功移除
     */
    removeTask(taskId) {
        const task = this.tasks.get(taskId);
        if (!task) return false;

        // 不能移除正在執行或等待中的任務
        if ([TASK_STATUS.QUEUED, TASK_STATUS.RUNNING].includes(task.status)) {
            return false;
        }

        this.tasks.delete(taskId);
        return true;
    }

    /**
     * 取得佇列狀態概覽。
     *
     * @returns {{ currentTask: Object|null, queueLength: number, totalTasks: number }}
     */
    getQueueStatus() {
        const currentTask = this.currentTaskId
            ? this._sanitizeTask(this.tasks.get(this.currentTaskId))
            : null;

        return {
            currentTask,
            queueLength: this.queue.length,
            totalTasks: this.tasks.size
        };
    }

    // ========================================
    // 內部方法
    // ========================================

    /**
     * FIFO 佇列處理迴圈。
     *
     * 自動從佇列取出下一個任務並執行，直到佇列為空。
     * 使用 setImmediate 確保不阻塞事件迴圈。
     *
     * @private
     */
    async _processQueue() {
        // 若已有處理器在運行，不重複啟動
        if (this.processing) return;

        this.processing = true;

        while (this.queue.length > 0) {
            const taskId = this.queue.shift();
            const task = this.tasks.get(taskId);

            // 任務可能已被取消（從佇列中移除前被 cancel）
            if (!task || task.status !== TASK_STATUS.QUEUED) {
                continue;
            }

            this.currentTaskId = taskId;

            // 標記為 RUNNING
            task.status = TASK_STATUS.RUNNING;
            task.message = '開始執行...';
            task.updatedAt = new Date().toISOString();
            this._broadcastUpdate(task);

            // 建立 taskContext（注入給 executeFn 使用）
            const ctx = this._createTaskContext(taskId);

            try {
                // 使用 setImmediate 包裝，確保 UI 有機會更新
                await new Promise(resolve => setImmediate(resolve));

                const result = await task._internal.executeFn(ctx);
                this.completeTask(taskId, result);
            } catch (error) {
                console.error(`[TaskManager] 任務 ${taskId} 執行錯誤:`, error);
                this.failTask(taskId, error);
            }

            this.currentTaskId = null;

            // 每個任務執行完後，釋放事件迴圈給 UI
            await new Promise(resolve => setImmediate(resolve));
        }

        this.processing = false;
    }

    /**
     * 建立任務執行上下文。
     *
     * 提供給 executeFn 使用的介面，讓任務可以更新進度與寫入日誌。
     *
     * @param {string} taskId - 任務 ID
     * @returns {{ updateProgress: Function, log: Function, taskId: string }}
     * @private
     */
    _createTaskContext(taskId) {
        return {
            taskId,
            /**
             * 更新任務進度
             * @param {number} progress - 進度百分比 (0-100)
             * @param {string} [message] - 進度訊息
             */
            updateProgress: (progress, message) => {
                this.updateProgress(taskId, progress, message);
            },
            /**
             * 新增任務日誌
             * @param {string} message - 日誌訊息
             * @param {string} [level='info'] - 日誌等級
             */
            log: (message, level = 'info') => {
                this.log(taskId, message, level);
            },
            /**
             * 讓出事件迴圈（用於同步密集操作之間）
             * @returns {Promise<void>}
             */
            yield: () => new Promise(resolve => setImmediate(resolve))
        };
    }

    /**
     * 廣播任務更新事件。
     *
     * 將任務資料（去除 _internal）透過事件發送給 IPC 層。
     *
     * @param {Object} task - 原始任務物件
     * @private
     */
    _broadcastUpdate(task) {
        this.emit('task:update', this._sanitizeTask(task));
    }

    /**
     * 清理任務物件（移除 _internal 欄位）。
     *
     * 避免將 executeFn 等內部資料傳送到渲染層。
     *
     * @param {Object} task - 原始任務物件
     * @returns {Object} 已清理的任務物件
     * @private
     */
    _sanitizeTask(task) {
        if (!task) return null;
        const { _internal, ...sanitized } = task;
        return sanitized;
    }
}

// ========================================
// Singleton 實例
// ========================================

const taskManager = new TaskManager();
export default taskManager;
