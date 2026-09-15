<template>
  <Teleport to="body">
    <Transition name="hover-card-fade">
      <div
        v-if="visible && targetRect"
        ref="cardRef"
        tabindex="-1"
        class="bom-cell-hover-card fixed z-[99999] flex flex-col rounded-md shadow-2xl border text-xs overflow-hidden select-text outline-none"
        :class="[
          'bg-white dark:bg-[#1e1e1e] text-slate-800 dark:text-[#cccccc]',
          'border-slate-300 dark:border-[#454545]'
        ]"
        :style="cardPositionStyle"
        @mouseenter="handleMouseEnter"
        @mouseleave="$emit('card-mouse-leave')"
        @keydown="handleKeyDownOnCard"
      >
        <!-- 1. Description 或 Location 模式：單純顯示完整內容，無多餘標題與按鈕 -->
        <div
          v-if="field === 'description' || field === 'locations'"
          ref="contentRef"
          class="hover-card-content max-h-64 overflow-y-auto custom-scrollbar select-text text-[12px] leading-relaxed break-all whitespace-normal"
          :class="field === 'locations' ? 'font-mono' : 'font-sans'"
          style="padding: 6px 8px; box-sizing: border-box;"
        >
          {{ content || (field === 'locations' ? '（無位置標號）' : '（無規格描述）') }}
        </div>

        <!-- 2. Notes 欄位模式：保留檢視與多行文字編輯能力 -->
        <div v-else-if="field === 'notes'" class="flex flex-col">
          <!-- Notes 檢視模式 -->
          <div
            v-if="!isEditing"
            class="relative group max-h-64 overflow-y-auto custom-scrollbar"
            style="padding: 6px 8px; box-sizing: border-box;"
          >
            <div
              ref="contentRef"
              class="hover-card-content select-text text-[12px] leading-relaxed break-all whitespace-normal font-sans pr-12 min-h-[32px]"
              :class="{ 'text-slate-400 dark:text-slate-500 italic': !content }"
            >
              {{ content || '（目前尚無註記，點擊編輯按鈕新增）' }}
            </div>

            <!-- 簡約編輯按鈕 -->
            <button
              type="button"
              class="absolute top-2 right-2 flex items-center gap-1 px-1.5 py-0.5 rounded text-[11px] bg-slate-100 dark:bg-[#2d2d2d] hover:bg-primary hover:text-white text-slate-600 dark:text-slate-300 transition-colors shadow-xs select-none"
              title="編輯註記說明"
              @click.stop="$emit('start-editing')"
            >
              <i class="pi pi-pencil text-[10px]" />
              <span>編輯</span>
            </button>
          </div>

          <!-- Notes 多行編輯模式 -->
          <div v-else class="p-3 flex flex-col gap-2 min-w-[340px]">
            <textarea
              ref="textareaRef"
              v-model="localDraftNotes"
              rows="4"
              class="w-full text-[12px] p-2 rounded border focus:outline-none focus:ring-1 focus:ring-primary font-sans resize-y bg-white dark:bg-[#252526] text-slate-800 dark:text-slate-100 border-slate-300 dark:border-[#444444]"
              placeholder="請輸入註記說明 (支援多行文字)..."
              @keydown.ctrl.enter="handleSaveNotes"
              @keydown.meta.enter="handleSaveNotes"
            />
            <div class="flex items-center justify-between select-none pt-0.5">
              <span class="text-[10px] text-slate-400 dark:text-slate-500">
                按 Ctrl+Enter 快速儲存
              </span>
              <div class="flex items-center gap-1.5">
                <button
                  type="button"
                  class="px-2 py-0.5 rounded text-[11px] border border-slate-300 dark:border-[#444] hover:bg-slate-100 dark:hover:bg-[#333] text-slate-600 dark:text-slate-300 transition-colors"
                  @click="$emit('cancel-editing')"
                >
                  取消
                </button>
                <button
                  type="button"
                  class="px-2.5 py-0.5 rounded text-[11px] bg-primary hover:bg-primary-emphasis text-white font-medium transition-colors shadow-sm"
                  @click="handleSaveNotes"
                >
                  儲存
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
/**
 * @file BOMCellHoverCard.vue
 * @description BOM 表格單元格極簡浮動懸停內容視窗
 * 
 * 特色：
 * 1. 極簡無干擾：無冗餘標題與按鈕，直覺呈現儲存格完整內容。
 * 2. 快捷鍵支援：支援滑鼠自由選取與 Ctrl+C 複製，並支援 Ctrl+A / Cmd+A 一鍵全選內容文字。
 * 3. 自動折行：使用 break-all 與 overflow-wrap: anywhere，保證 Location 等長串無空格文字在卡片範圍內精確折行，絕不溢出。
 * 4. Notes 編輯能力：Notes 欄位保留多行文字編輯與儲存介面。
 */

import { ref, computed, watch, nextTick } from 'vue'
import type { BOMDisplayRow } from '../types'
import type { HoverCardField, CellRect } from '../composables/useCellHoverCard'

const props = defineProps<{
  visible: boolean
  field: HoverCardField | null
  row: BOMDisplayRow | null
  content: string
  targetRect: CellRect | null
  isEditing: boolean
  draftNotes: string
}>()

const emit = defineEmits<{
  (e: 'card-mouse-enter'): void
  (e: 'card-mouse-leave'): void
  (e: 'close'): void
  (e: 'start-editing'): void
  (e: 'cancel-editing'): void
  (e: 'update:draftNotes', value: string): void
  (e: 'save-notes', notes: string): void
}>()

/** 卡片根節點 DOM 參考 */
const cardRef = ref<HTMLElement | null>(null)

/** 文字內容區 DOM 參考 */
const contentRef = ref<HTMLElement | null>(null)

/** 多行文字輸入區 DOM 參考 */
const textareaRef = ref<HTMLTextAreaElement | null>(null)

/**
 * 雙向同步 draftNotes
 */
const localDraftNotes = computed({
  get: () => props.draftNotes,
  set: (val: string) => emit('update:draftNotes', val)
})

/**
 * 當進入編輯模式時，自動聚焦文字輸入框
 */
watch(
  () => props.isEditing,
  (editing) => {
    if (editing) {
      nextTick(() => {
        if (textareaRef.value) {
          textareaRef.value.focus()
          const len = textareaRef.value.value.length
          textareaRef.value.setSelectionRange(len, len)
        }
      })
    }
  }
)

/**
 * 滑鼠移入卡片本體時，觸發父層維持開啟，並聚焦卡片以利快捷鍵響應
 */
function handleMouseEnter(): void {
  emit('card-mouse-enter')
  if (!props.isEditing && cardRef.value) {
    cardRef.value.focus()
  }
}

/**
 * 鍵盤事件監聽 (支援 Ctrl+A 全選卡片內文字)
 * 
 * @param {KeyboardEvent} event - 鍵盤事件物件
 */
function handleKeyDownOnCard(event: KeyboardEvent): void {
  // 判斷 Ctrl+A 或 Cmd+A
  if ((event.ctrlKey || event.metaKey) && (event.key === 'a' || event.key === 'A')) {
    // 若當前焦點在 textarea 內，走原生文字輸入框全選
    if (event.target instanceof HTMLTextAreaElement || event.target instanceof HTMLInputElement) {
      return
    }

    // 全選卡片內容文字
    if (contentRef.value) {
      event.preventDefault()
      event.stopPropagation()
      const selection = window.getSelection()
      if (selection) {
        const range = document.createRange()
        range.selectNodeContents(contentRef.value)
        selection.removeAllRanges()
        selection.addRange(range)
      }
    }
  }
}

/**
 * 卡片定位座標計算 (依據目標儲存格邊界自適應放置，並限制寬度防止溢出螢幕)
 */
const cardPositionStyle = computed(() => {
  if (!props.targetRect) return { display: 'none' }

  const rect = props.targetRect
  const viewportWidth = window.innerWidth
  const viewportHeight = window.innerHeight

  // 舒適閱讀寬度 (上限 440px，並保留兩側 16px 邊界)
  const CARD_WIDTH = Math.min(440, viewportWidth - 32)
  const ESTIMATED_HEIGHT = 200

  // 1. 水平定位計算 (預設貼齊儲存格左側，並防止溢出右側)
  let left = rect.left
  if (left + CARD_WIDTH > viewportWidth - 16) {
    left = Math.max(16, viewportWidth - CARD_WIDTH - 16)
  }
  if (left < 16) {
    left = 16
  }

  // 2. 垂直定位計算 (若下方空間足夠則向下彈出，否則向上翻轉)
  const spaceBelow = viewportHeight - rect.bottom
  const shouldFlipUp = spaceBelow < ESTIMATED_HEIGHT && rect.top > ESTIMATED_HEIGHT

  if (shouldFlipUp) {
    return {
      position: 'fixed' as const,
      zIndex: 99999,
      width: `${CARD_WIDTH}px`,
      maxWidth: `${CARD_WIDTH}px`,
      left: `${left}px`,
      top: 'auto',
      bottom: `${viewportHeight - rect.top + 6}px`,
    }
  }

  return {
    position: 'fixed' as const,
    zIndex: 99999,
    width: `${CARD_WIDTH}px`,
    maxWidth: `${CARD_WIDTH}px`,
    left: `${left}px`,
    top: `${rect.bottom + 6}px`,
    bottom: 'auto',
  }
})

/**
 * 處理儲存 Notes
 */
function handleSaveNotes(): void {
  emit('save-notes', localDraftNotes.value)
}
</script>

<style scoped>
.bom-cell-hover-card {
  position: fixed !important;
  z-index: 99999 !important;
  pointer-events: auto !important;
  background-color: #ffffff !important;
  border-radius: 6px !important;
  box-shadow: 0 10px 30px -5px rgba(0, 0, 0, 0.35), 0 0 1px 1px rgba(0, 0, 0, 0.15) !important;
  opacity: 1 !important;
  box-sizing: border-box !important;
}

:global(.app-dark) .bom-cell-hover-card,
:global(.dark) .bom-cell-hover-card {
  background-color: #1e1e1e !important;
  border-color: #454545 !important;
  box-shadow: 0 12px 36px -4px rgba(0, 0, 0, 0.7), 0 0 1px 1px #3c3c3c !important;
}

/* 強制長文字、無空格標號在卡片邊界處安全折行並保持充裕內距與行距 */
.hover-card-content {
  padding: 6px 8px !important;
  word-break: break-all !important;
  overflow-wrap: anywhere !important;
  white-space: normal !important;
  line-height: 1.65 !important;
  font-size: 12px !important;
  letter-spacing: 0.2px !important;
  box-sizing: border-box !important;
}

.custom-scrollbar::-webkit-scrollbar {
  width: 5px;
  height: 5px;
}

.custom-scrollbar::-webkit-scrollbar-thumb {
  background-color: rgba(156, 163, 175, 0.5);
  border-radius: 9999px;
}

.custom-scrollbar::-webkit-scrollbar-thumb:hover {
  background-color: rgba(107, 114, 128, 0.7);
}

.hover-card-fade-enter-active,
.hover-card-fade-leave-active {
  transition: opacity 0.12s ease, transform 0.12s ease;
}

.hover-card-fade-enter-from,
.hover-card-fade-leave-to {
  opacity: 0;
  transform: scale(0.98) translateY(-2px);
}
</style>
