<template>
  <div class="ai-chat-page select-text">
    <!-- 1. 頂部資訊工具列 (Header) -->
    <header class="chat-header select-none">
      <div class="chat-header-left">
        <div class="chat-avatar-badge">
          <i class="pi pi-sparkles"></i>
        </div>
        <div class="chat-header-meta">
          <h1 class="chat-header-title">AI 對話助手</h1>
          <p class="chat-header-status">
            <span class="status-indicator-dot"></span>
            <span class="truncate">已連接系列：{{ activeSeriesName }}</span>
          </p>
        </div>
      </div>

      <!-- 右側快捷操作按鈕 -->
      <div class="chat-header-actions">
        <!-- 清空歷史按鈕 -->
        <Button
          icon="pi pi-trash"
          label="清空對話"
          severity="secondary"
          size="small"
          text
          :disabled="!aiChatStore.hasMessages || aiChatStore.isGenerating"
          @click="handleClear"
          title="清空目前對話紀錄"
        />
      </div>
    </header>

    <!-- 2. 中間訊息滾動視區 (獨立可控 margin, padding, scroll) -->
    <main
      ref="messageContainerRef"
      class="chat-messages-viewport"
      @scroll="handleScroll"
    >
      <div class="chat-messages-body">
        <!-- 歡迎畫面 (尚未有訊息時顯示) -->
        <div v-if="!aiChatStore.hasMessages" class="chat-welcome-container select-text">
          <div class="chat-welcome-icon select-none">
            <i class="pi pi-sparkles"></i>
          </div>
          <h2 class="chat-welcome-title">
            歡迎使用 BOMIX AI 助手
          </h2>
          <p class="chat-welcome-subtitle">
            您可以透過自然語言查詢當前開啟的系列資料庫，支援零件追蹤、版本差異比對、料號跨專案統計等智慧功能。
          </p>

          <!-- 推薦快捷提問卡片清單 -->
          <div class="chat-prompts-grid select-none">
            <PromptChip
              icon="pi pi-list"
              title="列出所有專案與版本"
              description="取得當前系列內的所有專案清單與各 BOM Revision 詳細資訊"
              @click="sendPrompt('請列出目前系列中包含哪些專案，以及每個專案分別有哪些 BOM 版本？')"
            />
            <PromptChip
              icon="pi pi-search"
              title="物料跨專案使用查詢"
              description="查詢特定物料（如 HHPN 或供應商料號）在哪些專案與版本被使用"
              @click="sendPrompt('請查詢 GRM188R60J226MEA0D 物料目前被哪些專案與 BOM 版本使用？')"
            />
            <PromptChip
              icon="pi pi-arrows-h"
              title="版本差異精準比對"
              description="使用後端 Diff 演算法分析兩份 BOM 的新增、移除與用量修改"
              @click="sendPrompt('請幫我比對目前專案中最新兩個 BOM 版本的零件差異')"
            />
            <PromptChip
              icon="pi pi-chart-line"
              title="特定零件變更歷程"
              description="追蹤特定零件或位置代號 (RefDes 如 LR1) 在各版本的演變過程"
              @click="sendPrompt('我想追蹤 LR1 零件在專案中的版本變動歷程')"
            />
          </div>
        </div>

        <!-- 訊息清單 -->
        <template v-else>
          <MessageBubble
            v-for="msg in aiChatStore.messages"
            :key="msg.id"
            :role="msg.role"
            :content="msg.content"
            :tool-calls="msg.toolCalls"
            :timestamp="msg.timestamp"
            :is-streaming="msg.isStreaming"
          />

          <!-- 進行中的工具呼叫 (未完成的 tool call 卡片) -->
          <div v-if="aiChatStore.activeToolCalls.length > 0 && aiChatStore.isGenerating" class="chat-active-tools-wrapper">
            <ToolCallCard
              v-for="tc in aiChatStore.activeToolCalls"
              :key="tc.id"
              :tool-name="tc.name"
              :arguments="tc.arguments"
              :result="tc.result"
              :status="tc.status"
              :explanation="tc.explanation"
            />
          </div>
        </template>

        <!-- 底部捲動錨點 (緩衝高度，確保滾動到底部時訊息與輸入框間距舒適) -->
        <div ref="bottomAnchorRef" class="chat-scroll-anchor"></div>
      </div>

      <!-- 回到最底部懸浮按鈕 (仿 OpenWebUI) -->
      <transition name="fade">
        <Button
          v-if="showScrollBottomBtn"
          type="button"
          icon="pi pi-arrow-down"
          rounded
          severity="secondary"
          class="chat-scroll-bottom-btn"
          @click="scrollToBottom(true)"
          title="回到底部最新訊息"
        />
      </transition>
    </main>

    <!-- 3. 底部輸入區域 (Footer - Antigravity 風格一體化大圓角緊湊卡片) -->
    <footer class="chat-input-footer select-none">
      <div class="chat-input-container">
        <!-- Antigravity 經典整合式大圓角卡片輸入框 -->
        <div class="antigravity-input-card">
          <!-- 上層：多行自動調節 Textarea (緊湊微距，減少文字與邊框距離) -->
          <textarea
            ref="inputRef"
            v-model="inputPrompt"
            rows="1"
            class="antigravity-textarea select-text"
            placeholder="Ask anything, @ to mention, / for actions"
            :disabled="aiChatStore.isGenerating"
            @keydown="handleKeyDown"
            @input="adjustTextareaHeight"
          ></textarea>

          <!-- 下層：最底下一行控制列 (較小字體：模型下拉選單靠左，Token 統計靠右，傳送/停止按鈕最右) -->
          <div class="antigravity-bottom-bar select-none">
            <!-- 靠左：模型切換下拉選單按鈕 (無 '+' 符號，Antigravity 極簡字體) -->
            <div class="model-picker-wrapper relative">
              <button
                type="button"
                class="model-picker-btn"
                :title="'點擊切換 AI 模型' + (aiChatStore.currentModel ? ` (目前：${aiChatStore.currentModel})` : ' (尚未選擇模型)')"
                @click.stop="toggleModelMenu"
              >
                <span class="model-picker-name truncate">{{ aiChatStore.currentModel || '選擇模型' }}</span>
                <i class="pi pi-angle-up text-[10px] opacity-60 ml-1"></i>
              </button>

              <!-- 模型切換彈出選單 -->
              <div v-if="showModelMenu" class="model-dropdown-menu select-none">
                <div class="model-dropdown-header flex justify-between items-center">
                  <span>可用 AI 模型</span>
                  <button
                    v-if="aiChatStore.isLoadingModels"
                    type="button"
                    class="text-[10px] text-primary"
                    disabled
                  >
                    <i class="pi pi-spin pi-spinner"></i>
                  </button>
                  <button
                    v-else
                    type="button"
                    class="text-[10px] opacity-70 hover:opacity-100 hover:text-primary transition-opacity"
                    title="重新整理伺服器模型清單"
                    @click.stop="aiChatStore.fetchAvailableModels()"
                  >
                    <i class="pi pi-refresh"></i>
                  </button>
                </div>
                <div class="model-options-list">
                  <template v-if="displayModels.length > 0">
                    <button
                      v-for="m in displayModels"
                      :key="m"
                      type="button"
                      :class="['model-option-item', { active: aiChatStore.currentModel === m }]"
                      @click="selectModel(m)"
                    >
                      <span class="model-option-title truncate">{{ m }}</span>
                      <i v-if="aiChatStore.currentModel === m" class="pi pi-check text-[10px] text-primary shrink-0 ml-1"></i>
                    </button>
                  </template>
                  <div v-else class="px-2 py-3 text-center text-xs opacity-60">
                    無可用模型<br />
                    <span class="text-[10px] opacity-80">(請至設定確認 Base URL 或設定自訂模型)</span>
                  </div>
                </div>
              </div>
            </div>

            <!-- 靠右區域：本輪 | 累計 Token 數量 (靠右)，傳送/停止按鈕 (最右)，強制同橫行 -->
            <div class="antigravity-bottom-right">
              <!-- Token 數量統計 (本輪 | 累計) -->
              <div
                class="token-counter-badge font-mono select-none"
                :title="`本輪消耗：${aiChatStore.currentRoundTokens} tokens\n累計消耗：${aiChatStore.totalTokens} tokens`"
              >
                <span>本輪 {{ formatTokens(aiChatStore.currentRoundTokens) }}</span>
                <span class="token-separator">&nbsp;|&nbsp;</span>
                <span>累計 {{ formatTokens(aiChatStore.totalTokens) }} tokens</span>
              </div>

              <!-- 傳送 / 停止按鈕 (最右側圓形按鈕) -->
              <button
                v-if="aiChatStore.isGenerating"
                type="button"
                class="antigravity-action-btn action-btn-stop"
                title="停止生成"
                @click="handleStop"
              >
                <i class="pi pi-stop text-[10px]"></i>
              </button>
              <button
                v-else
                type="button"
                class="antigravity-action-btn action-btn-send"
                :class="{ 'action-btn-active': canSend }"
                :disabled="!canSend"
                :title="!aiChatStore.currentModel ? '請先設定或選擇 AI 模型' : '發送提問 (Enter)'"
                @click="handleSubmit"
              >
                <i class="pi pi-arrow-right text-[11px]"></i>
              </button>
            </div>
          </div>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, onMounted, onUnmounted, watch } from 'vue'
import Button from 'primevue/button'
import PromptChip from '../components/ai/PromptChip.vue'
import MessageBubble from '../components/ai/MessageBubble.vue'
import ToolCallCard from '../components/ai/ToolCallCard.vue'
import { useAIChatStore } from '../stores/aiChat'
import { useAppStore } from '../stores/app'

const aiChatStore = useAIChatStore()
const appStore = useAppStore()

const inputPrompt = ref('')
const inputRef = ref<HTMLTextAreaElement | null>(null)
const messageContainerRef = ref<HTMLElement | null>(null)
const bottomAnchorRef = ref<HTMLElement | null>(null)
const showScrollBottomBtn = ref(false)

// 模型切換選單狀態
const showModelMenu = ref(false)

// 供對話頁面展示之可用模型清單 (包含伺服器取得之模型與目前已選定的模型)
const displayModels = computed(() => {
  const list = [...aiChatStore.availableModels]
  const current = aiChatStore.currentModel?.trim()
  if (current && !list.includes(current)) {
    list.unshift(current)
  }
  return list
})

// 切換選單開關
function toggleModelMenu(): void {
  showModelMenu.value = !showModelMenu.value
}

// 關閉選單
function closeModelMenu(): void {
  showModelMenu.value = false
}

// 選擇模型
function selectModel(modelId: string): void {
  aiChatStore.switchModel(modelId)
  showModelMenu.value = false
}

// 格式化 Token 顯示文字 (例如 150 或 1.4k)
function formatTokens(count: number): string {
  if (!count) return '0'
  if (count >= 10000) {
    return (count / 1000).toFixed(1) + 'k'
  }
  return count.toLocaleString()
}

// 全域點擊關閉模型選單
function handleDocumentClick(e: MouseEvent): void {
  const target = e.target as HTMLElement
  if (!target.closest('.model-picker-wrapper')) {
    showModelMenu.value = false
  }
}

// 當前開啟系列名稱
const activeSeriesName = computed(() => {
  return appStore.seriesInfo?.name || '未命名系列'
})

// 是否允許點擊發送 (遵照要求：若無任何模型則無法送出查詢)
const canSend = computed(() => {
  const hasModel = !!aiChatStore.currentModel?.trim()
  return inputPrompt.value.trim().length > 0 && !aiChatStore.isGenerating && hasModel
})

// 初始化事件監聽與載入模型
onMounted(async () => {
  aiChatStore.initEventListeners()
  await aiChatStore.fetchCurrentModel()
  // 若目前模型清單為空，自動向伺服器拉取
  if (aiChatStore.availableModels.length === 0) {
    await aiChatStore.fetchAvailableModels()
  }
  window.addEventListener('click', handleDocumentClick)
  focusInput()
})

onUnmounted(() => {
  window.removeEventListener('click', handleDocumentClick)
})

// 自動聚焦輸入框
function focusInput(): void {
  nextTick(() => {
    inputRef.value?.focus()
  })
}

// 動態調整文字框高度 (最高 160px)
function adjustTextareaHeight(): void {
  const el = inputRef.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 160) + 'px'
}

// 滾動事件監聽（判斷是否顯示回到底部按鈕）
function handleScroll(): void {
  const el = messageContainerRef.value
  if (!el) return
  const distanceFromBottom = el.scrollHeight - el.scrollTop - el.clientHeight
  showScrollBottomBtn.value = distanceFromBottom > 160
}

// 捲動到底部
function scrollToBottom(smooth = true): void {
  nextTick(() => {
    if (bottomAnchorRef.value) {
      bottomAnchorRef.value.scrollIntoView({
        behavior: smooth ? 'smooth' : 'auto',
        block: 'end',
      })
    }
  })
}

// 監聽訊息變化自動滾動
watch(
  () => aiChatStore.messages.length,
  () => scrollToBottom(true)
)

watch(
  () => aiChatStore.messages[aiChatStore.messages.length - 1]?.content,
  () => {
    // 只有當使用者距離底部不遠時才自動吸附底部，避免打擾使用者回看歷史訊息
    const el = messageContainerRef.value
    if (!el || el.scrollHeight - el.scrollTop - el.clientHeight < 240) {
      scrollToBottom(false)
    }
  }
)

watch(
  () => aiChatStore.activeToolCalls.length,
  () => scrollToBottom(true)
)

// 處理鍵盤按鍵事件
function handleKeyDown(e: KeyboardEvent): void {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    handleSubmit()
  }
}

// 提交提問
async function handleSubmit(): Promise<void> {
  const text = inputPrompt.value.trim()
  if (!text || aiChatStore.isGenerating) return

  inputPrompt.value = ''
  if (inputRef.value) {
    inputRef.value.style.height = 'auto'
  }

  scrollToBottom(true)
  await aiChatStore.sendMessage(text)
  focusInput()
}

// 點擊快捷提問晶片
async function sendPrompt(text: string): Promise<void> {
  inputPrompt.value = text
  await handleSubmit()
}

// 停止生成
async function handleStop(): Promise<void> {
  await aiChatStore.stopGeneration()
  focusInput()
}

// 清空對話
function handleClear(): void {
  aiChatStore.clearMessages()
  focusInput()
}
</script>

<style scoped>
/* ==========================================================================
   1. 根容器佈局 (Root Flexbox Container)
   保證 100% 填滿 App 的 Main Content 區域，並隨視窗與側邊欄伸縮自適應
   ========================================================================== */
.ai-chat-page {
  display: flex;
  flex-direction: column;
  flex: 1 1 0%;
  min-height: 0;
  min-width: 0;
  height: 100%;
  width: 100%;
  overflow: hidden;
  position: relative;
  background-color: var(--surface-ground);
  color: var(--text-color);
  user-select: text !important;
  -webkit-user-select: text !important;
}

/* ==========================================================================
   2. 頂部資訊列 (Chat Header)
   ========================================================================== */
.chat-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 48px;
  padding: 0 1.5rem;
  border-bottom: 1px solid var(--surface-border);
  background-color: var(--surface-card);
  flex-shrink: 0;
  z-index: 10;
}

.chat-header-left {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  min-width: 0;
}

.chat-avatar-badge {
  width: 30px;
  height: 30px;
  border-radius: 8px;
  background: color-mix(in srgb, var(--primary-color) 15%, transparent);
  color: var(--primary-color);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.95rem;
  flex-shrink: 0;
}

.chat-header-meta {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.chat-header-title {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--text-color);
  line-height: 1.2;
}

.chat-header-status {
  font-size: 0.75rem;
  color: var(--text-color-secondary);
  line-height: 1.2;
  margin-top: 0.15rem;
  display: flex;
  align-items: center;
  gap: 0.35rem;
}

.status-indicator-dot {
  display: inline-block;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background-color: #22c55e;
  flex-shrink: 0;
}

.chat-header-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-shrink: 0;
}

/* ==========================================================================
   3. 中間訊息滾動視區 (Chat Messages Viewport & Scroll Control)
   獨立控制此區域的滾動行為、卷軸外觀、寬度與內外邊距 (margin, padding)
   ========================================================================== */
.chat-messages-viewport {
  flex: 1 1 0%;
  min-height: 0;
  min-width: 0;
  width: 100%;
  /* 保證內容超出時自動出現垂直卷軸 */
  overflow-y: auto;
  overflow-x: hidden;
  position: relative;
  scroll-behavior: smooth;
  user-select: text !important;
  -webkit-user-select: text !important;

  /* Firefox 捲動條設定 */
  scrollbar-width: thin;
  scrollbar-color: var(--surface-border) transparent;
}

/* Webkit (Chromium, WebView2) 捲動條深度客製 - 高對比度、平滑圓角 */
.chat-messages-viewport::-webkit-scrollbar {
  width: 8px;
}

.chat-messages-viewport::-webkit-scrollbar-track {
  background: transparent;
}

.chat-messages-viewport::-webkit-scrollbar-thumb {
  background-color: var(--surface-border);
  border-radius: 9999px;
  border: 2px solid transparent;
  background-clip: content-box;
  transition: background-color 0.2s ease;
}

.chat-messages-viewport::-webkit-scrollbar-thumb:hover {
  background-color: var(--text-color-secondary);
}

/* 訊息內容主體 (控制對齊、邊界 padding 與最大寬度) */
.chat-messages-body {
  max-width: 860px;
  width: 100%;
  margin: 0 auto;
  padding: 1.25rem 1.5rem 1.5rem 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  box-sizing: border-box;
}

/* 底部滾動緩衝錨點 */
.chat-scroll-anchor {
  height: 1.5rem;
  flex-shrink: 0;
}

/* 進行中的工具卡片容器 */
.chat-active-tools-wrapper {
  max-width: 92%;
  margin-bottom: 0.5rem;
}

/* 回到最底部懸浮按鈕 (OpenWebUI Floating Arrow) */
.chat-scroll-bottom-btn {
  position: absolute !important;
  bottom: 1.25rem;
  right: 1.75rem;
  width: 36px !important;
  height: 36px !important;
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.15) !important;
  z-index: 20;
  background-color: var(--surface-card) !important;
  border: 1px solid var(--surface-border) !important;
  color: var(--text-color) !important;
}

.chat-scroll-bottom-btn:hover {
  color: var(--primary-color) !important;
  border-color: var(--primary-color) !important;
}

/* ==========================================================================
   4. 歡迎卡片區塊 (Welcome / Prompts Screen)
   ========================================================================== */
.chat-welcome-container {
  padding: 2.5rem 0 1.5rem 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  width: 100%;
}

.chat-welcome-icon {
  width: 52px;
  height: 52px;
  border-radius: 16px;
  background: linear-gradient(135deg, var(--primary-color), #6366f1);
  color: #ffffff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.5rem;
  box-shadow: 0 8px 20px color-mix(in srgb, var(--primary-color) 35%, transparent);
  margin-bottom: 1rem;
}

.chat-welcome-title {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--text-color);
  margin-bottom: 0.5rem;
}

.chat-welcome-subtitle {
  font-size: 0.85rem;
  color: var(--text-color-secondary);
  max-width: 480px;
  line-height: 1.6;
  margin-bottom: 2rem;
}

.chat-prompts-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 0.75rem;
  width: 100%;
  text-align: left;
}

/* ==========================================================================
   5. 底部輸入區域 (Chat Input Footer - Antigravity Style Dock)
   作為同層 Flex 兄弟節點，固定於底部，絕不遮擋上方訊息內容
   ========================================================================== */
.chat-input-footer {
  flex-shrink: 0;
  width: 100%;
  background-color: var(--surface-card);
  border-top: 1px solid var(--surface-border);
  padding: 0.35rem 1.25rem 0.5rem 1.25rem;
  box-sizing: border-box;
  z-index: 10;
}

.chat-input-container {
  max-width: 860px;
  width: 100%;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

/* ==========================================================================
   Antigravity 經典整合式緊湊大圓角卡片輸入框
   縮短文字與卡片邊框之間的內距，兼具極客簡約與現代感
   ========================================================================== */
.antigravity-input-card {
  display: flex;
  flex-direction: column;
  padding: 0.25rem 0.5rem 0.2rem 0.55rem; /* 極致緊湊微距，縮小與邊框及行距空間 */
  border-radius: 0.85rem; /* ~14px 經典圓角 */
  background-color: #ffffff;
  border: 1px solid var(--surface-border, #e2e8f0);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
  transition: border-color 0.15s ease, box-shadow 0.15s ease, background-color 0.15s ease;
}

:global(.app-dark) .antigravity-input-card,
:global(.dark) .antigravity-input-card,
:global(html.app-dark) .antigravity-input-card {
  background-color: #232326 !important; /* 質感深灰黑，與主背景 #1e1e1e 舒適區隔 */
  border-color: #38383f !important;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.25) !important;
}

.antigravity-input-card:focus-within {
  border-color: var(--primary-color, #3b82f6) !important;
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--primary-color) 25%, transparent) !important;
}

/* 上層多行文字輸入框 (Textarea) */
.antigravity-textarea {
  width: 100%;
  min-height: 28px; /* 緊湊起始高度，爭取垂直空間 */
  max-height: 160px;
  padding: 0.05rem 0.1rem;
  margin-bottom: 0.05rem; /* 顯著縮小與底部控制列之間的間距 */
  background: transparent;
  border: none;
  outline: none;
  font-family: inherit;
  font-size: 0.875rem;
  line-height: 1.4;
  color: var(--text-color, #1e293b);
  resize: none;
  box-sizing: border-box;
}

:global(.app-dark) .antigravity-textarea,
:global(.dark) .antigravity-textarea,
:global(html.app-dark) .antigravity-textarea {
  color: #e2e8f0 !important;
}

.antigravity-textarea::placeholder {
  color: var(--text-color-secondary, #94a3b8);
  opacity: 0.65;
}

:global(.app-dark) .antigravity-textarea::placeholder,
:global(.dark) .antigravity-textarea::placeholder,
:global(html.app-dark) .antigravity-textarea::placeholder {
  color: #6e7681 !important;
  opacity: 0.8;
}

/* 下層：底端控制列 (模型切換、Token 統計、傳送按鈕) */
.antigravity-bottom-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding-top: 0; /* 零上內距，緊靠文字輸入區 */
  min-height: 24px;
}

/* 靠左：模型選擇下拉按鈕 (Antigravity 專屬字型與無 '+' 符號極簡風格) */
.model-picker-wrapper {
  position: relative;
  display: inline-flex;
}

.model-picker-btn {
  display: inline-flex;
  align-items: center;
  padding: 0.2rem 0.45rem;
  border-radius: 4px;
  background: transparent;
  border: none;
  cursor: pointer;
  /* Antigravity / VS Code 經典 UI 字型 */
  font-family: -apple-system, BlinkMacSystemFont, "Segoe WPC", "Segoe UI", system-ui, "Ubuntu", "Droid Sans", sans-serif !important;
  font-size: 0.72rem; /* ~11.5px 較小字體 */
  line-height: 1.2;
  color: var(--text-color-secondary, #64748b);
  font-weight: 500;
  letter-spacing: 0.01em;
  max-width: 240px;
  transition: background-color 0.15s ease, color 0.15s ease;
}

.model-picker-name {
  font-family: inherit;
  font-size: inherit;
  font-weight: inherit;
}

.model-picker-btn:hover {
  background-color: rgba(125, 125, 125, 0.12);
  color: var(--text-color, #1e293b);
}

:global(.app-dark) .model-picker-btn,
:global(.dark) .model-picker-btn,
:global(html.app-dark) .model-picker-btn {
  color: #a1a1aa !important;
}

:global(.app-dark) .model-picker-btn:hover,
:global(.dark) .model-picker-btn:hover,
:global(html.app-dark) .model-picker-btn:hover {
  background-color: rgba(255, 255, 255, 0.08) !important;
  color: #ffffff !important;
}

/* 模型切換彈出浮動選單 (Popover) */
.model-dropdown-menu {
  position: absolute;
  bottom: calc(100% + 6px);
  left: 0;
  width: 260px;
  background-color: #ffffff;
  border: 1px solid var(--surface-border, #e2e8f0);
  border-radius: 8px;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.12);
  padding: 0.4rem;
  z-index: 100;
  box-sizing: border-box;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe WPC", "Segoe UI", system-ui, "Ubuntu", "Droid Sans", sans-serif !important;
}

:global(.app-dark) .model-dropdown-menu,
:global(.dark) .model-dropdown-menu,
:global(html.app-dark) .model-dropdown-menu {
  background-color: #1e1e21 !important;
  border-color: #38383f !important;
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.6) !important;
}

.model-dropdown-header {
  font-size: 0.68rem;
  font-weight: 600;
  padding: 0.2rem 0.4rem 0.35rem 0.4rem;
  color: var(--text-color-secondary, #94a3b8);
  border-bottom: 1px solid var(--surface-border, #f1f5f9);
  margin-bottom: 0.3rem;
  letter-spacing: 0.02em;
}

:global(.app-dark) .model-dropdown-header,
:global(.dark) .model-dropdown-header,
:global(html.app-dark) .model-dropdown-header {
  border-bottom-color: rgba(255, 255, 255, 0.08) !important;
  color: #858585 !important;
}

.model-options-list {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  max-height: 210px;
  overflow-y: auto;
}

.model-option-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 0.35rem 0.5rem;
  border-radius: 4px;
  background: transparent;
  border: none;
  cursor: pointer;
  transition: background-color 0.12s ease;
  color: var(--text-color, #1e293b);
}

.model-option-item:hover {
  background-color: rgba(125, 125, 125, 0.1);
}

:global(.app-dark) .model-option-item,
:global(.dark) .model-option-item,
:global(html.app-dark) .model-option-item {
  color: #cccccc !important;
}

:global(.app-dark) .model-option-item:hover,
:global(.dark) .model-option-item:hover,
:global(html.app-dark) .model-option-item:hover {
  background-color: rgba(255, 255, 255, 0.08) !important;
  color: #ffffff !important;
}

.model-option-item.active {
  background-color: color-mix(in srgb, var(--primary-color, #3b82f6) 12%, transparent);
  color: var(--primary-color, #3b82f6) !important;
}

.model-option-title {
  font-size: 0.75rem;
  font-weight: 500;
}

.model-option-desc {
  font-size: 0.65rem;
  opacity: 0.75;
}


/* 靠右區域：Token 與 送出/停止按鈕 嚴格同行水平排列，按鈕在最右側 */
.antigravity-bottom-right {
  display: flex !important;
  flex-direction: row !important;
  align-items: center !important;
  justify-content: flex-end !important;
  gap: 0.5rem !important;
  flex-shrink: 0 !important;
  flex-wrap: nowrap !important;
  white-space: nowrap !important;
}

/* Token 數量統計徽章 (較小字體，等寬數字，強制單行不折行) */
.token-counter-badge {
  display: inline-flex !important;
  align-items: center !important;
  flex-direction: row !important;
  white-space: nowrap !important;
  flex-shrink: 0 !important;
  font-size: 0.68rem; /* ~11px */
  color: #94a3b8;
  letter-spacing: 0.02em;
  padding: 0 0.15rem;
  line-height: 1;
}

:global(.app-dark) .token-counter-badge,
:global(.dark) .token-counter-badge,
:global(html.app-dark) .token-counter-badge {
  color: #71717a !important;
}

/* Token 分隔垂直線樣式 */
.token-separator {
  display: inline-block;
  margin: 0 0.15rem;
  opacity: 0.45;
  user-select: none;
}

/* 最右側圓形操作按鈕 (傳送/停止按鈕，永不被擠壓) */
.antigravity-action-btn {
  display: inline-flex !important;
  align-items: center !important;
  justify-content: center !important;
  flex-shrink: 0 !important;
  width: 26px;
  height: 26px;
  border-radius: 50%;
  border: none;
  padding: 0;
  transition: all 0.15s ease;
}

/* 發送按鈕預設不可用態 (低飽和) */
.action-btn-send {
  background-color: #f1f5f9;
  color: #94a3b8;
  cursor: default;
  opacity: 0.65;
}

:global(.app-dark) .action-btn-send,
:global(.dark) .action-btn-send,
:global(html.app-dark) .action-btn-send {
  background-color: #2b2b2e !important;
  color: #52525b !important;
  opacity: 0.7;
}

/* 發送按鈕啟用態 (使用品牌 Primary Color) */
.action-btn-send.action-btn-active {
  background-color: var(--primary-color, #3b82f6) !important;
  color: #ffffff !important;
  cursor: pointer;
  opacity: 1;
  box-shadow: 0 1px 4px color-mix(in srgb, var(--primary-color, #3b82f6) 35%, transparent);
}

.action-btn-send.action-btn-active:hover {
  background-color: color-mix(in srgb, var(--primary-color, #3b82f6) 88%, #000000) !important;
  box-shadow: 0 2px 6px color-mix(in srgb, var(--primary-color, #3b82f6) 45%, transparent);
}

:global(.app-dark) .action-btn-send.action-btn-active,
:global(.dark) .action-btn-send.action-btn-active,
:global(html.app-dark) .action-btn-send.action-btn-active {
  background-color: var(--primary-color, #3b82f6) !important;
  color: #ffffff !important;
  opacity: 1;
  box-shadow: 0 1px 6px color-mix(in srgb, var(--primary-color, #3b82f6) 40%, transparent);
}

:global(.app-dark) .action-btn-send.action-btn-active:hover,
:global(.dark) .action-btn-send.action-btn-active:hover,
:global(html.app-dark) .action-btn-send.action-btn-active:hover {
  background-color: color-mix(in srgb, var(--primary-color, #3b82f6) 85%, #ffffff) !important;
}

/* 停止生成按鈕 (紅色) */
.action-btn-stop {
  background-color: #ef4444;
  color: #ffffff;
  cursor: pointer;
  box-shadow: 0 0 6px rgba(239, 68, 68, 0.4);
}

.action-btn-stop:hover {
  background-color: #dc2626;
}

/* ==========================================================================
   6. 動畫效果 (Transitions)
   ========================================================================== */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(6px);
}
</style>
