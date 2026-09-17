<template>
  <div class="tool-call-card rounded-lg border border-surface-200 dark:border-surface-700/80 bg-surface-50 dark:bg-surface-900/60 overflow-hidden text-xs my-2 transition-all">
    <!-- 工具標題條 -->
    <div
      class="flex items-center justify-between px-3 py-2 cursor-pointer hover:bg-surface-100 dark:hover:bg-surface-800/60 select-none transition-colors"
      @click="isExpanded = !isExpanded"
    >
      <div class="flex items-center gap-2 min-w-0">
        <!-- 工具狀態圖示 -->
        <div class="flex items-center justify-center w-5 h-5 rounded bg-surface-200/80 dark:bg-surface-800 text-surface-600 dark:text-surface-300 shrink-0">
          <i :class="toolIcon" class="text-xs"></i>
        </div>

        <!-- 工具名稱與說明 -->
        <span class="font-semibold text-surface-800 dark:text-surface-200 truncate">
          {{ toolFriendlyName }}
        </span>

        <span v-if="explanation" class="text-surface-500 dark:text-surface-400 text-[11px] truncate max-w-[260px]">
          — {{ explanation }}
        </span>
      </div>

      <div class="flex items-center gap-2 shrink-0 ml-2">
        <!-- 狀態徽章 -->
        <span v-if="status === 'calling'" class="inline-flex items-center gap-1 text-primary-600 dark:text-primary-400 font-medium">
          <i class="pi pi-spin pi-spinner text-[11px]"></i>
          <span>執行中</span>
        </span>
        <span v-else-if="status === 'success'" class="inline-flex items-center gap-1 text-emerald-600 dark:text-emerald-400 font-medium">
          <i class="pi pi-check text-[11px]"></i>
          <span>完成</span>
        </span>
        <span v-else-if="status === 'error'" class="inline-flex items-center gap-1 text-rose-500 dark:text-rose-400 font-medium">
          <i class="pi pi-times text-[11px]"></i>
          <span>失敗</span>
        </span>

        <!-- 展開箭頭 -->
        <i
          :class="isExpanded ? 'pi pi-chevron-down' : 'pi pi-chevron-right'"
          class="text-[10px] text-surface-400 transition-transform duration-200"
        ></i>
      </div>
    </div>

    <!-- 摺疊展開區塊：參數與回傳結果 -->
    <div v-show="isExpanded" class="border-t border-surface-200 dark:border-surface-700/80 p-3 bg-surface-100/50 dark:bg-surface-950/40 space-y-2.5">
      <!-- 呼叫參數 -->
      <div>
        <div class="text-[11px] font-medium text-surface-500 dark:text-surface-400 mb-1 flex items-center gap-1">
          <i class="pi pi-arrow-circle-right text-[10px]"></i>
          輸入參數 (Arguments)
        </div>
        <pre class="bg-surface-900 text-surface-100 p-2.5 rounded text-[11px] font-mono overflow-x-auto whitespace-pre-wrap leading-relaxed max-h-48 overflow-y-auto">{{ formattedArguments }}</pre>
      </div>

      <!-- 執行結果 -->
      <div v-if="result">
        <div class="text-[11px] font-medium text-surface-500 dark:text-surface-400 mb-1 flex items-center gap-1">
          <i class="pi pi-arrow-circle-left text-[10px]"></i>
          執行結果 (Result)
        </div>
        <pre class="bg-surface-900 text-surface-100 p-2.5 rounded text-[11px] font-mono overflow-x-auto whitespace-pre-wrap leading-relaxed max-h-60 overflow-y-auto">{{ formattedResult }}</pre>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'

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

// 工具對應圖示
const toolIcon = computed(() => {
  switch (props.toolName) {
    case 'get_database_schema':
      return 'pi pi-database'
    case 'execute_readonly_sql':
      return 'pi pi-code'
    case 'get_series_overview':
      return 'pi pi-folder-open'
    case 'compare_revisions_diff':
      return 'pi pi-arrows-h'
    default:
      return 'pi pi-cog'
  }
})

// 格式化參數 JSON
const formattedArguments = computed(() => {
  try {
    const obj = JSON.parse(props.arguments)
    return JSON.stringify(obj, null, 2)
  } catch {
    return props.arguments || '{}'
  }
})

// 格式化結果 JSON
const formattedResult = computed(() => {
  if (!props.result) return ''
  try {
    const obj = JSON.parse(props.result)
    return JSON.stringify(obj, null, 2)
  } catch {
    return props.result
  }
})
</script>
