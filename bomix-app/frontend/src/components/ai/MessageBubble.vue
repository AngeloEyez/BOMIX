<template>
  <!-- 1. 使用者訊息：Antigravity 風格膠囊卡片 (通欄、淡底色、右側小字時間與純圖標複製按鈕) -->
  <div v-if="isUser" class="user-message-row w-full mb-4 select-text">
    <div class="user-message-pill">
      <!-- 左側使用者輸入文字 -->
      <div class="user-message-text select-text">
        {{ content }}
      </div>

      <!-- 右側元資訊：時間與純圖標複製按鈕 -->
      <div class="user-message-meta select-none">
        <span class="user-message-time">{{ formattedTime }}</span>
        <button
          type="button"
          class="user-copy-icon-btn"
          :title="copied ? '已複製！' : '複製內容'"
          @click="copyContent"
        >
          <i :class="copied ? 'pi pi-check text-emerald-500' : 'pi pi-copy'"></i>
        </button>
      </div>
    </div>
  </div>

  <!-- 2. Assistant 訊息：極簡現代排版，移除頭像與助手名稱，時間採用小字體靠右 -->
  <div
    v-else
    class="assistant-message-row flex flex-col w-full mb-5 select-text"
    @click="handleContainerClick"
  >
    <!-- 頂部時間標籤 (淡色小字體靠右) -->
    <div class="assistant-meta-bar select-none">
      <span class="assistant-time font-mono">{{ formattedTime }}</span>
    </div>

    <!-- 工具呼叫清單 (Antigravity 緊湊折疊條) -->
    <div v-if="toolCalls && toolCalls.length > 0" class="w-full mb-2 space-y-1">
      <ToolCallCard
        v-for="tc in toolCalls"
        :key="tc.id"
        :tool-name="tc.name"
        :arguments="tc.arguments"
        :result="tc.result"
        :status="tc.status"
        :explanation="tc.explanation"
      />
    </div>

    <!-- 訊息內容主體 (安全 Markdown 渲染) -->
    <div
      class="assistant-content-wrapper relative w-full text-sm leading-relaxed select-text"
    >
      <div
        v-if="renderedHTML"
        class="markdown-body select-text"
        v-html="renderedHTML"
      ></div>
      <div v-else-if="isStreaming && (!toolCalls || toolCalls.length === 0)" class="text-surface-400 dark:text-surface-500 text-xs italic py-1 opacity-75">
        正在思考與整理資料...
      </div>

      <!-- 串流中打字游標 -->
      <span v-if="isStreaming" class="streaming-cursor inline-block w-1.5 h-4 ml-0.5 bg-primary align-middle animate-pulse"></span>
    </div>

    <!-- 底部快捷動作工具列 (純圖標複製) -->
    <div v-if="!isStreaming && content" class="flex items-center gap-2 mt-2 select-none">
      <button
        type="button"
        class="user-copy-icon-btn text-surface-400 hover:text-surface-600 dark:hover:text-surface-200 cursor-pointer p-1 rounded hover:bg-surface-100 dark:hover:bg-surface-800/80 transition-all"
        :title="copied ? '已複製！' : '複製內容'"
        @click="copyContent"
      >
        <i :class="copied ? 'pi pi-check text-emerald-500' : 'pi pi-copy'" class="text-[11px]"></i>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import hljs from 'highlight.js'
import 'highlight.js/styles/github-dark.css'
import ToolCallCard from './ToolCallCard.vue'
import type { ToolCallItem } from '../../stores/aiChat'

const props = defineProps<{
  role: 'user' | 'assistant' | 'system'
  content: string
  toolCalls?: ToolCallItem[]
  timestamp: number
  isStreaming?: boolean
}>()

const isUser = computed(() => props.role === 'user')
const copied = ref(false)

// 格式化時間戳 (輸出如 8:30 AM)
const formattedTime = computed(() => {
  const d = new Date(props.timestamp)
  return d.toLocaleTimeString([], { hour: 'numeric', minute: '2-digit', hour12: true })
})

// 設定 marked 與代碼高亮及一鍵純圖標複製代碼按鈕
marked.use({
  breaks: true,
  gfm: true,
  renderer: {
    code({ text, lang }: { text: string; lang?: string }) {
      const language = lang && hljs.getLanguage(lang) ? lang : 'plaintext'
      let highlighted = text
      try {
        highlighted = hljs.highlight(text, { language }).value
      } catch {
        highlighted = text
      }
      const encodedCode = encodeURIComponent(text)
      return `<div class="code-block my-3 rounded-lg overflow-hidden border border-surface-200 dark:border-surface-700/80 bg-surface-950 text-surface-100 text-xs shadow-xs"><div class="flex items-center justify-between px-3 py-1.5 bg-surface-900 border-b border-surface-800 text-[11px] text-surface-400 font-mono select-none"><span>${language}</span><button type="button" class="copy-code-btn inline-flex items-center justify-center w-5 h-5 rounded text-surface-400 hover:text-surface-100 bg-surface-800 hover:bg-surface-700 transition cursor-pointer border border-surface-700/60" title="複製代碼" data-raw="${encodedCode}"><i class="pi pi-copy text-[10px]"></i></button></div><pre class="p-3.5 overflow-x-auto font-mono leading-relaxed select-text selectable-text"><code class="hljs language-${language} select-text">${highlighted}</code></pre></div>`
    },
  },
})

// Markdown 渲染並進行 XSS 清洗
const renderedHTML = computed(() => {
  if (!props.content) return ''
  try {
    const rawHTML = marked.parse(props.content) as string
    return DOMPurify.sanitize(rawHTML)
  } catch (err) {
    console.error('Markdown 解析失敗:', err)
    return props.content
  }
})

// 處理容器內部點擊（例如代碼區塊右上角的純圖標複製代碼按鈕）
async function handleContainerClick(e: MouseEvent): Promise<void> {
  const target = e.target as HTMLElement
  const btn = target.closest('.copy-code-btn') as HTMLElement | null
  if (btn && btn.dataset.raw) {
    try {
      const rawText = decodeURIComponent(btn.dataset.raw)
      await navigator.clipboard.writeText(rawText)
      const icon = btn.querySelector('i')
      btn.title = '已複製！'
      if (icon) icon.className = 'pi pi-check text-emerald-400 text-[10px]'
      setTimeout(() => {
        btn.title = '複製代碼'
        if (icon) icon.className = 'pi pi-copy text-[10px]'
      }, 2000)
    } catch (err) {
      console.error('複製代碼失敗:', err)
    }
  }
}

// 複製整則內文至剪貼簿
async function copyContent(): Promise<void> {
  try {
    await navigator.clipboard.writeText(props.content)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  } catch (err) {
    console.error('複製內容失敗:', err)
  }
}
</script>

<style scoped>
.streaming-cursor {
  vertical-align: text-bottom;
}

/* ==========================================================================
   Antigravity 風格使用者訊息膠囊卡片 (User Message Pill)
   ========================================================================== */
.user-message-row {
  display: flex;
  width: 100%;
}

.user-message-pill {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  gap: 1rem;
  padding: 0.65rem 1rem;
  border-radius: 0.85rem; /* ~14px 圓角，吻合截圖風格 */
  box-sizing: border-box;

  /* 透過 CSS 變數動態依據主題無縫切換，保證深色模式下百分之百為質感深灰色 (#2e2e32) */
  background-color: var(--user-bubble-bg, #f1f5f9) !important;
  border: 1px solid var(--user-bubble-border, #e2e8f0) !important;
  color: var(--user-bubble-text, #1e293b) !important;
  transition: background-color 0.15s ease, border-color 0.15s ease, color 0.15s ease;
}

.user-message-text {
  flex: 1 1 0%;
  font-size: 0.875rem;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-word;
  user-select: text !important;
  -webkit-user-select: text !important;
}

.user-message-meta {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-shrink: 0;
}

.user-message-time {
  font-size: 0.72rem;
  color: var(--user-bubble-time, #94a3b8) !important;
  opacity: 0.9;
  font-variant-numeric: tabular-nums;
  letter-spacing: 0.02em;
}

.user-copy-icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 4px;
  background: transparent;
  border: none;
  cursor: pointer;
  color: var(--user-bubble-time, #94a3b8) !important;
  opacity: 0.75;
  transition: opacity 0.15s ease, background-color 0.15s ease, color 0.15s ease;
  padding: 0;
}

.user-copy-icon-btn:hover {
  opacity: 1;
  color: var(--user-bubble-text, #1e293b) !important;
  background-color: rgba(125, 125, 125, 0.15);
}

.user-copy-icon-btn i {
  font-size: 0.75rem;
}

/* ==========================================================================
   Assistant 訊息極簡現代排版
   ========================================================================== */
.assistant-message-row {
  width: 100%;
}

.assistant-meta-bar {
  display: flex !important;
  justify-content: flex-end !important;
  align-items: center !important;
  width: 100% !important;
  line-height: 1;
  margin-bottom: 0.25rem;
}

.assistant-time {
  font-size: 0.72rem;
  color: #94a3b8;
  opacity: 0.8;
  letter-spacing: 0.02em;
}

:global(.app-dark) .assistant-time,
:global(.dark) .assistant-time,
:global(html.app-dark) .assistant-time {
  color: #71717a !important;
}

.assistant-content-wrapper {
  color: var(--text-color, #1e293b);
}

:global(.app-dark) .assistant-content-wrapper,
:global(.dark) .assistant-content-wrapper,
:global(html.app-dark) .assistant-content-wrapper {
  color: #e2e8f0 !important;
}

/* 強制文字選取 */
.user-message-pill,
.user-message-text,
.assistant-message-row,
.assistant-content-wrapper,
.markdown-body,
.markdown-body :deep(p),
.markdown-body :deep(span),
.markdown-body :deep(li),
.markdown-body :deep(h1),
.markdown-body :deep(h2),
.markdown-body :deep(h3),
.markdown-body :deep(h4),
.markdown-body :deep(blockquote),
.markdown-body :deep(table),
.markdown-body :deep(thead),
.markdown-body :deep(tbody),
.markdown-body :deep(tr),
.markdown-body :deep(th),
.markdown-body :deep(td),
.markdown-body :deep(pre),
.markdown-body :deep(code) {
  -webkit-user-select: text !important;
  user-select: text !important;
}

/* Markdown 樣式深度客製 */
.markdown-body :deep(p) {
  margin-bottom: 0.65rem;
  line-height: 1.65;
  color: #1e293b;
}

:global(.app-dark) .markdown-body :deep(p),
:global(.dark) .markdown-body :deep(p),
:global(html.app-dark) .markdown-body :deep(p) {
  color: #f1f5f9 !important;
}

.markdown-body :deep(p:last-child) {
  margin-bottom: 0;
}

.markdown-body :deep(h1),
.markdown-body :deep(h2),
.markdown-body :deep(h3),
.markdown-body :deep(h4) {
  font-weight: 600;
  margin-top: 1rem;
  margin-bottom: 0.5rem;
  line-height: 1.3;
  color: #0f172a;
}

:global(.app-dark) .markdown-body :deep(h1),
:global(.app-dark) .markdown-body :deep(h2),
:global(.app-dark) .markdown-body :deep(h3),
:global(.app-dark) .markdown-body :deep(h4),
:global(.dark) .markdown-body :deep(h1),
:global(.dark) .markdown-body :deep(h2),
:global(.dark) .markdown-body :deep(h3),
:global(.dark) .markdown-body :deep(h4),
:global(html.app-dark) .markdown-body :deep(h1),
:global(html.app-dark) .markdown-body :deep(h2),
:global(html.app-dark) .markdown-body :deep(h3),
:global(html.app-dark) .markdown-body :deep(h4) {
  color: #ffffff !important;
}

.markdown-body :deep(h1) { font-size: 1.25rem; }
.markdown-body :deep(h2) { font-size: 1.15rem; }
.markdown-body :deep(h3) { font-size: 1.05rem; }

.markdown-body :deep(ul),
.markdown-body :deep(ol) {
  padding-left: 1.25rem;
  margin-bottom: 0.65rem;
}

.markdown-body :deep(li) {
  margin-bottom: 0.25rem;
}

.markdown-body :deep(blockquote) {
  border-left: 3px solid var(--p-primary-500, #3b82f6);
  padding: 0.35rem 0.75rem;
  margin: 0.65rem 0;
  background-color: rgba(59, 130, 246, 0.06);
  border-radius: 0 4px 4px 0;
  color: var(--p-surface-600, #4b5563);
}

.markdown-body :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 0.85rem 0;
  font-size: 0.825rem;
  border-radius: 6px;
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.markdown-body :deep(th) {
  background-color: rgba(0, 0, 0, 0.05);
  font-weight: 600;
  text-align: left;
  padding: 6px 10px;
  border: 1px solid rgba(125, 125, 125, 0.2);
}

.markdown-body :deep(td) {
  padding: 6px 10px;
  border: 1px solid rgba(125, 125, 125, 0.2);
}

.markdown-body :deep(tr:nth-child(even)) {
  background-color: rgba(125, 125, 125, 0.04);
}

.markdown-body :deep(code:not(.hljs)) {
  background-color: rgba(125, 125, 125, 0.15);
  padding: 2px 5px;
  border-radius: 4px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.875em;
}
</style>
