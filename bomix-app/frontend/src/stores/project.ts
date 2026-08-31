import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { GetProjects, GetRevisions } from '../services/api'
import { useLogStore } from './log'

export interface Project {
  id: number
  seriesId: number
  name?: string
  code: string
  description: string
  createdAt: string
  updatedAt: string
  revisions?: BomRevision[]
}

export interface TreeNode {
  key: string
  label: string
  type: 'project' | 'revision'
  data?: any
  children?: TreeNode[]
}

export interface BomRevision {
  id: number
  projectId: number
  phase: string
  version: string
  description: string
  schematicVersion: string
  pcbVersion: string
  pcaPn: string
  date: string
  sourceFile: string
  modelCount?: number
  createdAt: string
  updatedAt: string
}

export const useProjectStore = defineStore('project', () => {
  // State
  const projects = ref<Project[]>([])
  const selectedProjectId = ref<number | null>(null)
  const selectedRevisionId = ref<number | null>(null)
  const selectedRevisionIds = ref<number[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  // Getters
  const selectedProject = computed(() =>
    projects.value.find(p => p.id === selectedProjectId.value)
  )

  /**
   * 取得主選取的單一 Revision（多選時回傳第 1 個選取的項目，維持向下相容）
   */
  const selectedRevision = computed<BomRevision | null>(() => {
    const targetId = selectedRevisionIds.value.length > 0 ? selectedRevisionIds.value[0] : selectedRevisionId.value
    if (!targetId) return null

    for (const p of projects.value) {
      const found = p.revisions?.find(r => r.id === targetId)
      if (found) return found
    }
    return null
  })

  /**
   * 取得所有被選取的 BomRevision 物件陣列
   */
  const selectedRevisions = computed<BomRevision[]>(() => {
    if (selectedRevisionIds.value.length === 0) return []
    const set = new Set(selectedRevisionIds.value)
    const result: BomRevision[] = []
    for (const p of projects.value) {
      if (p.revisions) {
        for (const r of p.revisions) {
          if (set.has(r.id)) {
            result.push(r)
          }
        }
      }
    }
    return result
  })

  const currentBom = computed(() => {
    // This will be populated when BOM data is loaded
    return null
  })

  const projectTree = computed<TreeNode[]>(() => {
    return projects.value.map(project => ({
      key: `p_${project.id}`,
      label: project.name || project.code || `Project ${project.id}`,
      type: 'project',
      data: project,
      children: (project.revisions || []).map(rev => ({
        key: `${rev.id}`,
        label: `${rev.phase} ${rev.version}`,
        type: 'revision',
        data: rev
      }))
    }))
  })

  // Actions
  /**
   * 載入特定系列下的所有專案及其 BOM 版本清單
   * @param {number} seriesId - 系列 ID
   */
  async function loadProjects(seriesId: number): Promise<void> {
    const logStore = useLogStore()
    //logStore.addLogEntry('DEBUG', `[loadProjects] 開始載入專案列表 (seriesId: ${seriesId})`)
    isLoading.value = true
    error.value = null
    try {
      const data = await GetProjects(seriesId)
      //logStore.addLogEntry('DEBUG', `[loadProjects] 成功查詢專案 (seriesId: ${seriesId})，共 ${data?.length || 0} 個專案`)
      projects.value = data || []

      // Auto-load revisions for each project to populate the tree
      for (const p of projects.value) {
        await loadRevisions(p.id)
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err)
      logStore.addLogEntry('ERROR', `[loadProjects] 載入專案列表失敗 (seriesId: ${seriesId})：${msg}`)
      error.value = msg
      throw err
    } finally {
      isLoading.value = false
    }
  }

  /**
   * 載入指定專案的所有 BOM 版本
   * @param {number} projectId - 專案 ID
   * @returns {Promise<BomRevision[]>} 版本清單陣列
   */
  async function loadRevisions(projectId: number): Promise<BomRevision[]> {
    const logStore = useLogStore()
    try {
      const revisions = await GetRevisions(projectId)
      //logStore.addLogEntry('DEBUG', `[loadRevisions] 專案 (ID: ${projectId}) 載入 ${revisions?.length || 0} 個版本`)
      // Update the project with revisions
      const project = projects.value.find(p => p.id === projectId)
      if (project) {
        project.revisions = revisions
      }
      return revisions
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err)
      logStore.addLogEntry('ERROR', `[loadRevisions] 載入版本失敗 (projectId: ${projectId})：${msg}`)
      console.error('Failed to load revisions:', err)
      return []
    }
  }

  /**
   * 選取指定專案，並清除版本多選狀態
   * @param {number} projectId - 專案 ID
   */
  function selectProject(projectId: number): void {
    selectedProjectId.value = projectId
    selectedRevisionId.value = null
    selectedRevisionIds.value = []
  }

  /**
   * 設定多選的 BOM Revision ID 清單，並自動同步單選相容狀態與所屬專案 ID
   * @param {number[]} revisionIds - 選取的 BOM Revision ID 陣列
   */
  function setSelectedRevisionIds(revisionIds: number[]): void {
    selectedRevisionIds.value = [...revisionIds]
    if (revisionIds.length > 0) {
      selectedRevisionId.value = revisionIds[0]
      // 尋找第一個選取 revision 所屬的專案
      for (const p of projects.value) {
        if (p.revisions?.some(r => r.id === revisionIds[0])) {
          selectedProjectId.value = p.id
          break
        }
      }
    } else {
      selectedRevisionId.value = null
    }

    const logStore = useLogStore()
    logStore.addLogEntry('DEBUG', `[setSelectedRevisionIds] 目前選取 ${revisionIds.length} 個版本: [${revisionIds.join(', ')}]`)
  }

  /**
   * 選取單一 BOM Revision，支援單選或多選切換模式
   * @param {number} revisionId - BOM Revision ID
   * @param {boolean} [isMulti=false] - 是否為多選累加模式
   */
  function selectRevision(revisionId: number, isMulti = false): void {
    if (isMulti) {
      const idx = selectedRevisionIds.value.indexOf(revisionId)
      const newIds = [...selectedRevisionIds.value]
      if (idx >= 0) {
        newIds.splice(idx, 1)
      } else {
        newIds.push(revisionId)
      }
      setSelectedRevisionIds(newIds)
    } else {
      setSelectedRevisionIds([revisionId])
    }
  }

  /**
   * 清除所有專案與版本選取狀態
   */
  function clearSelection(): void {
    selectedProjectId.value = null
    selectedRevisionId.value = null
    selectedRevisionIds.value = []
  }

  /**
   * 清除專案列表與選取狀態
   */
  function clearProjects(): void {
    projects.value = []
    clearSelection()
  }

  /**
   * 清除錯誤訊息
   */
  function clearError(): void {
    error.value = null
  }

  return {
    // State
    projects,
    selectedProjectId,
    selectedRevisionId,
    selectedRevisionIds,
    isLoading,
    error,
    // Getters
    selectedProject,
    selectedRevision,
    selectedRevisions,
    currentBom,
    projectTree,
    // Actions
    loadProjects,
    loadRevisions,
    selectProject,
    selectRevision,
    setSelectedRevisionIds,
    clearSelection,
    clearProjects,
    clearError,
  }
})
