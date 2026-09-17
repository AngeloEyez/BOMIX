<template>
  <div :class="['message-row flex w-full mb-4', isUser ? 'justify-end' : 'justify-start']">
    <!-- AI 頭像 (左側) -->
    <div
      v-if="!isUser"
      class="w-7 h-7 rounded-full bg-gradient-to-tr from-primary-600 to-indigo-500 text-white flex items-center justify-center text-xs shrink-0 mr-2.5 mt-1 shadow-sm"
    >
      <i class="pi pi-sparkles text-[11px]"></i>
    </div>

    <!-- 訊息內容主容器 -->
    <div :class="['message-bubble-wrapper flex flex-col', isUser ? 'items-end max-w-[85%]' : 'items-start max-w-[90%]']">
      <!-- 角色標籤與時間 (可選) -->
      <div class="text-[11px] text-surface-400 mb-1 px-1 flex items-center gap-1.5 select-none">
        <span>{{ isUser ? '您' : 'BOMIX 助手' }}</span>
        <span class="text-[10px] text-surface-400/60">•</span>
        <span class="text-[10px]">{{ formattedTime }}</span>
      </div>

      <!-- 工具呼叫卡片清單 (僅 Assistant 訊息展示) -->
      <div v-if="toolCalls && toolCalls.length > 0" class="w-full mb-1">
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

      <!-- 訊息泡泡本體 -->
      <div
        :class="[
          'message-bubble relative rounded-2xl px-4 py-3 text-sm leading-relaxed transition-shadow',
          isUser
            ? 'bg-primary text-primary-contrast rounded-tr-xs shadow-sm select-text whitespace-pre-wrap'
            : 'bg-surface-0 dark:bg-surface-800 border border-surface-200 dark:border-surface-700/70 text-surface-800 dark:text-surface-100 rounded-tl-xs shadow-sm w-full'
        ]"
      >
        <!-- 使用者訊息直接輸出文字 -->
        <template v-if="isUser">
          {{ content }}
        </template>

        <!-- Assistant 訊息使用安全 Markdown 渲染 -->
        <template v-else>
          <div
            v-if="renderedHTML"
            class="markdown-body"
            v-html="renderedHTML"
          ></div>
          <div v-else-if="isStreaming" class="text-surface-400 text-xs italic">
            正在思考與整理資料...
          </div>

          <!-- 串流中打字游標 -->
          <span v-if="isStreaming" class="streaming-cursor inline-block w-1.5 h-4 ml-0.5 bg-primary align-middle animate-pulse"></span>
        </template>
      </div>

      <!-- 底部快捷動作工具列 (複製) -->
      <div v-if="!isStreaming && content" class="flex items-center gap-2 mt-1 px-1 opacity-0 hover:opacity-100 focus-within:opacity-100 transition-opacity">
        <button
          type="button"
          class="text-[11px] text-surface-400 hover:text-surface-600 dark:hover:text-surface-200 flex items-center gap-1 cursor-pointer bg-transparent border-none p-1 rounded hover:bg-surface-100 dark:hover:bg-surface-800"
          :title="copied ? '已複製！' : '複製內容'"
          @click="copyContent"
        >
          <i :class="copied ? 'pi pi-check text-emerald-500' : 'pi pi-copy'" class="text-[11px]"></i>
          <span>{{ copied ? '已複製' : '複製' }}</span>
        </button>
      </div>
    </div>

    <!-- 使用者頭像 (右側) -->
    <div
      v-if="isUser"
      class="w-7 h-7 rounded-full bg-surface-200 dark:bg-surface-700 text-surface-700 dark:text-surface-200 flex items-center justify-center text-xs shrink-0 ml-2.5 mt-1 shadow-sm"
    >
      <i class="pi pi-user text-[11px]"></i>
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

// 格式化時間戳
const formattedTime = computed(() => {
  const d = new Date(props.timestamp)
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
})

// 設定 marked 與代碼高亮
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
      return `<div class="code-block my-2.5 rounded-lg overflow-hidden border border-surface-200 dark:border-surface-700 bg-surface-900 text-surface-100 text-xs"><div class="flex items-center justify-between px-3 py-1 bg-surface-950/80 text-[11px] text-surface-400 font-mono select-none"><span>${language}</span></div><pre class="p-3 overflow-x-auto font-mono leading-relaxed"><code class="hljs language-${language}">${highlighted}</code></pre></div>`
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

// 複製內文至剪貼簿
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

/* Markdown 樣式深度客製 */
.markdown-body :deep(p) {
  margin-bottom: 0.65rem;
  line-height: 1.65;
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
