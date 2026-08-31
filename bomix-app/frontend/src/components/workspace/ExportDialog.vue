<template>
  <Dialog
    v-model:visible="visibleModel"
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
                  @change="saveProjectOrderFromCards"
                  @update:modelValue="saveProjectOrderFromCards"
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
        severity="secondary"
        @click="visibleModel = false"
      />
      <Button
        label="Export"
        icon="pi pi-check"
        @click="executeExport"
        :disabled="exportRevisions.length === 0"
      />
    </template>
  </Dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import MultiSelect from 'primevue/multiselect'
import Select from 'primevue/select'
import InputNumber from 'primevue/inputnumber'
import InputText from 'primevue/inputtext'
import { useAppStore, useLogStore } from '../../stores'
import {
  ExportExcel,
  SaveProjectExportOrder,
  SaveFileDialog,
  SelectFolderDialog,
  type ExportOptions
} from '../../services/api'

/**
 * 選取 Revision 卡片資料型態定義
 */
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

/**
 * Revision 下拉選單項目介面
 */
export interface RevisionOption {
  id: number
  projectId: number
  projectCode: string
  phase: string
  version: string
  label: string
  modelCount: number
}

/**
 * Component Props 定義
 */
const props = defineProps<{
  visible: boolean
  allRevisions: RevisionOption[]
}>()

/**
 * Component Emits 定義
 */
const emit = defineEmits<{
  (e: 'update:visible', val: boolean): void
  (e: 'exportSuccess', paths: string[]): void
}>()

const appStore = useAppStore()
const logStore = useLogStore()

/**
 * v-model:visible 雙向代理計算屬性
 */
const visibleModel = computed({
  get: () => props.visible,
  set: (val: boolean) => emit('update:visible', val)
})

const exportFormat = ref('BigMatrix')
const exportRevisions = ref<number[]>([])
const exportDescription = ref('')
const exportOutputPath = ref('')
const selectedCards = ref<SelectedRevisionCard[]>([])
const draggedIndex = ref<number | null>(null)

// 匯出格式選項
const exportFormatOptions = [
  { label: 'BigMatrix', value: 'BigMatrix' },
  { label: 'Matrix', value: 'Matrix' }
]

/**
 * 當開啟對話框時，初始化狀態與預設路徑
 */
watch(
  () => props.visible,
  (newVal) => {
    if (newVal) {
      initExportDialogState()
    }
  }
)

/**
 * 監聽匯出格式 (exportFormat) 的切換，即時更新匯出路徑與模式
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
 * 初始化對話框狀態與檔案輸出路徑
 */
function initExportDialogState(): void {
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
}

/**
 * 當 BigMatrix 模式下選取的 Card 發生改變時，即時更新預設檔名
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
    if (a.recordIndex === -1 && b.recordIndex !== -1) return -1
    if (a.recordIndex !== -1 && b.recordIndex === -1) return 1
    if (a.recordIndex !== -1 && b.recordIndex !== -1) {
      if (a.recordIndex !== b.recordIndex) {
        return a.recordIndex - b.recordIndex
      }
    }
    return a.originalIdx - b.originalIdx
  })

  return indexed.map(item => item.card)
}

/**
 * 從目前 selectedCards 的順序與 ModelCount 提取 Project 設定，並即時儲存至 Series 資料表
 */
async function saveProjectOrderFromCards(): Promise<void> {
  const settings: Array<{ projectCode: string; modelCount: number }> = []
  const projectOrder: string[] = []
  const projectModelCounts: Record<string, number> = {}

  for (const card of selectedCards.value) {
    if (card.projectCode && !projectOrder.includes(card.projectCode)) {
      projectOrder.push(card.projectCode)
      projectModelCounts[card.projectCode] = card.modelCount
      settings.push({
        projectCode: card.projectCode,
        modelCount: card.modelCount
      })
    }
  }

  if (appStore.seriesInfo) {
    appStore.seriesInfo.projectExportOrder = projectOrder
    appStore.seriesInfo.projectModelCounts = projectModelCounts
  }

  try {
    await SaveProjectExportOrder(settings)
    logStore.addLogEntry('DEBUG', `已即時更新 Project 匯出排序與 Model 數量紀錄：[${projectOrder.map(c => `${c}:${projectModelCounts[c]}`).join(', ')}]`)
  } catch (error) {
    const msg = error instanceof Error ? error.message : String(error)
    logStore.addLogEntry('WARN', `儲存 Project 匯出排序與 Model 數量紀錄失敗：${msg}`)
  }
}

/**
 * 處理 Revision 多選變更事件
 */
function onRevisionsChange(): void {
  const currentSelectedIds = new Set(exportRevisions.value)

  // 移除不再被選擇的卡片
  selectedCards.value = selectedCards.value.filter(c => currentSelectedIds.has(c.id))

  // 加入新選取的卡片
  const existingCardIds = new Set(selectedCards.value.map(c => c.id))
  for (const id of exportRevisions.value) {
    if (!existingCardIds.has(id)) {
      const opt = props.allRevisions.find(r => r.id === id)
      if (opt) {
        const dbCount = opt.modelCount || 0
        const savedCount = appStore.seriesInfo?.projectModelCounts?.[opt.projectCode]
        const initialCount = (savedCount && savedCount > 0) ? savedCount : (dbCount > 0 ? dbCount : 3)
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

  // 依歷史 Project 排序紀錄重新排序
  selectedCards.value = sortCardsByProjectRecord(selectedCards.value)

  // 套用歷史 Model 數量設定
  for (const card of selectedCards.value) {
    const savedCount = appStore.seriesInfo?.projectModelCounts?.[card.projectCode]
    if (savedCount && savedCount > 0) {
      card.modelCount = savedCount
    }
  }

  exportRevisions.value = selectedCards.value.map(c => c.id)
  updateBigMatrixFilenameIfNeed()
}

/**
 * 移除單一卡片
 * @param {number} index - 欲移除之卡片索引
 */
function removeCard(index: number): void {
  selectedCards.value.splice(index, 1)
  exportRevisions.value = selectedCards.value.map(c => c.id)
  updateBigMatrixFilenameIfNeed()
  saveProjectOrderFromCards()
}

// 拖曳排序處理函數
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
  saveProjectOrderFromCards()
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
 * 瀏覽選取匯出檔案或目錄路徑
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
 * 執行 Excel 匯出任務
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
    visibleModel.value = false
    logStore.addLogEntry('INFO', `匯出作業完成，共產生 ${exportedPaths?.length || 0} 個檔案`)
    emit('exportSuccess', exportedPaths || [])
  } catch (error) {
    const msg = error instanceof Error ? error.message : String(error)
    logStore.addLogEntry('ERROR', `匯出作業失敗：${msg}`)
  }
}
</script>

<style scoped>
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
