<template>
  <div class="workspace-page">
    <!-- Main Content Panel (依據 appStore.workspaceView 切換顯示 BOMTable 或 ExportView) -->
    <div class="main-content">
      <!-- 1. Export 視圖 -->
      <ExportView
        v-if="appStore.workspaceView === 'export'"
        @exportSuccess="onExportSuccess"
      />

      <!-- 2. BOM Table 視圖 (選取版本時顯示 BOMTable，否則顯示專案 Dashboard) -->
      <template v-else>
        <BOMTable
          v-if="projectStore.selectedRevisionIds.length > 0"
          :revision-ids="projectStore.selectedRevisionIds"
        />
        <div v-else class="placeholder-content">
          <div v-if="projectStore.projects.length === 0" class="empty-state">
            <p>No projects found in this series.</p>
            <Button
              label="Import BOM"
              icon="pi pi-upload"
              text
              severity="secondary"
              @click="appStore.openImportDialog()"
            />
          </div>
          
          <div v-else class="dashboard-stats">
            <div class="projects-list">
              <div class="projects-header">
                <span class="projects-title">Latest Revisions</span>
                <span class="projects-count">{{ projectStore.projects.length }} Projects</span>
              </div>
              <div class="project-items">
                <div v-for="p in sortedProjects" :key="p.id" class="project-item">
                  <div class="project-info">
                    <span class="project-code">{{ p.code || p.name || `Project ${p.id}` }}</span>
                    <span class="project-desc" v-if="getImportDate(p)">{{ getImportDate(p) }}</span>
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
      </template>
    </div>

    <!-- Sub-components Dialogs -->
    <ImportDialog
      v-model:visible="appStore.importDialogVisible"
      @importSuccess="onImportSuccess"
    />
    
    <ImportResultsDialog
      v-model:visible="importResultDialogVisible"
      :results="importResults"
    />

    <CopyMatrixDialog
      v-model:visible="appStore.copyMatrixDialogVisible"
      :allRevisions="allRevisions"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import Button from 'primevue/button'
import { useAppStore, useProjectStore, useLogStore, useTaskStore } from '../stores'
import BOMTable from '../components/BOMTable.vue'
import ImportDialog from '../components/workspace/ImportDialog.vue'
import ImportResultsDialog from '../components/workspace/ImportResultsDialog.vue'
import ExportView, { type RevisionOption } from '../components/workspace/ExportView.vue'
import CopyMatrixDialog from '../components/workspace/CopyMatrixDialog.vue'
import type { ImportResult as BackendImportResult } from '../services/api'
import type { Project, BomRevision } from '../stores/project'

const appStore = useAppStore()
const projectStore = useProjectStore()
const logStore = useLogStore()
const taskStore = useTaskStore()

const importResultDialogVisible = ref(false)

// 匯入結果與版本選項列表
const importResults = ref<BackendImportResult[]>([])
const allRevisions = ref<RevisionOption[]>([])

/**
 * 依據專案最新 BOM 版本的 ID 降冪排序，最新匯入的專案排在最前面
 */
const sortedProjects = computed(() => {
  if (!projectStore.projects || projectStore.projects.length === 0) return []
  return [...projectStore.projects].sort((a, b) => {
    const revA = getLatestRevisionObj(a)
    const revB = getLatestRevisionObj(b)

    // 若均有 Revision，依 Revision ID (或 CreatedAt / Date) 降冪排序 (ID 越大表示越新建立/匯入)
    if (revA && revB) {
      if (revA.id !== revB.id) {
        return revB.id - revA.id
      }
      const timeA = revA.createdAt ? new Date(revA.createdAt).getTime() : 0
      const timeB = revB.createdAt ? new Date(revB.createdAt).getTime() : 0
      if (timeA !== timeB) {
        return timeB - timeA
      }
    } else if (revA) {
      return -1
    } else if (revB) {
      return 1
    }

    return b.id - a.id
  })
})

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
 * 取得專案最新版本的 BomRevision 物件
 * @param {Project} project - 專案物件
 * @returns {BomRevision | null} 最新版本物件，若無版本則回傳 null
 */
function getLatestRevisionObj(project: Project): BomRevision | null {
  if (!project.revisions || project.revisions.length === 0) return null
  const sorted = [...project.revisions].sort((a, b) => b.id - a.id)
  return sorted[0]
}

/**
 * 取得專案最新版本的名稱標籤 (Phase + Version)
 * @param {Project} project - 專案物件
 * @returns {string | null} 最新版本標籤字串，若無版本則回傳 null
 */
function getLatestRevision(project: Project): string | null {
  const latest = getLatestRevisionObj(project)
  if (!latest) return null
  return `${latest.phase} ${latest.version}`
}

/**
 * 取得專案最新版本的匯入日期字串
 * @param {Project} project - 專案物件
 * @returns {string} 格式化後的匯入日期字串
 */
function getImportDate(project: Project): string {
  const latest = getLatestRevisionObj(project)
  const rawDate = latest?.createdAt || latest?.date || project.createdAt || project.updatedAt
  if (!rawDate) return ''

  try {
    const d = new Date(rawDate)
    if (!isNaN(d.getTime())) {
      const year = d.getFullYear()
      const month = String(d.getMonth() + 1).padStart(2, '0')
      const day = String(d.getDate()).padStart(2, '0')
      const hours = String(d.getHours()).padStart(2, '0')
      const minutes = String(d.getMinutes()).padStart(2, '0')
      if (hours === '00' && minutes === '00' && rawDate.length <= 10) {
        return `${year}-${month}-${day}`
      }
      return `${year}-${month}-${day} ${hours}:${minutes}`
    }
  } catch (_) {}
  return rawDate
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
 * 處理匯出完成事件
 * @param {string[]} paths - 匯出檔案路徑清單
 */
function onExportSuccess(paths: string[]): void {
  logStore.addLogEntry('INFO', `匯出作業成功完成，檔案路徑: ${paths.join(', ')}`)
}
</script>

<style scoped>
.workspace-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  width: 100%;
  min-width: 0;
  overflow: hidden;
}

.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  height: 100%;
  width: 100%;
  min-width: 0;
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
