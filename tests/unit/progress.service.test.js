
import { describe, it, expect } from 'vitest';
import progressService, { TASK_STATUS } from '../../src/main/services/progress.service.js';

describe('ProgressService', () => {
    it('should create a task with PENDING status', () => {
        const taskId = progressService.createTask('TEST_TASK', { foo: 'bar' });
        const task = progressService.getTask(taskId);

        expect(task).toBeDefined();
        expect(task.id).toBe(taskId);
        expect(task.type).toBe('TEST_TASK');
        expect(task.status).toBe(TASK_STATUS.PENDING);
        expect(task.metadata).toEqual({ foo: 'bar' });
    });

    it('should update progress', () => {
        const taskId = progressService.createTask('TEST_TASK');

        progressService.updateProgress(taskId, 50, 'Halfway');
        let task = progressService.getTask(taskId);
        expect(task.progress).toBe(50);
        expect(task.message).toBe('Halfway');
        expect(task.status).toBe(TASK_STATUS.RUNNING);

        progressService.updateProgress(taskId, 150); // Should cap at 100
        task = progressService.getTask(taskId);
        expect(task.progress).toBe(100);
    });

    it('should complete task', () => {
        const taskId = progressService.createTask('TEST_TASK');
        progressService.completeTask(taskId, { success: true });

        const task = progressService.getTask(taskId);
        expect(task.status).toBe(TASK_STATUS.COMPLETED);
        expect(task.progress).toBe(100);
        expect(task.result).toEqual({ success: true });
    });

    it('should fail task', () => {
        const taskId = progressService.createTask('TEST_TASK');
        progressService.failTask(taskId, new Error('Something went wrong'));

        const task = progressService.getTask(taskId);
        expect(task.status).toBe(TASK_STATUS.FAILED);
        expect(task.error).toBe('Something went wrong');
    });

    it('should emit events', () => {
        return new Promise(resolve => {
            const taskId = progressService.createTask('EVENT_TEST');

            progressService.once('task:update', (task) => {
                expect(task.id).toBe(taskId);
                expect(task.progress).toBe(10);
                resolve();
            });

            progressService.updateProgress(taskId, 10);
        });
    });
});
