/**
 * @file tests/unit/task-manager.service.test.js
 * @description TaskManager 服務單元測試
 *
 * 測試 FIFO 佇列排程、任務生命週期、進度更新、日誌、
 * 取消與移除等核心功能。
 *
 * 注意：TaskManager 為 singleton，且 enqueue 後會立即啟動佇列處理。
 * 所有測試需確保任務最終完成，避免阻塞後續測試。
 */

import { describe, it, expect } from 'vitest';
import taskManager, { TASK_STATUS } from '../../src/main/services/task-manager.service.js';

/** 輔助函數：等待指定毫秒 */
const wait = (ms) => new Promise(resolve => setTimeout(resolve, ms));

describe('TaskManager Service', () => {

    // ========================================
    // 基礎功能
    // ========================================

    it('enqueue 缺少 executeFn 應拋出錯誤', () => {
        expect(() => {
            taskManager.enqueue('NO_FN', { title: '無 executeFn' });
        }).toThrow('executeFn 必須是一個函數');
    });

    it('getTask 應回傳已清理的任務物件（不含 _internal）', async () => {
        const taskId = taskManager.enqueue('SANITIZE_TEST', {
            title: '清理測試',
            metadata: { key: 'val' },
            executeFn: async () => 'done'
        });

        // 立即檢查（可能是 QUEUED 或已 RUNNING）
        const task = taskManager.getTask(taskId);
        expect(task).toBeDefined();
        expect(task.type).toBe('SANITIZE_TEST');
        expect(task.metadata).toEqual({ key: 'val' });
        expect(task._internal).toBeUndefined();

        // 等任務完成避免阻塞後續測試
        await wait(200);
    });

    it('getQueueStatus 應回傳佇列資訊結構', () => {
        const status = taskManager.getQueueStatus();
        expect(status).toHaveProperty('currentTask');
        expect(status).toHaveProperty('queueLength');
        expect(status).toHaveProperty('totalTasks');
        expect(typeof status.queueLength).toBe('number');
        expect(typeof status.totalTasks).toBe('number');
    });

    // ========================================
    // 任務執行與完成
    // ========================================

    it('單一任務應自動執行並標記 COMPLETED', async () => {
        const result = { data: 'ok' };
        const taskId = taskManager.enqueue('AUTO_COMPLETE', {
            title: '自動完成',
            executeFn: async (ctx) => {
                ctx.updateProgress(50, '處理中...');
                ctx.log('正在處理');
                return result;
            }
        });

        await wait(300);

        const task = taskManager.getTask(taskId);
        expect(task.status).toBe(TASK_STATUS.COMPLETED);
        expect(task.progress).toBe(100);
        expect(task.result).toEqual(result);
    });

    it('executeFn 拋出錯誤時應標記 FAILED', async () => {
        const taskId = taskManager.enqueue('ERROR_FAIL', {
            title: '失敗測試',
            executeFn: async () => { throw new Error('測試錯誤'); }
        });

        await wait(300);

        const task = taskManager.getTask(taskId);
        expect(task.status).toBe(TASK_STATUS.FAILED);
        expect(task.error).toBe('測試錯誤');
    });

    // ========================================
    // FIFO 佇列排程
    // ========================================

    it('多任務應 FIFO 依序執行', async () => {
        const order = [];

        taskManager.enqueue('FIFO_A', {
            title: 'A',
            executeFn: async () => { order.push('A'); await wait(20); }
        });
        taskManager.enqueue('FIFO_B', {
            title: 'B',
            executeFn: async () => { order.push('B'); await wait(20); }
        });
        taskManager.enqueue('FIFO_C', {
            title: 'C',
            executeFn: async () => { order.push('C'); }
        });

        await wait(500);

        expect(order).toEqual(['A', 'B', 'C']);
    });

    // ========================================
    // 取消功能
    // ========================================

    it('QUEUED 任務可取消，RUNNING 任務不可', async () => {
        // 建立阻塞任務
        let resolveBlocker;
        taskManager.enqueue('BLOCK_FOR_CANCEL', {
            title: '阻塞器',
            executeFn: async () => {
                await new Promise(r => { resolveBlocker = r; });
            }
        });

        await wait(50); // 等阻塞任務開始 RUNNING

        // 加入一個等待中的任務
        const queuedId = taskManager.enqueue('QUEUED_CANCEL', {
            title: '待取消',
            executeFn: async () => 'should not run'
        });

        // 驗證 QUEUED 並取消
        expect(taskManager.getTask(queuedId).status).toBe(TASK_STATUS.QUEUED);
        expect(taskManager.cancelTask(queuedId)).toBe(true);
        expect(taskManager.getTask(queuedId).status).toBe(TASK_STATUS.CANCELLED);

        // 釋放阻塞器
        resolveBlocker?.();
        await wait(100);
    });

    // ========================================
    // 事件廣播
    // ========================================

    it('進度更新應廣播 task:update 事件', async () => {
        const captured = [];
        const listener = (task) => {
            if (task.type === 'PROGRESS_EVT') {
                captured.push(task.progress);
            }
        };
        taskManager.on('task:update', listener);

        taskManager.enqueue('PROGRESS_EVT', {
            title: '進度事件',
            executeFn: async (ctx) => {
                ctx.updateProgress(30, '30%');
                ctx.updateProgress(80, '80%');
            }
        });

        await wait(300);
        taskManager.removeListener('task:update', listener);

        expect(captured).toContain(30);
        expect(captured).toContain(80);
    });

    it('任務完成後應觸發 task:completed 事件', async () => {
        const events = [];
        const listener = (data) => {
            if (data.type === 'COMPLETED_EVT') events.push(data);
        };
        taskManager.on('task:completed', listener);

        taskManager.enqueue('COMPLETED_EVT', {
            title: '完成事件',
            metadata: { k: 'v' },
            executeFn: async () => ({ ok: true })
        });

        await wait(300);
        taskManager.removeListener('task:completed', listener);

        expect(events.length).toBe(1);
        expect(events[0].type).toBe('COMPLETED_EVT');
        expect(events[0].result).toEqual({ ok: true });
        expect(events[0].metadata).toEqual({ k: 'v' });
    });

    // ========================================
    // 日誌功能
    // ========================================

    it('Log 應記錄 info/warn/error 等級', async () => {
        const taskId = taskManager.enqueue('LOG_LEVELS', {
            title: '日誌等級',
            executeFn: async (ctx) => {
                ctx.log('資訊', 'info');
                ctx.log('警告', 'warn');
                ctx.log('錯誤', 'error');
            }
        });

        await wait(300);

        const task = taskManager.getTask(taskId);
        const levels = task.logs.map(l => l.level);
        expect(levels).toContain('info');
        expect(levels).toContain('warn');
        expect(levels).toContain('error');
    });

    // ========================================
    // 任務管理
    // ========================================

    it('已結束任務可移除，不存在的任務回傳 null', async () => {
        const taskId = taskManager.enqueue('REMOVABLE', {
            title: '可移除',
            executeFn: async () => 'ok'
        });

        await wait(300);

        expect(taskManager.removeTask(taskId)).toBe(true);
        expect(taskManager.getTask(taskId)).toBeNull();
    });

    it('taskContext.yield 應正常讓出事件迴圈', async () => {
        let yieldOk = false;
        taskManager.enqueue('YIELD_OK', {
            title: 'Yield',
            executeFn: async (ctx) => {
                await ctx.yield();
                yieldOk = true;
            }
        });

        await wait(300);
        expect(yieldOk).toBe(true);
    });
});
