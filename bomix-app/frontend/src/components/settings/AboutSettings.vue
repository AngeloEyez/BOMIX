<!--
  @file AboutSettings.vue
  @description 系統關於與版本資訊展示元件 (About Settings)
  
  職責說明：
  1. 版本元資料呈現：展示自後端 SSOT 取得之應用程式版本號、Git Commit 雜湊碼與建置時間。
  2. 應用程式簡介與 Logo：展示 BOMIX 應用程式標誌、名稱與技術架構資訊。
  3. 獨立展示區塊：作為 Settings 頁面最末端的「About」章節，支援側邊欄平滑捲動導航。
-->

<template>
  <!-- ==================== 6. About ==================== -->
  <section id="category-about" class="category-section">
    <div class="category-header">
      <div class="category-title">
        <i class="pi pi-info-circle category-icon"></i>
        <h2>About</h2>
      </div>
    </div>

    <!-- 應用程式產品資訊卡片 -->
    <div class="vscode-setting-item about-card">
      <div class="about-header">
        <img src="/app-logo.png" alt="BOMIX Logo" class="about-logo" />
        <div class="about-title-group">
          <div class="about-title-row">
            <span class="about-app-name">BOMIX</span>
            <Badge
              :value="displayBadge"
              :severity="isDevVersion ? 'warn' : 'info'"
              class="version-badge"
            />
          </div>
          <p class="about-app-sub">高效能 Excel BOM 處理與矩陣分析工具</p>
        </div>
      </div>

      <!-- 詳細版本與建置元資料列表 -->
      <div class="about-meta-list">
        <!-- 1. 版本號 -->
        <div class="about-meta-row">
          <span class="meta-label">版本號 (Version)</span>
          <span class="meta-value font-mono">{{ isDevVersion ? 'dev' : (appInfo.version || 'dev') }}</span>
        </div>

        <!-- 2. Git Commit -->
        <div class="about-meta-row">
          <span class="meta-label">Git Commit</span>
          <span class="meta-value font-mono">{{ appInfo.gitCommit || 'none' }}</span>
        </div>

        <!-- 3. 建置時間 -->
        <div class="about-meta-row">
          <span class="meta-label">建置時間 (Build Time)</span>
          <span class="meta-value">{{ formattedBuildTime }}</span>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
/**
 * @file AboutSettings.vue
 * @description 系統關於與版本展示子元件
 */
import { ref, onMounted, computed } from 'vue'
import Badge from 'primevue/badge'
import { GetAppInfo, type AppInfo } from '../../services/api'

/**
 * 應用程式版本與建置詳細資訊
 */
const appInfo = ref<AppInfo>({
  version: 'dev',
  gitCommit: 'none',
  buildTime: 'unknown',
})

/**
 * 組件掛載時向後端非同步取得版本元資料
 */
onMounted(async () => {
  try {
    const info = await GetAppInfo()
    appInfo.value = info
  } catch (error) {
    console.error('Failed to load app info in AboutSettings:', error)
  }
})

/**
 * 判定當前是否為開發或未知版本
 */
const isDevVersion = computed(() => {
  const ver = appInfo.value.version?.trim().toLowerCase()
  return !ver || ver === 'dev' || ver === 'unknown'
})

/**
 * 計算頂部 Badge 顯示字串
 * 若為 dev 版本則顯示 "dev"（不帶 v 前綴）
 * 若為正式發布版本 (如 1.0.0)，統一帶上標準 "v" 前綴 (如 "v1.0.0")
 */
const displayBadge = computed(() => {
  if (isDevVersion.value) return 'dev'
  const clean = appInfo.value.version.trim().replace(/^v/, '')
  return `v${clean}`
})

/**
 * 格式化呈現建置時間字串
 */
const formattedBuildTime = computed(() => {
  if (!appInfo.value.buildTime || appInfo.value.buildTime === 'unknown') {
    return '本地開發階段 (Local Dev)'
  }
  try {
    const d = new Date(appInfo.value.buildTime)
    if (isNaN(d.getTime())) return appInfo.value.buildTime
    return d.toLocaleString()
  } catch {
    return appInfo.value.buildTime
  }
})
</script>

<style scoped>
@import './styles/settings.css';

.about-card {
  padding: 1.25rem;
  background: var(--surface-card, rgba(255, 255, 255, 0.03));
  border: 1px solid var(--surface-border, rgba(255, 255, 255, 0.08));
  border-radius: 8px;
}

.about-header {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 1.25rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--surface-border, rgba(255, 255, 255, 0.08));
}

.about-logo {
  width: 44px;
  height: 44px;
  object-fit: contain;
  border-radius: 6px;
}

.about-title-group {
  display: flex;
  flex-direction: column;
}

.about-title-row {
  display: flex;
  align-items: center;
  gap: 0.75rem; /* 12px 舒適間隔 */
}

.about-app-name {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--text-color, #fff);
  letter-spacing: 0.05em;
  line-height: 1.2;
}

.version-badge {
  font-size: 0.72rem;
  padding: 0.12rem 0.5rem;
  margin-left: 0.25rem; /* 顯式 margin 保障與 BOMIX 文字保持間距 */
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.about-app-sub {
  font-size: 0.8rem;
  color: var(--text-color-secondary, #94a3b8);
  margin-top: 0.15rem;
}

.about-meta-list {
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
}

.about-meta-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.82rem;
  padding: 0.25rem 0;
}

.meta-label {
  color: var(--text-color-secondary, #94a3b8);
}

.meta-value {
  color: var(--text-color, #f1f5f9);
  font-size: 0.82rem;
}
</style>
