<template>
  <div class="welcome-page">
    <div class="welcome-container">
      <div class="logo-section">
        <h1 class="title">BOMIX</h1>
        <p class="subtitle">BOM Management System</p>
      </div>

      <div class="action-section">
        <div class="file-selection-group">
          <label class="field-label">選擇或開啟現有專案：</label>
          <div class="file-input-row">
            <InputText 
              v-model="selectedFilePath" 
              readonly 
              placeholder="請選擇 .bomx 檔案..." 
              class="path-input"
            />
            <Button
              label="瀏覽..."
              icon="pi pi-search"
              severity="secondary"
              @click="browseFile"
            />
            <Button
              label="開啟"
              icon="pi pi-folder-open"
              :disabled="!selectedFilePath"
              @click="openSelectedFile"
            />
          </div>
        </div>

        <div class="divider">
          <span>或</span>
        </div>

        <div class="new-file-group">
          <Button
            label="新增檔案 (New...)"
            icon="pi pi-plus"
            severity="success"
            class="action-button"
            @click="handleCreateSeries"
          />
        </div>
      </div>

      <div v-if="recentFiles.length > 0" class="recent-section">
        <h3 class="section-title">
          <i class="pi pi-clock"></i>
          最近開啟
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
import InputText from 'primevue/inputtext'
import { useAppStore } from '../stores'
import { OpenFileDialog, GetRecentSeries, CreateSeries as BackendCreateSeries } from '../services/api'

const appStore = useAppStore()

interface RecentFile {
  path: string
  name: string
  lastOpened: string
}

const recentFiles = ref<RecentFile[]>([])
const selectedFilePath = ref('')

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

async function browseFile(): Promise<void> {
  try {
    const path = await OpenFileDialog({
      title: '選擇 BOMIX 檔案',
      filters: [{ name: 'BOMIX Series', extensions: ['bomx'] }],
      buttonLabel: '選取',
    })

    if (path) {
      selectedFilePath.value = path
    }
  } catch (error) {
    console.error('Failed to browse file:', error)
  }
}

async function openSelectedFile(): Promise<void> {
  if (selectedFilePath.value) {
    try {
      await appStore.openSeries(selectedFilePath.value)
    } catch (error) {
      console.error('Failed to open series:', error)
    }
  }
}

async function handleCreateSeries(): Promise<void> {
  try {
    const path = await OpenFileDialog({
      title: '建立新 BOMIX 檔案',
      filters: [{ name: 'BOMIX Series', extensions: ['bomx'] }],
      buttonLabel: '建立',
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
  height: 100%;
  overflow-y: auto;
  background: var(--surface-ground);
  padding: 1.5rem;
  color: var(--text-color);
}

.welcome-container {
  text-align: center;
  max-width: 500px;
  width: 100%;
  background: var(--surface-card);
  padding: 1.5rem 2rem;
  border-radius: var(--p-radius-base);
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.1);
  border: 1px solid var(--surface-border);
}

.logo-section {
  margin-bottom: 1.5rem;
}

.title {
  font-size: 2.25rem;
  font-weight: 700;
  color: var(--primary-color);
  margin: 0;
  letter-spacing: 0.05em;
}

.subtitle {
  font-size: 1rem;
  color: var(--text-color-secondary);
  margin: 0.25rem 0 0;
}

.action-section {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  margin-bottom: 1.5rem;
  text-align: left;
}

.file-selection-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.field-label {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--text-color-secondary);
}

.file-input-row {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

.path-input {
  flex: 1;
}

.divider {
  display: flex;
  align-items: center;
  text-align: center;
  color: var(--text-color-secondary);
  font-size: 0.875rem;
}

.divider::before,
.divider::after {
  content: '';
  flex: 1;
  border-bottom: 1px solid var(--surface-border);
}

.divider span {
  padding: 0 1rem;
}

.new-file-group {
  display: flex;
  justify-content: center;
}

.action-button {
  width: 100%;
}

.recent-section {
  text-align: left;
  border-top: 1px solid var(--surface-border);
  padding-top: 1rem;
}

.section-title {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--text-color-secondary);
  margin-bottom: 1rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  text-transform: uppercase;
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
  padding: 0.75rem 1rem;
  background: var(--surface-ground);
  border-radius: var(--p-radius-base);
  cursor: pointer;
  transition: background 0.2s;
  border: 1px solid transparent;
}

.recent-item:hover {
  background: var(--surface-hover);
  border-color: var(--surface-border);
}

.recent-item i {
  font-size: 1.25rem;
  color: var(--primary-color);
}

.file-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.1rem;
  overflow: hidden;
}

.file-name {
  color: var(--text-color);
  font-weight: 500;
  font-size: 0.9rem;
}

.file-path {
  color: var(--text-color-secondary);
  font-size: 0.75rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-date {
  color: var(--text-color-secondary);
  font-size: 0.75rem;
  white-space: nowrap;
}
</style>
