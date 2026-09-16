<template>
  <Dialog
    v-model:visible="dialogVisible"
    modal
    :header="dialogTitle"
    :style="{ width: '270px', maxWidth: '95vw' }"
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
      <!-- 縮減 Model 刪除勾選紀錄之警告提示條 (減少 Model 時顯示) -->
      <div v-if="isModelCountReduced" class="reduce-warning-box">
        <i class="pi pi-exclamation-triangle warning-icon"></i>
        <div class="warning-text-wrap">
          <div class="warning-title">警告：減少 Model 將同步清除勾選紀錄！</div>
          <div class="warning-desc">
            Model 數量由 {{ initialCount }} 減為 {{ modelList.length }}。儲存後，被減少之 Model（{{ removedModelNamesText }}）底下的所有物料勾選紀錄將被同步清除。
          </div>
        </div>
      </div>

      <!-- 工具列：同一行顯示「目前共 x 個機種」與純符號按鈕 -->
      <div class="model-edit-toolbar">
        <div class="count-text">
          目前共 <span class="count-num">{{ modelList.length }}</span> 個機種
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
            title="減少 Model (最少保留 1 個)"
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
            title="增加 Model"
          >
            <i class="pi pi-plus btn-icon plus-icon"></i>
          </Button>
        </div>
      </div>

      <!-- 中間區域：各 Model 顯示為 "Model A : [ 147 ]" 同一行格式 -->
      <div class="model-list-container">
        <div
          v-for="item in modelList"
          :key="item.sortOrder"
          class="model-item-row"
        >
          <!-- 左側：Model 名稱顯示 "Model A : " -->
          <span class="model-label font-mono">Model {{ item.modelAlias }} :</span>

          <!-- 右側：Model Qty 輸入框 (小巧緊湊，不超出顯示範圍) -->
          <InputNumber
            v-model="item.qty"
            :min="0"
            size="small"
            class="compact-input-number"
            inputClass="compact-input-text"
            :disabled="isSaving"
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
          @click="dialogVisible = false"
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
 * 1. 視窗尺寸：橫向寬度精簡收縮至 270px，邊框維持 10px 緊湊間距；
 * 2. 標題：僅顯示圖標與 projectcode phase version (粗體)；
 * 3. 工具列：「目前共 x 個機種」與圖標符號按鈕在同一行；
 * 4. 增減按鈕：純圖標符號且水平垂直絕對正中置中；
 * 5. Model 列表：以 "Model A : [ 147 ]" 同一行格式顯示，數字框寬度適中不超出範圍；
 * 6. 按鈕尺寸：全部嚴格限定為 small (高 24px)；
 * 7. 防呆警告：減少 Model 時提示將同步清除對應勾選紀錄；
 * 8. 確定寫入：儲存後即時更新表格內容。
 */

import { ref, computed, watch } from 'vue'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import InputNumber from 'primevue/inputnumber'
import { UpdateRevisionMatrixModels, type MatrixModelInput } from '../../../services/api'
import { useAppStore, useLogStore } from '../../../stores'
import type { MatrixModelColumnInfo } from '../types'

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
  /** 該 Revision 當前既有的 Model 清單 */
  initialModels: MatrixModelColumnInfo[]
}

interface Emits {
  (e: 'update:visible', val: boolean): void
  (e: 'saved'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const appStore = useAppStore()
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
}

/** 編輯中的 Model 清單 */
const modelList = ref<EditableModelItem[]>([])

/** 開啟對話框時的原始 Model 數量，用於判定是否減少了 Model */
const initialCount = ref(1)

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

/**
 * 判定當前 Model 數量是否小於開啟時的初始數量 (即使用者點擊了減少 Model)
 */
const isModelCountReduced = computed(() => {
  return modelList.value.length < initialCount.value
})

/**
 * 取得被減少之 Model 代號文字說明 (例如 "Model B, Model C")
 */
const removedModelNamesText = computed(() => {
  if (!isModelCountReduced.value) return ''
  const removed: string[] = []
  for (let i = modelList.value.length; i < initialCount.value; i++) {
    removed.push(`Model ${getModelOrderAlias(i)}`)
  }
  return removed.join(', ')
})

/**
 * 監聽 visible 開啟時，從 props.initialModels 初始化可編輯列表
 */
watch(
  () => props.visible,
  (newVal) => {
    if (newVal) {
      isSaving.value = false
      if (props.initialModels && props.initialModels.length > 0) {
        initialCount.value = props.initialModels.length
        modelList.value = props.initialModels.map((m, idx) => ({
          id: m.modelId || undefined,
          sortOrder: m.sortOrder ?? idx,
          modelAlias: m.modelAlias || getModelOrderAlias(idx),
          qty: m.qty ?? 1
        }))
      } else {
        // 保底：若無則提供 1 個預設 Model A
        initialCount.value = 1
        modelList.value = [
          {
            sortOrder: 0,
            modelAlias: 'A',
            qty: 1
          }
        ]
      }
    }
  },
  { immediate: true }
)

/**
 * 點擊 "+" 按鈕：增加一個 Model (代號如 C，預設 Qty: 1)
 */
function handleAddModel(): void {
  const nextIndex = modelList.value.length
  const alias = getModelOrderAlias(nextIndex)
  modelList.value.push({
    sortOrder: nextIndex,
    modelAlias: alias,
    qty: 1
  })
}

/**
 * 點擊 "-" 按鈕：減少最後一個 Model (最少保留 1 個)
 */
function handleRemoveModel(): void {
  if (modelList.value.length <= 1) return
  modelList.value.pop()
}

/**
 * 點擊「確定」按鈕：將窗口中的資料寫入資料庫並即時通知外部更新
 */
async function handleSave(): Promise<void> {
  if (!props.revisionId || props.revisionId <= 0) return

  isSaving.value = true
  try {
    const payload: MatrixModelInput[] = modelList.value.map((m, idx) => ({
      id: m.id || 0,
      sort_order: idx,
      model_name: `Model ${m.modelAlias}`,
      qty: Number(m.qty) || 0
    }))

    logStore.addLogEntry(
      'INFO',
      `[MatrixModelEditDialog] 準備更新 RevisionID=${props.revisionId} 的 Model 設定 (共 ${payload.length} 個 Model)`
    )

    // 1. 寫入資料庫
    await UpdateRevisionMatrixModels(props.revisionId, payload)

    // 2. 同步更新 seriesInfo 的自訂 Model 數量紀錄 (若有)
    if (appStore.seriesInfo) {
      if (!appStore.seriesInfo.projectModelCounts) {
        appStore.seriesInfo.projectModelCounts = {}
      }
      appStore.seriesInfo.projectModelCounts[props.projectCode] = payload.length
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

/* 警告框 */
.matrix-model-edit-dialog .reduce-warning-box {
  display: flex !important;
  flex-direction: row !important;
  align-items: flex-start !important;
  gap: 6px !important;
  padding: 6px 8px !important;
  border-radius: 3px !important;
  background: #fffbeb !important;
  border: 1px solid #fcd34d !important;
  color: #92400e !important;
  box-sizing: border-box !important;
  width: 100% !important;
}

.dark .matrix-model-edit-dialog .reduce-warning-box {
  background: rgba(120, 53, 15, 0.25) !important;
  border-color: #78350f !important;
  color: #fde68a !important;
}

.matrix-model-edit-dialog .warning-icon {
  font-size: 12px !important;
  margin-top: 2px !important;
  color: #d97706 !important;
  flex-shrink: 0 !important;
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

.matrix-model-edit-dialog .model-label {
  font-size: 12px !important;
  font-weight: 700 !important;
  color: var(--text-color, #1e293b) !important;
  white-space: nowrap !important;
  flex-shrink: 0 !important;
  user-select: none !important;
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
