<template>
  <Dialog
    v-model:visible="visibleModel"
    modal
    maximizable
    :style="{ width: '720px', maxWidth: '94vw' }"
    class="vscode-styled-dialog"
  >
    <!-- 自訂 VS Code 風格 Title Bar -->
    <template #header>
      <div class="vscode-titlebar-title flex items-center gap-2">
        <i class="pi pi-list-check text-primary text-xs"></i>
        <span class="title-main font-medium"> 匯入結果與即時狀態</span>
        <span class="title-sub text-xs text-color-secondary font-normal"> (Import Status & Results)</span>
      </div>
    </template>

    <div class="import-results-container">
      <!-- 頂部整體狀態摘要統計列 (VS Code Status Bar 風格) -->
      <div v-if="results && results.length > 0" class="summary-card">
        <!-- 上方：總體進度與統計切換 -->
        <div class="summary-header flex items-center justify-between">
          <div class="overall-progress-info flex items-center gap-2">
            <span class="summary-title text-xs">總體進度：</span>
            <span class="summary-progress-text text-xs font-semibold">
              {{ completedCount }}/{{ results.length }} 完成 ({{ overallPercent }}%)
            </span>
          </div>

          <!-- 快速狀態過濾器 (點擊可切換清單篩選) -->
          <div class="status-filters flex items-center gap-1">
            <button
              class="filter-pill"
              :class="{ 'is-active': activeFilter === 'all' }"
              @click="activeFilter = 'all'"
            >
              全部 <span class="pill-count">{{ results.length }}</span>
            </button>
            <button
              v-if="runningCount > 0"
              class="filter-pill pill-running"
              :class="{ 'is-active': activeFilter === 'running' }"
              @click="activeFilter = 'running'"
            >
              <i class="pi pi-spin pi-spinner mr-0.5"></i>
              處理中 <span class="pill-count">{{ runningCount }}</span>
            </button>
            <button
              v-if="queuedCount > 0"
              class="filter-pill pill-queued"
              :class="{ 'is-active': activeFilter === 'queued' }"
              @click="activeFilter = 'queued'"
            >
              排隊中 <span class="pill-count">{{ queuedCount }}</span>
            </button>
            <button
              v-if="successCount > 0"
              class="filter-pill pill-success"
              :class="{ 'is-active': activeFilter === 'completed' }"
              @click="activeFilter = 'completed'"
            >
              完成 <span class="pill-count">{{ successCount }}</span>
            </button>
            <button
              v-if="failedCount > 0"
              class="filter-pill pill-failed"
              :class="{ 'is-active': activeFilter === 'failed' }"
              @click="activeFilter = 'failed'"
            >
              失敗 <span class="pill-count">{{ failedCount }}</span>
            </button>
          </div>
        </div>

        <!-- 總體進度條 (細條型 3px) -->
        <div class="overall-progress-bar-wrapper">
          <div
            class="overall-progress-bar-fill"
            :style="{ width: `${overallPercent}%` }"
            :class="{ 'is-all-completed': completedCount === results.length }"
          ></div>
        </div>
      </div>

      <!-- 任務結果列表區域 (VS Code Diagnostics & Tasks 緊湊風格) -->
      <div class="results-list-container">
        <div v-if="filteredResults.length === 0" class="results-empty">
          <i class="pi pi-inbox empty-icon"></i>
          <span class="empty-text">無符合篩選條件的任務項目</span>
        </div>

        <div v-else class="results-list">
          <div
            v-for="(result, idx) in filteredResults"
            :key="result.taskID || idx"
            class="result-row"
            :class="getResultClass(result)"
          >
            <!-- 左側：狀態圖示 + 檔名 -->
            <div class="row-main-info flex items-center gap-2 min-w-0 flex-1">
              <i :class="['result-status-icon shrink-0', getResultIcon(result)]"></i>
              <div class="file-info-group min-w-0 flex-1">
                <div class="flex items-center gap-1.5">
                  <span class="result-file-name truncate" :title="result.fileName">
                    {{ result.fileName }}
                  </span>
                  <span v-if="isMatrix(result.fileName)" class="matrix-mini-tag shrink-0">
                    Matrix
                  </span>
                </div>
                <!-- 執行中之細緻訊息 / 錯誤提示 -->
                <div class="result-sub-message truncate" :title="getResultMessage(result)">
                  {{ getResultMessage(result) }}
                </div>
              </div>
            </div>

            <!-- 右側：即時狀態標籤與進度條 -->
            <div class="row-side-info flex flex-col items-end shrink-0 gap-1">
              <div class="flex items-center gap-1.5">
                <span
                  v-if="getResultStatus(result) === 'running'"
                  class="progress-percent-label font-mono text-xs text-blue-500"
                >
                  {{ getResultProgress(result) }}%
                </span>
                <Tag
                  :value="getResultStatusTag(result).label"
                  :severity="getResultStatusTag(result).severity"
                  class="compact-status-tag"
                />
              </div>

              <!-- 執行中動態微型進度條 -->
              <div
                v-if="getResultStatus(result) === 'running'"
                class="mini-progress-track"
              >
                <div
                  class="mini-progress-fill"
                  :style="{ width: `${getResultProgress(result)}%` }"
                ></div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 對話框底部操作列 -->
    <template #footer>
      <div class="vscode-statusbar-footer flex items-center justify-between w-full">
        <div class="footer-left text-xs text-color-secondary flex items-center gap-1.5">
          <i
            v-if="runningCount > 0 || queuedCount > 0"
            class="pi pi-spin pi-spinner text-blue-500"
          ></i>
          <span v-if="runningCount > 0 || queuedCount > 0">
            匯入任務正在背景持續處理中，可隨時關閉此視窗
          </span>
          <span v-else-if="results.length > 0" class="text-green-500 flex items-center gap-1">
            <i class="pi pi-check"></i> 所有匯入作業已處理完畢
          </span>
        </div>
        <div class="footer-right flex items-center gap-2">
          <Button
            label="關閉"
            icon="pi pi-check"
            size="small"
            class="vscode-btn vscode-primary-btn"
            @click="visibleModel = false"
          />
        </div>
      </div>
    </template>
  </Dialog>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import Tag from 'primevue/tag'
import { useTaskStore } from '../../stores'
import type { ImportResult as BackendImportResult } from '../../services/api'

/**
 * Component Props 定義
 */
const props = defineProps<{
  visible: boolean
  results: BackendImportResult[]
}>()

/**
 * Component Emits 定義
 */
const emit = defineEmits<{
  (e: 'update:visible', val: boolean): void
}>()

const taskStore = useTaskStore()

/** 篩選模式：'all' | 'running' | 'queued' | 'completed' | 'failed' */
const activeFilter = ref<'all' | 'running' | 'queued' | 'completed' | 'failed'>('all')

/**
 * v-model:visible 雙向代理計算屬性
 */
const visibleModel = computed({
  get: () => props.visible,
  set: (val: boolean) => emit('update:visible', val)
})

/**
 * 取得 TaskStore 中的即時 Task 物件
 * @param {BackendImportResult} result - 匯入結果項目
 * @returns {any} Task 物件或 null
 */
function getResultTask(result: BackendImportResult) {
  if (result.taskID) {
    return taskStore.getTask(result.taskID)
  }
  return null
}

/**
 * 判斷檔名是否為 Matrix
 * @param {string} fileName - 檔名
 * @returns {boolean} 是否為 Matrix
 */
function isMatrix(fileName?: string): boolean {
  if (!fileName) return false
  return fileName.toLowerCase().includes('matrix')
}

/**
 * 取得 Task 即時狀態
 * @param {BackendImportResult} result - 匯入結果項目
 * @returns {string} 狀態字串 (running/completed/failed/queued 等)
 */
function getResultStatus(result: BackendImportResult): string {
  const task = getResultTask(result)
  if (task) {
    return task.status
  }
  return result.status || 'queued'
}

/**
 * 取得 Task 即時進度百分比 (0 ~ 100)
 * @param {BackendImportResult} result - 匯入結果項目
 * @returns {number} 進度百分比
 */
function getResultProgress(result: BackendImportResult): number {
  const task = getResultTask(result)
  if (task) {
    return task.progress || 0
  }
  return 0
}

/**
 * 取得 Task 即時說明訊息
 * @param {BackendImportResult} result - 匯入結果項目
 * @returns {string} 訊息字串
 */
function getResultMessage(result: BackendImportResult): string {
  const task = getResultTask(result)
  if (task) {
    if (task.status === 'completed' || task.status === 'done') {
      return task.message || '匯入完成'
    }
    if (task.status === 'failed' || task.status === 'error') {
      return task.error || task.message || '匯入失敗'
    }
    if (task.status === 'warning') {
      return task.error || task.message || '需進行業務與格式確認'
    }
    if (task.status === 'running') {
      return task.message || '正在進行匯入解析...'
    }
    if (task.status === 'queued' || task.status === 'created') {
      return '佇列中 (等待解析)'
    }
    return task.message || ''
  }
  return result.message || ''
}

/**
 * 各狀態數量計算屬性
 */
const runningCount = computed(() => {
  if (!props.results) return 0
  return props.results.filter(r => getResultStatus(r) === 'running').length
})

const queuedCount = computed(() => {
  if (!props.results) return 0
  return props.results.filter(r => ['queued', 'created'].includes(getResultStatus(r))).length
})

const successCount = computed(() => {
  if (!props.results) return 0
  return props.results.filter(r => ['completed', 'done'].includes(getResultStatus(r))).length
})

const failedCount = computed(() => {
  if (!props.results) return 0
  return props.results.filter(r => ['failed', 'error', 'warning'].includes(getResultStatus(r))).length
})

const completedCount = computed(() => {
  return successCount.value + failedCount.value
})

/**
 * 總體完成百分比 (0 ~ 100)
 */
const overallPercent = computed(() => {
  if (!props.results || props.results.length === 0) return 0
  return Math.round((completedCount.value / props.results.length) * 100)
})

/**
 * 依 Filter 過濾後的結果列表
 */
const filteredResults = computed(() => {
  if (!props.results) return []
  if (activeFilter.value === 'all') return props.results

  return props.results.filter(r => {
    const s = getResultStatus(r)
    if (activeFilter.value === 'running') return s === 'running'
    if (activeFilter.value === 'queued') return s === 'queued' || s === 'created'
    if (activeFilter.value === 'completed') return s === 'completed' || s === 'done'
    if (activeFilter.value === 'failed') return s === 'failed' || s === 'error' || s === 'warning'
    return true
  })
})

/**
 * 取得 Status Tag 標籤與 Severity 級別
 * @param {BackendImportResult} result - 匯入結果項目
 * @returns {{ label: string; severity: string }} PrimeVue Tag 參數
 */
function getResultStatusTag(result: BackendImportResult): { label: string; severity: 'secondary' | 'info' | 'success' | 'warn' | 'danger' | 'contrast' } {
  const status = getResultStatus(result)
  switch (status) {
    case 'created':
    case 'queued':
      return { label: '排隊中', severity: 'secondary' }
    case 'running':
      return { label: '處理中', severity: 'info' }
    case 'completed':
    case 'done':
      return { label: '已完成', severity: 'success' }
    case 'warning':
      return { label: '警告', severity: 'warn' }
    case 'failed':
    case 'error':
      return { label: '失敗', severity: 'danger' }
    case 'cancelled':
      return { label: '已取消', severity: 'contrast' }
    default:
      return { label: status, severity: 'info' }
  }
}

/**
 * 取得即時狀態圖示 Class
 * @param {BackendImportResult} result - 匯入結果項目
 * @returns {string} FontAwesome/PrimeIcon Class
 */
function getResultIcon(result: BackendImportResult): string {
  const status = getResultStatus(result)
  switch (status) {
    case 'created':
    case 'queued':
      return 'pi pi-clock text-slate-400'
    case 'running':
      return 'pi pi-spin pi-spinner text-blue-500'
    case 'completed':
    case 'done':
      return 'pi pi-check-circle text-green-500'
    case 'warning':
      return 'pi pi-exclamation-triangle text-amber-500'
    case 'failed':
    case 'error':
      return 'pi pi-times-circle text-red-500'
    case 'cancelled':
      return 'pi pi-ban text-slate-500'
    default:
      return 'pi pi-info-circle'
  }
}

/**
 * 取得列表項目之 CSS Row Class
 * @param {BackendImportResult} result - 匯入結果項目
 * @returns {string} CSS Class
 */
function getResultClass(result: BackendImportResult): string {
  const status = getResultStatus(result)
  if (status === 'failed' || status === 'error') return 'row-failed'
  if (status === 'warning') return 'row-warning'
  if (status === 'completed' || status === 'done') return 'row-success'
  if (status === 'running') return 'row-running'
  return 'row-queued'
}
</script>

<style scoped>
.import-results-container {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

/* 頂部狀態摘要卡片 */
.summary-card {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  padding: 0.45rem 0.6rem;
  background: var(--surface-card);
  border: 1px solid var(--surface-border);
  border-radius: var(--p-radius-base, 2px);
}

.summary-title {
  color: var(--text-color-secondary);
}

.summary-progress-text {
  color: var(--text-color);
}

/* 篩選標籤按鈕 (Filter Pills) */
.status-filters {
  display: flex;
  flex-wrap: wrap;
}

.filter-pill {
  background: transparent;
  border: 1px solid transparent;
  color: var(--text-color-secondary);
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 2px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 3px;
  line-height: 1.4;
  transition: all 0.15s ease;
}

.filter-pill:hover {
  background: var(--surface-hover);
  color: var(--text-color);
}

.filter-pill.is-active {
  background: var(--surface-hover);
  border-color: var(--surface-border);
  color: var(--text-color);
  font-weight: 600;
}

.pill-count {
  background: rgba(0, 0, 0, 0.08);
  padding: 0 4px;
  border-radius: 2px;
  font-size: 10px;
}

html.app-dark .pill-count {
  background: rgba(255, 255, 255, 0.1);
}

.pill-running.is-active {
  border-color: rgba(59, 130, 246, 0.4);
  color: #3b82f6;
}

.pill-success.is-active {
  border-color: rgba(34, 197, 94, 0.4);
  color: #22c55e;
}

.pill-failed.is-active {
  border-color: rgba(239, 68, 68, 0.4);
  color: #ef4444;
}

/* 總體進度條 */
.overall-progress-bar-wrapper {
  width: 100%;
  height: 3px;
  background: var(--surface-ground);
  border-radius: 2px;
  overflow: hidden;
}

.overall-progress-bar-fill {
  height: 100%;
  background: #3b82f6;
  transition: width 0.3s ease;
}

.overall-progress-bar-fill.is-all-completed {
  background: #22c55e;
}

/* 任務列表容器 */
.results-list-container {
  border: 1px solid var(--surface-border);
  border-radius: var(--p-radius-base, 2px);
  background: var(--surface-card);
  overflow: hidden;
}

.results-list {
  min-height: 200px;
  max-height: 360px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}

.results-empty {
  min-height: 200px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.35rem;
  color: var(--text-color-secondary);
}

.empty-icon {
  font-size: 1.8rem;
  opacity: 0.5;
}

.empty-text {
  font-size: 12px;
}

/* 任務項目列 (緊湊 VS Code Row) */
.result-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  padding: 0.3rem 0.55rem;
  border-bottom: 1px solid var(--surface-border);
  font-size: 12px;
  line-height: 1.3;
  transition: background-color 0.1s ease;
}

.result-row:last-child {
  border-bottom: none;
}

.result-row:hover {
  background: var(--surface-hover);
}

.result-status-icon {
  font-size: 14px;
}

.result-file-name {
  font-weight: 500;
  color: var(--text-color);
  font-size: 12px;
}

.matrix-mini-tag {
  font-size: 9px;
  padding: 0 4px;
  border-radius: 2px;
  background: rgba(168, 85, 247, 0.12);
  color: #a855f7;
  border: 1px solid rgba(168, 85, 247, 0.25);
  line-height: 1.2;
}

.result-sub-message {
  font-size: 11px;
  color: var(--text-color-secondary);
  line-height: 1.2;
}

.row-failed .result-sub-message {
  color: #ef4444;
}

.row-warning .result-sub-message {
  color: #f59e0b;
}

.row-running .result-sub-message {
  color: #3b82f6;
}

/* 狀態標籤與進度條 */
.compact-status-tag {
  font-size: 10px !important;
  padding: 1px 5px !important;
  line-height: 1.1 !important;
  border-radius: 2px !important;
}

.mini-progress-track {
  width: 80px;
  height: 3px;
  background: var(--surface-ground);
  border-radius: 2px;
  overflow: hidden;
}

.mini-progress-fill {
  height: 100%;
  background: #3b82f6;
  transition: width 0.2s ease;
}

/* 底部 VS Code Status Bar 風格 */
.vscode-statusbar-footer {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  line-height: 1;
}

.vscode-btn {
  height: 26px !important;
  font-size: 12px !important;
  padding: 0 0.75rem !important;
  border-radius: var(--p-radius-base, 2px) !important;
}
</style>
