<template>
  <Teleport to="body">
    <Transition name="notes-editor-fade">
      <div
        v-if="visible && targetRect"
        ref="editorRef"
        tabindex="-1"
        class="bom-notes-editor"
        :style="editorPositionStyle"
        @pointerdown.stop
      >
        <!-- 極簡主體：純色底色、可縮放調整尺寸之多行輸入框 -->
        <textarea
          ref="textareaRef"
          v-model="localNotes"
          class="notes-textarea"
          placeholder="輸入註記說明 (Shift+Enter換行，Enter儲存)..."
          @keydown="handleTextareaKeyDown"
        />
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
/**
 * @file BOMNotesEditor.vue
 * @description Notes 欄位儲存格專用極簡小編輯視窗 (Popover Editor)
 * 
 * 核心特色：
 * 1. 極簡純粹設計：無標題列與說明文字，外框尺寸完全貼合 Textarea。
 * 2. 自由調整尺寸：Textarea 支援 resize: both，外層卡片自動隨之同步延伸。
 * 3. 純色不透明底色：容器與輸入框皆為 100% 不透明純色底色，避免視覺透底。
 * 4. 舒適均勻間距：外框與 Textarea 四周保持精準舒適之 padding。
 * 5. 快捷操作：Shift+Enter 換行，Enter 或點擊外部即時儲存，Esc 取消。
 */

import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import type { BOMDisplayRow } from '../types'
import type { CellRect } from '../composables/useCellHoverCard'

const props = defineProps<{
  /** 編輯視窗是否顯示 */
  visible: boolean
  /** 當前正在編輯的資料列物件 */
  row: BOMDisplayRow | null
  /** 觸發編輯之儲存格螢幕座標邊界矩形 */
  targetRect: CellRect | null
  /** 初始或當前的 Notes 內容 */
  initialNotes: string
}>()

const emit = defineEmits<{
  /** 觸發儲存事件 */
  (e: 'save', notes: string): void
  /** 觸發取消事件 */
  (e: 'cancel'): void
  /** 關閉編輯器 */
  (e: 'close'): void
}>()

/** 編輯器本體節點 DOM 參考 */
const editorRef = ref<HTMLElement | null>(null)

/** 文字輸入框 DOM 參考 */
const textareaRef = ref<HTMLTextAreaElement | null>(null)

/** 本機編輯中的草稿文字 */
const localNotes = ref('')

/** 標記是否剛開啟編輯視窗，防止當次點擊事件觸發外部點擊判定 */
let justOpened = false

// 當 visible 或 initialNotes 變動時同步更新草稿
watch(
  () => [props.visible, props.initialNotes],
  ([newVisible]) => {
    if (newVisible) {
      localNotes.value = props.initialNotes || ''
      justOpened = true
      nextTick(() => {
        setTimeout(() => {
          justOpened = false
        }, 80)
        if (textareaRef.value) {
          textareaRef.value.focus()
          const len = textareaRef.value.value.length
          textareaRef.value.setSelectionRange(len, len)
        }
      })
    }
  },
  { immediate: true }
)

/**
 * 處理鍵盤事件
 * 支援 Shift+Enter 換行，Enter 儲存，Esc 取消
 * 
 * @param {KeyboardEvent} event - 鍵盤事件物件
 */
function handleTextareaKeyDown(event: KeyboardEvent): void {
  if (event.key === 'Enter') {
    if (event.shiftKey) {
      // Shift + Enter: 允許原生換行，不阻止事件
      return
    }
    // 普通 Enter: 完成編輯並儲存
    event.preventDefault()
    event.stopPropagation()
    handleSave()
  } else if (event.key === 'Escape') {
    event.preventDefault()
    event.stopPropagation()
    handleCancel()
  }
}

/**
 * 執行儲存
 */
function handleSave(): void {
  emit('save', localNotes.value)
}

/**
 * 執行取消
 */
function handleCancel(): void {
  emit('cancel')
}

/**
 * 監聽視窗外部點擊事件 (Click Outside)
 * 若點擊在編輯視窗外部，自動儲存並關閉視窗
 * 
 * @param {PointerEvent} event - 指針事件物件
 */
function handlePointerDownOutside(event: PointerEvent): void {
  if (!props.visible || justOpened) return

  const target = event.target as Node | null
  if (!target) return

  // 若點擊在編輯器本身內部，不處理
  if (editorRef.value && editorRef.value.contains(target)) {
    return
  }

  // 點擊在編輯器外部，完成編輯並儲存
  handleSave()
}

/**
 * 全域按鍵監聽 (例如當焦點游標可能在視窗外時按 Esc)
 * 
 * @param {KeyboardEvent} event - 鍵盤事件物件
 */
function handleGlobalKeyDown(event: KeyboardEvent): void {
  if (!props.visible) return

  if (event.key === 'Escape') {
    handleCancel()
  }
}

onMounted(() => {
  window.addEventListener('pointerdown', handlePointerDownOutside, true)
  window.addEventListener('keydown', handleGlobalKeyDown)
})

onUnmounted(() => {
  window.removeEventListener('pointerdown', handlePointerDownOutside, true)
  window.removeEventListener('keydown', handleGlobalKeyDown)
})

/**
 * 彈出視窗定位計算
 * 僅負責座標計算 (left, top, bottom)，不寫死 width/maxWidth，
 * 讓外層容器尺寸自然貼合內部 Textarea 與周圍 padding。
 */
const editorPositionStyle = computed(() => {
  if (!props.targetRect) return { display: 'none' }

  const rect = props.targetRect
  const viewportWidth = window.innerWidth
  const viewportHeight = window.innerHeight

  // 預設初始寬度 280px + padding 16px ≈ 296px
  const ESTIMATED_WIDTH = 300
  const ESTIMATED_HEIGHT = 115

  // 1. 水平定位：貼齊儲存格左側，若貼齊後會超出螢幕右邊界，則向左平移
  let left = rect.left
  if (left + ESTIMATED_WIDTH > viewportWidth - 16) {
    left = Math.max(16, viewportWidth - ESTIMATED_WIDTH - 16)
  }
  if (left < 16) {
    left = 16
  }

  // 2. 垂直定位：下方空間足夠時向下展開，否則向上翻轉
  const spaceBelow = viewportHeight - rect.bottom
  const shouldFlipUp = spaceBelow < ESTIMATED_HEIGHT && rect.top > ESTIMATED_HEIGHT

  if (shouldFlipUp) {
    return {
      position: 'fixed' as const,
      zIndex: 999999,
      left: `${left}px`,
      top: 'auto',
      bottom: `${viewportHeight - rect.top + 4}px`,
    }
  }

  return {
    position: 'fixed' as const,
    zIndex: 999999,
    left: `${left}px`,
    top: `${rect.bottom + 4}px`,
    bottom: 'auto',
  }
})
</script>

<style scoped>
/* 外層容器：純色不透明背景，以 fit-content 與 inline-flex 緊密包覆 Textarea，四周保持精準舒適 padding */
.bom-notes-editor {
  position: fixed !important;
  z-index: 999999 !important;
  pointer-events: auto !important;
  box-sizing: border-box !important;
  display: inline-flex !important;
  flex-direction: column !important;
  width: fit-content !important;
  height: fit-content !important;
  max-width: none !important;
  padding: 8px !important;
  background-color: var(--bom-card-bg, #ffffff) !important;
  border: 1px solid var(--bom-card-border, #cbd5e1) !important;
  border-radius: 8px !important;
  box-shadow: var(--bom-card-shadow, 0 12px 32px -4px rgba(0, 0, 0, 0.25), 0 0 1px 1px rgba(0, 0, 0, 0.08)) !important;
  outline: none !important;
}

:global(.app-dark .bom-notes-editor),
:global(.dark .bom-notes-editor),
:global(html.app-dark .bom-notes-editor) {
  background-color: #1e1e1e !important;
  border-color: #454545 !important;
  box-shadow: 0 16px 36px -4px rgba(0, 0, 0, 0.75), 0 0 1px 1px #3c3c3c !important;
}

/* 內部 Textarea：純色底色、可調整大小 (resize: both)，舒適行距與內距 */
.notes-textarea {
  display: block;
  width: 280px;
  height: 96px;
  min-width: 180px;
  min-height: 60px;
  max-width: calc(100vw - 48px);
  max-height: calc(100vh - 48px);
  resize: both;
  overflow: auto;
  box-sizing: border-box;
  padding: 8px 10px;
  border: 1px solid var(--bom-card-input-border, #cbd5e1);
  border-radius: 6px;
  font-family: inherit;
  font-size: 12px;
  line-height: 1.5;
  background-color: var(--bom-card-input-bg, #ffffff);
  color: var(--bom-card-input-text, #1e293b);
  outline: none;
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}

.notes-textarea::placeholder {
  color: var(--bom-card-text-muted, #94a3b8);
}

:global(.app-dark .notes-textarea),
:global(.dark .notes-textarea),
:global(html.app-dark .notes-textarea) {
  background-color: #252526 !important;
  color: #e2e8f0 !important;
  border-color: #525252 !important;
}

:global(.app-dark .notes-textarea::placeholder),
:global(.dark .notes-textarea::placeholder) {
  color: #666666 !important;
}

.notes-textarea:focus {
  border-color: var(--p-primary-500, #10b981) !important;
  box-shadow: 0 0 0 2px rgba(16, 185, 129, 0.2) !important;
}

.notes-editor-fade-enter-active,
.notes-editor-fade-leave-active {
  transition: opacity 0.12s ease, transform 0.12s ease;
}

.notes-editor-fade-enter-from,
.notes-editor-fade-leave-to {
  opacity: 0;
  transform: scale(0.97) translateY(-2px);
}
</style>
