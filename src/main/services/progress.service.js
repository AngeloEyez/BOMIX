/**
 * @file src/main/services/progress.service.js
 * @description 進度追蹤服務 (Progress Tracking Service)
 * @module services/progress
 */

import { EventEmitter } from 'events';
import { randomUUID } from 'crypto';

export const TASK_STATUS = {
    PENDING: 'PENDING',
    RUNNING: 'RUNNING',
    COMPLETED: 'COMPLETED',
    FAILED: 'FAILED',
    CANCELLED: 'CANCELLED'
};

class ProgressService extends EventEmitter {
    constructor() {
        super();
        this.tasks = new Map();
    }

    /**
     * 建立新的進度任務
     * @param {string} type - 任務類型 (e.g., 'EXPORT_BOM')
     * @param {Object} metadata - 額外資訊 (e.g., { fileName: 'test.xlsx' })
     * @returns {string} taskId
     */
    createTask(type, metadata = {}) {
        const taskId = randomUUID();
        const task = {
            id: taskId,
            type,
            status: TASK_STATUS.PENDING,
            progress: 0,
            message: 'Initializing...',
            result: null,
            error: null,
            metadata,
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString()
        };

        this.tasks.set(taskId, task);
        this.emit('task:created', task);
        return taskId;
    }

    /**
     * 更新任務進度
     * @param {string} taskId
     * @param {number} progress - 進度百分比 (0-100)
     * @param {string} message - 進度訊息 (Optional)
     */
    updateProgress(taskId, progress, message = null) {
        const task = this.tasks.get(taskId);
        if (!task) return;

        if (task.status === TASK_STATUS.PENDING) {
            task.status = TASK_STATUS.RUNNING;
        }

        // Prevent updates if task is already finished
        if ([TASK_STATUS.COMPLETED, TASK_STATUS.FAILED, TASK_STATUS.CANCELLED].includes(task.status)) {
            return;
        }

        task.progress = Math.min(100, Math.max(0, progress));
        if (message) {
            task.message = message;
        }
        task.updatedAt = new Date().toISOString();

        this.emit('task:update', task);
    }

    /**
     * 完成任務
     * @param {string} taskId
     * @param {Object} result - 任務結果
     */
    completeTask(taskId, result = null) {
        const task = this.tasks.get(taskId);
        if (!task) return;

        task.status = TASK_STATUS.COMPLETED;
        task.progress = 100;
        task.message = 'Completed';
        task.result = result;
        task.updatedAt = new Date().toISOString();

        this.emit('task:complete', task);
        // Also emit generic update so listeners can just listen to 'task:update' if they want
        this.emit('task:update', task);
    }

    /**
     * 任務失敗
     * @param {string} taskId
     * @param {string|Error} error - 錯誤訊息或物件
     */
    failTask(taskId, error) {
        const task = this.tasks.get(taskId);
        if (!task) return;

        task.status = TASK_STATUS.FAILED;
        task.error = error instanceof Error ? error.message : String(error);
        task.updatedAt = new Date().toISOString();

        this.emit('task:error', task);
        this.emit('task:update', task);
    }

    /**
     * 取消任務
     * @param {string} taskId
     */
    cancelTask(taskId) {
        const task = this.tasks.get(taskId);
        if (!task) return;

        if ([TASK_STATUS.COMPLETED, TASK_STATUS.FAILED].includes(task.status)) {
            return;
        }

        task.status = TASK_STATUS.CANCELLED;
        task.message = 'Cancelled by user';
        task.updatedAt = new Date().toISOString();

        this.emit('task:cancel', task);
        this.emit('task:update', task);
    }

    /**
     * 取得任務資訊
     * @param {string} taskId
     * @returns {Object|null}
     */
    getTask(taskId) {
        return this.tasks.get(taskId) || null;
    }

    /**
     * 清除任務 (Optional: 可以用於定期清理)
     * @param {string} taskId
     */
    removeTask(taskId) {
        this.tasks.delete(taskId);
    }
}

// Singleton Instance
const progressService = new ProgressService();
export default progressService;
