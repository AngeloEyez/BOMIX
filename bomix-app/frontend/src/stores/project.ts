import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { GetProjects, GetRevisions } from '../services/api'

export interface Project {
  id: number
  seriesId: number
  code: string
  description: string
  createdAt: string
  updatedAt: string
  revisions?: BomRevision[]
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

  // Actions
  async function loadProjects(seriesId: number): Promise<void> {
    isLoading.value = true
    error.value = null
    try {
      const data = await GetProjects(seriesId)
      projects.value = data
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to load projects'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function loadRevisions(projectId: number): Promise<BomRevision[]> {
    try {
      const revisions = await GetRevisions(projectId)
      // Update the project with revisions
      const project = projects.value.find(p => p.id === projectId)
      if (project) {
        project.revisions = revisions
      }
      return revisions
    } catch (err) {
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
