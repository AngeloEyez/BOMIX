<template>
  <Dialog
    v-model:visible="visibleModel"
    modal
    header="Import BOM Files (多檔案匯入)"
    :style="{ width: '560px' }"
  >
    <div class="import-dialog-content">
      <p class="drag-hint">
        選擇要匯入的 Excel 檔案（支援 EBOM, BigMatrix, Matrix 格式，可按住 Ctrl/Shift 選擇多個檔案）
      </p>

      <div class="button-group flex gap-2">
        <Button
          label="瀏覽選取檔案..."
          icon="pi pi-folder-open"
          @click="browseFiles"
        />
        <Button
          v-if="importFilePaths.length > 0"
          label="清除全部"
          icon="pi pi-trash"
          text
          severity="danger"
          @click="clearImportFiles"
        />
      </div>

      <div v-if="importFilePaths.length > 0" class="file-list-container">
        <div class="file-list-header">
          已選取 {{ importFilePaths.length }} 個檔案：
        </div>
        <div class="file-list">
          <div class="file-item" v-for="(path, idx) in importFilePaths" :key="idx" :title="path">
            <i
              class="pi pi-times remove-icon"
              @click="removeImportFile(idx)"
              title="移除此檔案"
            ></i>
            <i class="pi pi-file-excel file-type-icon text-green-500"></i>
            <span class="file-name">{{ getFileName(path) }}</span>
          </div>
        </div>
      </div>

      <div class="import-options">
        <Checkbox
          v-model="confirmOverwrite"
          inputId="confirmOverwrite"
          :binary="true"
        />
        <label for="confirmOverwrite">匯入覆蓋現有 BOM 前提示確認</label>
      </div>
    </div>

    <template #footer>
      <Button
        label="取消"
        icon="pi pi-times"
        text
        severity="secondary"
        @click="visibleModel = false"
      />
      <Button
        label="開始匯入"
        icon="pi pi-upload"
        @click="executeImport"
        :disabled="importFilePaths.length === 0"
      />
    </template>
  </Dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import Checkbox from 'primevue/checkbox'
import { useLogStore, useTaskStore } from '../../stores'
import {
  ImportExcel,
  OpenMultipleFilesDialog,
  type ImportResult as BackendImportResult
} from '../../services/api'

/**
 * Component Props 定義
 */
const props = defineProps<{
  visible: boolean
}>()

/**
 * Component Emits 定義
 */
const emit = defineEmits<{
  (e: 'update:visible', val: boolean): void
  (e: 'importSuccess', results: BackendImportResult[]): void
}>()

const logStore = useLogStore()
const taskStore = useTaskStore()

/**
 * v-model:visible 雙向代理計算屬性
 */
const visibleModel = computed({
  get: () => props.visible,
  set: (val: boolean) => emit('update:visible', val)
})

const importFilePath = ref('')
const importFilePaths = ref<string[]>([])
const confirmOverwrite = ref(false)

/**
 * 當開啟對話框時，自動重置選取檔案列表
 */
watch(
  () => props.visible,
  (newVal) => {
    if (newVal) {
      importFilePaths.value = []
      importFilePath.value = ''
    }
  }
)

/**
 * 從完整檔案路徑中擷取檔名
 * @param {string} filePath - 傳入的檔案完整路徑
 * @returns {string} 檔名
 */
function getFileName(filePath: string): string {
  if (!filePath) return ''
  return filePath.split(/[\\/]/).pop() || filePath
}

/**
 * 移除指定索引之匯入檔案
 * @param {number} index - 欲移除之檔案索引
 */
function removeImportFile(index: number): void {
  importFilePaths.value.splice(index, 1)
  if (importFilePaths.value.length === 0) {
    importFilePath.value = ''
  } else {
    importFilePath.value = `已選取 ${importFilePaths.value.length} 個檔案`
  }
}

/**
 * 清除所有已選取的匯入檔案
 */
function clearImportFiles(): void {
  importFilePaths.value = []
  importFilePath.value = ''
}

/**
 * 開啟作業系統原生多檔案選擇對話框
 */
async function browseFiles(): Promise<void> {
  try {
    const selectedPaths = await OpenMultipleFilesDialog({
      title: '選擇要匯入的 BOM Excel 檔案 (可按住 Ctrl/Shift 多選)',
      filters: [
        { name: 'Excel Files', extensions: ['xlsx', 'xls'] }
      ],
      selectFiles: true,
      multiSelect: true
    })

    if (selectedPaths && selectedPaths.length > 0) {
      const currentSet = new Set(importFilePaths.value)
      for (const p of selectedPaths) {
        currentSet.add(p)
      }
      importFilePaths.value = Array.from(currentSet)
      importFilePath.value = `已選取 ${importFilePaths.value.length} 個檔案`
      logStore.addLogEntry('DEBUG', `已選取 ${selectedPaths.length} 個匯入檔案`)
    }
  } catch (error) {
    const msg = error instanceof Error ? error.message : String(error)
    logStore.addLogEntry('ERROR', `選取匯入檔案發生錯誤：${msg}`)
  }
}

/**
 * 執行 Excel BOM 匯入任務
 */
async function executeImport(): Promise<void> {
  try {
    const results = await ImportExcel(importFilePaths.value)
    
    // 向 taskStore 登記任務狀態
    for (const r of results) {
      if (r.taskID && !taskStore.getTask(r.taskID)) {
        taskStore.updateTask(r.taskID, {
          id: r.taskID,
          name: `Import: ${r.fileName}`,
          type: 'Import',
          status: 'queued',
          message: 'Task created',
          progress: 0,
        })
      }
    }

    logStore.addLogEntry('INFO', `已提交 ${results.length} 個匯入作業任務`)
    emit('importSuccess', results)
    visibleModel.value = false
  } catch (error) {
    const msg = error instanceof Error ? error.message : String(error)
    logStore.addLogEntry('ERROR', `匯入作業失敗：${msg}`)
  }
}
</script>

<style scoped>
.import-dialog-content {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.drag-hint {
  color: var(--text-color-secondary);
  font-size: 0.875rem;
}

.file-list-container {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.file-list-header {
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--text-color-secondary);
}

.file-list {
  max-height: 160px;
  overflow-y: auto;
  border: 1px solid var(--surface-border);
  border-radius: 6px;
  padding: 0.35rem 0.4rem;
  background: var(--surface-ground);
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.file-item {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.2rem 0.45rem;
  background: var(--surface-card);
  border: 1px solid var(--surface-border);
  border-radius: 4px;
  font-size: 0.78rem;
  line-height: 1.2;
}

.remove-icon {
  font-size: 0.75rem;
  color: var(--text-color-secondary);
  cursor: pointer;
  padding: 2px;
  border-radius: 3px;
  transition: color 0.15s ease, background-color 0.15s ease;
}

.remove-icon:hover {
  color: #ef4444;
  background-color: rgba(239, 68, 68, 0.1);
}

.file-type-icon {
  font-size: 0.8rem;
}

.file-item .file-name {
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 0.78rem;
}

.import-options {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
</style>
