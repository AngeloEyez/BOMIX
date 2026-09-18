<template>
  <div class="log-panel" ref="panelRef">
    <!--
      篩選列：當面板高度超過三行（且能容納工具列與至少三行日誌）時才顯示。
      高度拉開小於或等於三行時隱藏，騰出最大可視空間呈現最新日誌。
    -->
    <div v-show="showFilterToolbar" class="panel-header">
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
    </div>

    <!-- 日誌顯示區：flex-direction: column，新資訊在下 -->
    <div
      class="log-content"
      ref="logContentRef"
      :class="{ 'is-single-line': isSingleLine }"
      @contextmenu.stop.prevent="onPanelContextMenu"
    >
      <!--
        無日誌時：顯示單一 "Ready" 狀態文字（對齊底部，仿 VS Code 狀態列）。
      -->
      <div v-if="filteredLogs.length === 0" class="log-ready">
        Ready
      </div>

      <!-- 單行模式：顯示最新一條 log（完整支援時間、TASK 標籤、主次標題內容與細節按鈕） -->
      <div
        v-else-if="isSingleLine && latestLog"
        class="log-item log-single-line"
        :class="[`log-level-${latestLog.level.toLowerCase()}`, { 'log-item-task': latestLog.isTaskTracker }]"
        @click="latestLog.isTaskTracker ? openTaskDetails(latestLog) : null"
        @dblclick="latestLog.isTaskTracker ? openTaskDetails(latestLog) : null"
        @contextmenu.stop.prevent="onLogItemContextMenu($event, latestLog)"
        :style="latestLog.isTaskTracker ? 'cursor: pointer;' : ''"
        :title="latestLog.isTaskTracker ? '點擊檢視任務內部細節日誌' : ''"
      >
        <!-- 左側色條指示 log 等級 -->
        <span class="log-level-indicator"></span>

        <!-- Task Tracker 顯示模式 -->
        <template v-if="latestLog.isTaskTracker">
          <span class="log-time">{{ formatTime(latestLog.timestamp) }}</span>
          <span class="log-task-indicator">TASK</span>
          <span class="log-status-tag" :class="`status-${latestLog.status}`">
            <i v-if="latestLog.status?.toLowerCase() === 'running'" class="pi pi-spin pi-spinner mr-1 text-[8px]"></i>
            {{ latestLog.status }}
          </span>
          <span class="log-message">
            <span v-if="latestLog.attrs?.name" class="task-name-label">{{ latestLog.attrs.name }} - </span>
            {{ latestLog.message }}
          </span>
          <button
            type="button"
            class="task-detail-pill-btn"
            title="點開檢視內部細節日誌"
            @click.stop="openTaskDetails(latestLog)"
          >
            <i class="pi pi-list text-[10px]"></i>
            <span>細節</span>
          </button>
        </template>

        <!-- 一般 Log 顯示模式 -->
        <template v-else>
          <span class="log-time">{{ formatTime(latestLog.timestamp) }}</span>
          <span class="log-level-tag">{{ latestLog.level }}</span>
          <span class="log-message">{{ latestLog.message }}</span>
          <span v-if="latestLog.attrs && Object.keys(latestLog.attrs).length > 0" class="log-attrs">
            {{ formatAttrs(latestLog.attrs) }}
          </span>
        </template>
      </div>

      <!-- 多行模式：顯示完整日誌列表（新資訊在下方） -->
      <template v-else>
        <!-- 彈性頂部填充：當日誌行數未填滿容器時，自動吸收頂部空間將日誌推至底部對齊，確保最新日誌優先可見 -->
        <div class="log-content-spacer"></div>
        <div
          v-for="log in filteredLogs"
          :key="log.id || log.timestamp"
          class="log-item"
          :class="[`log-level-${log.level.toLowerCase()}`, { 'log-item-task': log.isTaskTracker }]"
          @click="log.isTaskTracker ? openTaskDetails(log) : null"
          @dblclick="log.isTaskTracker ? openTaskDetails(log) : null"
          @contextmenu.stop.prevent="onLogItemContextMenu($event, log)"
          :style="log.isTaskTracker ? 'cursor: pointer;' : ''"
          :title="log.isTaskTracker ? '點擊檢視任務內部細節日誌' : ''"
        >
          <!-- 左側色條指示 log 等級 -->
          <span class="log-level-indicator"></span>
          
          <!-- Task Tracker 顯示模式 -->
          <template v-if="log.isTaskTracker">
            <span class="log-time">{{ formatTime(log.timestamp) }}</span>
            <span class="log-task-indicator">TASK</span>
            <span class="log-status-tag" :class="`status-${log.status}`">
              <i v-if="log.status?.toLowerCase() === 'running'" class="pi pi-spin pi-spinner mr-1 text-[8px]"></i>
              {{ log.status }}
            </span>
            <span class="log-message">
              <span v-if="log.attrs?.name" class="task-name-label">{{ log.attrs.name }} - </span>
              {{ log.message }}
            </span>
            <button
              type="button"
              class="task-detail-pill-btn"
              title="點開檢視內部細節日誌"
              @click.stop="openTaskDetails(log)"
            >
              <i class="pi pi-list text-[10px]"></i>
              <span>細節</span>
            </button>
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
          <span class="task-message">{{ task.message || '' }}</span>
        </div>
      </div>
    </div>

    <!-- 右鍵選單元件 -->
    <ContextMenu ref="contextMenuRef" :model="contextMenuItems" />

    <!-- 任務日誌詳細彈窗 -->
    <Dialog
      v-model:visible="taskDialogVisible"
      modal
      resizable
      maximizable
      :header="selectedTaskTracker?.attrs?.name ? `Task Details: ${selectedTaskTracker.attrs.name}` : 'Task Details'"
      :style="{ width: '800px', height: '600px', minWidth: '400px', minHeight: '300px' }"
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
        <div class="task-log-content" @contextmenu.stop.prevent="onPanelContextMenu">
          <div
            v-for="log in filteredTaskHistory"
            :key="log.id"
            class="log-item"
            :class="`log-level-${log.level.toLowerCase()}`"
            @contextmenu.stop.prevent="onLogItemContextMenu($event, log)"
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
import ContextMenu from 'primevue/contextmenu'
import type { MenuItem } from 'primevue/menuitem'
import { useLogStore, useTaskStore } from '../stores'
import type { LogEntry } from '../stores/log'

const logStore = useLogStore()
const taskStore = useTaskStore()

/** ContextMenu 元件 ref */
const contextMenuRef = ref()
/** 當前右鍵點擊選中的日誌項目 */
const selectedLogEntry = ref<LogEntry | null>(null)

/** 面板根元素 ref，用於 ResizeObserver 偵測高度 */
const panelRef = ref<HTMLElement | null>(null)
/** 日誌內容容器 ref，用於自動捲動到底部 */
const logContentRef = ref<HTMLElement | null>(null)

/** 目前面板高度（px），由 ResizeObserver 動態更新 */
const panelHeight = ref(0)

/** 單條日誌項目固定行高 (px) */
const LOG_LINE_HEIGHT = 18
/** 篩選工具列固定高度 (px，含 1px 下框線) */
const TOOLBAR_HEIGHT = 23
/**
 * 單行模式高度閾值 (px)。
 * 面板高度在此數值以內時切換為單行極簡模式（僅顯示最新一條日誌）。
 */
const SINGLE_LINE_THRESHOLD = 28
/**
 * 篩選工具列顯示閾值 (px)。
 * 超過三行日誌高度（且能容納篩選列與至少三行日誌）時才顯示。
 * 23px (篩選列) + 3 * 18px (三行日誌) = 77px。
 */
const FILTER_TOOLBAR_THRESHOLD = TOOLBAR_HEIGHT + 3 * LOG_LINE_HEIGHT

/** 是否處於單行模式（面板高度縮至僅能顯示單行） */
const isSingleLine = computed(() =>
  panelHeight.value > 0 && panelHeight.value <= SINGLE_LINE_THRESHOLD
)

/** 是否顯示篩選工具列（拉開高度超過三行後才顯示） */
const showFilterToolbar = computed(() =>
  panelHeight.value > FILTER_TOOLBAR_THRESHOLD
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

/**
 * 活躍任務列表
 * 注意：排除 AIChat 任務，取消 AI 對話進行中於日誌面板覆蓋之進度條
 */
const activeTasks = computed(() =>
  taskStore.activeTasks.filter(task => task.type !== 'AIChat')
)

// ── 自動捲動到底部 ─────────────────────────────────────
/**
 * 將日誌內容區域平滑捲動至最底部，確保最新資訊永遠優先可見
 */
function scrollToBottom(): void {
  nextTick(() => {
    if (logContentRef.value) {
      logContentRef.value.scrollTop = logContentRef.value.scrollHeight
    }
  })
}

/**
 * 監聽 filteredLogs 長度變化，新增日誌時自動捲動到最底部
 */
watch(
  () => filteredLogs.value.length,
  () => {
    scrollToBottom()
  }
)

/**
 * 監聽面板高度變更（拉伸調整高度）時自動重新定位至最底部，確保最新日誌始終優先可見
 */
watch(panelHeight, () => {
  scrollToBottom()
})

/**
 * 監聽分類標籤切換時自動滾動至最底部
 */
watch(activeTab, () => {
  scrollToBottom()
})

// ── ResizeObserver 偵測面板高度 ────────────────────────
let resizeObserver: ResizeObserver | null = null

onMounted(() => {
  if (panelRef.value) {
    // 使用 ResizeObserver 監聽面板大小變化，即時同步滾動至底部保持最新日誌可見
    resizeObserver = new ResizeObserver((entries) => {
      for (const entry of entries) {
        panelHeight.value = entry.contentRect.height
      }
      scrollToBottom()
    })
    resizeObserver.observe(panelRef.value)
    // 記錄初始高度並滾動至底部
    panelHeight.value = panelRef.value.clientHeight
    scrollToBottom()
  }
})

onUnmounted(() => {
  // 元件銷毀時停止觀察，避免記憶體洩漏
  resizeObserver?.disconnect()
})

// ── 工具函式 ───────────────────────────────────────────

/**
 * 格式化時間戳記。
 * 當設定選項或篩選層級為 DEBUG 時，顯示到毫秒 (MM-DD hh:mm:ss.SSS)；否則僅顯示到秒 (MM-DD hh:mm:ss)。
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

    const isDebug = logStore.globalLogLevel.toUpperCase() === 'DEBUG' || logStore.filterLevel === 'DEBUG'
    if (isDebug) {
      const ms = String(date.getMilliseconds()).padStart(3, '0')
      return `${MM}-${DD} ${hh}:${mm}:${ss}.${ms}`
    }

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

// ── 右鍵選單操作函式 ───────────────────────────────────────

/**
 * 複製純文字至系統剪貼簿
 * @param text - 要複製的字串內容
 */
async function copyText(text?: string): Promise<void> {
  if (!text) return
  try {
    await navigator.clipboard.writeText(text)
  } catch (err) {
    console.error('Failed to copy text to clipboard:', err)
  }
}

/**
 * 複製單條日誌的完整內容（含時間、層級與屬性）
 * @param log - 日誌物件
 */
function copyFullLog(log?: LogEntry | null): void {
  if (!log) return
  const time = formatTime(log.timestamp)
  let line = `[${time}] [${log.level}] ${log.message}`
  if (log.attrs && Object.keys(log.attrs).length > 0) {
    line += ` | ${formatAttrs(log.attrs)}`
  }
  copyText(line)
}

/**
 * 複製當前篩選視圖下的所有日誌
 */
function copyAllLogs(): void {
  const lines = filteredLogs.value.map(log => {
    const time = formatTime(log.timestamp)
    let line = `[${time}] [${log.level}] ${log.message}`
    if (log.attrs && Object.keys(log.attrs).length > 0) {
      line += ` | ${formatAttrs(log.attrs)}`
    }
    return line
  })
  copyText(lines.join('\n'))
}

/**
 * 處理日誌條目右鍵事件，彈出自定義選單
 * @param event - 原生滑鼠事件
 * @param log - 當前右鍵點擊的日誌條目
 */
function onLogItemContextMenu(event: MouseEvent, log: LogEntry): void {
  selectedLogEntry.value = log
  contextMenuRef.value?.show(event)
}

/**
 * 處理日誌面板空白區域右鍵事件
 * @param event - 原生滑鼠事件
 */
function onPanelContextMenu(event: MouseEvent): void {
  selectedLogEntry.value = null
  contextMenuRef.value?.show(event)
}

/**
 * 動態計算右鍵選單項目
 */
const contextMenuItems = computed(() => {
  if (selectedLogEntry.value) {
    const items: MenuItem[] = []
    if (selectedLogEntry.value.isTaskTracker) {
      items.push({
        label: '查看任務細節日誌 (View Task Details)',
        icon: 'pi pi-external-link',
        command: () => openTaskDetails(selectedLogEntry.value!)
      })
      items.push({ separator: true })
    }
    items.push(
      {
        label: '複製訊息 (Copy Message)',
        icon: 'pi pi-copy',
        command: () => copyText(selectedLogEntry.value?.message)
      },
      {
        label: '複製整條日誌 (Copy Full Log)',
        icon: 'pi pi-file',
        command: () => copyFullLog(selectedLogEntry.value)
      },
      { separator: true },
      {
        label: '複製所有日誌 (Copy All Logs)',
        icon: 'pi pi-clone',
        disabled: filteredLogs.value.length === 0,
        command: copyAllLogs
      },
      {
        label: '清除日誌 (Clear Logs)',
        icon: 'pi pi-trash',
        disabled: logStore.logs.length === 0,
        command: handleClearLogs
      }
    )
    return items
  }

  return [
    {
      label: '複製所有日誌 (Copy All Logs)',
      icon: 'pi pi-clone',
      disabled: filteredLogs.value.length === 0,
      command: copyAllLogs
    },
    {
      label: '清除日誌 (Clear Logs)',
      icon: 'pi pi-trash',
      disabled: logStore.logs.length === 0,
      command: handleClearLogs
    }
  ]
})

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

/* 單行模式隱藏垂直捲軸並垂直置中，避免縮放小數點誤差導致短暫出現捲軸 */
.log-content.is-single-line {
  overflow-y: hidden;
  justify-content: center;
}

.log-content.is-single-line .log-single-line {
  margin-top: 0;
}

/* 彈性頂部間隔：日誌未填滿時自動推到底部，填滿溢出時收縮為 0px 支援正常捲動 */
.log-content-spacer {
  margin-top: auto;
  flex-shrink: 0;
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
  min-width: 0;
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
  font-weight: 600;
  letter-spacing: 0.03em;
  padding: 0 3px;
  border-radius: 2px;
  color: #2196f3;
  background-color: rgba(33,150,243,0.12);
  margin-right: 0.5rem;
  text-transform: uppercase;
  line-height: 13px;
  height: 13px;
  display: flex;
  align-items: center;
}

.log-status-tag {
  font-size: 9px;
  font-weight: 600;
  letter-spacing: 0.03em;
  padding: 0 4px;
  border-radius: 2px;
  background-color: var(--surface-hover);
  color: var(--text-color-secondary);
  margin-right: 0.5rem;
  text-transform: uppercase;
  line-height: 13px;
  height: 13px;
  display: inline-flex;
  align-items: center;
  gap: 2px;
}

.log-status-tag.status-error {
  color: #f44336;
  background-color: rgba(244,67,54,0.12);
}

.log-status-tag.status-warning,
.log-status-tag.status-warn {
  color: #ff9800;
  background-color: rgba(255,152,0,0.12);
}

.log-status-tag.status-running {
  color: #2196f3;
  background-color: rgba(33,150,243,0.12);
}

.log-status-tag.status-done {
  color: #4caf50;
  background-color: rgba(76,175,80,0.12);
}

.log-status-tag.status-cancelled {
  color: #9e9e9e;
  background-color: rgba(158,158,158,0.12);
}

.log-item-task {
  transition: background-color 0.15s ease;
}

.log-item-task:hover {
  background-color: var(--surface-hover);
}

.task-detail-pill-btn {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  margin-left: 8px;
  padding: 1px 6px;
  font-size: 10px;
  font-weight: 500;
  border-radius: 9999px;
  border: 1px solid var(--surface-border);
  background-color: var(--surface-card);
  color: var(--text-color-secondary);
  cursor: pointer;
  transition: all 0.15s ease;
  vertical-align: middle;
  flex-shrink: 0;
}

.task-detail-pill-btn:hover {
  background-color: var(--primary-color, #3b82f6);
  color: #ffffff;
  border-color: var(--primary-color, #3b82f6);
}

.task-name-label {
  font-weight: 600;
  color: var(--text-color);
}

/* Task Dialog Styles */
.task-log-dialog .p-dialog-content {
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

.task-log-content {
  flex: 1;
  overflow-x: auto;
  overflow-y: auto;
  padding: 0.5rem;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 0.85rem;
}

.task-log-content .log-item {
  white-space: nowrap;
  min-width: max-content;
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
