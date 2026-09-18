<template>
  <div class="antigravity-tool-item my-1 select-none">
    <!-- 單行緊湊折疊列 (Antigravity Style，採用顯著淡色感) -->
    <div
      class="tool-trigger-row inline-flex items-center gap-1.5 py-0.5 px-1.5 rounded cursor-pointer transition-colors group select-none"
      @click="isExpanded = !isExpanded"
    >
      <!-- 狀態圖示 (淡色) -->
      <span v-if="status === 'calling'" class="tool-icon-status tool-icon-calling inline-flex items-center shrink-0">
        <i class="pi pi-spin pi-spinner text-[10px]"></i>
      </span>
      <span v-else-if="status === 'error'" class="tool-icon-status tool-icon-error inline-flex items-center shrink-0">
        <i class="pi pi-times-circle text-[10px]"></i>
      </span>
      <span v-else class="tool-icon-status tool-icon-success inline-flex items-center shrink-0">
        <i class="pi pi-check text-[10px]"></i>
      </span>

      <!-- 動作標題文字 (顯著淡色感，明確次要階層) -->
      <span class="tool-label-text">
        {{ displayActionTitle }}
      </span>

      <!-- 展開 / 摺疊箭頭指示 (仿 Antigravity `>`) -->
      <i
        :class="isExpanded ? 'pi pi-chevron-down' : 'pi pi-chevron-right'"
        class="tool-chevron-icon text-[9px] shrink-0 ml-0.5"
      ></i>
    </div>

    <!-- 展開區塊：JSON 代碼塊與執行結果 -->
    <div v-show="isExpanded" class="tool-expand-panel mt-1 mb-2 pl-3 border-l-2 border-surface-200 dark:border-surface-700/60 space-y-2">
      <!-- 呼叫參數 (JSON Code Block) -->
      <div v-if="arguments" class="code-block-wrapper rounded-lg overflow-hidden border border-surface-200 dark:border-surface-700/80 bg-surface-950 text-surface-100 text-xs shadow-xs">
        <div class="code-block-header flex items-center justify-between px-3 py-1.5 bg-surface-900 border-b border-surface-800 text-[11px] text-surface-400 font-mono select-none">
          <span class="text-surface-300 font-medium">輸入參數 (Arguments: JSON)</span>
        </div>
        <pre class="p-3 overflow-x-auto font-mono text-xs leading-relaxed bg-surface-950 text-surface-100 max-h-56 overflow-y-auto select-text selectable-text"><code class="hljs language-json select-text" v-html="highlightedArguments"></code></pre>
      </div>

      <!-- 執行結果 (JSON 或文字 Code Block) -->
      <div v-if="result" class="code-block-wrapper rounded-lg overflow-hidden border border-surface-200 dark:border-surface-700/80 bg-surface-950 text-surface-100 text-xs shadow-xs">
        <div class="code-block-header flex items-center justify-between px-3 py-1.5 bg-surface-900 border-b border-surface-800 text-[11px] text-surface-400 font-mono select-none">
          <span class="text-surface-300 font-medium">執行結果 (Result: {{ isResultJson ? 'JSON' : 'Text' }})</span>
        </div>
        <pre class="p-3 overflow-x-auto font-mono text-xs leading-relaxed bg-surface-950 text-surface-100 max-h-64 overflow-y-auto select-text selectable-text"><code :class="['hljs', isResultJson ? 'language-json' : '']" class="select-text" v-html="highlightedResult"></code></pre>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import hljs from 'highlight.js'
import 'highlight.js/styles/github-dark.css'

const props = defineProps<{
  toolName: string
  arguments: string
  result?: string
  status: 'calling' | 'success' | 'error'
  explanation?: string
}>()

const isExpanded = ref(false)

// 友善工具名稱對照
const toolFriendlyName = computed(() => {
  switch (props.toolName) {
    case 'get_database_schema':
      return '查詢資料庫結構 (Schema)'
    case 'execute_readonly_sql':
      return '執行唯讀 SQL 查詢'
    case 'get_series_overview':
      return '取得系列與專案總覽'
    case 'compare_revisions_diff':
      return '比對 BOM 版本差異'
    default:
      return props.toolName
  }
})

// 顯示於折疊列的簡明標題
const displayActionTitle = computed(() => {
  const base = props.explanation || toolFriendlyName.value
  if (props.status === 'calling') {
    return `${base} 執行中...`
  }
  if (props.status === 'error') {
    return `${base} 失敗`
  }
  return base
})

// 格式化參數 (供原始文字複製使用)
const rawFormattedArguments = computed(() => {
  try {
    const obj = JSON.parse(props.arguments)
    return JSON.stringify(obj, null, 2)
  } catch {
    return props.arguments || '{}'
  }
})

// Highlight 上色後的參數 (HTML)
const highlightedArguments = computed(() => {
  try {
    return hljs.highlight(rawFormattedArguments.value, { language: 'json' }).value
  } catch {
    return rawFormattedArguments.value
  }
})

// 判斷回傳結果是否為 JSON
const isResultJson = computed(() => {
  if (!props.result) return false
  try {
    JSON.parse(props.result)
    return true
  } catch {
    return false
  }
})

// 格式化結果 (供原始文字複製使用)
const rawFormattedResult = computed(() => {
  if (!props.result) return ''
  try {
    const obj = JSON.parse(props.result)
    return JSON.stringify(obj, null, 2)
  } catch {
    return props.result
  }
})

// Highlight 上色後的回傳結果 (HTML)
const highlightedResult = computed(() => {
  if (!rawFormattedResult.value) return ''
  try {
    const lang = isResultJson.value ? 'json' : 'plaintext'
    return hljs.highlight(rawFormattedResult.value, { language: lang }).value
  } catch {
    return rawFormattedResult.value
  }
})
</script>

<style scoped>
.antigravity-tool-item {
  width: 100%;
}

.tool-trigger-row {
  max-width: 100%;
  border-radius: 4px;
  transition: background-color 0.15s ease;
}

.tool-trigger-row:hover {
  background-color: rgba(125, 125, 125, 0.08);
}

/* ==========================================================================
   動作標題文字：強化淡色感（次要層次），不搶主輸出視覺焦點
   ========================================================================== */

/* 1. Light Theme (預設)：文字明顯淡化為 Slate-400，字級微縮 */
.tool-label-text {
  font-size: 0.72rem; /* ~11.5px */
  line-height: 1.4;
  color: #94a3b8; /* 明顯淡色，與正文黑字形成鮮明反差 */
  font-weight: 400;
  letter-spacing: 0.01em;
  transition: color 0.15s ease;
}

.tool-trigger-row:hover .tool-label-text {
  color: #64748b; /* 滑鼠懸停時微調提亮 */
}

/* 狀態圖示淡化 */
.tool-icon-status {
  font-size: 10px;
  color: #94a3b8;
  opacity: 0.8;
}

.tool-icon-calling {
  color: #60a5fa;
}

.tool-icon-error {
  color: #f87171;
}

.tool-icon-success {
  color: #10b981;
  opacity: 0.75;
}

.tool-chevron-icon {
  color: #cbd5e1;
  opacity: 0.8;
  transition: color 0.15s ease, opacity 0.15s ease;
}

.tool-trigger-row:hover .tool-chevron-icon {
  color: #94a3b8;
  opacity: 1;
}

/* ==========================================================================
   2. Dark Theme：暗灰淡色（Zinc 500），對比度顯著低於正式輸出的高亮白字
   ========================================================================== */
:global(.app-dark) .tool-label-text,
:global(.dark) .tool-label-text,
:global(html.app-dark) .tool-label-text {
  color: #71717a !important; /* 暗淡次要灰，徹底拉開與正文白字的層次 */
}

:global(.app-dark) .tool-trigger-row:hover .tool-label-text,
:global(.dark) .tool-trigger-row:hover .tool-label-text,
:global(html.app-dark) .tool-trigger-row:hover .tool-label-text {
  color: #a1a1aa !important; /* 滑鼠懸停時提亮 */
}

:global(.app-dark) .tool-icon-status,
:global(.dark) .tool-icon-status,
:global(html.app-dark) .tool-icon-status {
  color: #71717a !important;
}

:global(.app-dark) .tool-icon-calling,
:global(.dark) .tool-icon-calling,
:global(html.app-dark) .tool-icon-calling {
  color: #60a5fa !important;
}

:global(.app-dark) .tool-icon-error,
:global(.dark) .tool-icon-error,
:global(html.app-dark) .tool-icon-error {
  color: #f87171 !important;
}

:global(.app-dark) .tool-icon-success,
:global(.dark) .tool-icon-success,
:global(html.app-dark) .tool-icon-success {
  color: #34d399 !important;
  opacity: 0.65;
}

:global(.app-dark) .tool-chevron-icon,
:global(.dark) .tool-chevron-icon,
:global(html.app-dark) .tool-chevron-icon {
  color: #52525b !important;
  opacity: 0.7;
}

:global(.app-dark) .tool-trigger-row:hover .tool-chevron-icon,
:global(.dark) .tool-trigger-row:hover .tool-chevron-icon,
:global(html.app-dark) .tool-trigger-row:hover .tool-chevron-icon {
  color: #a1a1aa !important;
  opacity: 1;
}

pre,
pre code {
  -webkit-user-select: text !important;
  user-select: text !important;
}
</style>
