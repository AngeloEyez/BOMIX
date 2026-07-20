<template>
  <div class="app-container">
    <!-- Header / Title Bar -->
    <header class="header">
      <div class="header-left">
        <div class="logo">
          <i class="pi pi-box"></i>
          <span class="logo-text">BOMIX</span>
        </div>
        <span v-if="appStore.seriesInfo" class="series-title">
          - {{ appStore.seriesInfo.name }}
        </span>
      </div>
      <div class="header-right">
        <Button
          v-if="appStore.isOpen"
          icon="pi pi-sign-out"
          text
          severity="secondary"
          @click="handleCloseSeries"
          title="Close Series"
        />
        <Button
          icon="pi pi-home"
          label="BOM"
          text
          severity="secondary"
          @click="$router.push(appStore.isOpen ? '/workspace' : '/')"
          title="Main View"
        />
        <Button
          icon="pi pi-cog"
          text
          severity="secondary"
          @click="$router.push('/settings')"
          title="Settings"
        />
      </div>
    </header>

    <!-- Main Content with Splitter -->
    <Splitter class="main-splitter">
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
        <router-view />
      </SplitterPanel>
    </Splitter>

    <!-- Bottom Log Panel -->
    <div class="bottom-panel" :style="{ height: `${bottomPanelHeight}px` }">
      <div
        class="resize-handle"
        @mousedown="startBottomResize"
        @dblclick="resetBottomHeight"
      >
        <i class="pi pi-bars"></i>
      </div>
      <LogPanel />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import Splitter from 'primevue/splitter'
import SplitterPanel from 'primevue/splitterpanel'
import Tree from 'primevue/tree'
import Button from 'primevue/button'
import { useAppStore, useProjectStore, useLogStore, useTaskStore } from './stores'
import LogPanel from './components/LogPanel.vue'

const router = useRouter()
const appStore = useAppStore()
const projectStore = useProjectStore()
const logStore = useLogStore()
const taskStore = useTaskStore()

// Header height
const headerHeight = 40

// Sidebar width management
const sidebarWidth = ref(20) // Default 20%

// Bottom panel height
const bottomPanelHeight = ref(200) // Default 200px
let isResizingBottom = false
let startY = 0
let startHeight = 0

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

onMounted(() => {
  // Start listening to events
  logStore.startListening()
  taskStore.startListening()

  // Load initial data if series is open
  if (appStore.isOpen) {
    loadProjects()
  }

  // Check for auto-open last file
  checkAutoOpen()
})

// Watch for series open changes
watch(() => appStore.isOpen, (isOpen) => {
  if (isOpen) {
    loadProjects()
    router.push('/workspace')
  } else {
    projectTree.value = []
    router.push('/')
  }
})

async function loadProjects(): Promise<void> {
  // This will be implemented when we have the series ID
  try {
    // const projects = await GetProjects(seriesId)
    // Build tree from projects
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
            data: { phase: 'PV', version: '0.1' },
          },
          {
            key: '0-1',
            label: 'PV 0.2',
            type: 'revision',
            data: { phase: 'PV', version: '0.2' },
          },
        ],
      },
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

function startBottomResize(event: MouseEvent): void {
  isResizingBottom = true
  startY = event.clientY
  startHeight = bottomPanelHeight.value
  document.addEventListener('mousemove', handleBottomResize)
  document.addEventListener('mouseup', stopBottomResize)
}

function handleBottomResize(event: MouseEvent): void {
  if (!isResizingBottom) return
  const deltaY = event.clientY - startY
  const newHeight = startHeight - deltaY
  // Minimum height is 30px, maximum is 50% of viewport
  bottomPanelHeight.value = Math.max(30, Math.min(newHeight, window.innerHeight * 0.5))
}

function stopBottomResize(): void {
  isResizingBottom = false
  document.removeEventListener('mousemove', handleBottomResize)
  document.removeEventListener('mouseup', stopBottomResize)
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

async function handleCloseSeries(): Promise<void> {
  await appStore.closeSeries()
  projectStore.clearProjects()
}

async function checkAutoOpen(): Promise<void> {
  // This will check the settings and auto-open the last file if enabled
  // Implementation depends on settings being loaded
}
</script>

<style>
/* Global styles */
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html,
body {
  height: 100%;
  overflow: hidden;
}

#app {
  height: 100%;
}

/* PrimeVue overrides */
:root {
  --p-primary-500: #6366f1;
  --p-primary-600: #4f46e5;
}
</style>

<style scoped>
.app-container {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: var(--surface-ground);
}

/* Header */
.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  height: 36px;
  padding: 0 0.5rem;
  background: var(--surface-card);
  border-bottom: 1px solid var(--surface-border);
  flex-shrink: 0;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.logo {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.logo i {
  font-size: 1.5rem;
  color: var(--primary-color);
}

.logo-text {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--text-color);
  letter-spacing: 0.05em;
}

.series-title {
  font-size: 0.9rem;
  color: var(--text-color-secondary);
}

.header-right {
  display: flex;
  gap: 0.25rem;
}

/* Main Splitter */
.main-splitter {
  flex: 1;
  overflow: hidden;
  border: none;
}

:deep(.p-splitter-gutter) {
  background-color: var(--surface-border) !important;
  width: 4px !important;
  transition: background-color 0.2s;
  cursor: col-resize;
  border-left: 1px solid var(--surface-hover);
  border-right: 1px solid var(--surface-hover);
}

:deep(.p-splitter-gutter:hover) {
  background-color: var(--primary-color) !important;
}

/* Sidebar */
.sidebar {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.sidebar-header {
  padding: 0.25rem 0.5rem;
  border-bottom: 1px solid var(--surface-border);
}

.sidebar-title {
  font-weight: 600;
  font-size: 0.75rem;
  color: var(--text-color-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.sidebar-content {
  flex: 1;
  overflow-y: auto;
  padding: 0.25rem;
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
  font-size: 0.875rem;
}

.node-label {
  flex: 1;
  font-size: 0.875rem;
}

/* Bottom Panel */
.bottom-panel {
  display: flex;
  flex-direction: column;
  border-top: 1px solid var(--surface-border);
  background: var(--surface-card);
  flex-shrink: 0;
}

.resize-handle {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 20px;
  background: var(--surface-ground);
  border-bottom: 1px solid var(--surface-border);
  cursor: ns-resize;
  transition: background 0.2s;
}

.resize-handle:hover {
  background: var(--surface-hover);
}

.resize-handle i {
  font-size: 0.75rem;
  color: var(--text-color-secondary);
}
</style>
