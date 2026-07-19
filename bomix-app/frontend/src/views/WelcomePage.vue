<template>
  <div class="welcome-page">
    <div class="welcome-container">
      <div class="logo-section">
        <h1 class="title">BOMIX</h1>
        <p class="subtitle">BOM Management System</p>
      </div>

      <div class="action-section">
        <Button
          label="Create New Series"
          icon="pi pi-plus"
          class="p-button-primary action-button"
          @click="handleCreateSeries"
        />
        <Button
          label="Open Series"
          icon="pi pi-folder-open"
          class="p-button-secondary action-button"
          @click="handleOpenSeries"
        />
      </div>

      <div v-if="recentFiles.length > 0" class="recent-section">
        <h3 class="section-title">
          <i class="pi pi-clock"></i>
          Recently Opened
        </h3>
        <div class="recent-list">
          <div
            v-for="file in recentFiles"
            :key="file.path"
            class="recent-item"
            @click="openRecentFile(file.path)"
          >
            <i class="pi pi-file"></i>
            <div class="file-info">
              <span class="file-name">{{ file.name }}</span>
              <span class="file-path">{{ file.path }}</span>
            </div>
            <span class="file-date">{{ formatDate(file.lastOpened) }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Button from 'primevue/button'
import { useAppStore } from '../stores'
import { OpenFileDialog, GetRecentSeries, OpenSeries, CreateSeries as BackendCreateSeries } from '../services/api'

const appStore = useAppStore()

interface RecentFile {
  path: string
  name: string
  lastOpened: string
}

const recentFiles = ref<RecentFile[]>([])

onMounted(async () => {
  await loadRecentFiles()
})

async function loadRecentFiles(): Promise<void> {
  try {
    const files = await GetRecentSeries()
    recentFiles.value = files
  } catch (error) {
    console.error('Failed to load recent files:', error)
  }
}

function formatDate(dateStr: string): string {
  try {
    return new Date(dateStr).toLocaleDateString()
  } catch {
    return dateStr
  }
}

async function handleCreateSeries(): Promise<void> {
  try {
    const path = await OpenFileDialog({
      title: 'Create New Series',
      filters: [{ name: 'BOMIX Series', extensions: ['bomx'] }],
      buttonLabel: 'Create',
    })

    if (path) {
      // Extract name from path
      const name = path.split(/[\\/]/).pop()?.replace('.bomx', '') || 'Untitled'
      await BackendCreateSeries(path, name, '')

      // Open the newly created series
      await appStore.openSeries(path)
    }
  } catch (error) {
    console.error('Failed to create series:', error)
  }
}

async function handleOpenSeries(): Promise<void> {
  try {
    const path = await OpenFileDialog({
      title: 'Open Series',
      filters: [{ name: 'BOMIX Series', extensions: ['bomx'] }],
      buttonLabel: 'Open',
    })

    if (path) {
      await appStore.openSeries(path)
    }
  } catch (error) {
    console.error('Failed to open series:', error)
  }
}

async function openRecentFile(path: string): Promise<void> {
  try {
    await appStore.openSeries(path)
  } catch (error) {
    console.error('Failed to open recent file:', error)
  }
}
</script>

<style scoped>
.welcome-page {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 2rem;
}

.welcome-container {
  text-align: center;
  max-width: 500px;
  width: 100%;
}

.logo-section {
  margin-bottom: 3rem;
}

.title {
  font-size: 4rem;
  font-weight: 700;
  color: white;
  margin: 0;
  letter-spacing: 0.1em;
}

.subtitle {
  font-size: 1.25rem;
  color: rgba(255, 255, 255, 0.8);
  margin: 0.5rem 0 0;
}

.action-section {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  margin-bottom: 2rem;
}

.action-button {
  width: 100%;
  padding: 1rem 2rem;
  font-size: 1.1rem;
}

.recent-section {
  text-align: left;
}

.section-title {
  font-size: 1rem;
  color: white;
  margin-bottom: 1rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.recent-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.recent-item {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.2s;
}

.recent-item:hover {
  background: rgba(255, 255, 255, 0.2);
}

.recent-item i {
  font-size: 1.5rem;
  color: white;
}

.file-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.file-name {
  color: white;
  font-weight: 500;
}

.file-path {
  color: rgba(255, 255, 255, 0.6);
  font-size: 0.875rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-date {
  color: rgba(255, 255, 255, 0.7);
  font-size: 0.875rem;
  white-space: nowrap;
}
</style>
