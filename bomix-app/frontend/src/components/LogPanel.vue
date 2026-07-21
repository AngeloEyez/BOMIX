<template>
  <div class="log-panel" ref="panelRef">
    <!--
      篩選列：當面板高度足夠時顯示（非單行模式）。
      單行模式下隱藏，讓最新一條 log 佔滿整個面板高度。
    -->
    <div v-show="!isSingleLine" class="panel-header">
      <div class="tabs-wrapper">
        <button
          v-for="(tab, index) in tabs"
          :key="tab.value"
          :class="['tab-button', { active: activeTab === index }]"
          @click="activeTab = index"
        >
          {{ tab.label }}
          <!-- 非 All 且有計數時顯示計數徽章 -->
          <span
            v-if="tab.value !== 'ALL' && tab.count > 0"
            class="tab-count"
          >{{ tab.count }}</span>
        </button>
      </div>
      <div class="header-actions">
        <!-- 清除日誌按鈕 -->
        <button
          class="action-btn"
          title="Clear Logs"
          @click="handleClearLogs"
        >
          <i class="pi pi-trash"></i>
        </button>
      </div>
    </div>

    <!-- 日誌顯示區：flex-direction: column，新資訊在下 -->
    <div class="log-content" ref="logContentRef">
      <!--
        無日誌時：顯示單一 "Ready" 狀態文字（對齊底部，仿 VS Code 狀態列）。
      -->
      <div v-if="filteredLogs.length === 0" class="log-ready">
        Ready
      </div>

      <!-- 單行模式：只顯示最新一條 log（推至底部），不顯示時間戳記 -->
      <div
        v-else-if="isSingleLine && latestLog"
        class="log-item log-single-line"
        :class="`log-level-${latestLog.level.toLowerCase()}`"
      >
        <span class="log-level-indicator"></span>
        <span class="log-level-tag">{{ latestLog.level }}</span>
        <span class="log-message">{{ latestLog.message }}</span>
      </div>

      <!-- 多行模式：顯示完整日誌列表（新資訊在下方） -->
      <template v-else>
        <div
          v-for="log in filteredLogs"
          :key="log.id || log.timestamp"
          class="log-item"
          :class="`log-level-${log.level.toLowerCase()}`"
          @dblclick="log.isTaskTracker ? openTaskDetails(log) : null"
          :style="log.isTaskTracker ? 'cursor: pointer;' : ''"
          :title="log.isTaskTracker ? 'Double click to view task details' : ''"
        >
          <!-- 左側色條指示 log 等級 -->
          <span class="log-level-indicator"></span>
          
          <!-- Task Tracker 顯示模式 -->
          <template v-if="log.isTaskTracker">
            <span class="log-time">{{ formatTime(log.timestamp) }}</span>
            <span class="log-task-indicator">TASK</span>
            <span class="log-status-tag" :class="`status-${log.status}`">{{ log.status }}</span>
            <span class="log-message">
              <span v-if="log.attrs?.name" class="task-name-label">{{ log.attrs.name }} - </span>
              {{ log.message }}
            </span>
          </template>
          
          <!-- 一般 Log 顯示模式 -->
          <template v-else>
            <span class="log-time">{{ formatTime(log.timestamp) }}</span>
            <span class="log-level-tag">{{ log.level }}</span>
            <span class="log-message">{{ log.message }}</span>
            <span v-if="log.attrs && Object.keys(log.attrs).length > 0" class="log-attrs">
              {{ formatAttrs(log.attrs) }}
            </span>
          </template>
        </div>
      </template>
    </div>

    <!-- 任務區域：僅在非單行模式且有活躍任務時顯示 -->
    <div v-if="!isSingleLine && activeTasks.length > 0" class="task-section">
      <div class="task-list">
        <div
          v-for="task in activeTasks"
          :key="task.id"
          class="task-item"
          :class="`task-status-${task.status}`"
        >
          <span class="task-name">{{ task.name }}</span>
          <span class="task-status-tag">{{ task.status }}</span>
          <!-- 極細進度條 -->
          <div class="task-progress-bar">
            <div
              class="task-progress-fill"
              :class="`progress-${task.status}`"
              :style="{ width: `${task.progress}%` }"
            ></div>
          </div>
          <span class="task-message">{{ task.message || '' }}</span>
        </div>
      </div>
    </div>

    <!-- 任務日誌詳細彈窗 -->
    <Dialog
      v-model:visible="taskDialogVisible"
      modal
      :header="selectedTaskTracker?.attrs?.name ? `Task Details: ${selectedTaskTracker.attrs.name}` : 'Task Details'"
      :style="{ width: '800px', height: '600px' }"
      class="task-log-dialog"
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
        <div class="task-log-content">
          <div
            v-for="log in filteredTaskHistory"
            :key="log.id"
            class="log-item"
            :class="`log-level-${log.level.toLowerCase()}`"
          >
            <span class="log-level-indicator"></span>
            <span class="log-time">{{ formatTime(log.timestamp) }}</span>
            <span class="log-level-tag">{{ log.level }}</span>
            <span class="log-message">{{ log.message }}</span>
            <span v-if="log.attrs && Object.keys(log.attrs).filter(k => k !== 'taskID' && k !== 'name').length > 0" class="log-attrs">
              {{ formatAttrs(log.attrs, ['taskID', 'name']) }}
            </span>
          </div>
          <div v-if="filteredTaskHistory.length === 0" class="log-ready">
            No logs to display for this filter.
          </div>
        </div>
      </div>
    </Dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import Dialog from 'primevue/dialog'
import { useLogStore, useTaskStore } from '../stores'
import type { LogEntry } from '../stores/log'

const logStore = useLogStore()
const taskStore = useTaskStore()

/** 面板根元素 ref，用於 ResizeObserver 偵測高度 */
const panelRef = ref<HTMLElement | null>(null)
/** 日誌內容容器 ref，用於自動捲動到底部 */
const logContentRef = ref<HTMLElement | null>(null)

/** 目前面板高度（px），由 ResizeObserver 動態更新 */
const panelHeight = ref(0)

/**
 * 單行模式閾值（px）。
 * 篩選列高度約 22px，單條 log 行高約 18px。
 * 當面板總高度縮至此值以下時，切換為單行模式。
 */
const SINGLE_LINE_THRESHOLD = 44

/** 是否處於單行模式（面板縮至最小） */
const isSingleLine = computed(() =>
  panelHeight.value > 0 && panelHeight.value <= SINGLE_LINE_THRESHOLD
)

// ── 標籤頁管理 ─────────────────────────────────────────
const activeTab = ref(0)

const tabs = computed(() => {
  const level = logStore.globalLogLevel.toUpperCase()
  
  const allTabs = [
    { label: 'All',   value: 'ALL',   count: logStore.logs.filter(log => logStore.filteredLogs.includes(log)).length },
  ]
  
  if (level === 'DEBUG') {
    allTabs.push({ label: 'Debug', value: 'DEBUG', count: logStore.debugCount })
  }
  
  if (level === 'DEBUG' || level === 'INFO') {
    allTabs.push({ label: 'Info',  value: 'INFO',  count: logStore.infoCount })
  }
  
  if (level === 'DEBUG' || level === 'INFO' || level === 'WARN') {
    allTabs.push({ label: 'Warn',  value: 'WARN',  count: logStore.warnCount })
  }
  
  allTabs.push({ label: 'Error', value: 'ERROR', count: logStore.errorCount })
  
  return allTabs
})

/**
 * 根據目前標籤篩選後的日誌列表。
 * 同時觸發 store 的 setFilterLevel，讓 filteredLogs 計算屬性生效。
 */
const filteredLogs = computed(() => {
  const level = tabs.value[activeTab.value].value as 'ALL' | 'DEBUG' | 'INFO' | 'WARN' | 'ERROR'
  logStore.setFilterLevel(level)
  return logStore.filteredLogs
})

/** 最新一條日誌（用於單行模式顯示最新訊息） */
const latestLog = computed<LogEntry | null>(() =>
  filteredLogs.value.length > 0
    ? filteredLogs.value[filteredLogs.value.length - 1]
    : null
)

/** 活躍任務列表 */
const activeTasks = computed(() => taskStore.activeTasks)

// ── 自動捲動到底部 ─────────────────────────────────────
/**
 * 監聽 filteredLogs 長度變化，新增日誌時自動捲動到最底部，
 * 確保最新資訊永遠可見。
 */
watch(
  () => filteredLogs.value.length,
  async () => {
    await nextTick()
    if (logContentRef.value) {
      logContentRef.value.scrollTop = logContentRef.value.scrollHeight
    }
  }
)

// ── ResizeObserver 偵測面板高度 ────────────────────────
let resizeObserver: ResizeObserver | null = null

onMounted(() => {
  if (panelRef.value) {
    // 使用 ResizeObserver 監聽面板大小變化
    resizeObserver = new ResizeObserver((entries) => {
      for (const entry of entries) {
        panelHeight.value = entry.contentRect.height
      }
    })
    resizeObserver.observe(panelRef.value)
    // 記錄初始高度
    panelHeight.value = panelRef.value.clientHeight
  }
})

onUnmounted(() => {
  // 元件銷毀時停止觀察，避免記憶體洩漏
  resizeObserver?.disconnect()
})

// ── 工具函式 ───────────────────────────────────────────

/**
 * 格式化時間戳記為 MM-DD hh:mm:ss 格式。
 * @param timestamp - ISO 8601 格式的時間字串
 * @returns 格式化後的時間字串
 */
function formatTime(timestamp: string): string {
  try {
    const date = new Date(timestamp)
    const MM = String(date.getMonth() + 1).padStart(2, '0')
    const DD = String(date.getDate()).padStart(2, '0')
    const hh = String(date.getHours()).padStart(2, '0')
    const mm = String(date.getMinutes()).padStart(2, '0')
    const ss = String(date.getSeconds()).padStart(2, '0')
    return `${MM}-${DD} ${hh}:${mm}:${ss}`
  } catch {
    return timestamp
  }
}

/**
 * 格式化 log 附加屬性為可讀字串，可選忽略特定 key。
 * @param attrs - key-value 屬性對
 * @param ignoreKeys - 要忽略不顯示的 key 陣列
 * @returns 格式化後的屬性字串
 */
function formatAttrs(attrs: Record<string, string>, ignoreKeys: string[] = []): string {
  return Object.entries(attrs)
    .filter(([key]) => !ignoreKeys.includes(key))
    .map(([key, value]) => `${key}: ${value}`)
    .join('  ')
}

/** 清除所有日誌記錄 */
function handleClearLogs(): void {
  logStore.clearLogs()
}

// ── 任務彈窗邏輯 ──────────────────────────────────────────
const taskDialogVisible = ref(false)
const selectedTaskTracker = ref<LogEntry | null>(null)
const taskDetailActiveTab = ref('ALL')
const taskDetailTabs = [
  { label: 'All',   value: 'ALL' },
  { label: 'Debug', value: 'DEBUG' },
  { label: 'Info',  value: 'INFO' },
  { label: 'Warn',  value: 'WARN' },
  { label: 'Error', value: 'ERROR' },
]

function openTaskDetails(tracker: LogEntry) {
  selectedTaskTracker.value = tracker
  taskDetailActiveTab.value = 'ALL'
  taskDialogVisible.value = true
}

function getTaskLogCount(level: string): number {
  if (!selectedTaskTracker.value?.history) return 0
  return selectedTaskTracker.value.history.filter(log => log.level === level).length
}

const filteredTaskHistory = computed(() => {
  if (!selectedTaskTracker.value?.history) return []
  if (taskDetailActiveTab.value === 'ALL') {
    return selectedTaskTracker.value.history
  }
  return selectedTaskTracker.value.history.filter(log => log.level === taskDetailActiveTab.value)
})
</script>

<style scoped>
/* ── 根容器 ────────────────────────────────────────────── */
.log-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--surface-ground);
  overflow: hidden;
  font-size: 11px;
  line-height: 1.4;
}

/* ── 篩選標籤列 ─────────────────────────────────────────── */
.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  /* 固定高度，與 sidebar-header 相同的緊湊感 */
  height: 22px;
  padding: 0 4px;
  border-bottom: 1px solid var(--surface-border);
  background: var(--surface-card);
  flex-shrink: 0;
  gap: 4px;
  overflow: hidden;
}

/* 標籤按鈕群組 */
.tabs-wrapper {
  display: flex;
  gap: 2px;
  align-items: center;
  overflow: hidden;
  flex: 1;
}

.tab-button {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 1px 6px;
  border: none;
  background: transparent;
  /* 與 sidebar-title 相同的次要文字顏色 */
  color: var(--text-color-secondary);
  cursor: pointer;
  border-radius: 3px;
  font-size: 11px;
  font-family: inherit;
  white-space: nowrap;
  transition: background 0.15s, color 0.15s;
  height: 18px;
}

.tab-button:hover {
  background: var(--surface-hover);
  color: var(--text-color);
}

.tab-button.active {
  /* 與主分隔器 gutter hover 使用相同的 primary-color */
  background: var(--primary-color);
  color: #fff;
}

/* 計數徽章：極小尺寸，不使用 PrimeVue Badge 避免過大 */
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

/* 操作按鈕區（清除等） */
.header-actions {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border: none;
  background: transparent;
  color: var(--text-color-secondary);
  cursor: pointer;
  border-radius: 3px;
  transition: background 0.15s, color 0.15s;
  padding: 0;
}

.action-btn:hover {
  background: var(--surface-hover);
  color: var(--text-color);
}

.action-btn i {
  font-size: 10px;
}

/* ── 日誌內容區 ─────────────────────────────────────────── */
.log-content {
  flex: 1;
  overflow-y: auto;
  /* 細化捲軸，與 VS Code 相近 */
  scrollbar-width: thin;
  scrollbar-color: var(--surface-border) transparent;
  /* 使日誌項目從上往下堆疊（新資訊在下） */
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.log-content::-webkit-scrollbar {
  width: 6px;
}
.log-content::-webkit-scrollbar-track {
  background: transparent;
}
.log-content::-webkit-scrollbar-thumb {
  background: var(--surface-border);
  border-radius: 3px;
}

/* "Ready" 空狀態：推到底部，以斜體小字顯示 */
.log-ready {
  display: flex;
  align-items: center;
  padding: 0 8px;
  height: 18px;
  color: var(--text-color-secondary);
  font-size: 11px;
  font-style: italic;
  /* auto margin 推至底部，確保視覺上對齊最後一行 */
  margin-top: auto;
}

/* ── 單條 Log 行 ─────────────────────────────────────────── */
.log-item {
  display: flex;
  align-items: center;
  gap: 5px;
  /* 固定行高，緊湊排列 */
  height: 18px;
  min-height: 18px;
  padding: 0 4px;
  font-size: 11px;
  font-family: 'Consolas', 'Cascadia Code', 'Fira Code', monospace;
  white-space: nowrap;
  overflow: hidden;
  flex-shrink: 0;
  transition: background 0.1s;
}

.log-item:hover {
  background: var(--surface-hover);
}

/* 單行模式：唯一的 log 行推至底部 */
.log-single-line {
  margin-top: auto;
  height: 100%;
  max-height: 18px;
}

/* 左側色條（替代原 border-left，更清晰） */
.log-level-indicator {
  width: 2px;
  height: 12px;
  border-radius: 1px;
  flex-shrink: 0;
}

.log-level-debug  .log-level-indicator { background: #6c757d; }
.log-level-info   .log-level-indicator { background: #4caf50; }
.log-level-warn   .log-level-indicator { background: #ff9800; }
.log-level-error  .log-level-indicator { background: #f44336; }

/* 時間戳記：等寬字體，淡色 */
.log-time {
  color: var(--text-color-secondary);
  font-size: 10px;
  flex-shrink: 0;
  min-width: 90px;
}

/* 等級標籤：取代 PrimeVue Badge，更小更緊湊 */
.log-level-tag {
  font-size: 9px;
  font-weight: 600;
  letter-spacing: 0.03em;
  padding: 0 3px;
  border-radius: 2px;
  flex-shrink: 0;
  line-height: 13px;
  height: 13px;
  display: flex;
  align-items: center;
}

.log-level-debug  .log-level-tag { color: #6c757d; background: rgba(108,117,125,0.12); }
.log-level-info   .log-level-tag { color: #4caf50; background: rgba(76,175,80,0.12);  }
.log-level-warn   .log-level-tag { color: #ff9800; background: rgba(255,152,0,0.12);  }
.log-level-error  .log-level-tag { color: #f44336; background: rgba(244,67,54,0.12);  }

/* 主訊息文字 */
.log-message {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-color);
}

/* 附加屬性：更淡色 */
.log-attrs {
  color: var(--text-color-secondary);
  font-size: 0.8em;
  margin-left: 0.5rem;
}

/* Task Tracker Styles */
.log-task-indicator {
  font-size: 9px;
  padding: 0 4px;
  border-radius: 2px;
  background-color: #2196f3; /* Literal blue color */
  color: white;
  margin-right: 0.5rem;
  text-transform: uppercase;
  font-weight: bold;
  line-height: 13px;
  height: 13px;
  display: flex;
  align-items: center;
}

.log-status-tag {
  font-size: 9px;
  font-weight: 600;
  letter-spacing: 0.03em;
  padding: 0 3px;
  border-radius: 2px;
  background-color: var(--surface-hover);
  color: var(--text-color-secondary);
  margin-right: 0.5rem;
  text-transform: uppercase;
  line-height: 13px;
  height: 13px;
  display: flex;
  align-items: center;
}

.log-status-tag.status-error {
  color: #f44336;
  background-color: rgba(244,67,54,0.12);
}

.log-status-tag.status-running {
  color: #2196f3;
  background-color: rgba(33,150,243,0.12);
}

.log-status-tag.status-done {
  color: #4caf50;
  background-color: rgba(76,175,80,0.12);
}

.task-name-label {
  font-weight: 600;
  color: var(--text-color);
}

/* Task Dialog Styles */
.task-log-dialog .p-dialog-content {
  padding: 0;
  overflow: hidden;
}

.task-log-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  background-color: var(--surface-card);
}

.task-log-filters {
  display: flex;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background-color: var(--surface-section);
  border-bottom: 1px solid var(--surface-border);
}

.task-log-content {
  flex: 1;
  overflow-y: auto;
  padding: 0.5rem;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 0.85rem;
}

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
