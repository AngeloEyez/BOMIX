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
      header="Import BOM"
      :style="{ width: '500px' }"
    >
      <div class="import-dialog-content">
        <p class="drag-hint">
          Select Excel files to import (EBOM, BigMatrix formats)
        </p>

        <InputText
          v-model="importFilePath"
          placeholder="Select files..."
          class="file-path-input"
          readonly
        />

        <div class="button-group">
          <Button
            label="Browse Files"
            icon="pi pi-folder-open"
            @click="browseFiles"
          />
        </div>

        <div v-if="importFilePaths.length > 0" class="file-list">
          <div class="file-item" v-for="(path, idx) in importFilePaths" :key="idx">
            <i class="pi pi-file"></i>
            <span>{{ path }}</span>
          </div>
        </div>

        <div class="import-options">
          <Checkbox
            v-model="confirmOverwrite"
            inputId="confirmOverwrite"
            :binary="true"
          />
          <label for="confirmOverwrite">Confirm before overwriting existing BOM</label>
        </div>
      </div>

      <template #footer>
        <Button
          label="Cancel"
          icon="pi pi-times"
          text
          @click="importDialogVisible = false"
        />
        <Button
          label="Import"
          icon="pi pi-check"
          @click="executeImport"
          :disabled="importFilePaths.length === 0"
        />
      </template>
    </Dialog>

    <!-- Import Results Dialog -->
    <Dialog
      v-model:visible="importResultDialogVisible"
      modal
      header="Import Results"
      :style="{ width: '600px' }"
    >
      <div class="import-results">
        <div v-if="importResults.length > 0" class="results-list">
          <div
            v-for="(result, idx) in importResults"
            :key="idx"
            class="result-item"
            :class="{ 'result-error': result.status === 'failed' }"
          >
            <i :class="['pi', result.status === 'completed' ? 'pi-check-circle' : 'pi-times-circle']"></i>
            <span class="result-name">{{ result.fileName }}</span>
            <span class="result-status">{{ result.status }}</span>
            <span v-if="result.partsCount" class="result-count">
              {{ result.partsCount }} parts
            </span>
          </div>
        </div>
        <div v-else class="no-results">
          No results to display
        </div>
      </div>

      <template #footer>
        <Button
          label="Close"
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
                  <span class="card-ver">{{ card.phase }} {{ card.version }}</span>
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

        <!-- Matrix specific options -->
        <div v-if="exportFormat.toLowerCase() === 'matrix'" class="export-options">
          <div class="form-group">
            <label for="exportOutputDir">Output Directory</label>
            <InputText
              v-model="exportOutputPath"
              placeholder="Select output directory"
              id="exportOutputDir"
              readonly
            />
            <Button
              label="Browse"
              icon="pi pi-folder-open"
              class="p-button-text"
              @click="browseOutputDir"
            />
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
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import MultiSelect from 'primevue/multiselect'
import Select from 'primevue/select'
import InputNumber from 'primevue/inputnumber'
import InputText from 'primevue/inputtext'
import Checkbox from 'primevue/checkbox'
import { useAppStore, useProjectStore, useLogStore } from '../stores'
import BOMTable from '../components/BOMTable.vue'
import {
  ImportExcel,
  ExportExcel,
  OpenFileDialog,
  SelectFolderDialog,
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

onMounted(() => {
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
            list.push({
              id: r.id,
              projectId: p.id,
              projectCode: pCode,
              phase: r.phase,
              version: r.version,
              label: `${pCode} - ${r.phase} ${r.version}`,
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
}

function removeCard(index: number): void {
  selectedCards.value.splice(index, 1)
  exportRevisions.value = selectedCards.value.map(c => c.id)
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
}

// Import functions
function openImportDialog(): void {
  importFilePaths.value = []
  importFilePath.value = ''
  importResults.value = []
  importDialogVisible.value = true
}

async function browseFiles(): Promise<void> {
  try {
    const filePath = await OpenFileDialog({
      title: 'Select BOM files to import',
      filters: [
        { name: 'Excel Files', extensions: ['xlsx', 'xls'] }
      ],
      selectFiles: true,
      multiSelect: true
    })

    if (filePath) {
      importFilePath.value = filePath
      importFilePaths.value = [filePath]
      logStore.addLogEntry('DEBUG', `已選取匯入檔案：${filePath}`)
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
    importResultDialogVisible.value = true
    importDialogVisible.value = false
  } catch (error) {
    const msg = error instanceof Error ? error.message : String(error)
    logStore.addLogEntry('ERROR', `匯入作業失敗：${msg}`)
  }
}

// Export functions
function openExportDialog(): void {
  loadProjects()
  exportRevisions.value = []
  selectedCards.value = []
  exportDescription.value = ''
  exportOutputPath.value = ''
  exportDialogVisible.value = true
}

async function browseOutputDir(): Promise<void> {
  try {
    const dirPath = await SelectFolderDialog({
      title: 'Select output directory'
    })

    if (dirPath) {
      exportOutputPath.value = dirPath
      logStore.addLogEntry('INFO', `已選取匯出目錄：${dirPath}`)
    }
  } catch (error) {
    const msg = error instanceof Error ? error.message : String(error)
    logStore.addLogEntry('ERROR', `選取匯出目錄發生錯誤：${msg}`)
  }
}

async function executeExport(): Promise<void> {
  logStore.addLogEntry('INFO', `開始執行匯出作業...`)
  try {
    const sortedRevisionIds = selectedCards.value.map(c => c.id)
    const modelCountOverrides: Record<string, number> = {}
    selectedCards.value.forEach(card => {
      modelCountOverrides[String(card.id)] = card.modelCount
    })

    const options: ExportOptions = {
      format: exportFormat.value,
      revisionIds: sortedRevisionIds,
      description: exportDescription.value,
      outputPath: exportOutputPath.value,
      outputDir: exportOutputPath.value,
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
  height: 100vh;
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

.file-path-input {
  flex: 1;
}

.button-group {
  display: flex;
  gap: 0.5rem;
}

.file-list {
  max-height: 150px;
  overflow-y: auto;
  border: 1px solid var(--surface-border);
  border-radius: 4px;
  padding: 0.5rem;
}

.file-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.25rem 0;
  font-size: 0.875rem;
}

.import-options {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

/* Import Results */
.import-results {
  max-height: 300px;
  overflow-y: auto;
}

.results-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.result-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem;
  background: var(--surface-ground);
  border-radius: 4px;
}

.result-error {
  background: #fee2e2;
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
