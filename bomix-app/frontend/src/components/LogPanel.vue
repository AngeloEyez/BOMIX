<template>
  <div class="log-panel">
    <!-- Panel Header -->
    <div class="panel-header">
      <div class="header-left">
        <div class="tabs-wrapper">
          <button
            v-for="(tab, index) in tabs"
            :key="tab.value"
            :class="['tab-button', { active: activeTab === index }]"
            @click="activeTab = index"
          >
            {{ tab.label }}
            <Badge
              v-if="tab.value !== 'ALL' && tab.count > 0"
              :value="tab.count"
              severity="secondary"
              class="tab-badge"
            />
          </button>
        </div>
      </div>
      <div class="header-right">
        <Button
          icon="pi pi-trash"
          text
          severity="secondary"
          @click="handleClearLogs"
          title="Clear Logs"
        />
      </div>
    </div>

    <!-- Log Content -->
    <div class="log-content">
      <div v-if="filteredLogs.length === 0" class="empty-logs">
        <i class="pi pi-inbox"></i>
        <span>No logs available</span>
      </div>
      <div v-else class="log-list">
        <div
          v-for="log in filteredLogs"
          :key="log.id || log.timestamp"
          class="log-item"
          :class="`log-level-${log.level.toLowerCase()}`"
        >
          <span class="log-time">{{ formatTime(log.timestamp) }}</span>
          <Badge
            :value="log.level"
            :severity="getLevelSeverity(log.level)"
            class="log-badge"
          />
          <span class="log-message">{{ log.message }}</span>
          <span
            v-if="log.attrs && Object.keys(log.attrs).length > 0"
            class="log-attrs"
          >
            {{ formatAttrs(log.attrs) }}
          </span>
        </div>
      </div>
    </div>

    <!-- Task Panel (if there are active tasks) -->
    <div v-if="activeTasks.length > 0" class="task-section">
      <div class="task-header">
        <span class="task-title">
          <i class="pi pi-cog"></i>
          Active Tasks
        </span>
      </div>
      <div class="task-list">
        <div
          v-for="task in activeTasks"
          :key="task.id"
          class="task-item"
          :class="`task-status-${task.status}`"
        >
          <div class="task-info">
            <span class="task-name">{{ task.name }}</span>
            <Badge
              :value="task.status"
              :severity="getTaskStatusSeverity(task.status)"
            />
          </div>
          <div class="task-progress">
            <ProgressBar
              :value="task.progress"
              :show-value="false"
              :class="`progress-${task.status}`"
            />
            <span class="task-message">{{ task.message || '' }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import Badge from 'primevue/badge'
import Button from 'primevue/button'
import ProgressBar from 'primevue/progressbar'
import { useLogStore, useTaskStore } from '../stores'
import type { LogEntry } from '../stores/log'

const logStore = useLogStore()
const taskStore = useTaskStore()

// Tab management
const activeTab = ref(0)

const tabs = computed(() => [
  { label: 'All', value: 'ALL', count: logStore.logs.length },
  { label: 'INFO', value: 'INFO', count: logStore.infoCount },
  { label: 'WARN', value: 'WARN', count: logStore.warnCount },
  { label: 'ERROR', value: 'ERROR', count: logStore.errorCount },
])

// Filtered logs based on active tab
const filteredLogs = computed(() => {
  const level = tabs.value[activeTab.value].value as 'ALL' | 'DEBUG' | 'INFO' | 'WARN' | 'ERROR'
  logStore.setFilterLevel(level)
  return logStore.filteredLogs
})

// Active tasks
const activeTasks = computed(() => taskStore.activeTasks)

// Format timestamp
function formatTime(timestamp: string): string {
  try {
    const date = new Date(timestamp)
    return date.toLocaleTimeString('en-US', {
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
  } catch {
    return timestamp
  }
}

// Format attributes
function formatAttrs(attrs: Record<string, string>): string {
  return Object.entries(attrs)
    .map(([key, value]) => `${key}: ${value}`)
    .join(', ')
}

// Get badge severity based on log level
function getLevelSeverity(level: string): 'success' | 'info' | 'warn' | 'danger' | 'secondary' | 'contrast' {
  switch (level) {
    case 'DEBUG':
      return 'secondary'
    case 'INFO':
      return 'success'
    case 'WARN':
      return 'warn'
    case 'ERROR':
      return 'danger'
    default:
      return 'secondary'
  }
}

// Get task status severity
function getTaskStatusSeverity(status: string): 'success' | 'info' | 'warn' | 'danger' | 'secondary' {
  switch (status) {
    case 'completed':
      return 'success'
    case 'running':
      return 'info'
    case 'queued':
      return 'secondary'
    case 'failed':
      return 'danger'
    case 'cancelled':
      return 'warn'
    default:
      return 'secondary'
  }
}

// Clear logs
function handleClearLogs(): void {
  logStore.clearLogs()
}
</script>

<style scoped>
.log-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--surface-ground);
}

/* Panel Header */
.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.5rem 1rem;
  border-bottom: 1px solid var(--surface-border);
  background: var(--surface-card);
}

.header-left {
  flex: 1;
}

.header-right {
  display: flex;
  gap: 0.5rem;
}

/* Custom Tabs */
.tabs-wrapper {
  display: flex;
  gap: 0.25rem;
}

.tab-button {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.375rem 0.75rem;
  border: none;
  background: transparent;
  color: var(--text-color-secondary);
  cursor: pointer;
  border-radius: 4px;
  font-size: 0.875rem;
  transition: background 0.2s;
}

.tab-button:hover {
  background: var(--surface-hover);
}

.tab-button.active {
  background: var(--primary-color);
  color: white;
}

.tab-badge {
  margin-left: 0.25rem;
}

/* Log Content */
.log-content {
  flex: 1;
  overflow-y: auto;
  padding: 0.5rem;
}

.empty-logs {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--text-color-secondary);
  gap: 0.5rem;
}

.empty-logs i {
  font-size: 2rem;
}

.log-list {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.log-item {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.875rem;
  line-height: 1.4;
}

.log-item:hover {
  background: var(--surface-hover);
}

.log-time {
  color: var(--text-color-secondary);
  font-family: monospace;
  min-width: 60px;
  flex-shrink: 0;
}

.log-badge {
  font-size: 0.75rem;
  padding: 0.125rem 0.375rem;
  flex-shrink: 0;
}

.log-message {
  flex: 1;
  word-break: break-word;
}

.log-attrs {
  color: var(--text-color-secondary);
  font-style: italic;
  font-size: 0.8rem;
}

/* Log level colors */
.log-level-debug {
  border-left: 3px solid #6c757d;
}

.log-level-info {
  border-left: 3px solid #4caf50;
}

.log-level-warn {
  border-left: 3px solid #ff9800;
}

.log-level-error {
  border-left: 3px solid #f44336;
}

/* Task Section */
.task-section {
  border-top: 1px solid var(--surface-border);
  background: var(--surface-card);
}

.task-header {
  padding: 0.5rem 1rem;
  font-weight: 600;
  font-size: 0.875rem;
  color: var(--text-color);
}

.task-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.task-list {
  display: flex;
  flex-direction: column;
  padding: 0.5rem;
  gap: 0.5rem;
}

.task-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  padding: 0.5rem;
  border-radius: 4px;
  background: var(--surface-ground);
}

.task-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.task-name {
  font-weight: 500;
  flex: 1;
}

.task-progress {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.task-progress :deep(.p-progressbar-value) {
  transition: width 0.3s ease;
}

.progress-running :deep(.p-progressbar-value) {
  background: linear-gradient(90deg, #2196f3, #64b5f6);
}

.progress-completed :deep(.p-progressbar-value) {
  background: #4caf50;
}

.progress-failed :deep(.p-progressbar-value) {
  background: #f44336;
}

.progress-cancelled :deep(.p-progressbar-value) {
  background: #9e9e9e;
}

.task-message {
  font-size: 0.75rem;
  color: var(--text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}
</style>
