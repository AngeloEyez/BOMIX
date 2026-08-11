<template>
  <div class="workspace-page">
    <!-- Top Toolbar -->
    <div class="top-toolbar">
      <div class="toolbar-left">
        <span class="series-name" v-if="appStore.seriesInfo">
          {{ appStore.seriesInfo.name }}
        </span>
      </div>
      <div class="toolbar-right">
        <Button
          label="Import"
          icon="pi pi-upload"
          class="p-button-success"
          @click="openImportDialog"
        />
        <Button
          label="Export"
          icon="pi pi-download"
          class="p-button-warning"
          @click="openExportDialog"
        />
        <Button
          label="複製 Matrix"
          icon="pi pi-copy"
          class="p-button-outlined p-button-info"
          @click="openCopyMatrixDialog"
          title="手動從指定版本複製 Matrix Selection 到另一版本"
        />
      </div>
    </div>

    <!-- Main Content Panel -->
    <div class="main-content">
      <BOMTable
        v-if="projectStore.selectedRevision"
        :revision-id="projectStore.selectedRevision?.id"
      />
      <div v-else class="placeholder-content">
        <div class="dashboard-header">
          <i class="pi pi-box"></i>
          <h2>BOM Workspace</h2>
        </div>
        
        <div v-if="projectStore.projects.length === 0" class="empty-state">
          <p>No projects found in this series.</p>
          <Button label="Import BOM" icon="pi pi-upload" @click="openImportDialog" class="p-button-outlined" />
        </div>
        
        <div v-else class="dashboard-stats">
          <div class="stat-cards">
            <div class="stat-card">
              <span class="stat-title">Projects</span>
              <span class="stat-value">{{ projectStore.projects.length }}</span>
            </div>
            <div class="stat-card">
              <span class="stat-title">Total Revisions</span>
              <span class="stat-value">{{ totalRevisions }}</span>
            </div>
          </div>
          
          <div class="projects-list">
            <h3>Latest Revisions</h3>
            <div class="project-items">
              <div v-for="p in projectStore.projects" :key="p.id" class="project-item">
                <div class="project-info">
                  <span class="project-code">{{ p.code || p.name || `Project ${p.id}` }}</span>
                  <span class="project-desc" v-if="p.description">{{ p.description }}</span>
                </div>
                <div class="revision-info">
                  <span class="latest-rev" v-if="getLatestRevision(p)">{{ getLatestRevision(p) }}</span>
                  <span class="no-rev" v-else>No revisions</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Import Dialog -->
    <Dialog
      v-model:visible="importDialogVisible"
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
            class="p-button-outlined p-button-danger"
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
          @click="importDialogVisible = false"
        />
        <Button
          label="開始匯入"
          icon="pi pi-upload"
          @click="executeImport"
          :disabled="importFilePaths.length === 0"
        />
      </template>
    </Dialog>

    <!-- Import Results Dialog -->
    <Dialog
      v-model:visible="importResultDialogVisible"
      modal
      header="Import Results (匯入結果與即時 Task 狀態)"
      :style="{ width: '640px' }"
    >
      <div class="import-results">
        <div v-if="importResults.length > 0" class="results-list">
          <div
            v-for="(result, idx) in importResults"
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
          @click="importResultDialogVisible = false"
        />
      </template>
    </Dialog>

    <!-- Export Dialog -->
    <Dialog
      v-model:visible="exportDialogVisible"
      modal
      header="Export BOM"
      :style="{ width: '600px' }"
    >
      <div class="export-dialog-content">
        <!-- Format Selection -->
        <div class="form-group">
          <label for="exportFormat">Export Format</label>
          <Select
            v-model="exportFormat"
            :options="exportFormatOptions"
            option-label="label"
            option-value="value"
            placeholder="Select format"
            id="exportFormat"
          />
        </div>

        <!-- Revisions Selection -->
        <div class="form-group">
          <label for="exportRevisions">Select Revisions</label>
          <MultiSelect
            v-model="exportRevisions"
            :options="allRevisions"
            option-label="label"
            option-value="id"
            placeholder="Select revisions"
            id="exportRevisions"
            class="revision-multiselect"
            @change="onRevisionsChange"
          />
        </div>

        <!-- Selected Revisions Cards (Drag & Drop + Model Count) -->
        <div v-if="selectedCards.length > 0" class="form-group selected-revisions-section">
          <label>Selected Revisions & Model Counts (Drag to reorder)</label>
          <div class="selected-cards-list">
            <div
              v-for="(card, index) in selectedCards"
              :key="card.id"
              class="revision-card"
              draggable="true"
              @dragstart="onDragStart($event, index)"
              @dragover.prevent="onDragOver($event, index)"
              @drop="onDrop($event, index)"
            >
              <div class="card-left">
                <i class="pi pi-bars drag-handle" title="Drag to reorder"></i>
                <div class="card-info">
                  <span class="card-code">{{ card.projectCode }}</span>
                  <span class="card-ver">{{ [card.phase, card.version].filter(Boolean).join(' ') }}</span>
                </div>
              </div>

              <div class="card-right">
                <div class="model-count-group">
                  <span class="mc-label">Model Count:</span>
                  <InputNumber
                    v-model="card.modelCount"
                    :showButtons="true"
                    :min="1"
                    :disabled="exportFormat.toLowerCase() === 'matrix'"
                    class="card-model-input"
                  />
                </div>
                <Button
                  icon="pi pi-times"
                  class="p-button-text p-button-danger p-button-sm remove-card-btn"
                  @click="removeCard(index)"
                  title="Remove"
                />
              </div>
            </div>
          </div>
        </div>

        <!-- BigMatrix specific options -->
        <div v-if="exportFormat.toLowerCase() === 'bigmatrix'" class="export-options">
          <div class="form-group">
            <label for="exportDescription">Description (optional)</label>
            <InputText
              v-model="exportDescription"
              placeholder="Enter description"
              id="exportDescription"
            />
          </div>
        </div>

        <!-- Export Path Selection -->
        <div class="export-options">
          <div class="form-group">
            <label for="exportOutputPath">
              {{ exportFormat.toLowerCase() === 'bigmatrix' ? 'Output File' : 'Output Directory' }}
            </label>
            <div style="display: flex; gap: 0.5rem; align-items: center;">
              <InputText
                v-model="exportOutputPath"
                :placeholder="exportFormat.toLowerCase() === 'bigmatrix' ? 'Select output file' : 'Select output directory'"
                id="exportOutputPath"
                style="flex: 1;"
              />
              <Button
                label="Browse"
                icon="pi pi-folder-open"
                @click="browseExportPath"
              />
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <Button
          label="Cancel"
          icon="pi pi-times"
          text
          @click="exportDialogVisible = false"
        />
        <Button
          label="Export"
          icon="pi pi-check"
          @click="executeExport"
          :disabled="exportRevisions.length === 0"
        />
      </template>
    </Dialog>

    <!-- Copy Matrix Dialog -->
    <Dialog
      v-model:visible="copyMatrixDialogVisible"
      modal
      header="Matrix Selection 版本複製"
      :style="{ width: '520px' }"
    >
      <div class="copy-matrix-dialog-content">
        <div class="copy-matrix-warning mb-4 flex items-center gap-2 p-3 rounded" style="background: var(--p-yellow-50, #fffbeb); border: 1px solid var(--p-yellow-300, #fcd34d);">
          <i class="pi pi-exclamation-triangle text-amber-500"></i>
          <span>此操作將<strong>覆蓋</strong>目標版本的現有 Matrix Model 與 Selection！</span>
        </div>

        <div class="form-group">
          <label for="copyMatrixSource">來源版本 (Source)</label>
          <Select
            v-model="copyMatrixSourceId"
            :options="allRevisions"
            option-label="label"
            option-value="id"
            placeholder="選擇來源版本..."
            id="copyMatrixSource"
            class="w-full"
          />
        </div>

        <div class="form-group">
          <label for="copyMatrixTarget">目標版本 (Target)</label>
          <Select
            v-model="copyMatrixTargetId"
            :options="allRevisions"
            option-label="label"
            option-value="id"
            placeholder="選擇目標版本..."
            id="copyMatrixTarget"
            class="w-full"
          />
        </div>

        <p v-if="copyMatrixSourceId === copyMatrixTargetId && copyMatrixSourceId !== null" class="text-red-500 text-sm">
          來源版本與目標版本不可相同
        </p>
      </div>

      <template #footer>
        <Button
          label="取消"
          icon="pi pi-times"
          text
          @click="copyMatrixDialogVisible = false"
        />
        <Button
          label="開始複製"
          icon="pi pi-copy"
          severity="warning"
          @click="executeCopyMatrix"
          :disabled="!copyMatrixSourceId || !copyMatrixTargetId || copyMatrixSourceId === copyMatrixTargetId"
        />
      </template>
    </Dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import MultiSelect from 'primevue/multiselect'
import Select from 'primevue/select'
import InputNumber from 'primevue/inputnumber'
import InputText from 'primevue/inputtext'
import Checkbox from 'primevue/checkbox'
import Tag from 'primevue/tag'
import ProgressBar from 'primevue/progressbar'
import { useAppStore, useProjectStore, useLogStore, useTaskStore } from '../stores'
import BOMTable from '../components/BOMTable.vue'
import {
  ImportExcel,
  ExportExcel,
  CopyMatrixSelections,
  SaveProjectExportOrder,
  OpenFileDialog,
  OpenMultipleFilesDialog,
  SelectFolderDialog,
  SaveFileDialog,
  type ImportResult as BackendImportResult,
  type ExportOptions
} from '../services/api'
import type { Project } from '../stores/project'

const appStore = useAppStore()
const projectStore = useProjectStore()
const logStore = useLogStore()

// Computed dashboard stats
const totalRevisions = computed(() => {
  return projectStore.projects.reduce((sum, p) => sum + (p.revisions?.length || 0), 0)
})

function getLatestRevision(project: Project): string | null {
  if (!project.revisions || project.revisions.length === 0) return null
  // Assuming the last one in the array is the latest, or sort by id
  const sorted = [...project.revisions].sort((a, b) => b.id - a.id)
  const latest = sorted[0]
  return `${latest.phase} ${latest.version}`
}

// Import dialog
const importDialogVisible = ref(false)
const importFilePath = ref('')
const importFilePaths = ref<string[]>([])
const importResults = ref<BackendImportResult[]>([])
const importResultDialogVisible = ref(false)
const confirmOverwrite = ref(false)

// Export dialog
export interface SelectedRevisionCard {
  id: number
  projectId: number
  projectCode: string
  phase: string
  version: string
  label: string
  modelCount: number
  dbModelCount: number
}

const exportDialogVisible = ref(false)
const exportFormat = ref('BigMatrix')
const exportRevisions = ref<number[]>([])
const exportDescription = ref('')
const exportOutputPath = ref('')
const allRevisions = ref<any[]>([])
const selectedCards = ref<SelectedRevisionCard[]>([])
const draggedIndex = ref<number | null>(null)

// Export format options
const exportFormatOptions = [
  { label: 'BigMatrix', value: 'BigMatrix' },
  { label: 'Matrix', value: 'Matrix' }
]

// Copy Matrix Dialog state
const copyMatrixDialogVisible = ref(false)
const copyMatrixSourceId = ref<number | null>(null)
const copyMatrixTargetId = ref<number | null>(null)

const taskStore = useTaskStore()

onMounted(() => {
  taskStore.startListening()
  if (appStore.isOpen) {
    loadProjects()
  }
})

async function loadProjects(): Promise<void> {
  try {
    const list: any[] = []
    if (projectStore.projects && projectStore.projects.length > 0) {
      for (const p of projectStore.projects) {
        if (p.revisions) {
          for (const r of p.revisions) {
            const pCode = p.code || p.name || `Project ${p.id}`
            const phaseStr = (r.phase || '').trim()
            const verStr = (r.version || '').trim()
            const phaseVer = [phaseStr, verStr].filter(Boolean).join(' ')
            list.push({
              id: r.id,
              projectId: p.id,
              projectCode: pCode,
              phase: phaseStr,
              version: verStr,
              label: phaseVer ? `${pCode} - ${phaseVer}` : pCode,
              modelCount: r.modelCount || 0
            })
          }
        }
      }
    }
    if (list.length === 0) {
      list.push(
        { id: 1, projectId: 101, projectCode: 'PROJECT-A', phase: 'PV', version: '0.1', label: 'PROJECT-A - PV 0.1', modelCount: 0 },
        { id: 2, projectId: 101, projectCode: 'PROJECT-A', phase: 'PV', version: '0.2', label: 'PROJECT-A - PV 0.2', modelCount: 2 }
      )
    }
    allRevisions.value = list
  } catch (error) {
    const msg = error instanceof Error ? error.message : String(error)
    logStore.addLogEntry('ERROR', `載入專案資料失敗：${msg}`)
  }
}

/**
 * 當 BigMatrix 模式下選取的 Card 發生改變時，即時更新檔名
 */
function updateBigMatrixFilenameIfNeed(): void {
  if (exportFormat.value.toLowerCase() === 'bigmatrix') {
    const dir = getDirectory(exportOutputPath.value)
    const defaultFilename = generateBigMatrixFilename()
    exportOutputPath.value = dir ? `${dir}\\${defaultFilename}` : defaultFilename
  }
}

/**
 * 依據 Series 中儲存的 Project 排序紀錄，自動為選取之 Revisions 卡片進行排序
 * 排序規則：
 * 1. 若選中的 revision 中的 project 沒有存在上次紀錄，則排在最前面 (rank = -1)
 * 2. 若存在於上次紀錄中，則依紀錄位置升冪排序 (rank >= 0)
 * @param {SelectedRevisionCard[]} cards - 要排序的卡片陣列
 * @returns {SelectedRevisionCard[]} 排序後的卡片陣列
 */
function sortCardsByProjectRecord(cards: SelectedRevisionCard[]): SelectedRevisionCard[] {
  const savedOrder = appStore.seriesInfo?.projectExportOrder || []
  if (savedOrder.length === 0 || cards.length <= 1) return cards

  const orderMap = new Map<string, number>()
  savedOrder.forEach((code, idx) => {
    orderMap.set(code, idx)
  })

  const indexed = cards.map((card, originalIdx) => {
    const code = card.projectCode || ''
    const recordIndex = orderMap.has(code) ? orderMap.get(code)! : -1
    return { card, recordIndex, originalIdx }
  })

  indexed.sort((a, b) => {
    // 規則 2: 未在紀錄中的 Project (recordIndex === -1) 排在最前面
    if (a.recordIndex === -1 && b.recordIndex !== -1) return -1
    if (a.recordIndex !== -1 && b.recordIndex === -1) return 1

    // 規則 1: 若均在紀錄中，依紀錄索引比較
    if (a.recordIndex !== -1 && b.recordIndex !== -1) {
      if (a.recordIndex !== b.recordIndex) {
        return a.recordIndex - b.recordIndex
      }
    }

    // 若屬於同一 Project 或均未在紀錄中，維持原始選擇順序
    return a.originalIdx - b.originalIdx
  })

  return indexed.map(item => item.card)
}

/**
 * 從目前 selectedCards 的順序提取不重複的 Project Code 序列，並即時儲存至 Series 資料表
 */
async function saveProjectOrderFromCards(): Promise<void> {
  const projectOrder: string[] = []
  for (const card of selectedCards.value) {
    if (card.projectCode && !projectOrder.includes(card.projectCode)) {
      projectOrder.push(card.projectCode)
    }
  }

  // 即時更新 AppStore 記憶體快取
  if (appStore.seriesInfo) {
    appStore.seriesInfo.projectExportOrder = projectOrder
  }

  // 寫入 DB
  try {
    await SaveProjectExportOrder(projectOrder)
    logStore.addLogEntry('DEBUG', `已即時更新 Project 匯出排序紀錄：[${projectOrder.join(', ')}]`)
  } catch (error) {
    const msg = error instanceof Error ? error.message : String(error)
    logStore.addLogEntry('WARN', `儲存 Project 匯出排序紀錄失敗：${msg}`)
  }
}

function onRevisionsChange(): void {
  const currentSelectedIds = new Set(exportRevisions.value)

  // Remove cards no longer selected
  selectedCards.value = selectedCards.value.filter(c => currentSelectedIds.has(c.id))

  // Add new cards
  const existingCardIds = new Set(selectedCards.value.map(c => c.id))
  for (const id of exportRevisions.value) {
    if (!existingCardIds.has(id)) {
      const opt = allRevisions.value.find(r => r.id === id)
      if (opt) {
        const dbCount = opt.modelCount || 0
        const initialCount = dbCount > 0 ? dbCount : 3
        selectedCards.value.push({
          id: opt.id,
          projectId: opt.projectId,
          projectCode: opt.projectCode,
          phase: opt.phase,
          version: opt.version,
          label: opt.label,
          modelCount: initialCount,
          dbModelCount: dbCount
        })
      }
    }
  }

  // 依據歷史 Project 排序紀錄自動重新排序（未在紀錄中排前面，其餘依紀錄排序）
  selectedCards.value = sortCardsByProjectRecord(selectedCards.value)
  exportRevisions.value = selectedCards.value.map(c => c.id)

  updateBigMatrixFilenameIfNeed()
}

function removeCard(index: number): void {
  selectedCards.value.splice(index, 1)
  exportRevisions.value = selectedCards.value.map(c => c.id)
  updateBigMatrixFilenameIfNeed()
  saveProjectOrderFromCards()
}

// Drag and drop ordering logic
function onDragStart(event: DragEvent, index: number): void {
  draggedIndex.value = index
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
  }
}

function onDragOver(event: DragEvent, index: number): void {
  event.preventDefault()
}

function onDrop(event: DragEvent, dropIndex: number): void {
  event.preventDefault()
  if (draggedIndex.value === null || draggedIndex.value === dropIndex) return

  const itemToMove = selectedCards.value[draggedIndex.value]
  selectedCards.value.splice(draggedIndex.value, 1)
  selectedCards.value.splice(dropIndex, 0, itemToMove)

  exportRevisions.value = selectedCards.value.map(c => c.id)
  draggedIndex.value = null
  updateBigMatrixFilenameIfNeed()

  // 拖曳調整排序後，即時將 Project 排序紀錄寫入 Series 資料庫
  saveProjectOrderFromCards()
}

// ==================== Copy Matrix ====================

/**
 * 開啟 Matrix 複製對話框
 */
function openCopyMatrixDialog(): void {
  copyMatrixSourceId.value = null
  copyMatrixTargetId.value = null
  copyMatrixDialogVisible.value = true
}

/**
 * 執行 Matrix Selection 複製任務
 */
async function executeCopyMatrix(): Promise<void> {
  if (!copyMatrixSourceId.value || !copyMatrixTargetId.value) return
  if (copyMatrixSourceId.value === copyMatrixTargetId.value) {
    logStore.addLogEntry('WARN', '來源版本與目標版本不可相同')
    return
  }

  try {
    const taskId = await CopyMatrixSelections(copyMatrixSourceId.value, copyMatrixTargetId.value)
    if (taskId) {
      taskStore.updateTask(taskId, {
        id: taskId,
        name: 'Copy Matrix',
        type: 'CopyMatrix',
        status: 'queued',
        message: '複製任務已建立',
        progress: 0,
      })
      logStore.addLogEntry('INFO', `Matrix 複製任務已提交 (taskID: ${taskId})`)
    }
    copyMatrixDialogVisible.value = false
  } catch (error) {
    const msg = error instanceof Error ? error.message : String(error)
    logStore.addLogEntry('ERROR', `Matrix 複製失敗：${msg}`)
  }
}

// Import functions
function openImportDialog(): void {
  importFilePaths.value = []
  importFilePath.value = ''
  importResults.value = []
  importDialogVisible.value = true
}

function getFileName(filePath: string): string {
  if (!filePath) return ''
  return filePath.split(/[\\/]/).pop() || filePath
}

function removeImportFile(index: number): void {
  importFilePaths.value.splice(index, 1)
  if (importFilePaths.value.length === 0) {
    importFilePath.value = ''
  } else {
    importFilePath.value = `已選取 ${importFilePaths.value.length} 個檔案`
  }
}

function clearImportFiles(): void {
  importFilePaths.value = []
  importFilePath.value = ''
}

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

async function executeImport(): Promise<void> {
  try {
    const results = await ImportExcel(importFilePaths.value)
    importResults.value = results
    
    // 初始化或向 taskStore 登記任務狀態（若背景事件尚未建立才新增）
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

    importResultDialogVisible.value = true
    importDialogVisible.value = false
    logStore.addLogEntry('INFO', `已提交 ${results.length} 個匯入作業任務`)
  } catch (error) {
    const msg = error instanceof Error ? error.message : String(error)
    logStore.addLogEntry('ERROR', `匯入作業失敗：${msg}`)
  }
}

// 取得即時 Task 物件
function getResultTask(result: BackendImportResult) {
  if (result.taskID) {
    return taskStore.getTask(result.taskID)
  }
  return null
}

// 取得 Task 即時狀態
function getResultStatus(result: BackendImportResult): string {
  const task = getResultTask(result)
  if (task) {
    return task.status
  }
  return result.status || 'queued'
}

// 取得 Task 即時進度 (0 ~ 100)
function getResultProgress(result: BackendImportResult): number {
  const task = getResultTask(result)
  if (task) {
    return task.progress || 0
  }
  return 0
}

// 取得 Task 即時說明訊息
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

// 取得 Status Tag 標籤與 Severity
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

// 取得即時圖示
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

// 取得列表項 CSS 樣式
function getResultClass(result: BackendImportResult): string {
  const status = getResultStatus(result)
  if (status === 'failed' || status === 'error') return 'result-error'
  if (status === 'warning') return 'result-warning'
  if (status === 'completed' || status === 'done') return 'result-success'
  if (status === 'running') return 'result-running'
  return 'result-queued'
}

// Export functions
/**
 * 監聽匯出格式 (exportFormat) 的切換，即時更新匯出路徑與模式
 * 切換為 Matrix 時轉換為純目錄模式，切換為 BigMatrix 時轉換為檔案路徑模式
 */
watch(exportFormat, (newFormat) => {
  const dir = getDirectory(exportOutputPath.value)
  if (newFormat.toLowerCase() === 'bigmatrix') {
    const defaultFilename = generateBigMatrixFilename()
    exportOutputPath.value = dir ? `${dir}\\${defaultFilename}` : defaultFilename
  } else {
    exportOutputPath.value = dir
  }
})

/**
 * 開啟匯出對話框並初始化預設匯出路徑與檔名
 */
function openExportDialog(): void {
  loadProjects()
  exportRevisions.value = []
  selectedCards.value = []
  exportDescription.value = ''
  
  const lastPath = appStore.seriesInfo?.lastExportPath
  let initialDir = ''
  if (lastPath) {
    initialDir = getDirectory(lastPath)
  } else if (appStore.seriesInfo?.path) {
    const pathStr = appStore.seriesInfo.path
    const lastSlash = Math.max(pathStr.lastIndexOf('/'), pathStr.lastIndexOf('\\'))
    initialDir = lastSlash >= 0 ? pathStr.substring(0, lastSlash) : ''
  }

  if (exportFormat.value.toLowerCase() === 'bigmatrix') {
    const defaultFilename = generateBigMatrixFilename()
    exportOutputPath.value = initialDir ? `${initialDir}\\${defaultFilename}` : defaultFilename
  } else {
    exportOutputPath.value = initialDir
  }
  
  exportDialogVisible.value = true
}

/**
 * 依據選取的 Revision 資訊與日期產生 BigMatrix 預設檔名
 * @returns {string} 預設的 BigMatrix Excel 檔名
 */
function generateBigMatrixFilename(): string {
  const seriesName = appStore.seriesInfo?.name || 'Unknown'
  
  let allSamePhase = true
  let allSameVersion = true
  let firstPhase = ''
  let firstVersion = ''
  
  if (selectedCards.value.length > 0) {
    firstPhase = selectedCards.value[0].phase
    firstVersion = selectedCards.value[0].version
    for (const c of selectedCards.value) {
      if (c.phase !== firstPhase) allSamePhase = false
      if (c.version !== firstVersion) allSameVersion = false
    }
  } else {
    allSamePhase = false
    allSameVersion = false
  }
  
  const dateStr = new Date().toISOString().slice(0,10).replace(/-/g, '')
  
  let parts = [seriesName, 'BigMatrix']
  if (allSamePhase && firstPhase) {
    parts.push(firstPhase)
    if (allSameVersion && firstVersion) {
      parts.push(firstVersion)
    }
  }
  parts.push(dateStr)
  
  return parts.join('_') + '.xlsx'
}

/**
 * 從檔案或目錄路徑中提取目錄部分
 * @param {string} pathStr - 傳入的路徑字串
 * @returns {string} 提取出來的目錄路徑
 */
function getDirectory(pathStr: string): string {
  if (!pathStr) return ''
  if (pathStr.toLowerCase().endsWith('.xlsx') || pathStr.toLowerCase().endsWith('.xls')) {
    const lastSlash = Math.max(pathStr.lastIndexOf('/'), pathStr.lastIndexOf('\\'))
    return lastSlash >= 0 ? pathStr.substring(0, lastSlash) : ''
  }
  return pathStr
}

/**
 * 瀏覽並選取匯出檔案或目錄路徑
 */
async function browseExportPath(): Promise<void> {
  try {
    if (exportFormat.value.toLowerCase() === 'bigmatrix') {
      const dir = getDirectory(exportOutputPath.value)
      const defaultName = generateBigMatrixFilename()
      const defaultPath = dir ? `${dir}\\${defaultName}` : defaultName
      
      const filePath = await SaveFileDialog({
        title: 'Save BigMatrix Excel',
        defaultPath: defaultPath,
        filters: [{ name: 'Excel Files', extensions: ['xlsx'] }]
      })
      if (filePath) {
        exportOutputPath.value = filePath
        logStore.addLogEntry('INFO', `已選取匯出檔案：${filePath}`)
      }
    } else {
      const dir = getDirectory(exportOutputPath.value)
      const dirPath = await SelectFolderDialog({
        title: 'Select output directory',
        defaultPath: dir
      })
      if (dirPath) {
        exportOutputPath.value = dirPath
        logStore.addLogEntry('INFO', `已選取匯出目錄：${dirPath}`)
      }
    }
  } catch (error) {
    const msg = error instanceof Error ? error.message : String(error)
    logStore.addLogEntry('ERROR', `選取匯出路徑發生錯誤：${msg}`)
  }
}

/**
 * 執行 Excel 匯出作業
 */
async function executeExport(): Promise<void> {
  logStore.addLogEntry('INFO', `開始執行匯出作業...`)
  try {
    let sortedRevisionIds = selectedCards.value.map(c => c.id)
    if (sortedRevisionIds.length === 0 && exportRevisions.value.length > 0) {
      sortedRevisionIds = [...exportRevisions.value]
    }
    const modelCountOverrides: Record<string, number> = {}
    selectedCards.value.forEach(card => {
      modelCountOverrides[String(card.id)] = card.modelCount
    })

    const isBigMatrix = exportFormat.value.toLowerCase() === 'bigmatrix'
    let finalOutputPath = exportOutputPath.value
    const dir = getDirectory(finalOutputPath)
    
    if (isBigMatrix && !finalOutputPath.toLowerCase().endsWith('.xlsx')) {
      const defaultName = generateBigMatrixFilename()
      finalOutputPath = dir ? `${dir}\\${defaultName}` : defaultName
    }

    const options: ExportOptions = {
      format: exportFormat.value,
      revisionIds: sortedRevisionIds,
      description: exportDescription.value,
      outputPath: finalOutputPath,
      outputDir: dir,
      modelCountOverrides: modelCountOverrides
    }

    const exportedPaths = await ExportExcel(options)
    exportDialogVisible.value = false
    logStore.addLogEntry('INFO', `匯出作業完成，共產生 ${exportedPaths?.length || 0} 個檔案`)
  } catch (error) {
    const msg = error instanceof Error ? error.message : String(error)
    logStore.addLogEntry('ERROR', `匯出作業失敗：${msg}`)
  }
}
</script>

<style scoped>
.workspace-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.top-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  height: 50px;
  padding: 0 1rem;
  background: var(--surface-ground);
  border-bottom: 1px solid var(--surface-border);
}

.toolbar-left {
  display: flex;
  align-items: center;
}

.series-name {
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--text-color);
}

.toolbar-right {
  display: flex;
  gap: 0.5rem;
}

.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--surface-ground);
}

.placeholder-content {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 2rem;
  color: var(--text-color);
  overflow-y: auto;
}

.dashboard-header {
  text-align: center;
  margin-bottom: 2rem;
  color: var(--text-color-secondary);
}

.dashboard-header i {
  font-size: 3rem;
  margin-bottom: 1rem;
  color: var(--surface-border);
}

.dashboard-header h2 {
  font-size: 1.5rem;
  margin: 0;
  color: var(--text-color);
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
  margin-top: 2rem;
  color: var(--text-color-secondary);
}

.dashboard-stats {
  max-width: 800px;
  margin: 0 auto;
  width: 100%;
}

.stat-cards {
  display: flex;
  gap: 1.5rem;
  margin-bottom: 2rem;
}

.stat-card {
  flex: 1;
  background: var(--surface-card);
  padding: 1.5rem;
  border-radius: 8px;
  border: 1px solid var(--surface-border);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
}

.stat-title {
  color: var(--text-color-secondary);
  font-size: 0.875rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.stat-value {
  font-size: 2rem;
  font-weight: 600;
  color: var(--primary-color);
}

.projects-list h3 {
  margin: 0 0 1rem 0;
  font-size: 1.1rem;
  color: var(--text-color-secondary);
}

.project-items {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.project-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--surface-card);
  padding: 1rem;
  border-radius: 6px;
  border: 1px solid var(--surface-border);
}

.project-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.project-code {
  font-weight: 600;
}

.project-desc {
  font-size: 0.875rem;
  color: var(--text-color-secondary);
}

.latest-rev {
  background: var(--primary-color);
  color: white;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.875rem;
  font-weight: 500;
}

.no-rev {
  color: var(--text-color-secondary);
  font-size: 0.875rem;
  font-style: italic;
}

/* Bottom Splitter */
.bottom-splitter {
  border-top: 1px solid var(--surface-border);
}

/* Import Dialog */
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

/* Import Results */
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

.result-status {
  font-size: 0.75rem;
  color: var(--text-color-secondary);
}

.result-count {
  font-size: 0.75rem;
  color: var(--text-color-secondary);
}

.no-results {
  text-align: center;
  padding: 2rem;
  color: var(--text-color-secondary);
}

/* Export Dialog */
.export-dialog-content {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.form-group label {
  font-weight: 500;
  font-size: 0.875rem;
}

.revision-multiselect {
  min-height: 100px;
}

.export-options {
  border-top: 1px solid var(--surface-border);
  padding-top: 1rem;
}

/* Selected Revisions Cards & Drag-and-Drop */
.selected-revisions-section {
  margin-top: 0.5rem;
}

.selected-cards-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  max-height: 220px;
  overflow-y: auto;
  padding: 0.35rem;
  border: 1px dashed var(--surface-border);
  border-radius: 6px;
  background: var(--surface-ground);
}

.revision-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.5rem 0.75rem;
  background: var(--surface-card);
  border: 1px solid var(--surface-border);
  border-radius: 6px;
  cursor: grab;
  user-select: none;
  transition: transform 0.15s ease, box-shadow 0.15s ease, border-color 0.15s ease;
}

.revision-card:active {
  cursor: grabbing;
}

.revision-card:hover {
  border-color: var(--primary-color);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

.card-left {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.drag-handle {
  color: var(--text-color-secondary);
  cursor: grab;
  font-size: 0.9rem;
}

.card-info {
  display: flex;
  flex-direction: column;
}

.card-code {
  font-weight: 600;
  font-size: 0.9rem;
  color: var(--text-color);
}

.card-ver {
  font-size: 0.8rem;
  color: var(--text-color-secondary);
}

.card-right {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.model-count-group {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}

.mc-label {
  font-size: 0.85rem;
  color: var(--text-color-secondary);
}

.card-model-input {
  width: 90px;
}
</style>
