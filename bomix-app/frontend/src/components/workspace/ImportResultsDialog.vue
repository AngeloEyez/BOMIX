<template>
  <Dialog
    v-model:visible="visibleModel"
    modal
    header="Import Results (匯入結果與即時 Task 狀態)"
    :style="{ width: '640px' }"
  >
    <div class="import-results">
      <div v-if="results && results.length > 0" class="results-list">
        <div
          v-for="(result, idx) in results"
          :key="result.taskID || idx"
          class="result-item"
          :class="getResultClass(result)"
        >
          <div class="result-header">
            <div class="result-name-group">
              <i :class="['result-status-icon', getResultIcon(result)]"></i>
              <span class="result-name" :title="result.fileName">{{ result.fileName }}</span>
            </div>
            <Tag :value="getResultStatusTag(result).label" :severity="getResultStatusTag(result).severity" />
          </div>

          <!-- 執行中動態進度條 -->
          <div v-if="getResultStatus(result) === 'running'" class="result-progress-container">
            <ProgressBar :value="getResultProgress(result)" :showValue="true" style="height: 14px; margin-top: 6px;" />
            <div class="result-message text-xs mt-1 text-blue-400">
              {{ getResultMessage(result) }}
            </div>
          </div>

          <!-- 排隊 / 完成 / 失敗訊息 -->
          <div v-else class="result-footer-msg mt-1">
            <span class="result-msg text-xs text-color-secondary">{{ getResultMessage(result) }}</span>
          </div>
        </div>
      </div>
      <div v-else class="no-results">
        目前沒有匯入作業結果
      </div>
    </div>

    <template #footer>
      <Button
        label="關閉"
        icon="pi pi-check"
        @click="visibleModel = false"
      />
    </template>
  </Dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import Tag from 'primevue/tag'
import ProgressBar from 'primevue/progressbar'
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
 * @returns {Task | null} Task 物件或 null
 */
function getResultTask(result: BackendImportResult) {
  if (result.taskID) {
    return taskStore.getTask(result.taskID)
  }
  return null
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
    if (task.status === 'completed') {
      return task.message || '匯入完成'
    }
    if (task.status === 'failed') {
      return task.error || task.message || '匯入失敗'
    }
    if (task.status === 'warning') {
      return task.error || task.message || '需進行業務與格式確認'
    }
    if (task.status === 'running') {
      return task.message || '正在進行匯入解析...'
    }
    if (task.status === 'queued' || task.status === 'created') {
      return '佇列中 (排隊中)'
    }
    return task.message || ''
  }
  return result.message || ''
}

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
      return { label: '排隊中 (Queued)', severity: 'secondary' }
    case 'running':
      return { label: '處理中 (Running)', severity: 'info' }
    case 'completed':
    case 'done':
      return { label: '已完成 (Completed)', severity: 'success' }
    case 'warning':
      return { label: '警告 (Warning)', severity: 'warn' }
    case 'failed':
    case 'error':
      return { label: '失敗 (Failed)', severity: 'danger' }
    case 'cancelled':
      return { label: '已取消 (Cancelled)', severity: 'contrast' }
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
      return 'pi pi-clock text-gray-400'
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
      return 'pi pi-ban text-gray-500'
    default:
      return 'pi pi-info-circle'
  }
}

/**
 * 取得列表項目之 CSS Card Class
 * @param {BackendImportResult} result - 匯入結果項目
 * @returns {string} CSS Class
 */
function getResultClass(result: BackendImportResult): string {
  const status = getResultStatus(result)
  if (status === 'failed' || status === 'error') return 'result-error'
  if (status === 'warning') return 'result-warning'
  if (status === 'completed' || status === 'done') return 'result-success'
  if (status === 'running') return 'result-running'
  return 'result-queued'
}
</script>

<style scoped>
.import-results {
  max-height: 380px;
  overflow-y: auto;
}

.results-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.result-item {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 0.75rem;
  background: var(--surface-ground);
  border: 1px solid var(--surface-border);
  border-radius: 6px;
  transition: all 0.2s ease;
}

.result-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.result-name-group {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-weight: 500;
  font-size: 0.9rem;
}

.result-status-icon {
  font-size: 1.1rem;
}

.result-error {
  border-color: rgba(239, 68, 68, 0.4);
  background: rgba(239, 68, 68, 0.05);
}

.result-warning {
  border-color: rgba(245, 158, 11, 0.4);
  background: rgba(245, 158, 11, 0.05);
}

.result-success {
  border-color: rgba(34, 197, 94, 0.3);
}

.result-running {
  border-color: rgba(59, 130, 246, 0.4);
  background: rgba(59, 130, 246, 0.03);
}

.result-name {
  flex: 1;
  font-weight: 500;
}

.no-results {
  text-align: center;
  padding: 2rem;
  color: var(--text-color-secondary);
}
</style>
