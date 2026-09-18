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
      <!-- 無日誌時：顯示單一 "Ready" 狀態文字（對齊底部，仿 VS Code 狀態列） -->
      <div v-if="filteredLogs.length === 0" class="log-ready">
        Ready
      </div>

      <!-- 單行模式：顯示最新一條 log -->
      <LogItem
        v-else-if="isSingleLine && latestLog"
        :log="latestLog"
        :is-single-line="true"
        @open-details="openTaskDetails"
        @contextmenu="onLogItemContextMenu"
      />

      <!-- 多行模式：顯示完整日誌列表（新資訊在下方） -->
      <template v-else>
        <!-- 彈性頂部填充：當日誌行數未填滿容器時，自動吸收頂部空間將日誌推至底部對齊，確保最新日誌優先可見 -->
        <div class="log-content-spacer"></div>
        <LogItem
          v-for="log in filteredLogs"
          :key="log.id || log.timestamp"
          :log="log"
          @open-details="openTaskDetails"
          @contextmenu="onLogItemContextMenu"
        />
      </template>
    </div>

    <!-- 任務區域：僅在非單行模式且有活躍任務時顯示 -->
    <LogActiveTasks
      v-if="!isSingleLine && activeTasks.length > 0"
      :tasks="activeTasks"
    />

    <!-- 右鍵選單元件 -->
    <ContextMenu ref="contextMenuRef" :model="contextMenuItems" />

    <!-- 任務日誌詳細彈窗 -->
    <TaskLogDialog
      v-model:visible="taskDialogVisible"
      :tracker="selectedTaskTracker"
      @contextmenu-item="onLogItemContextMenu"
      @contextmenu-panel="onPanelContextMenu"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import ContextMenu from 'primevue/contextmenu'
import type { MenuItem } from 'primevue/menuitem'
import { useLogStore, useTaskStore } from '../stores'
import type { LogEntry } from '../stores/log'
import LogItem from './log/LogItem.vue'
import TaskLogDialog from './log/TaskLogDialog.vue'
import LogActiveTasks from './log/LogActiveTasks.vue'

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
 * 格式化時間戳記（供右鍵選單複製功能使用）。
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
 * 格式化 log 附加屬性為可讀字串（供右鍵選單複製功能使用）。
 * @param attrs - key-value 屬性對
 * @returns 格式化後的屬性字串
 */
function formatAttrs(attrs: Record<string, string>): string {
  return Object.entries(attrs)
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

/**
 * 開啟任務詳細日誌彈窗
 * @param tracker - 選取的任務 Tracker 日誌
 */
function openTaskDetails(tracker: LogEntry): void {
  selectedTaskTracker.value = tracker
  taskDialogVisible.value = true
}
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
  background: var(--primary-color);
  color: #fff;
}

/* 計數徽章 */
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
  scrollbar-width: thin;
  scrollbar-color: var(--surface-border) transparent;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

/* 單行模式隱藏垂直捲軸並垂直置中 */
.log-content.is-single-line {
  overflow-y: hidden;
  justify-content: center;
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
  margin-top: auto;
}
</style>
