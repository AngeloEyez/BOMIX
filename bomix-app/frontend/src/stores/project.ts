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
  mode: string
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
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  // Getters
  const selectedProject = computed(() =>
    projects.value.find(p => p.id === selectedProjectId.value)
  )

  const selectedRevision = computed(() => {
    if (!selectedProject.value) return null
    return selectedProject.value.revisions?.find(r => r.id === selectedRevisionId.value)
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
  async function loadProjects(seriesId: number): Promise<void> {
    const logStore = useLogStore()
    logStore.addLogEntry('DEBUG', `[loadProjects] 開始載入專案列表 (seriesId: ${seriesId})`)
    isLoading.value = true
    error.value = null
    try {
      const data = await GetProjects(seriesId)
      logStore.addLogEntry('DEBUG', `[loadProjects] 成功查詢專案 (seriesId: ${seriesId})，共 ${data?.length || 0} 個專案`)
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

  async function loadRevisions(projectId: number): Promise<BomRevision[]> {
    const logStore = useLogStore()
    try {
      const revisions = await GetRevisions(projectId)
      logStore.addLogEntry('DEBUG', `[loadRevisions] 專案 (ID: ${projectId}) 載入 ${revisions?.length || 0} 個版本`)
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

  function selectProject(projectId: number): void {
    selectedProjectId.value = projectId
    selectedRevisionId.value = null
  }

  function selectRevision(revisionId: number): void {
    selectedRevisionId.value = revisionId
  }

  function clearSelection(): void {
    selectedProjectId.value = null
    selectedRevisionId.value = null
  }

  function clearProjects(): void {
    projects.value = []
    clearSelection()
  }

  function clearError(): void {
    error.value = null
  }

  return {
    // State
    projects,
    selectedProjectId,
    selectedRevisionId,
    isLoading,
    error,
    // Getters
    selectedProject,
    selectedRevision,
    currentBom,
    projectTree,
    // Actions
    loadProjects,
    loadRevisions,
    selectProject,
    selectRevision,
    clearSelection,
    clearProjects,
    clearError,
  }
})
