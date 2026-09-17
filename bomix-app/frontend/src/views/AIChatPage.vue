<template>
  <div class="ai-chat-page flex flex-col h-full bg-surface-50 dark:bg-surface-900 overflow-hidden">
    <!-- 頂部資訊工具列 -->
    <header class="flex items-center justify-between px-6 py-3 border-b border-surface-200 dark:border-surface-800 bg-surface-0 dark:bg-surface-900/90 shrink-0 select-none">
      <div class="flex items-center gap-2.5">
        <div class="w-8 h-8 rounded-lg bg-primary-100 dark:bg-primary-950/60 text-primary flex items-center justify-center">
          <i class="pi pi-sparkles text-base"></i>
        </div>
        <div>
          <h1 class="text-sm font-semibold text-surface-900 dark:text-surface-100 leading-none">
            AI 對話助手
          </h1>
          <p class="text-xs text-surface-500 dark:text-surface-400 mt-1 leading-none flex items-center gap-1.5">
            <span class="inline-block w-2 h-2 rounded-full bg-emerald-500"></span>
            <span>已連接當前系列：{{ activeSeriesName }}</span>
          </p>
        </div>
      </div>

      <!-- 右側操作按鈕 -->
      <div class="flex items-center gap-2">
        <!-- 停止生成按鈕 (僅在生成中顯示) -->
        <Button
          v-if="aiChatStore.isGenerating"
          label="停止生成"
          icon="pi pi-stop-circle"
          severity="danger"
          size="small"
          outlined
          @click="handleStop"
        />

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

    <!-- 對話訊息捲動區域 -->
    <main
      ref="messageContainerRef"
      class="flex-1 overflow-y-auto p-6 space-y-4"
    >
      <!-- 歡迎畫面 (尚未有訊息時顯示) -->
      <div v-if="!aiChatStore.hasMessages" class="max-w-2xl mx-auto my-auto py-12 flex flex-col items-center text-center">
        <div class="w-14 h-14 rounded-2xl bg-gradient-to-tr from-primary-500 to-indigo-600 text-white flex items-center justify-center text-2xl shadow-lg shadow-primary-500/20 mb-4">
          <i class="pi pi-sparkles"></i>
        </div>
        <h2 class="text-xl font-bold text-surface-900 dark:text-surface-100 mb-2">
          歡迎使用 BOMIX AI 助手
        </h2>
        <p class="text-sm text-surface-500 dark:text-surface-400 max-w-md mb-8 leading-relaxed">
          您可以透過自然語言查詢當前開啟的系列資料庫，支援零件追蹤、版本差異比對、料號跨專案統計等智慧功能。
        </p>

        <!-- 推薦快捷提問卡片 -->
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3 w-full text-left">
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
            @click="sendPrompt('請查詢 ASM1024 物料目前被哪些專案與 BOM 版本使用？')"
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
        <div v-if="aiChatStore.activeToolCalls.length > 0 && aiChatStore.isGenerating" class="max-w-[90%]">
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

      <!-- 底部捲動錨點 -->
      <div ref="bottomAnchorRef" class="h-2"></div>
    </main>

    <!-- 底部輸入區域 -->
    <footer class="p-4 border-t border-surface-200 dark:border-surface-800 bg-surface-0 dark:bg-surface-900/90 shrink-0">
      <div class="max-w-4xl mx-auto">
        <!-- 輸入框外框 -->
        <div class="flex items-end gap-2.5 p-2 rounded-2xl border border-surface-300 dark:border-surface-700 bg-surface-50 dark:bg-surface-800/80 focus-within:border-primary-500 focus-within:ring-2 focus-within:ring-primary-500/20 transition-all shadow-xs">
          <!-- 文本輸入框 -->
          <textarea
            ref="inputRef"
            v-model="inputPrompt"
            rows="1"
            class="flex-1 max-h-36 p-2 bg-transparent text-sm text-surface-900 dark:text-surface-100 placeholder-surface-400 dark:placeholder-surface-500 resize-none outline-none leading-relaxed"
            placeholder="輸入您的問題... (Enter 發送，Shift + Enter 換行)"
            :disabled="aiChatStore.isGenerating"
            @keydown="handleKeyDown"
            @input="adjustTextareaHeight"
          ></textarea>

          <!-- 發送按鈕 -->
          <Button
            type="button"
            icon="pi pi-arrow-up"
            rounded
            size="small"
            class="shrink-0 mb-0.5"
            :disabled="!canSend"
            :loading="aiChatStore.isGenerating"
            @click="handleSubmit"
            title="發送 (Enter)"
          />
        </div>

        <!-- 底部狀態與快捷提示文字 -->
        <div class="flex items-center justify-between px-2 pt-2 text-[11px] text-surface-400 select-none">
          <div class="flex items-center gap-1.5">
            <span v-if="aiChatStore.currentStatus" class="flex items-center gap-1.5 text-primary font-medium animate-pulse">
              <i class="pi pi-spin pi-spinner text-[10px]"></i>
              <span>{{ aiChatStore.currentStatus }}</span>
            </span>
            <span v-else>
              由 OpenAI 相容 API 與雙軌資料庫引擎驅動
            </span>
          </div>
          <div>
            Shift + Enter 換行 • Enter 發送
          </div>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, onMounted, watch } from 'vue'
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

// 當前開啟系列名稱
const activeSeriesName = computed(() => {
  return appStore.seriesInfo?.name || '未命名系列'
})

// 是否允許點擊發送
const canSend = computed(() => {
  return inputPrompt.value.trim().length > 0 && !aiChatStore.isGenerating
})

// 初始化事件監聽
onMounted(() => {
  aiChatStore.initEventListeners()
  focusInput()
})

// 自動聚焦輸入框
function focusInput(): void {
  nextTick(() => {
    inputRef.value?.focus()
  })
}

// 動態調整文字框高度
function adjustTextareaHeight(): void {
  const el = inputRef.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 140) + 'px'
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
  () => scrollToBottom(false)
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
.ai-chat-page {
  height: 100%;
}
</style>
