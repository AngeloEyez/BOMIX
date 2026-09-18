<template>
  <div class="task-section">
    <div class="task-list">
      <div
        v-for="task in tasks"
        :key="task.id"
        class="task-item"
        :class="`task-status-${task.status}`"
      >
        <span class="task-name" :title="task.name">{{ task.name }}</span>
        <span class="task-status-tag">
          <i v-if="task.status?.toLowerCase() === 'running'" class="pi pi-spin pi-spinner mr-1 text-[8px]"></i>
          {{ task.status }}
        </span>
        <!-- 極細進度條 -->
        <div class="task-progress-bar">
          <div
            class="task-progress-fill"
            :class="`progress-${task.status}`"
            :style="{ width: `${task.progress}%` }"
          ></div>
        </div>
        <span class="task-message" :title="task.message || ''">{{ task.message || '' }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Task } from '../../stores/task'

interface Props {
  /** 當前活躍任務列表 */
  tasks: Task[]
}

defineProps<Props>()
</script>

<style scoped>
/* ── 任務區域（緊湊版） ─────────────────────────────────── */
.task-section {
  border-top: 1px solid var(--surface-border);
  background: var(--surface-card);
  flex-shrink: 0;
}

.task-list {
  display: flex;
  flex-direction: column;
  padding: 2px 4px;
  gap: 2px;
}

.task-item {
  display: flex;
  align-items: center;
  gap: 5px;
  height: 18px;
  font-size: 11px;
  font-family: inherit;
}

.task-name {
  flex-shrink: 0;
  color: var(--text-color);
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-status-tag {
  font-size: 9px;
  color: var(--text-color-secondary);
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  gap: 2px;
}

/* 進度條：極細（3px 高） */
.task-progress-bar {
  flex: 1;
  height: 3px;
  background: var(--surface-hover);
  border-radius: 2px;
  overflow: hidden;
}

.task-progress-fill {
  height: 100%;
  border-radius: 2px;
  transition: width 0.3s ease;
}

.progress-running   { background: linear-gradient(90deg, #2196f3, #64b5f6); }
.progress-completed { background: #4caf50; }
.progress-failed    { background: #f44336; }
.progress-cancelled { background: #9e9e9e; }
.progress-queued    { background: var(--surface-border); }

.task-message {
  font-size: 10px;
  color: var(--text-color-secondary);
  flex-shrink: 0;
  max-width: 160px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
