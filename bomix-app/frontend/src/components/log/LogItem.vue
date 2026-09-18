<template>
  <div
    class="log-item"
    :class="[
      `log-level-${log.level.toLowerCase()}`,
      {
        'log-item-task': log.isTaskTracker,
        'log-single-line': isSingleLine,
      }
    ]"
    :style="log.isTaskTracker ? 'cursor: pointer;' : ''"
    :title="log.isTaskTracker ? '點擊檢視任務內部細節日誌' : ''"
    @click="log.isTaskTracker ? emit('openDetails', log) : emit('click', log)"
    @dblclick="log.isTaskTracker ? emit('openDetails', log) : emit('dblclick', log)"
    @contextmenu.stop.prevent="emit('contextmenu', $event, log)"
  >
    <!-- 左側色條指示 log 等級 -->
    <span class="log-level-indicator"></span>

    <!-- Task Tracker 顯示模式 -->
    <template v-if="log.isTaskTracker">
      <span class="log-time">{{ formattedTime }}</span>
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
        @click.stop="emit('openDetails', log)"
      >
        <i class="pi pi-list text-[10px]"></i>
        <span>細節</span>
      </button>
    </template>

    <!-- 一般 Log 顯示模式 -->
    <template v-else>
      <span class="log-time">{{ formattedTime }}</span>
      <span class="log-level-tag">{{ log.level }}</span>
      <span class="log-message">{{ log.message }}</span>
      <span v-if="hasVisibleAttrs" class="log-attrs">
        {{ formattedAttrs }}
      </span>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useLogStore } from '../../stores'
import type { LogEntry } from '../../stores/log'

interface Props {
  /** 日誌物件資料 */
  log: LogEntry
  /** 是否為單行顯示模式 */
  isSingleLine?: boolean
  /** 格式化時忽略的屬性鍵名稱列表 */
  ignoreAttrs?: string[]
}

const props = withDefaults(defineProps<Props>(), {
  isSingleLine: false,
  ignoreAttrs: () => [],
})

const emit = defineEmits<{
  (e: 'click', log: LogEntry): void
  (e: 'dblclick', log: LogEntry): void
  (e: 'contextmenu', event: MouseEvent, log: LogEntry): void
  (e: 'openDetails', log: LogEntry): void
}>()

const logStore = useLogStore()

/**
 * 格式化時間戳記。
 * 當全域日誌設定或篩選層級為 DEBUG 時，顯示到毫秒 (MM-DD hh:mm:ss.SSS)；否則顯示到秒 (MM-DD hh:mm:ss)。
 */
const formattedTime = computed(() => {
  try {
    const date = new Date(props.log.timestamp)
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
    return props.log.timestamp
  }
})

/** 是否有需顯示的附加屬性 */
const hasVisibleAttrs = computed(() => {
  if (!props.log.attrs) return false
  const keys = Object.keys(props.log.attrs).filter(k => !props.ignoreAttrs.includes(k))
  return keys.length > 0
})

/** 格式化後的屬性字串 */
const formattedAttrs = computed(() => {
  if (!props.log.attrs) return ''
  return Object.entries(props.log.attrs)
    .filter(([key]) => !props.ignoreAttrs.includes(key))
    .map(([key, value]) => `${key}: ${value}`)
    .join('  ')
})
</script>

<style scoped>
/* ── 單條 Log 行 ─────────────────────────────────────────── */
.log-item {
  display: flex;
  align-items: center;
  gap: 5px;
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

/* 左側色條指示 log 等級 */
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

/* 時間戳記：等寬字體，次要色 */
.log-time {
  color: var(--text-color-secondary);
  font-size: 10px;
  flex-shrink: 0;
  min-width: 90px;
}

/* 等級標籤 */
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

/* 附加屬性 */
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
</style>
