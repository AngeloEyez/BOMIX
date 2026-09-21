<template>
  <Teleport to="body">
    <Transition name="hover-card-fade">
      <div
        v-if="visible && targetRect"
        ref="cardRef"
        tabindex="-1"
        class="bom-cell-hover-card"
        :style="cardPositionStyle"
        @mouseenter="handleMouseEnter"
        @mouseleave="$emit('card-mouse-leave')"
        @keydown="handleKeyDownOnCard"
      >
        <!-- 純粹顯示完整內容，無多餘標題與編輯按鈕 (支援 Description, Location, Notes) -->
        <div
          ref="contentRef"
          class="hover-card-content custom-scrollbar select-text text-[12px] leading-relaxed break-all whitespace-normal"
          :class="[
            field === 'locations' ? 'font-mono' : 'font-sans',
            { 'is-empty': !content }
          ]"
        >
          {{ content || getEmptyPlaceholder(field) }}
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
/**
 * @file BOMCellHoverCard.vue
 * @description BOM 表格單元格極簡浮動懸停內容視窗 (純檢視預覽卡片)
 * 
 * 特色：
 * 1. 極簡純粹預覽：無多餘按鈕干擾，直覺呈現儲存格完整內容（Description、Location、Notes）。
 * 2. 快捷鍵支援：支援滑鼠自由選取與 Ctrl+C 複製，並支援 Ctrl+A / Cmd+A 一鍵全選內容文字。
 * 3. 自動折行：使用 break-all 與 overflow-wrap: anywhere，保證長串文字在卡片範圍內精確折行，絕不溢出。
 * 4. 主題支援：完整支援 Light / Dark 主題自適應切換。
 */

import { ref, computed } from 'vue'
import type { BOMDisplayRow } from '../types'
import type { HoverCardField, CellRect } from '../composables/useCellHoverCard'

const props = withDefaults(
  defineProps<{
    visible: boolean
    field: HoverCardField | null
    row?: BOMDisplayRow | null
    content: string
    targetRect: CellRect | null
    isEditing?: boolean
    draftNotes?: string
  }>(),
  {
    row: null,
    isEditing: false,
    draftNotes: ''
  }
)

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

/**
 * 取得空白時之預設佔位文字
 * 
 * @param {HoverCardField | null} f - 欄位名稱
 * @returns {string} 佔位文字
 */
function getEmptyPlaceholder(f: HoverCardField | null): string {
  if (f === 'locations') return '（無位置標號）'
  if (f === 'notes') return '（無註記）'
  return '（無規格描述）'
}

/**
 * 滑鼠移入卡片本體時，觸發父層維持開啟，並聚焦卡片以利快捷鍵響應
 */
function handleMouseEnter(): void {
  emit('card-mouse-enter')
  if (cardRef.value) {
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
</script>

<style scoped>
/* 浮動懸停卡片根容器：支援 Light / Dark 主題自適應，採用純色高不透明背景與精緻投影 */
.bom-cell-hover-card {
  position: fixed !important;
  z-index: 99999 !important;
  pointer-events: auto !important;
  display: flex !important;
  flex-direction: column !important;
  border-radius: 6px !important;
  border: 1px solid var(--bom-card-border, #cbd5e1) !important;
  background-color: var(--bom-card-bg, #ffffff) !important;
  color: var(--bom-card-text, #1e293b) !important;
  box-shadow: var(--bom-card-shadow, 0 10px 30px -5px rgba(0, 0, 0, 0.25), 0 0 1px 1px rgba(0, 0, 0, 0.08)) !important;
  box-sizing: border-box !important;
  overflow: hidden !important;
  outline: none !important;
  user-select: text !important;
  -webkit-user-select: text !important;
  font-size: 12px !important;
}

/* 深色主題顯式全域備援覆寫 (包覆完整選擇器以避免 SFC 作用域解析器移除標籤) */
:global(.app-dark .bom-cell-hover-card),
:global(.dark .bom-cell-hover-card),
:global(html.app-dark .bom-cell-hover-card) {
  background-color: #1e1e1e !important;
  border-color: #454545 !important;
  color: #cccccc !important;
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
  color: var(--bom-card-text, #1e293b) !important;
  box-sizing: border-box !important;
  user-select: text !important;
  -webkit-user-select: text !important;
  max-height: 16rem;
  overflow-y: auto;
}

/* 內容為空時的次要文字色彩 */
.hover-card-content.is-empty {
  color: var(--bom-card-text-muted, #94a3b8) !important;
  font-style: italic !important;
}

:global(.app-dark .hover-card-content.is-empty),
:global(.dark .hover-card-content.is-empty) {
  color: #858585 !important;
}

/* 自訂捲軸樣式 (支援 Light / Dark 主題) */
.custom-scrollbar::-webkit-scrollbar {
  width: 5px;
  height: 5px;
}

.custom-scrollbar::-webkit-scrollbar-thumb {
  background-color: var(--bom-card-scrollbar-thumb, rgba(156, 163, 175, 0.5));
  border-radius: 9999px;
}

.custom-scrollbar::-webkit-scrollbar-thumb:hover {
  background-color: var(--bom-card-scrollbar-thumb-hover, rgba(107, 114, 128, 0.7));
}

/* 進出場過渡動畫 */
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
