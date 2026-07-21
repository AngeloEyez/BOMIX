<template>
  <div class="welcome-page">
    <div class="vscode-welcome-container">
      
      <!-- 標題區域 -->
      <div class="header-section">
        <h1 class="main-title">BOMIX</h1>
        <p class="subtitle">BOM Management System</p>
      </div>

      <div class="content-grid">
        <!-- 左側：開始 (Start) -->
        <div class="column start-column">
          <h2 class="section-title">開始 (Start)</h2>
          <ul class="action-list">
            <li>
              <a href="#" class="action-link" @click.prevent="handleCreateSeries">
                <i class="pi pi-file-plus"></i>
                <span>新增系列 (New Series...)</span>
              </a>
            </li>
            <li>
              <a href="#" class="action-link" @click.prevent="browseFile">
                <i class="pi pi-folder-open"></i>
                <span>開啟現有系列 (Open Series...)</span>
              </a>
            </li>
          </ul>
        </div>

        <!-- 右側：最近開啟 (Recent) -->
        <div class="column recent-column">
          <h2 class="section-title">最近開啟 (Recent)</h2>
          <div v-if="recentFiles.length > 0" class="recent-list">
            <a
              href="#"
              v-for="file in recentFiles"
              :key="file.path"
              class="recent-link"
              @click.prevent="openRecentFile(file.path)"
              :title="file.path"
            >
              <div class="recent-file-main">
                <span class="file-name">{{ file.name }}</span>
                <span class="file-date">{{ formatDate(file.lastOpened) }}</span>
              </div>
              <div class="file-path">{{ file.path }}</div>
            </a>
          </div>
          <div v-else class="no-recent">
            目前沒有最近開啟的系列
          </div>
        </div>
      </div>

    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useAppStore } from '../stores'
import { OpenFileDialog, SaveFileDialog, GetRecentSeries, CreateSeries as BackendCreateSeries } from '../services/api'

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
    // 限制最多顯示 10 筆
    recentFiles.value = files.slice(0, 10)
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

// 建立新系列
async function handleCreateSeries(): Promise<void> {
  try {
    const path = await SaveFileDialog({
      title: '建立新系列 (選擇儲存位置)',
      filters: [{ name: 'BOMIX Series', extensions: ['bomx'] }],
      buttonLabel: '建立',
    })

    if (path) {
      let finalPath = path
      if (!finalPath.toLowerCase().endsWith('.bomx')) {
        finalPath += '.bomx'
      }

      const name = finalPath.split(/[\\/]/).pop()?.replace('.bomx', '') || 'Untitled'
      
      await BackendCreateSeries(finalPath, name, '')
      await appStore.openSeries(finalPath)
    }
  } catch (error) {
    console.error('Failed to create series:', error)
  }
}

// 瀏覽開啟現有系列
async function browseFile(): Promise<void> {
  try {
    const path = await OpenFileDialog({
      title: '選擇 BOMIX 系列',
      filters: [{ name: 'BOMIX Series', extensions: ['bomx'] }],
      buttonLabel: '選取',
    })

    if (path) {
      await appStore.openSeries(path)
    }
  } catch (error) {
    console.error('Failed to browse file:', error)
  }
}

// 點擊最近項目開啟
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
  height: 100%;
  width: 100%;
  overflow-y: auto;
  background: var(--surface-card); /* VS Code typically uses the editor background */
  color: var(--text-color);
  padding: 4rem; /* generously spaced like VS Code */
}

.vscode-welcome-container {
  max-width: 900px;
  width: 100%;
  margin: 0 auto;
}

/* Header */
.header-section {
  margin-bottom: 3rem;
}

.main-title {
  font-size: 2rem;
  font-weight: 400;
  color: var(--text-color);
  margin: 0 0 0.25rem 0;
}

.subtitle {
  font-size: 1.1rem;
  font-weight: 300;
  color: var(--text-color-secondary);
  margin: 0;
}

/* Grid Layout */
.content-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 4rem;
}

.column {
  display: flex;
  flex-direction: column;
}

.section-title {
  font-size: 1.25rem;
  font-weight: 400;
  color: var(--text-color);
  margin-bottom: 1.25rem;
}

/* Action Links (Start Column) */
.action-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.action-link {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.35rem 0;
  color: var(--primary-color);
  text-decoration: none;
  font-size: 13px;
  transition: opacity 0.15s;
}

.action-link:hover {
  opacity: 0.8;
  text-decoration: underline;
}

.action-link i {
  font-size: 1.1rem;
}

/* Recent List */
.recent-list {
  display: flex;
  flex-direction: column;
}

.recent-link {
  display: flex;
  flex-direction: column;
  padding: 0.35rem 0.5rem;
  margin-left: -0.5rem;
  text-decoration: none;
  border-radius: var(--p-radius-base);
  transition: background-color 0.1s;
}

.recent-link:hover {
  background-color: var(--surface-hover);
}

.recent-file-main {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.15rem;
}

.file-name {
  color: var(--primary-color);
  font-weight: 500;
  font-size: 13px;
}

.file-date {
  color: var(--text-color-secondary);
  font-size: 11px;
}

.file-path {
  color: var(--text-color-secondary);
  font-size: 11px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  direction: rtl; /* Truncates the left side of long paths (keeps filename visible if very long) */
  text-align: left; /* Keep text left aligned */
}

/* Reset direction so the text reads normally but truncates left */
.file-path::after {
  content: '\200E'; /* Left-to-Right Mark */
}

.no-recent {
  color: var(--text-color-secondary);
  font-style: italic;
  font-size: 13px;
}
</style>

