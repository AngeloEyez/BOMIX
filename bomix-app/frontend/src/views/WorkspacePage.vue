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
          @click="importDialogVisible = true"
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
          @click="copyMatrixDialogVisible = true"
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
        <div v-if="projectStore.projects.length === 0" class="empty-state">
          <p>No projects found in this series.</p>
          <Button label="Import BOM" icon="pi pi-upload" @click="importDialogVisible = true" class="p-button-outlined" />
        </div>
        
        <div v-else class="dashboard-stats">
          <div class="projects-list">
            <div class="projects-header">
              <span class="projects-title">Latest Revisions</span>
              <span class="projects-count">{{ projectStore.projects.length }} Projects</span>
            </div>
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

    <!-- Sub-components Dialogs -->
    <ImportDialog
      v-model:visible="importDialogVisible"
      @importSuccess="onImportSuccess"
    />
    
    <ImportResultsDialog
      v-model:visible="importResultDialogVisible"
      :results="importResults"
    />

    <ExportDialog
      v-model:visible="exportDialogVisible"
      :allRevisions="allRevisions"
    />

    <CopyMatrixDialog
      v-model:visible="copyMatrixDialogVisible"
      :allRevisions="allRevisions"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Button from 'primevue/button'
import { useAppStore, useProjectStore, useLogStore, useTaskStore } from '../stores'
import BOMTable from '../components/BOMTable.vue'
import ImportDialog from '../components/workspace/ImportDialog.vue'
import ImportResultsDialog from '../components/workspace/ImportResultsDialog.vue'
import ExportDialog, { type RevisionOption } from '../components/workspace/ExportDialog.vue'
import CopyMatrixDialog from '../components/workspace/CopyMatrixDialog.vue'
import type { ImportResult as BackendImportResult } from '../services/api'
import type { Project } from '../stores/project'

const appStore = useAppStore()
const projectStore = useProjectStore()
const logStore = useLogStore()
const taskStore = useTaskStore()

// 對話框顯示控制狀態
const importDialogVisible = ref(false)
const importResultDialogVisible = ref(false)
const exportDialogVisible = ref(false)
const copyMatrixDialogVisible = ref(false)

// 匯入結果與版本選項列表
const importResults = ref<BackendImportResult[]>([])
const allRevisions = ref<RevisionOption[]>([])

onMounted(() => {
  taskStore.startListening()
  if (appStore.isOpen) {
    loadProjects()
  }
})

/**
 * 載入當前 Series 下所有專案與 Revisions 選項
 */
async function loadProjects(): Promise<void> {
  try {
    const list: RevisionOption[] = []
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
 * 取得專案最新版本的名稱標籤 (Phase + Version)
 * @param {Project} project - 專案物件
 * @returns {string | null} 最新版本標籤字串，若無版本則回傳 null
 */
function getLatestRevision(project: Project): string | null {
  if (!project.revisions || project.revisions.length === 0) return null
  const sorted = [...project.revisions].sort((a, b) => b.id - a.id)
  const latest = sorted[0]
  return `${latest.phase} ${latest.version}`
}

/**
 * 處理匯入完成事件，彈出進度監控對話框
 * @param {BackendImportResult[]} results - 匯入結果項目清單
 */
function onImportSuccess(results: BackendImportResult[]): void {
  importResults.value = results
  importResultDialogVisible.value = true
}

/**
 * 開啟匯出對話框前重新載入專案資料
 */
function openExportDialog(): void {
  loadProjects()
  exportDialogVisible.value = true
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
  padding: 0.75rem 1rem;
  color: var(--text-color);
  overflow-y: auto;
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
  width: 100%;
}

.projects-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.projects-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.25rem 0.25rem 0.5rem 0.25rem;
  border-bottom: 1px solid var(--surface-border);
  margin-bottom: 0.25rem;
}

.projects-title {
  font-size: 0.75rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--text-color-secondary);
}

.projects-count {
  font-size: 0.75rem;
  color: var(--text-color-secondary);
}

.project-items {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.project-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--surface-card);
  padding: 0.35rem 0.65rem;
  border-radius: 4px;
  border: 1px solid var(--surface-border);
  transition: background-color 0.15s ease, border-color 0.15s ease;
}

.project-item:hover {
  border-color: var(--primary-color);
  background: var(--surface-hover, rgba(255, 255, 255, 0.03));
}

.project-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.project-code {
  font-weight: 600;
  font-size: 0.85rem;
}

.project-desc {
  font-size: 0.78rem;
  color: var(--text-color-secondary);
}

.latest-rev {
  background: var(--primary-color);
  color: white;
  padding: 0.125rem 0.4rem;
  border-radius: 3px;
  font-size: 0.75rem;
  font-weight: 600;
  line-height: 1.2;
}

.no-rev {
  color: var(--text-color-secondary);
  font-size: 0.75rem;
  font-style: italic;
}
</style>
