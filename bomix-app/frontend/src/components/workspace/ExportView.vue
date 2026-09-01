<template>
  <div class="export-view-container">
    <!-- 頂部區塊：匯出格式與輸出檔案/目錄設定 -->
    <div class="export-section top-config-section">
      <!-- 1. Export Format (SelectButton) -->
      <div class="form-row format-row">
        <label class="section-label">Export Format</label>
        <SelectButton
          v-model="exportFormat"
          :options="exportFormatOptions"
          option-label="label"
          option-value="value"
          :allow-empty="false"
          size="small"
          class="format-select-btn"
        />
      </div>

      <!-- 2. Output File / Output Directory -->
      <div class="form-row path-row">
        <label class="section-label" for="exportOutputPath">
          {{ exportFormat.toLowerCase() === 'bigmatrix' ? 'Output File' : 'Output Directory' }}
        </label>
        <div class="path-input-group">
          <InputText
            v-model="exportOutputPath"
            :placeholder="exportFormat.toLowerCase() === 'bigmatrix' ? 'Select output file (.xlsx)' : 'Select output directory'"
            id="exportOutputPath"
            class="path-input"
          />
          <Button
            label="Browse"
            icon="pi pi-folder-open"
            size="small"
            severity="secondary"
            class="browse-btn"
            @click="browseExportPath"
            title="瀏覽選取輸出路徑"
          />
        </div>
      </div>
    </div>

    <!-- 3. Select Revisions & Model Counts (與 Sidepanel 雙向同步 + 拖曳排序) -->
    <div class="export-section revisions-section">
      <div class="revisions-header">
        <span class="section-label">
          Selected Revisions & Model Counts
          <span v-if="selectedCards.length > 0" class="revisions-count-badge">
            {{ selectedCards.length }}
          </span>
        </span>
      </div>

      <!-- 卡片清單區塊 -->
      <div class="selected-cards-container">
        <!-- 未選擇任何版本時的提示 -->
        <div v-if="selectedCards.length === 0" class="empty-revisions-tip">
          <i class="pi pi-info-circle"></i>
          <span>請在左側側邊欄 (Sidepanel) 勾選要匯出的 Revision 版本</span>
        </div>

        <!-- VSCode Style 緊湊版本卡片 (僅 BigMatrix 模式支援拖曳排序) -->
        <div
          v-for="(card, index) in selectedCards"
          :key="card.id"
          :class="['revision-card', { 'is-draggable': exportFormat.toLowerCase() === 'bigmatrix' }]"
          :draggable="exportFormat.toLowerCase() === 'bigmatrix'"
          @dragstart="onDragStart($event, index)"
          @dragover.prevent="onDragOver($event, index)"
          @drop="onDrop($event, index)"
        >
          <!-- 左側：拖曳把手 (僅 BigMatrix 顯示)、專案代碼、版本資訊 -->
          <div class="card-left">
            <i
              v-if="exportFormat.toLowerCase() === 'bigmatrix'"
              class="pi pi-bars drag-handle"
              title="拖曳以重新排序"
            ></i>
            <span class="card-code" :title="card.projectCode">{{ card.projectCode }}</span>
            <span class="card-ver" v-if="card.phase || card.version">
              {{ [card.phase, card.version].filter(Boolean).join(' ') }}
            </span>
          </div>

          <!-- 右側：Model 數量輸入與移除按鈕 -->
          <div class="card-right">
            <div class="model-count-group" v-if="exportFormat.toLowerCase() === 'bigmatrix'">
              <span class="mc-label">Model Count:</span>
              <InputNumber
                v-model="card.modelCount"
                :showButtons="true"
                :min="1"
                size="small"
                class="card-model-input"
                @change="saveProjectOrderFromCards"
                @update:modelValue="saveProjectOrderFromCards"
              />
            </div>
            <Button
              icon="pi pi-times"
              text
              rounded
              severity="secondary"
              class="remove-card-btn"
              @click="removeCard(index)"
              title="從匯出選取中移除 (將同步取消側邊欄勾選)"
            />
          </div>
        </div>
      </div>
    </div>

    <!-- 4. Description (optional) & 5. 匯出操作按鈕 -->
    <div class="export-section bottom-action-section">
      <!-- 4. Description (BigMatrix 專用或通用可選) -->
      <div v-if="exportFormat.toLowerCase() === 'bigmatrix'" class="form-row desc-row">
        <label class="section-label" for="exportDescription">Description (optional)</label>
        <InputText
          v-model="exportDescription"
          placeholder="Enter description for BigMatrix header..."
          id="exportDescription"
          class="desc-input"
        />
      </div>

      <!-- 5. Export Action 按鈕 (Primary 底色，無 Cancel 按鈕) -->
      <div class="action-btn-row">
        <div class="action-left-info">
          <span v-if="isExporting" class="exporting-spinner">
            <i class="pi pi-spin pi-spinner"></i> 正在處理匯出作業...
          </span>
        </div>
        <Button
          label="Export"
          icon="pi pi-download"
          severity="primary"
          :loading="isExporting"
          :disabled="selectedCards.length === 0 || isExporting"
          class="export-submit-btn"
          @click="executeExport"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import Button from 'primevue/button'
import SelectButton from 'primevue/selectbutton'
import InputNumber from 'primevue/inputnumber'
import InputText from 'primevue/inputtext'
import { useAppStore, useProjectStore, useLogStore } from '../../stores'
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
 * Revision 下拉選單/選項介面（向下相容 CopyMatrixDialog 等引用）
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
 * Component Emits 定義
 */
const emit = defineEmits<{
  (e: 'exportSuccess', paths: string[]): void
}>()

const appStore = useAppStore()
const projectStore = useProjectStore()
const logStore = useLogStore()

// 匯出設定狀態
const exportFormat = ref<'BigMatrix' | 'Matrix'>('BigMatrix')
const exportDescription = ref('')
const exportOutputPath = ref('')
const selectedCards = ref<SelectedRevisionCard[]>([])
const draggedIndex = ref<number | null>(null)
const isExporting = ref(false)

// 匯出格式選項清單
const exportFormatOptions = [
  { label: 'BigMatrix', value: 'BigMatrix' },
  { label: 'Matrix', value: 'Matrix' }
]

onMounted(() => {
  initExportPath()
  syncCardsFromProjectStore()
})

/**
 * 監聽 projectStore.selectedRevisionIds 變更，與側邊欄 (Sidepanel) 勾選保持雙向同步
 */
watch(
  () => projectStore.selectedRevisionIds,
  () => {
    syncCardsFromProjectStore()
  },
  { deep: true }
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
 * 初始化檔案輸出預設路徑
 */
function initExportPath(): void {
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
 * 從 projectStore 讀取側邊欄勾選的 Revision 並更新 selectedCards 清單
 */
function syncCardsFromProjectStore(): void {
  const currentSelectedIds = new Set(projectStore.selectedRevisionIds)

  // 1. 移除已取消勾選的卡片
  let updatedCards = selectedCards.value.filter(c => currentSelectedIds.has(c.id))

  // 2. 加入新勾選的 Revision 卡片
  const existingCardIds = new Set(updatedCards.map(c => c.id))
  let hasNewCard = false

  for (const id of projectStore.selectedRevisionIds) {
    if (!existingCardIds.has(id)) {
      // 從 projectStore.projects 中尋找對應的 revision 與專案資訊
      const revInfo = findRevisionById(id)
      if (revInfo) {
        const dbCount = revInfo.revision.modelCount || 0
        const savedCount = appStore.seriesInfo?.projectModelCounts?.[revInfo.projectCode]
        const initialCount = (savedCount && savedCount > 0) ? savedCount : (dbCount > 0 ? dbCount : 3)
        const phaseStr = (revInfo.revision.phase || '').trim()
        const verStr = (revInfo.revision.version || '').trim()
        const phaseVer = [phaseStr, verStr].filter(Boolean).join(' ')

        updatedCards.push({
          id: revInfo.revision.id,
          projectId: revInfo.projectId,
          projectCode: revInfo.projectCode,
          phase: phaseStr,
          version: verStr,
          label: phaseVer ? `${revInfo.projectCode} - ${phaseVer}` : revInfo.projectCode,
          modelCount: initialCount,
          dbModelCount: dbCount
        })
        hasNewCard = true
      }
    }
  }

  // 3. 若有新增卡片，依 Series 歷史記錄排序；若只是既有調整則保留拖曳順序
  if (hasNewCard) {
    updatedCards = sortCardsByProjectRecord(updatedCards)
  }

  selectedCards.value = updatedCards
  updateBigMatrixFilenameIfNeed()
}

/**
 * 依據 Revision ID 於 projectStore 中尋找對應的 revision 物件與所屬專案資訊
 * @param {number} revisionId - Revision ID
 */
function findRevisionById(revisionId: number) {
  if (!projectStore.projects) return null
  for (const p of projectStore.projects) {
    if (p.revisions) {
      const r = p.revisions.find(rev => rev.id === revisionId)
      if (r) {
        return {
          revision: r,
          projectId: p.id,
          projectCode: p.code || p.name || `Project ${p.id}`
        }
      }
    }
  }
  return null
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
 * 移除單一卡片，並同步更新 projectStore.selectedRevisionIds（Sidepanel 即時取消勾選）
 * @param {number} index - 欲移除之卡片索引
 */
function removeCard(index: number): void {
  selectedCards.value.splice(index, 1)
  const remainingIds = selectedCards.value.map(c => c.id)
  projectStore.setSelectedRevisionIds(remainingIds)
  updateBigMatrixFilenameIfNeed()
  saveProjectOrderFromCards()
}

// 拖曳排序處理函數 (僅 BigMatrix 模式啟用)
function onDragStart(event: DragEvent, index: number): void {
  if (exportFormat.value.toLowerCase() !== 'bigmatrix') return
  draggedIndex.value = index
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
  }
}

function onDragOver(event: DragEvent, _index: number): void {
  if (exportFormat.value.toLowerCase() !== 'bigmatrix') return
  event.preventDefault()
}

function onDrop(event: DragEvent, dropIndex: number): void {
  if (exportFormat.value.toLowerCase() !== 'bigmatrix') return
  event.preventDefault()
  if (draggedIndex.value === null || draggedIndex.value === dropIndex) return

  const itemToMove = selectedCards.value[draggedIndex.value]
  selectedCards.value.splice(draggedIndex.value, 1)
  selectedCards.value.splice(dropIndex, 0, itemToMove)

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
  if (selectedCards.value.length === 0) return
  isExporting.value = true
  logStore.addLogEntry('INFO', `開始執行匯出作業...`)

  try {
    const sortedRevisionIds = selectedCards.value.map(c => c.id)
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
    logStore.addLogEntry('INFO', `匯出作業完成，共產生 ${exportedPaths?.length || 0} 個檔案`)
    emit('exportSuccess', exportedPaths || [])
  } catch (error) {
    const msg = error instanceof Error ? error.message : String(error)
    logStore.addLogEntry('ERROR', `匯出作業失敗：${msg}`)
  } finally {
    isExporting.value = false
  }
}
</script>

<style scoped>
.export-view-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  padding: 0.65rem 0.85rem;
  gap: 0.5rem;
  background: var(--surface-ground);
}

/* 區塊共通樣式 */
.export-section {
  background: var(--surface-card);
  border: 1px solid var(--surface-border);
  border-radius: 4px;
  padding: 0.55rem 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.45rem;
}

.top-config-section {
  flex-shrink: 0;
  display: grid;
  grid-template-columns: auto 1fr;
  column-gap: 1.25rem;
  align-items: center;
}

.form-row {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.format-row {
  min-width: 170px;
}

.path-row {
  flex: 1;
}

.section-label {
  font-size: 0.72rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-color-secondary);
  display: flex;
  align-items: center;
  gap: 0.4rem;
}

.format-select-btn {
  height: 28px;
}

:deep(.format-select-btn .p-button) {
  padding: 0.2rem 0.6rem !important;
  font-size: 0.8rem !important;
}

.path-input-group {
  display: flex;
  gap: 0.35rem;
  align-items: center;
}

.path-input {
  flex: 1;
  height: 28px;
  font-size: 0.82rem;
  padding: 0.2rem 0.5rem;
}

.browse-btn {
  height: 28px;
  padding: 0 0.6rem;
  font-size: 0.8rem;
  white-space: nowrap;
}

/* 3. Revisions 區塊 (填滿中間剩餘空間，最大化可視面積) */
.revisions-section {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.revisions-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-shrink: 0;
}

.revisions-count-badge {
  background: var(--primary-color);
  color: white;
  padding: 0.05rem 0.35rem;
  border-radius: 10px;
  font-size: 0.68rem;
  font-weight: 600;
}

.selected-cards-container {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
  padding: 0.35rem;
  border: 1px solid var(--surface-border);
  border-radius: 4px;
  background: var(--surface-ground);
}

.empty-revisions-tip {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  height: 100%;
  min-height: 100px;
  color: var(--text-color-secondary);
  font-size: 0.82rem;
}

/* VSCode Style 緊湊卡片 */
.revision-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  box-sizing: border-box;
  padding: 0.35rem 0.6rem;
  background: var(--surface-card);
  border: 1px solid var(--surface-border);
  border-radius: 4px;
  cursor: default;
  user-select: none;
  transition: background-color 0.12s ease, border-color 0.12s ease;
}

.revision-card.is-draggable {
  cursor: grab;
}

.revision-card.is-draggable:active {
  cursor: grabbing;
}

.revision-card:hover {
  border-color: var(--primary-color);
  background: var(--surface-hover, rgba(255, 255, 255, 0.04));
}

.card-left {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  min-width: 0;
  flex: 1;
  overflow: hidden;
}

.drag-handle {
  color: var(--text-color-secondary);
  font-size: 0.75rem;
  cursor: grab;
  padding: 0.1rem;
  flex-shrink: 0;
}

.card-code {
  font-weight: 600;
  font-size: 0.82rem;
  color: var(--text-color);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.card-ver {
  font-size: 0.72rem;
  color: var(--text-color-secondary);
  background: var(--surface-ground);
  padding: 0.08rem 0.35rem;
  border-radius: 3px;
  border: 1px solid var(--surface-border);
  white-space: nowrap;
  flex-shrink: 0;
}

.card-right {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-shrink: 0;
}

.model-count-group {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  flex-shrink: 0;
}

.mc-label {
  font-size: 0.75rem;
  color: var(--text-color-secondary);
  white-space: nowrap;
}

.card-model-input {
  width: 58px !important;
  min-width: 58px !important;
  max-width: 58px !important;
  height: 26px !important;
  flex-shrink: 0;
}

:deep(.card-model-input.p-inputnumber) {
  width: 58px !important;
  min-width: 58px !important;
  max-width: 58px !important;
  height: 26px !important;
  display: inline-flex !important;
}

:deep(.card-model-input .p-inputnumber-input) {
  width: 36px !important;
  min-width: 0 !important;
  height: 24px !important;
  padding: 0 0.2rem !important;
  font-size: 0.78rem !important;
  text-align: center !important;
}

:deep(.card-model-input .p-inputnumber-button) {
  width: 18px !important;
  padding: 0 !important;
  height: 50% !important;
}

:deep(.card-model-input .p-inputnumber-button .p-icon) {
  font-size: 0.6rem !important;
  width: 0.6rem !important;
  height: 0.6rem !important;
}

.remove-card-btn {
  width: 22px !important;
  height: 22px !important;
  min-width: 22px !important;
  padding: 0 !important;
  display: flex !important;
  align-items: center !important;
  justify-content: center !important;
  flex-shrink: 0;
  font-size: 0.75rem !important;
}

.remove-card-btn:hover {
  color: var(--p-red-500, #ef4444) !important;
  background: var(--surface-hover, rgba(239, 68, 68, 0.08)) !important;
}

/* 4 & 5. 底部描述與按鈕區塊 */
.bottom-action-section {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 0.45rem;
}

.desc-input {
  height: 28px;
  font-size: 0.82rem;
  padding: 0.2rem 0.5rem;
}

.action-btn-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.action-left-info {
  font-size: 0.78rem;
  color: var(--text-color-secondary);
}

.exporting-spinner {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  color: var(--primary-color);
}

.export-submit-btn {
  height: 30px;
  padding: 0 1rem;
  font-size: 0.82rem;
  font-weight: 600;
}
</style>
