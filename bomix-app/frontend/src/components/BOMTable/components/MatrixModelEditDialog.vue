<template>
  <Dialog
    v-model:visible="dialogVisible"
    modal
    :header="dialogTitle"
    :style="{ width: '295px', maxWidth: '95vw' }"
    class="matrix-model-edit-dialog"
    :closable="!isSaving"
  >
    <!-- 自訂 Header：僅顯示圖標與 projectcode phase version，使用粗體 -->
    <template #header>
      <div class="model-edit-header">
        <i class="pi pi-sliders-h header-icon"></i>
        <span class="header-title">{{ dialogTitle }}</span>
      </div>
    </template>

    <div class="dialog-content-body">
      <!-- 隱藏機種保留勾選紀錄之提示條（當存在已勾選歷史資料的 Model 被設為隱藏時顯示） -->
      <div v-if="hiddenWithSelectionNamesText" class="hide-info-box">
        <i class="pi pi-info-circle info-icon"></i>
        <div class="warning-text-wrap">
          <div class="warning-title">已隱藏之機種將保留歷史勾選紀錄</div>
          <div class="warning-desc">
            {{ hiddenWithSelectionNamesText }} 在表格與匯出中將不顯示，但既有勾選紀錄與數量會保留於資料庫，日後重新勾選啟用即可恢復。
          </div>
        </div>
      </div>

      <!-- 工具列：同一行顯示啟用統計與純符號增減按鈕 -->
      <div class="model-edit-toolbar">
        <div class="count-text">
          已啟用 <span class="count-num">{{ enabledCount }}</span> / {{ modelList.length }} 個機種
        </div>
        <div class="btn-group">
          <!-- 減少按鈕 (純圖標符號，插槽自訂與絕對正中置中) -->
          <Button
            size="small"
            severity="secondary"
            outlined
            class="icon-only-btn"
            :disabled="modelList.length <= 1 || isSaving"
            @click="handleRemoveModel"
            title="減少尾端 Model 列 (最少保留 1 列)"
          >
            <i class="pi pi-minus btn-icon minus-icon"></i>
          </Button>
          <!-- 增加按鈕 (純圖標符號，插槽自訂與絕對正中置中) -->
          <Button
            size="small"
            severity="primary"
            class="icon-only-btn"
            :disabled="isSaving"
            @click="handleAddModel"
            title="增加尾端 Model"
          >
            <i class="pi pi-plus btn-icon plus-icon"></i>
          </Button>
        </div>
      </div>

      <!-- 中間區域：各 Model 顯示為 "[✓] Model A : [ 147 ]" 同一行格式 -->
      <div class="model-list-container">
        <div
          v-for="item in modelList"
          :key="item.sortOrder"
          class="model-item-row"
          :class="{ 'is-row-hidden': !item.enabled }"
        >
          <!-- 左側：啟用開關 + Model 名稱 -->
          <div class="model-left-group">
            <Checkbox
              v-model="item.enabled"
              binary
              :disabled="isSaving"
              :input-id="`model-enable-${item.sortOrder}`"
              @update:model-value="(val: boolean) => handleToggleModel(item, Boolean(val))"
            />
            <label
              :for="`model-enable-${item.sortOrder}`"
              class="model-label font-mono"
              :class="{ 'is-disabled-label': !item.enabled }"
            >
              Model {{ item.modelAlias }} :
            </label>
          </div>

          <!-- 右側：Model Qty 輸入框 (未啟用時鎖定唯讀，保留原數量供日後恢復) -->
          <InputNumber
            v-model="item.qty"
            :min="0"
            size="small"
            class="compact-input-number"
            inputClass="compact-input-text"
            :disabled="isSaving || !item.enabled"
          />
        </div>
      </div>
    </div>

    <!-- 底部操作按鈕 (全部 size="small") -->
    <template #footer>
      <div class="dialog-footer-box">
        <Button
          label="取消"
          icon="pi pi-times"
          text
          severity="secondary"
          size="small"
          class="footer-btn"
          :disabled="isSaving"
          @click="handleCancelDialog"
        />
        <Button
          label="確定"
          icon="pi pi-check"
          severity="primary"
          size="small"
          class="footer-btn"
          :loading="isSaving"
          @click="handleSave"
        />
      </div>
    </template>
  </Dialog>
</template>

<script setup lang="ts">
/**
 * @file MatrixModelEditDialog.vue
 * @description Matrix 模式 Model 表頭點擊機種設定編輯對話框元件 (VS Code Style 極致緊湊版)
 * 
 * 支援稀疏 Model 設定（例如僅啟用 Model C、隱藏 Model A/B）：
 * 1. 完整列出 0 .. maxSortOrder 的所有 Model 列，每列提供「啟用/隱藏 Checkbox + Qty 輸入框」；
 * 2. 取消勾選（隱藏）機種時保留資料庫中的舊勾選紀錄與數量，重新勾選啟用時自動恢復；
 * 3. 儲存時嚴格保留每列真實的 sort_order 與 is_hidden 狀態，避免錯位覆蓋。
 */

import { ref, computed, watch } from 'vue'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import InputNumber from 'primevue/inputnumber'
import Checkbox from 'primevue/checkbox'
import { UpdateRevisionMatrixModels, type MatrixModelInput } from '../../../services/api'
import { useProjectStore, useLogStore } from '../../../stores'
import type { MatrixModelColumnInfo, RevisionModelItem } from '../types'

interface Props {
  /** 控制對話框顯示狀態 */
  visible: boolean
  /** 所屬 Revision ID */
  revisionId: number
  /** 專案代碼 (例如：DEMO) */
  projectCode: string
  /** 階段 (例如：EVT) */
  phase: string
  /** 版本 (例如：1.0) */
  version: string
  /** 該 Revision 當前顯示中之 Model 欄位清單 */
  initialModels: MatrixModelColumnInfo[]
  /** 該 Revision 資料庫中所有 Model 記錄（含已隱藏者） */
  revisionModels?: RevisionModelItem[]
}

interface Emits {
  (e: 'update:visible', val: boolean): void
  (e: 'saved'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const projectStore = useProjectStore()
const logStore = useLogStore()

/** 內部對話框顯示狀態 */
const dialogVisible = computed({
  get: () => props.visible,
  set: (val: boolean) => emit('update:visible', val)
})

/** 內部暫存的 Model 編輯項目結構 */
interface EditableModelItem {
  id?: number
  sortOrder: number
  modelAlias: string
  qty: number
  enabled: boolean
  existsInDb: boolean
  hasSelection: boolean
}

/** 編輯中的 Model 清單 */
const modelList = ref<EditableModelItem[]>([])

/** 是否正在儲存中 */
const isSaving = ref(false)

/**
 * 依 0-based 索引轉換為大寫字母代號 (0->A, 1->B, 25->Z, 26->AA)
 * @param {number} index - 0-based 索引
 * @returns {string} 字母代號
 */
function getModelOrderAlias(index: number): string {
  if (index < 0) return 'A'
  let name = ''
  let cur = index
  while (cur >= 0) {
    name = String.fromCharCode(65 + (cur % 26)) + name
    cur = Math.floor(cur / 26) - 1
  }
  return name
}

/**
 * 視窗標題：僅顯示 projectcode phase version (例如 TARIS SI1 0.4)
 */
const dialogTitle = computed(() => {
  return [props.projectCode, props.phase, props.version].filter(Boolean).join(' ') || '機種設定'
})

/** 目前已啟用的機種數 */
const enabledCount = computed(() => {
  return modelList.value.filter((m) => m.enabled).length
})

/**
 * 取得已隱藏但具備歷史勾選紀錄的 Model 名稱清單文字（供提示使用者資料已保留）
 */
const hiddenWithSelectionNamesText = computed(() => {
  const hiddenWithSel = modelList.value
    .filter((m) => !m.enabled && m.hasSelection)
    .map((m) => `Model ${m.modelAlias}`)
  return hiddenWithSel.join(', ')
})

/**
 * 監聽 visible 開啟時，結合 revisionModels（含已隱藏 DB 資料）與 initialModels 建立完整的 0..maxSortOrder 編輯列
 */
watch(
  () => props.visible,
  (newVal) => {
    if (newVal) {
      isSaving.value = false
      const dbModels = props.revisionModels || []
      const colModels = props.initialModels || []

      let maxOrder = 0
      for (const m of dbModels) {
        if (m.sort_order > maxOrder) maxOrder = m.sort_order
      }
      for (const c of colModels) {
        if (c.sortOrder > maxOrder) maxOrder = c.sortOrder
      }

      const items: EditableModelItem[] = []
      for (let order = 0; order <= maxOrder; order++) {
        const dbItem = dbModels.find((m) => m.sort_order === order)
        const colItem = colModels.find((c) => c.sortOrder === order)
        const alias = getModelOrderAlias(order)

        if (dbItem) {
          const hasSel = Boolean(dbItem.has_selection)
          const isActive = !dbItem.is_hidden && (dbItem.qty > 0 || hasSel)
          items.push({
            id: dbItem.id,
            sortOrder: order,
            modelAlias: alias,
            qty: dbItem.qty ?? 0,
            enabled: isActive,
            existsInDb: true,
            hasSelection: hasSel
          })
        } else if (colItem) {
          const isReadOnly = Boolean(colItem.isReadOnly)
          const isActive = !isReadOnly && (colItem.qty > 0 || colModels.length > 0)
          items.push({
            id: colItem.modelId || undefined,
            sortOrder: order,
            modelAlias: alias,
            qty: colItem.qty ?? 0,
            enabled: isActive,
            existsInDb: Boolean(colItem.modelId),
            hasSelection: false
          })
        } else {
          // 前置未建立的稀疏空缺（例如僅有 Model C 時，此處列出未啟用的 Model A、B 供使用者隨時啟用）
          items.push({
            sortOrder: order,
            modelAlias: alias,
            qty: 0,
            enabled: false,
            existsInDb: false,
            hasSelection: false
          })
        }
      }

      modelList.value = items
    }
  },
  { immediate: true }
)

/**
 * 切換單一 Model 的啟用 / 隱藏狀態
 * - 啟用時：若原本 Qty 為 0 且無歷史勾選，預設填入 Qty = 1 使其立即生效顯示
 * - 隱藏時：保留原本 Qty 與資料庫勾選紀錄，供日後重新啟用時無損恢復
 *
 * @param {EditableModelItem} item - 目標 Model 編輯項目
 * @param {boolean} enabled - 新的啟用狀態
 */
function handleToggleModel(item: EditableModelItem, enabled: boolean): void {
  item.enabled = enabled
  if (enabled && (!item.qty || item.qty <= 0) && !item.hasSelection) {
    item.qty = 1
  }
}

/**
 * 點擊 "+" 按鈕：於尾端增加一個 Model (預設啟用，Qty: 1)
 */
function handleAddModel(): void {
  const nextOrder = modelList.value.length > 0
    ? modelList.value[modelList.value.length - 1].sortOrder + 1
    : 0
  const alias = getModelOrderAlias(nextOrder)
  modelList.value.push({
    sortOrder: nextOrder,
    modelAlias: alias,
    qty: 1,
    enabled: true,
    existsInDb: false,
    hasSelection: false
  })
}

/**
 * 點擊 "-" 按鈕：移除尾端最後一列 Model (最少保留 1 列；若尾端已有歷史勾選，後端儲存時會自動保留並標記隱藏)
 */
function handleRemoveModel(): void {
  if (modelList.value.length <= 1) return
  modelList.value.pop()
}

/**
 * 點擊「確定」按鈕：將每一列真實的 sort_order、qty 與 is_hidden 寫入資料庫並即時通知外部更新
 */
async function handleSave(): Promise<void> {
  if (!props.revisionId || props.revisionId <= 0) return

  isSaving.value = true
  try {
    // 僅送出已啟用，或原本已存在於資料庫中的 Model（避免為從未使用過的前置空欄建立多餘資料列）
    const payload: MatrixModelInput[] = modelList.value
      .filter((m) => m.enabled || m.existsInDb)
      .map((m) => ({
        id: m.id || 0,
        sort_order: m.sortOrder,
        model_name: `Model ${m.modelAlias}`,
        qty: Math.max(0, Number(m.qty) || 0),
        is_hidden: !m.enabled
      }))

    const activeCount = modelList.value.filter(
      (m) => m.enabled && (Number(m.qty) > 0 || m.hasSelection)
    ).length

    logStore.addLogEntry(
      'INFO',
      `[MatrixModelEditDialog] 準備更新 RevisionID=${props.revisionId} 的 Model 設定 (送出 ${payload.length} 筆，有效啟用 ${activeCount} 個)`
    )

    // 1. 寫入資料庫
    await UpdateRevisionMatrixModels(props.revisionId, payload)

    // 2. 僅同步更新該 Revision 本身在 projectStore 中的有效 modelCount
    for (const proj of projectStore.projects) {
      const targetRev = proj.revisions?.find((r) => r.id === props.revisionId)
      if (targetRev) {
        targetRev.modelCount = activeCount
        break
      }
    }

    logStore.addLogEntry('INFO', `[MatrixModelEditDialog] 成功更新 RevisionID=${props.revisionId} 的 Model 設定`)

    // 3. 觸發完成事件，通知父元件重新載入 view
    emit('saved')
    dialogVisible.value = false
  } catch (error) {
    const msg = error instanceof Error ? error.message : String(error)
    logStore.addLogEntry('ERROR', `[MatrixModelEditDialog] 更新 Model 設定失敗: ${msg}`)
  } finally {
    isSaving.value = false
  }
}

/**
 * 處理使用者取消機種設定編輯對話框操作
 */
function handleCancelDialog(): void {
  dialogVisible.value = false
  logStore.addLogEntry('DEBUG', `[MatrixModelEditDialog] 使用者已取消機種設定編輯對話框 (RevisionID=${props.revisionId})`)
}
</script>

<style>
/* ==========================================================================
   MatrixModelEditDialog 專屬 VS Code 風格全域樣式
   注意：因 PrimeVue Dialog 預設 teleport 至 body，必須使用全域樣式保證 100% 生效
   ========================================================================== */

/* 1. 視窗邊框與內容間距嚴格設定為 10px，禁止外溢與水平滾動條 */
.matrix-model-edit-dialog.p-dialog {
  border-radius: 4px !important;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.25) !important;
  overflow: hidden !important;
}

.matrix-model-edit-dialog .p-dialog-header {
  padding: 10px 10px 4px 10px !important;
  border-bottom: none !important;
}

.matrix-model-edit-dialog .p-dialog-content {
  padding: 0 10px !important;
  overflow-x: hidden !important;
  overflow-y: auto !important;
  box-sizing: border-box !important;
}

.matrix-model-edit-dialog .p-dialog-footer {
  padding: 8px 10px 10px 10px !important;
  border-top: none !important;
}

/* 2. 標題：圖標 + 粗體 projectcode phase version */
.matrix-model-edit-dialog .model-edit-header {
  display: flex !important;
  flex-direction: row !important;
  align-items: center !important;
  gap: 8px !important;
  user-select: none !important;
}

.matrix-model-edit-dialog .header-icon {
  font-size: 14px !important;
  color: var(--primary-color, #10b981) !important;
}

.matrix-model-edit-dialog .header-title {
  font-size: 14px !important;
  font-weight: 700 !important;
  font-family: var(--font-mono, monospace) !important;
  color: var(--text-color, #1e293b) !important;
  letter-spacing: -0.2px !important;
}

.dark .matrix-model-edit-dialog .header-title {
  color: #f1f5f9 !important;
}

/* 右上角關閉按鈕：正方形 20x20，內部圖標絕對置中 */
.matrix-model-edit-dialog .p-dialog-header-actions .p-dialog-close-button {
  width: 20px !important;
  height: 20px !important;
  min-width: 20px !important;
  max-width: 20px !important;
  min-height: 20px !important;
  max-height: 20px !important;
  padding: 0 !important;
  border-radius: 3px !important;
  display: flex !important;
  align-items: center !important;
  justify-content: center !important;
  box-sizing: border-box !important;
}

.matrix-model-edit-dialog .p-dialog-header-actions .p-dialog-close-button svg,
.matrix-model-edit-dialog .p-dialog-header-actions .p-dialog-close-button .p-icon {
  font-size: 10px !important;
  width: 10px !important;
  height: 10px !important;
  margin: auto !important;
  display: flex !important;
  align-items: center !important;
  justify-content: center !important;
}

/* 3. 內容主體容器 */
.matrix-model-edit-dialog .dialog-content-body {
  display: flex !important;
  flex-direction: column !important;
  gap: 6px !important;
  width: 100% !important;
  box-sizing: border-box !important;
  overflow-x: hidden !important;
}

/* 隱藏提示框 */
.matrix-model-edit-dialog .hide-info-box {
  display: flex !important;
  flex-direction: row !important;
  align-items: flex-start !important;
  gap: 6px !important;
  padding: 6px 8px !important;
  border-radius: 3px !important;
  background: #eff6ff !important;
  border: 1px solid #93c5fd !important;
  color: #1e40af !important;
  box-sizing: border-box !important;
  width: 100% !important;
}

.dark .matrix-model-edit-dialog .hide-info-box {
  background: rgba(30, 58, 138, 0.25) !important;
  border-color: #1d4ed8 !important;
  color: #bfdbfe !important;
}

.matrix-model-edit-dialog .info-icon {
  font-size: 12px !important;
  margin-top: 2px !important;
  color: #2563eb !important;
  flex-shrink: 0 !important;
}

.dark .matrix-model-edit-dialog .info-icon {
  color: #60a5fa !important;
}

.matrix-model-edit-dialog .warning-title {
  font-weight: 700 !important;
  font-size: 11px !important;
}

.matrix-model-edit-dialog .warning-desc {
  font-size: 10.5px !important;
  margin-top: 1px !important;
  line-height: 1.3 !important;
}

/* 4. 工具列：同一行左右分佈 */
.matrix-model-edit-dialog .model-edit-toolbar {
  display: flex !important;
  flex-direction: row !important;
  align-items: center !important;
  justify-content: space-between !important;
  width: 100% !important;
  padding: 2px 0 !important;
  box-sizing: border-box !important;
}

.matrix-model-edit-dialog .count-text {
  font-size: 12px !important;
  color: var(--text-color-secondary, #64748b) !important;
  user-select: none !important;
}

.matrix-model-edit-dialog .count-num {
  font-weight: 700 !important;
  color: var(--text-color, #1e293b) !important;
  font-family: var(--font-mono, monospace) !important;
}

.dark .matrix-model-edit-dialog .count-num {
  color: #e2e8f0 !important;
}

.matrix-model-edit-dialog .btn-group {
  display: flex !important;
  flex-direction: row !important;
  align-items: center !important;
  gap: 5px !important;
}

/* 5. 純圖標增減按鈕 (正方形 22x22，符號絕對正中置中) */
.matrix-model-edit-dialog .icon-only-btn {
  width: 22px !important;
  height: 22px !important;
  min-width: 22px !important;
  max-width: 22px !important;
  min-height: 22px !important;
  max-height: 22px !important;
  padding: 0 !important;
  border-radius: 3px !important;
  display: flex !important;
  align-items: center !important;
  justify-content: center !important;
  line-height: 1 !important;
  box-sizing: border-box !important;
}

.matrix-model-edit-dialog .icon-only-btn .btn-icon {
  font-size: 10px !important;
  margin: 0 !important;
  padding: 0 !important;
  line-height: 1 !important;
  display: flex !important;
  align-items: center !important;
  justify-content: center !important;
}

/* 減號與加號基線光學中心校正：克服圖標字型默認下沉偏差 */
.matrix-model-edit-dialog .icon-only-btn .minus-icon {
  transform: translateY(-1.5px) !important;
}

.matrix-model-edit-dialog .icon-only-btn .plus-icon {
  transform: translateY(-0.5px) !important;
}

/* Footer 按鈕 */
.matrix-model-edit-dialog .footer-btn {
  height: 24px !important;
  min-height: 24px !important;
  padding: 0 10px !important;
  font-size: 11.5px !important;
  line-height: 1 !important;
  border-radius: 3px !important;
}

.matrix-model-edit-dialog .footer-btn .p-button-icon {
  font-size: 10.5px !important;
  margin-right: 4px !important;
}

/* 6. Model 列表：Model A : [ 147 ] 同一行排版，嚴防水平滾動條 */
.matrix-model-edit-dialog .model-list-container {
  display: flex !important;
  flex-direction: column !important;
  gap: 5px !important;
  max-height: 240px !important;
  overflow-y: auto !important;
  overflow-x: hidden !important;
  padding: 1px 0 !important;
  width: 100% !important;
  box-sizing: border-box !important;
}

.matrix-model-edit-dialog .model-item-row {
  display: flex !important;
  flex-direction: row !important;
  align-items: center !important;
  justify-content: space-between !important;
  gap: 8px !important;
  padding: 3px 6px !important;
  border: 1px solid var(--surface-border, #e2e8f0) !important;
  border-radius: 3px !important;
  background: var(--surface-card, #ffffff) !important;
  box-sizing: border-box !important;
  width: 100% !important;
}

.dark .matrix-model-edit-dialog .model-item-row {
  background: #252526 !important;
  border-color: #333333 !important;
}

.matrix-model-edit-dialog .model-item-row.is-row-hidden {
  opacity: 0.65 !important;
  background: #f8fafc !important;
}

.dark .matrix-model-edit-dialog .model-item-row.is-row-hidden {
  background: #1e1e1e !important;
}

.matrix-model-edit-dialog .model-left-group {
  display: flex !important;
  flex-direction: row !important;
  align-items: center !important;
  gap: 6px !important;
  min-width: 0 !important;
}

.matrix-model-edit-dialog .model-label {
  font-size: 12px !important;
  font-weight: 700 !important;
  color: var(--text-color, #1e293b) !important;
  white-space: nowrap !important;
  flex-shrink: 0 !important;
  user-select: none !important;
  cursor: pointer !important;
}

.matrix-model-edit-dialog .model-label.is-disabled-label {
  color: var(--text-color-secondary, #94a3b8) !important;
  text-decoration: line-through !important;
}

.dark .matrix-model-edit-dialog .model-label {
  color: #e2e8f0 !important;
}

/* InputNumber 輸入框尺寸緊湊，適當寬度不超出視窗範圍 */
.matrix-model-edit-dialog .compact-input-number {
  width: 70px !important;
  max-width: 70px !important;
  height: 22px !important;
  flex-shrink: 0 !important;
  box-sizing: border-box !important;
}

.matrix-model-edit-dialog .compact-input-number .p-inputnumber-input {
  width: 100% !important;
  height: 22px !important;
  padding: 0 4px !important;
  font-size: 11.5px !important;
  text-align: center !important;
  font-family: var(--font-mono, monospace) !important;
  border-radius: 2px !important;
  color: var(--text-color, #1e293b) !important;
  box-sizing: border-box !important;
}

.dark .matrix-model-edit-dialog .compact-input-number .p-inputnumber-input {
  color: #f1f5f9 !important;
  background: #1e1e1e !important;
}

/* 7. Footer */
.matrix-model-edit-dialog .dialog-footer-box {
  display: flex !important;
  flex-direction: row !important;
  justify-content: flex-end !important;
  gap: 6px !important;
  width: 100% !important;
  box-sizing: border-box !important;
}
</style>
