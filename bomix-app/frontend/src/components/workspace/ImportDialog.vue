<template>
  <Dialog
    v-model:visible="visibleModel"
    modal
    maximizable
    :style="{ width: '680px', maxWidth: '94vw' }"
    class="vscode-styled-dialog"
  >
    <!-- 自訂 VS Code 風格 Title Bar -->
    <template #header>
      <div class="vscode-titlebar-title flex items-center gap-2">
        <i class="pi pi-upload text-primary text-xs"></i>
        <span class="title-main font-medium"> 匯入 BOM 檔案</span>
        <span class="title-sub text-xs text-color-secondary font-normal"> (Import BOM Files)</span>
      </div>
    </template>

    <div class="import-dialog-content">
      <!-- 頂部操作工具列 -->
      <div class="toolbar-row flex items-center justify-between">
        <div class="flex items-center gap-1.5">
          <Button
            label="瀏覽選取檔案..."
            icon="pi pi-folder-open"
            size="small"
            class="vscode-action-btn"
            @click="browseFiles"
          />
          <Button
            v-if="importFilePaths.length > 0"
            label="清除全部"
            icon="pi pi-trash"
            text
            size="small"
            severity="danger"
            class="vscode-action-btn vscode-danger-btn"
            @click="clearImportFiles"
          />
        </div>
      </div>

      <!-- 檔案列表區域 (VS Code 緊湊列表風格，超出窗口時支援垂直捲軸) -->
      <div class="file-list-container" data-file-drop-target="true">
        <div v-if="importFilePaths.length === 0" class="file-list-empty">
          <i class="pi pi-file-excel empty-icon"></i>
          <span class="empty-text">尚未選取任何檔案</span>
          <span class="empty-subtext">點擊上方「瀏覽選取檔案」或直接拖曳 Excel 檔案至此視窗 (支援 Ctrl/Shift 多選)</span>
        </div>

        <div v-else class="file-list">
          <div
            v-for="(path, idx) in importFilePaths"
            :key="path"
            class="file-item"
            :title="path"
          >
            <!-- 序號與 Excel 圖示 -->
            <span class="file-index">{{ idx + 1 }}</span>
            <i class="pi pi-file-excel file-type-icon"></i>

            <!-- 檔案名稱與路徑 -->
            <div class="file-info min-w-0 flex-1">
              <span class="file-name truncate">{{ getFileName(path) }}</span>
            </div>

            <!-- 僅 Matrix BOM 顯示 "Matrix" 標籤，一般 EBOM 不顯示標籤 -->
            <span
              v-if="isMatrix(path)"
              class="phase-badge matrix-badge"
              title="Matrix BOM 檔案"
            >
              Matrix
            </span>

            <!-- 移除按鈕 -->
            <button
              class="remove-btn"
              @click.stop="removeImportFile(idx)"
              title="從清單中移除"
            >
              <i class="pi pi-times"></i>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 對話框底部操作列 (VS Code 雙行舒適排版，確保文字與 Checkbox 完整清晰) -->
    <template #footer>
      <div class="vscode-statusbar-footer flex items-center justify-between w-full gap-3">
        <!-- 左側：上行計數資訊，下行覆蓋選項 Checkbox -->
        <div class="footer-left flex flex-col justify-center gap-1 min-w-0 flex-1">
          <!-- 上行：檔案計數與細項分類 -->
          <div class="stats-row flex items-center min-w-0">
            <span v-if="importFilePaths.length > 0" class="stats-label text-xs">
              已選取 <strong>{{ importFilePaths.length }}</strong> 個檔案
              <span class="stats-breakdown text-color-secondary font-normal ml-1">
                ( EBOM: <strong class="text-color">{{ nonMatrixFiles.length }}</strong> Matrix: <strong class="text-color">{{ matrixFiles.length }}</strong> )
              </span>
            </span>
            <span v-else class="text-xs text-color-secondary">未選取檔案</span>
          </div>

          <!-- 下行：覆蓋確認選項 Checkbox -->
          <div class="import-options flex items-center gap-1.5">
            <Checkbox
              v-model="confirmOverwrite"
              inputId="confirmOverwrite"
              :binary="true"
              class="compact-checkbox"
            />
            <label for="confirmOverwrite" class="options-label text-xs"> 匯入覆蓋現有 BOM 前提示確認</label>
          </div>
        </div>

        <!-- 右側：取消與開始匯入按鈕 -->
        <div class="footer-right flex items-center gap-2 shrink-0">
          <Button
            label="取消"
            icon="pi pi-times"
            text
            size="small"
            severity="secondary"
            class="vscode-btn vscode-ghost-btn"
            @click="visibleModel = false"
          />
          <Button
            label="開始匯入"
            icon="pi pi-upload"
            size="small"
            class="vscode-btn vscode-primary-btn"
            @click="executeImport"
            :disabled="importFilePaths.length === 0"
          />
        </div>
      </div>
    </template>
  </Dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import Checkbox from 'primevue/checkbox'
import { useLogStore, useTaskStore, useAppStore } from '../../stores'
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
const appStore = useAppStore()

/**
 * v-model:visible 雙向代理計算屬性
 */
const visibleModel = computed({
  get: () => props.visible,
  set: (val: boolean) => emit('update:visible', val)
})

const importFilePaths = ref<string[]>([])
const confirmOverwrite = ref(false)

/**
 * 當開啟對話框時，自動重置選取檔案列表，若有拖曳進來的檔案則自動載入
 */
watch(
  () => props.visible,
  (newVal) => {
    if (newVal) {
      if (appStore.droppedFiles && appStore.droppedFiles.length > 0) {
        importFilePaths.value = [...appStore.droppedFiles]
        appStore.clearDroppedFiles()
      } else {
        importFilePaths.value = []
      }
    }
  }
)

/**
 * 監聽外部是否有新的拖曳檔案傳入
 */
watch(
  () => appStore.droppedFiles,
  (newFiles) => {
    if (newFiles && newFiles.length > 0 && props.visible) {
      const set = new Set(importFilePaths.value)
      for (const f of newFiles) {
        set.add(f)
      }
      importFilePaths.value = Array.from(set)
      appStore.clearDroppedFiles()
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
 * 判斷檔案是否為 Matrix BOM 檔案（檔名包含 "matrix"，不區分大小寫）
 * @param {string} filePath - 傳入的檔案路徑
 * @returns {boolean} 是否包含 matrix
 */
function isMatrix(filePath: string): boolean {
  if (!filePath) return false
  const name = getFileName(filePath).toLowerCase()
  return name.includes('matrix')
}

/**
 * 非 Matrix（基礎 EBOM）檔案清單
 */
const nonMatrixFiles = computed(() => importFilePaths.value.filter(p => !isMatrix(p)))

/**
 * Matrix BOM 檔案清單
 */
const matrixFiles = computed(() => importFilePaths.value.filter(p => isMatrix(p)))

/**
 * 移除指定索引之匯入檔案
 * @param {number} index - 欲移除之檔案索引
 */
function removeImportFile(index: number): void {
  importFilePaths.value.splice(index, 1)
}

/**
 * 清除所有已選取的匯入檔案
 */
function clearImportFiles(): void {
  importFilePaths.value = []
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
      logStore.addLogEntry('DEBUG', `已選取 ${selectedPaths.length} 個匯入檔案，目前累計 ${importFilePaths.value.length} 個`)
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
/* 內容排版容器 */
.import-dialog-content {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  height: 100%;
}

/* 操作按鈕 (VS Code Action Btn) */
.vscode-action-btn {
  height: 26px !important;
  font-size: 12px !important;
  padding: 0 0.55rem !important;
  border-radius: var(--p-radius-base, 2px) !important;
}

.vscode-danger-btn {
  color: #ef4444 !important;
}

.vscode-danger-btn:hover {
  background: rgba(239, 68, 68, 0.1) !important;
}

/* 檔案列表容器 (VS Code 風格) */
.file-list-container {
  display: flex;
  flex-direction: column;
  border: 1px solid var(--surface-border);
  border-radius: var(--p-radius-base, 2px);
  background: var(--surface-card);
  overflow: hidden;
  flex: 1;
}

/* 檔案滾動清單：超出時呈現垂直捲軸 */
.file-list {
  min-height: 220px;
  max-height: 340px;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 2px 0;
  display: flex;
  flex-direction: column;
}

/* 自訂 VS Code 緊湊滾動條 */
.file-list::-webkit-scrollbar {
  width: 8px;
}

.file-list::-webkit-scrollbar-track {
  background: transparent;
}

.file-list::-webkit-scrollbar-thumb {
  background-color: var(--surface-border);
  border-radius: 4px;
}

.file-list::-webkit-scrollbar-thumb:hover {
  background-color: var(--text-color-secondary);
}

/* 空白狀態提示 */
.file-list-empty {
  min-height: 220px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.35rem;
  padding: 1.5rem;
  color: var(--text-color-secondary);
}

.empty-icon {
  font-size: 1.8rem;
  color: var(--text-color-secondary);
  opacity: 0.6;
}

.empty-text {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-color);
}

.empty-subtext {
  font-size: 11px;
  text-align: center;
  max-width: 360px;
  line-height: 1.3;
}

/* 檔案項目列 (緊湊 VS Code Row) */
.file-item {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  height: 26px;
  padding: 0 0.5rem;
  border-bottom: 1px solid var(--surface-border);
  font-size: 12px;
  line-height: 1;
  transition: background-color 0.1s ease;
}

.file-item:last-child {
  border-bottom: none;
}

.file-item:hover {
  background: var(--surface-hover);
}

.file-index {
  font-size: 11px;
  color: var(--text-color-secondary);
  width: 18px;
  text-align: right;
  flex-shrink: 0;
}

.file-type-icon {
  font-size: 13px;
  color: #22c55e;
  flex-shrink: 0;
}

.file-item .file-name {
  font-weight: 500;
  color: var(--text-color);
  font-size: 12px;
}

/* Matrix 標籤樣式 */
.phase-badge {
  font-size: 10px;
  padding: 1px 5px;
  border-radius: 2px;
  font-weight: 600;
  white-space: nowrap;
  flex-shrink: 0;
  letter-spacing: 0.02em;
}

.matrix-badge {
  background-color: rgba(168, 85, 247, 0.12);
  color: #a855f7;
  border: 1px solid rgba(168, 85, 247, 0.3);
}

/* 移除按鈕 */
.remove-btn {
  background: transparent;
  border: none;
  color: var(--text-color-secondary);
  cursor: pointer;
  padding: 3px;
  border-radius: 2px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  opacity: 0.6;
  transition: opacity 0.15s ease, color 0.15s ease, background-color 0.15s ease;
  flex-shrink: 0;
}

.file-item:hover .remove-btn {
  opacity: 1;
}

.remove-btn:hover {
  color: #ef4444;
  background-color: rgba(239, 68, 68, 0.12);
}

/* 底部 VS Code Status Bar 風格 */
.vscode-statusbar-footer {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.stats-label {
  color: var(--text-color);
  white-space: nowrap;
  line-height: 1.3;
}

.stats-breakdown {
  letter-spacing: 0.02em;
}

/* 覆蓋選項 Checkbox */
.import-options {
  cursor: pointer;
  user-select: none;
  line-height: 1.3;
}

.compact-checkbox {
  transform: scale(0.9);
}

.options-label {
  color: var(--text-color-secondary);
  cursor: pointer;
  white-space: nowrap;
  font-size: 11.5px !important;
}

.import-options:hover .options-label {
  color: var(--text-color);
}

/* Footer 按鈕樣式 */
.vscode-btn {
  height: 26px !important;
  font-size: 12px !important;
  padding: 0 0.65rem !important;
  border-radius: var(--p-radius-base, 2px) !important;
}

.vscode-ghost-btn {
  color: var(--text-color-secondary) !important;
}

.vscode-ghost-btn:hover {
  color: var(--text-color) !important;
  background: var(--surface-hover) !important;
}
</style>

<style>
/* ==========================================================================
   VS Code 視窗風格 Dialog 全域與深層覆蓋 (VS Code Dialog Styles)
   ========================================================================== */
.vscode-styled-dialog {
  border-radius: var(--p-radius-base, 2px) !important;
  overflow: hidden !important;
  box-shadow: 0 10px 28px rgba(0, 0, 0, 0.28), 0 0 1px rgba(0, 0, 0, 0.5) !important;
  border: 1px solid var(--surface-border) !important;
}

/* Title Bar 頂部標題列 */
.vscode-styled-dialog .p-dialog-header {
  height: 34px !important;
  padding: 0 0.5rem 0 0.75rem !important;
  background-color: var(--surface-ground) !important;
  border-bottom: 1px solid var(--surface-border) !important;
  display: flex !important;
  align-items: center !important;
}

.vscode-styled-dialog .p-dialog-title {
  font-size: 12px !important;
  font-weight: 500 !important;
}

.vscode-styled-dialog .p-dialog-header-actions {
  gap: 0.2rem !important;
}

.vscode-styled-dialog .p-dialog-header-actions button,
.vscode-styled-dialog .p-dialog-header-actions .p-dialog-close-button,
.vscode-styled-dialog .p-dialog-header-actions .p-dialog-maximize-button {
  width: 22px !important;
  height: 22px !important;
  border-radius: 2px !important;
  color: var(--text-color-secondary) !important;
  transition: all 0.1s ease !important;
}

.vscode-styled-dialog .p-dialog-header-actions button:hover {
  background-color: var(--surface-hover) !important;
  color: var(--text-color) !important;
}

.vscode-styled-dialog .p-dialog-header-actions .p-dialog-close-button:hover {
  background-color: #e81123 !important;
  color: #ffffff !important;
}

/* Dialog Content 區域 */
.vscode-styled-dialog .p-dialog-content {
  padding: 0.55rem 0.75rem !important;
  background-color: var(--surface-ground) !important;
}

/* Footer 底部狀態與操作列：高度自適應，保證雙行排版舒適無截斷 */
.vscode-styled-dialog .p-dialog-footer {
  min-height: 46px !important;
  height: auto !important;
  padding: 0.35rem 0.75rem !important;
  background-color: var(--surface-ground) !important;
  border-top: 1px solid var(--surface-border) !important;
  display: flex !important;
  align-items: center !important;
}
</style>
