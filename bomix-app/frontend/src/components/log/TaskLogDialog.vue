<template>
  <Dialog
    :visible="visible"
    modal
    resizable
    maximizable
    :header="dialogHeader"
    :style="{ width: '800px', height: '600px', minWidth: '400px', minHeight: '300px' }"
    class="task-log-dialog"
    @update:visible="(val: boolean) => emit('update:visible', val)"
  >
    <div class="task-log-container">
      <!-- 彈窗內的日誌過濾器 -->
      <div class="task-log-filters">
        <button
          v-for="tab in taskDetailTabs"
          :key="tab.value"
          :class="['tab-button', { active: taskDetailActiveTab === tab.value }]"
          @click="taskDetailActiveTab = tab.value"
        >
          {{ tab.label }}
          <span v-if="tab.value !== 'ALL' && getTaskLogCount(tab.value) > 0" class="tab-count">
            {{ getTaskLogCount(tab.value) }}
          </span>
        </button>
      </div>

      <!-- 彈窗內的日誌列表 -->
      <div class="task-log-content" @contextmenu.stop.prevent="emit('contextmenuPanel', $event)">
        <LogItem
          v-for="log in filteredTaskHistory"
          :key="log.id || log.timestamp"
          :log="log"
          :ignore-attrs="['taskID', 'name']"
          @contextmenu="(e, l) => emit('contextmenuItem', e, l)"
        />
        <div v-if="filteredTaskHistory.length === 0" class="log-ready">
          No logs to display for this filter.
        </div>
      </div>
    </div>
  </Dialog>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import Dialog from 'primevue/dialog'
import type { LogEntry } from '../../stores/log'
import LogItem from './LogItem.vue'

interface Props {
  /** 彈窗是否可見 */
  visible: boolean
  /** 當前選取的任務 Tracker 日誌 */
  tracker: LogEntry | null
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'contextmenuItem', event: MouseEvent, log: LogEntry): void
  (e: 'contextmenuPanel', event: MouseEvent): void
}>()

const taskDetailActiveTab = ref('ALL')
const taskDetailTabs = [
  { label: 'All',   value: 'ALL' },
  { label: 'Debug', value: 'DEBUG' },
  { label: 'Info',  value: 'INFO' },
  { label: 'Warn',  value: 'WARN' },
  { label: 'Error', value: 'ERROR' },
]

/** 彈窗標題 */
const dialogHeader = computed(() => {
  return props.tracker?.attrs?.name
    ? `Task Details: ${props.tracker.attrs.name}`
    : 'Task Details'
})

/** 計算指定等級之任務日誌數量 */
function getTaskLogCount(level: string): number {
  if (!props.tracker?.history) return 0
  return props.tracker.history.filter(log => log.level === level).length
}

/** 依選取標籤過濾後的任務歷史日誌清單 */
const filteredTaskHistory = computed(() => {
  if (!props.tracker?.history) return []
  if (taskDetailActiveTab.value === 'ALL') {
    return props.tracker.history
  }
  return props.tracker.history.filter(log => log.level === taskDetailActiveTab.value)
})
</script>

<style scoped>
/* ── Task Dialog Styles ─────────────────────────────────── */
:deep(.p-dialog-content) {
  padding: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  height: 100%;
}

.task-log-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  width: 100%;
  background-color: var(--surface-card);
  overflow: hidden;
}

.task-log-filters {
  display: flex;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background-color: var(--surface-section);
  border-bottom: 1px solid var(--surface-border);
  flex-shrink: 0;
}

.tab-button {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 2px 8px;
  border: none;
  background: transparent;
  color: var(--text-color-secondary);
  cursor: pointer;
  border-radius: 4px;
  font-size: 11px;
  font-family: inherit;
  white-space: nowrap;
  transition: background 0.15s, color 0.15s;
  height: 22px;
}

.tab-button:hover {
  background: var(--surface-hover);
  color: var(--text-color);
}

.tab-button.active {
  background: var(--primary-color);
  color: #fff;
}

.tab-count {
  font-size: 9px;
  background: var(--surface-hover);
  color: var(--text-color-secondary);
  border-radius: 8px;
  padding: 0 4px;
  min-width: 14px;
  text-align: center;
}

.tab-button.active .tab-count {
  background: rgba(255, 255, 255, 0.25);
  color: #fff;
}

.task-log-content {
  flex: 1;
  overflow-x: auto;
  overflow-y: auto;
  padding: 0.5rem;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 0.85rem;
  scrollbar-width: thin;
}

.task-log-content :deep(.log-item) {
  white-space: nowrap;
  min-width: max-content;
}

.log-ready {
  display: flex;
  align-items: center;
  padding: 8px;
  color: var(--text-color-secondary);
  font-size: 11px;
  font-style: italic;
}
</style>
