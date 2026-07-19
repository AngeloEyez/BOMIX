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

    <!-- Main Content with Splitter -->
    <Splitter
      class="workspace-splitter"
      :style="{ height: `calc(100% - ${toolbarHeight}px)` }"
    >
      <!-- Sidebar Panel -->
      <SplitterPanel
        :size="sidebarWidth"
        :min-size="0"
        @resize="onSidebarResize"
        @dblclick="resetSidebarWidth"
      >
        <div class="sidebar">
          <div class="sidebar-header">
            <span class="sidebar-title">Projects</span>
          </div>
          <div class="sidebar-content">
            <Tree
              :value="projectTree"
              :expanded-keys="expandedKeys"
              :selection-keys="selectionKeys"
              selection-mode="single"
              @node-select="onNodeSelect"
              @node-toggle="onNodeToggle"
            >
              <template #node="slotProps">
                <div class="tree-node">
                  <span v-if="slotProps.node.type === 'project'" class="node-icon">
                    <i class="pi pi-folder"></i>
                  </span>
                  <span v-if="slotProps.node.type === 'revision'" class="node-icon">
                    <i class="pi pi-version"></i>
                  </span>
                  <span class="node-label">{{ slotProps.node.label }}</span>
                </div>
              </template>
            </Tree>
          </div>
        </div>
      </SplitterPanel>

      <!-- Main Content Panel -->
      <SplitterPanel :size="100 - sidebarWidth">
        <div class="main-content">
          <BOMTable
            v-if="projectStore.selectedRevision"
            :revision-id="projectStore.selectedRevision?.id"
          />
          <div v-else class="placeholder-content">
            <i class="pi pi-box"></i>
            <h2>BOM Workspace</h2>
            <p>Select a project and revision from the sidebar to view BOM data.</p>
          </div>
        </div>
      </SplitterPanel>
    </Splitter>

    <!-- Bottom Log Panel -->
    <Splitter
      class="bottom-splitter"
      :style="{ height: `${bottomPanelHeight}px` }"
      @resize="onBottomResize"
      @dblclick="resetBottomHeight"
    >
      <SplitterPanel :size="100 - bottomPanelHeight" :min-size="50">
        <!-- Empty panel above log panel -->
      </SplitterPanel>
      <SplitterPanel :size="bottomPanelHeight" :min-size="30">
        <LogPanel />
      </SplitterPanel>
    </Splitter>

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
          />
        </div>

        <!-- BigMatrix specific options -->
        <div v-if="exportFormat === 'bigmatrix'" class="export-options">
          <div class="form-group">
            <label for="exportDescription">Description (optional)</label>
            <InputText
              v-model="exportDescription"
              placeholder="Enter description"
              id="exportDescription"
            />
          </div>

          <div class="form-group">
            <label for="exportModelCount">Model Count</label>
            <InputNumber
              v-model="exportModelCount"
              placeholder="1"
              id="exportModelCount"
              :min="1"
            />
          </div>
        </div>

        <!-- Matrix specific options -->
        <div v-if="exportFormat === 'matrix'" class="export-options">
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
import { ref, computed, onMounted, watch } from 'vue'
import Splitter from 'primevue/splitter'
import SplitterPanel from 'primevue/splitterpanel'
import Tree from 'primevue/tree'
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import MultiSelect from 'primevue/multiselect'
import Select from 'primevue/select'
import InputNumber from 'primevue/inputnumber'
import InputText from 'primevue/inputtext'
import Checkbox from 'primevue/checkbox'
import { useAppStore, useProjectStore } from '../stores'
import LogPanel from '../components/LogPanel.vue'
import BOMTable from '../components/BOMTable.vue'
import {
  ImportExcel,
  ExportExcel,
  OpenFileDialog,
  SelectFolderDialog,
  type ImportResult as BackendImportResult,
  type ExportOptions
} from '../services/api'

const appStore = useAppStore()
const projectStore = useProjectStore()

// Toolbar height
const toolbarHeight = 50

// Sidebar width management
const sidebarWidth = ref(20) // Default 20%
const sidebarWidthPx = ref(200) // Actual pixel width

// Bottom panel height
const bottomPanelHeight = ref(200) // Default 200px

// Import dialog
const importDialogVisible = ref(false)
const importFilePath = ref('')
const importFilePaths = ref<string[]>([])
const importResults = ref<BackendImportResult[]>([])
const importResultDialogVisible = ref(false)
const confirmOverwrite = ref(false)

// Export dialog
const exportDialogVisible = ref(false)
const exportFormat = ref('bigmatrix')
const exportRevisions = ref<number[]>([])
const exportDescription = ref('')
const exportModelCount = ref(1)
const exportOutputPath = ref('')
const allRevisions = ref<any[]>([])

// Tree data
interface TreeNode {
  key: string
  label: string
  type: 'project' | 'revision'
  data?: any
  children?: TreeNode[]
}

const projectTree = ref<TreeNode[]>([])

// Tree selection
const expandedKeys = ref<Record<string, boolean>>({})
const selectionKeys = ref<Record<string, string>>({})

// Export format options
const exportFormatOptions = [
  { label: 'BigMatrix', value: 'bigmatrix' },
  { label: 'Matrix', value: 'matrix' }
]

onMounted(() => {
  // Load initial data if series is open
  if (appStore.isOpen) {
    loadProjects()
  }
})

// Watch for series open changes
watch(() => appStore.isOpen, (isOpen) => {
  if (isOpen) {
    loadProjects()
  } else {
    projectTree.value = []
  }
})

async function loadProjects(): Promise<void> {
  try {
    // This will be implemented when we have the series ID
    // For now, we'll use a placeholder
    projectTree.value = [
      {
        key: '0',
        label: 'Sample Project',
        type: 'project',
        children: [
          {
            key: '0-0',
            label: 'PV 0.1',
            type: 'revision',
            data: { id: 1, phase: 'PV', version: '0.1' },
          },
          {
            key: '0-1',
            label: 'PV 0.2',
            type: 'revision',
            data: { id: 2, phase: 'PV', version: '0.2' },
          },
        ],
      },
    ]

    // Populate allRevisions for export
    allRevisions.value = [
      { id: 1, label: 'PV 0.1', phase: 'PV', version: '0.1' },
      { id: 2, label: 'PV 0.2', phase: 'PV', version: '0.2' },
    ]
  } catch (error) {
    console.error('Failed to load projects:', error)
  }
}

function onSidebarResize(event: any): void {
  sidebarWidth.value = event.size || event
}

function resetSidebarWidth(): void {
  sidebarWidth.value = 20
}

function onBottomResize(event: any): void {
  bottomPanelHeight.value = event.size || event
}

function resetBottomHeight(): void {
  bottomPanelHeight.value = 200
}

function onNodeSelect(node: any): void {
  if (node.type === 'project') {
    projectStore.selectProject(parseInt(node.key))
  } else if (node.type === 'revision') {
    projectStore.selectRevision(parseInt(node.key))
  }
}

function onNodeToggle(_node: any): void {
  // Tree handles this internally with expandedKeys
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
    }
  } catch (error) {
    console.error('Failed to browse files:', error)
  }
}

async function executeImport(): Promise<void> {
  try {
    const results = await ImportExcel(importFilePaths.value)
    importResults.value = results
    importResultDialogVisible.value = true
    importDialogVisible.value = false

    // Reload BOM data if a revision is selected
    if (projectStore.selectedRevision) {
      // Trigger reload
    }
  } catch (error) {
    console.error('Import failed:', error)
  }
}

// Export functions
function openExportDialog(): void {
  exportRevisions.value = []
  exportDescription.value = ''
  exportModelCount.value = 1
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
    }
  } catch (error) {
    console.error('Failed to browse directory:', error)
  }
}

async function executeExport(): Promise<void> {
  try {
    const options: ExportOptions = {
      format: exportFormat.value,
      revisionIds: exportRevisions.value,
      description: exportDescription.value,
      outputPath: exportOutputPath.value,
      outputDir: exportOutputPath.value
    }

    await ExportExcel(options)
    exportDialogVisible.value = false
  } catch (error) {
    console.error('Export failed:', error)
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

/* Top Toolbar */
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

/* Workspace Splitter */
.workspace-splitter {
  flex: 1;
  overflow: hidden;
}

/* Sidebar */
.sidebar {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.sidebar-header {
  padding: 1rem;
  border-bottom: 1px solid var(--surface-border);
}

.sidebar-title {
  font-weight: 600;
  font-size: 0.875rem;
  color: var(--text-color-secondary);
  text-transform: uppercase;
}

.sidebar-content {
  flex: 1;
  overflow-y: auto;
  padding: 0.5rem;
}

/* Tree Node */
.tree-node {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.25rem 0;
}

.node-icon {
  color: var(--primary-color);
  width: 1.25rem;
  text-align: center;
}

.node-label {
  flex: 1;
}

/* Main Content */
.main-content {
  height: 100%;
  overflow: auto;
  background: var(--surface-ground);
}

.placeholder-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  text-align: center;
  color: var(--text-color-secondary);
}

.placeholder-content i {
  font-size: 4rem;
  margin-bottom: 1rem;
  color: var(--surface-border);
}

.placeholder-content h2 {
  font-size: 1.5rem;
  margin: 0 0 0.5rem;
  color: var(--text-color);
}

.placeholder-content p {
  margin: 0 0 0.5rem;
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
</style>
