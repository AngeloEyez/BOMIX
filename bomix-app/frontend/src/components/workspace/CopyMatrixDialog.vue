<template>
  <Dialog
    v-model:visible="visibleModel"
    modal
    header="Matrix Selection 版本複製"
    :style="{ width: isAdvancedMode ? '640px' : '540px', maxWidth: '95vw' }"
    :closable="!isSubmitting"
  >
    <div class="copy-matrix-dialog-content">
      <!-- 動態警示橫幅：依整版模式或進階模式提示影響範圍，支援亮色與深色主題 -->
      <div
        class="flex items-start gap-2.5 p-3 rounded border bg-amber-50 border-amber-300 text-amber-900 dark:bg-amber-950/40 dark:border-amber-700/70 dark:text-amber-100"
      >
        <i class="pi pi-exclamation-triangle text-amber-500 dark:text-amber-400 text-lg flex-shrink-0 mt-0.5"></i>
        <span class="text-sm leading-relaxed" v-if="!isAdvancedMode">
          此操作將<strong>覆蓋</strong>目標版本的現有 Matrix Model 與 Selection！
        </span>
        <span class="text-sm leading-relaxed" v-else>
          此操作將<strong>覆蓋</strong>所指定目標 Model 的現有 Selection{{ copyQty ? '與打件數量 (Qty)' : '' }}（目標版本的其他 Model 將完整保留）！
        </span>
      </div>

      <div v-if="effectiveRevisions.length === 0" class="text-center py-4 text-sm text-gray-500 dark:text-gray-400">
        目前系列內尚無任何 BOM 版本資料。
      </div>

      <template v-else>
        <!-- 來源版本 -->
        <div class="form-group">
          <label for="copyMatrixSource" class="font-medium text-sm flex items-center justify-between">
            <span>來源版本 (Source)</span>
            <span v-if="selectedSource" class="text-xs text-gray-500 dark:text-gray-400">
              機種數: {{ selectedSource.modelCount }}
            </span>
          </label>
          <Select
            v-model="copyMatrixSourceId"
            :options="effectiveRevisions"
            option-label="label"
            option-value="id"
            placeholder="請選擇來源版本..."
            id="copyMatrixSource"
            class="w-full"
            :disabled="isSubmitting"
          >
            <template #option="slotProps">
              <div class="revision-option-row">
                <span class="revision-option-text">{{ slotProps.option.projectCode }} - {{ slotProps.option.phase }} {{ slotProps.option.version }}</span>
                <span
                  class="revision-model-badge"
                  :class="slotProps.option.modelCount > 0
                    ? 'bg-blue-50 text-blue-700 border border-blue-200 dark:bg-blue-950/60 dark:text-blue-300 dark:border-blue-800'
                    : 'bg-gray-100 text-gray-500 border border-gray-200 dark:bg-zinc-800 dark:text-gray-400 dark:border-zinc-700'"
                >
                  {{ slotProps.option.modelCount > 0 ? `${slotProps.option.modelCount} Models` : 'No Model' }}
                </span>
              </div>
            </template>
          </Select>

          <!-- 整版模式：沿用 modelCount 檢查 -->
          <p
            v-if="!isAdvancedMode && selectedSource && selectedSource.modelCount === 0"
            class="text-amber-600 dark:text-amber-400 text-xs flex items-center gap-1 mt-1"
          >
            <i class="pi pi-info-circle"></i>
            <span>此來源版本無任何機種 (No Model)，無法作為複製來源。</span>
          </p>

          <!-- 進階模式：依實際查詢到的未隱藏機種檢查 -->
          <p
            v-if="isAdvancedMode && copyMatrixSourceId && !isLoadingSourceModels && sourceModelsError"
            class="text-red-500 dark:text-red-400 text-xs flex items-center gap-1 mt-1"
          >
            <i class="pi pi-times-circle"></i>
            <span>載入來源機種失敗：{{ sourceModelsError }}</span>
          </p>
          <p
            v-else-if="isAdvancedMode && copyMatrixSourceId && !isLoadingSourceModels && sourceModels.length === 0"
            class="text-amber-600 dark:text-amber-400 text-xs flex items-center gap-1 mt-1"
          >
            <i class="pi pi-info-circle"></i>
            <span>此來源版本無任何啟用機種，無法作為複製來源。</span>
          </p>
        </div>

        <!-- 目標版本 -->
        <div class="form-group">
          <label for="copyMatrixTarget" class="font-medium text-sm flex items-center justify-between">
            <span>目標版本 (Target)</span>
            <span v-if="selectedTarget" class="text-xs text-gray-500 dark:text-gray-400">
              現有機種數: {{ selectedTarget.modelCount }}
            </span>
          </label>
          <Select
            v-model="copyMatrixTargetId"
            :options="effectiveRevisions"
            option-label="label"
            option-value="id"
            placeholder="請選擇目標版本..."
            id="copyMatrixTarget"
            class="w-full"
            :disabled="isSubmitting"
          >
            <template #option="slotProps">
              <div class="revision-option-row">
                <span class="revision-option-text">{{ slotProps.option.projectCode }} - {{ slotProps.option.phase }} {{ slotProps.option.version }}</span>
                <span
                  class="revision-model-badge"
                  :class="slotProps.option.modelCount > 0
                    ? 'bg-blue-50 text-blue-700 border border-blue-200 dark:bg-blue-950/60 dark:text-blue-300 dark:border-blue-800'
                    : 'bg-gray-100 text-gray-500 border border-gray-200 dark:bg-zinc-800 dark:text-gray-400 dark:border-zinc-700'"
                >
                  {{ slotProps.option.modelCount > 0 ? `${slotProps.option.modelCount} Models` : 'No Model' }}
                </span>
              </div>
            </template>
          </Select>

          <!-- 進階模式：目標版本無可用機種提醒 -->
          <p
            v-if="isAdvancedMode && copyMatrixTargetId && !isLoadingTargetModels && targetModelsError"
            class="text-red-500 dark:text-red-400 text-xs flex items-center gap-1 mt-1"
          >
            <i class="pi pi-times-circle"></i>
            <span>載入目標機種失敗：{{ targetModelsError }}</span>
          </p>
          <p
            v-else-if="isAdvancedMode && copyMatrixTargetId && !isLoadingTargetModels && targetModels.length === 0"
            class="text-amber-600 dark:text-amber-400 text-xs flex items-center gap-1 mt-1"
          >
            <i class="pi pi-info-circle"></i>
            <span>此目標版本尚無任何啟用機種，請先至機種設定建立機種，或關閉進階模式使用整版複製。</span>
          </p>
        </div>

        <!-- 同版本提示（整版模式阻擋、進階模式允許並提示） -->
        <p
          v-if="!isAdvancedMode && isSameRevision"
          class="text-red-500 dark:text-red-400 text-sm flex items-center gap-1"
        >
          <i class="pi pi-times-circle"></i>
          <span>來源版本與目標版本不可相同</span>
        </p>
        <p
          v-else-if="isAdvancedMode && isSameRevision"
          class="text-blue-600 dark:text-blue-400 text-xs flex items-center gap-1.5"
        >
          <i class="pi pi-info-circle"></i>
          <span>目前為同版本內機種複製模式，僅可複製至不同的目標 Model。</span>
        </p>

        <!-- 進階模式開關 -->
        <div
          class="flex items-center justify-between gap-3 pt-2 border-t border-gray-200 dark:border-zinc-700"
        >
          <div class="flex flex-col">
            <label for="copyMatrixAdvancedToggle" class="text-sm font-medium cursor-pointer select-none">
              進階模式（Model-to-Model 複製）
            </label>
            <span class="text-xs text-gray-500 dark:text-gray-400">
              指定單一或多個來源 Model 對應複製至目標既有 Model（支援同版本內複製）
            </span>
          </div>
          <ToggleSwitch
            v-model="isAdvancedMode"
            input-id="copyMatrixAdvancedToggle"
            :disabled="isSubmitting"
          />
        </div>

        <!-- 進階模式內容區塊 -->
        <div
          v-if="isAdvancedMode"
          class="flex flex-col gap-3 p-3 rounded border border-gray-200 bg-gray-50/70 dark:border-zinc-700 dark:bg-zinc-900/50"
        >
          <div class="flex items-center justify-between">
            <span class="text-sm font-medium">Model 對應清單</span>
            <Button
              label="新增對應"
              icon="pi pi-plus"
              size="small"
              outlined
              severity="secondary"
              :disabled="isAddMappingDisabled || isSubmitting"
              @click="handleAddMappingRow"
            />
          </div>

          <div
            v-if="!copyMatrixSourceId || !copyMatrixTargetId"
            class="text-xs text-gray-500 dark:text-gray-400 py-2 text-center"
          >
            請先選取來源版本與目標版本，以載入可用的 Model 清單。
          </div>

          <template v-else>
            <div class="flex flex-col gap-2.5 max-h-64 overflow-y-auto pr-0.5">
              <div
                v-for="(row, index) in mappingRows"
                :key="row.rowKey"
                class="flex flex-col gap-1"
              >
                <div class="flex items-center gap-2">
                  <!-- 來源 Model 選單 -->
                  <Select
                    v-model="row.sourceModelId"
                    :options="sourceModelOptions"
                    option-label="displayLabel"
                    option-value="id"
                    :loading="isLoadingSourceModels"
                    :disabled="isSubmitting || isLoadingSourceModels || sourceModels.length === 0"
                    :invalid="Boolean(rowValidations[index]?.isSameModelInSameRevision)"
                    placeholder="選擇來源 Model..."
                    class="flex-1 min-w-0"
                  >
                    <template #option="slotProps">
                      <div class="revision-option-row">
                        <span class="revision-option-text">Model {{ slotProps.option.modelAlias }}</span>
                        <span
                          class="revision-model-badge bg-gray-100 text-gray-600 border border-gray-200 dark:bg-zinc-800 dark:text-gray-300 dark:border-zinc-700"
                        >
                          Qty {{ slotProps.option.qty }} · {{ slotProps.option.selectionCount }} 筆
                        </span>
                      </div>
                    </template>
                  </Select>

                  <i class="pi pi-arrow-right text-gray-400 dark:text-gray-500 flex-shrink-0 text-sm"></i>

                  <!-- 目標 Model 選單 -->
                  <Select
                    v-model="row.targetModelId"
                    :options="targetModelOptions"
                    option-label="displayLabel"
                    option-value="id"
                    :loading="isLoadingTargetModels"
                    :disabled="isSubmitting || isLoadingTargetModels || targetModels.length === 0"
                    :invalid="Boolean(rowValidations[index]?.isSameModelInSameRevision || rowValidations[index]?.isDuplicateTarget)"
                    placeholder="選擇目標 Model..."
                    class="flex-1 min-w-0"
                  >
                    <template #option="slotProps">
                      <div class="revision-option-row">
                        <span class="revision-option-text">Model {{ slotProps.option.modelAlias }}</span>
                        <span
                          class="revision-model-badge bg-gray-100 text-gray-600 border border-gray-200 dark:bg-zinc-800 dark:text-gray-300 dark:border-zinc-700"
                        >
                          Qty {{ slotProps.option.qty }} · {{ slotProps.option.selectionCount }} 筆
                        </span>
                      </div>
                    </template>
                  </Select>

                  <!-- 移除該列按鈕 -->
                  <Button
                    icon="pi pi-trash"
                    text
                    rounded
                    severity="danger"
                    size="small"
                    class="flex-shrink-0"
                    :disabled="mappingRows.length <= 1 || isSubmitting"
                    title="移除此組對應"
                    @click="handleRemoveMappingRow(row.rowKey)"
                  />
                </div>

                <!-- 單列阻擋錯誤訊息 -->
                <p
                  v-if="rowValidations[index]?.errorMessage"
                  class="text-red-500 dark:text-red-400 text-xs flex items-center gap-1 pl-0.5"
                >
                  <i class="pi pi-times-circle"></i>
                  <span>{{ rowValidations[index]?.errorMessage }}</span>
                </p>

                <!-- 來源 Model 0 筆勾選之非阻擋黃色提醒 -->
                <p
                  v-else-if="rowValidations[index]?.isEmptySourceWarning"
                  class="text-amber-600 dark:text-amber-400 text-xs flex items-center gap-1 pl-0.5"
                >
                  <i class="pi pi-exclamation-circle"></i>
                  <span>來源 Model 無勾選紀錄，複製將清空對應目標 Model 的勾選。</span>
                </p>
              </div>
            </div>

            <!-- 進階附加選項：同時複製打件數量 (Qty) -->
            <div class="flex items-center gap-2 pt-2 border-t border-gray-200 dark:border-zinc-700">
              <Checkbox
                v-model="copyQty"
                binary
                input-id="copyMatrixCopyQty"
                :disabled="isSubmitting"
              />
              <label for="copyMatrixCopyQty" class="text-sm cursor-pointer select-none">
                同時複製打件數量 (Qty)
              </label>
            </div>
          </template>
        </div>
      </template>
    </div>

    <template #footer>
      <Button
        label="取消"
        icon="pi pi-times"
        text
        severity="secondary"
        :disabled="isSubmitting"
        @click="handleCancelDialog"
      />
      <Button
        label="開始複製"
        icon="pi pi-copy"
        severity="warning"
        :loading="isSubmitting"
        :disabled="isCopyDisabled"
        @click="executeCopyMatrix"
      />
    </template>
  </Dialog>
</template>

<script setup lang="ts">
/**
 * @file CopyMatrixDialog.vue
 * @description Matrix Selection 版本複製對話框，支援「整版複製」與「進階 Model-to-Model 複製」兩種模式
 */

import { ref, computed, watch } from 'vue'
import Button from 'primevue/button'
import Checkbox from 'primevue/checkbox'
import Dialog from 'primevue/dialog'
import Select from 'primevue/select'
import ToggleSwitch from 'primevue/toggleswitch'
import { useAppStore, useProjectStore, useLogStore, useTaskStore, useBOMTableStore } from '../../stores'
import {
  CopyMatrixSelections,
  CopyMatrixModelSelections,
  GetRevisionMatrixModels,
  type RevisionMatrixModelInfo
} from '../../services/api'
import type { RevisionOption } from '../../stores/project'

/**
 * 附加顯示標籤的 Model 下拉選項資料結構
 */
interface ModelSelectOption extends RevisionMatrixModelInfo {
  displayLabel: string
}

/**
 * 單列「來源 Model → 目標 Model」映射狀態
 */
interface ModelMappingRow {
  rowKey: number
  sourceModelId: number | null
  targetModelId: number | null
}

/**
 * 單列映射驗證結果
 */
interface RowValidationResult {
  isSameModelInSameRevision: boolean
  isDuplicateTarget: boolean
  isEmptySourceWarning: boolean
  errorMessage: string | null
}

/**
 * Component Props 定義
 */
const props = withDefaults(
  defineProps<{
    visible: boolean
    allRevisions?: RevisionOption[]
  }>(),
  {
    allRevisions: undefined
  }
)

/**
 * Component Emits 定義
 */
const emit = defineEmits<{
  (e: 'update:visible', val: boolean): void
}>()

const appStore = useAppStore()
const projectStore = useProjectStore()
const logStore = useLogStore()
const taskStore = useTaskStore()
const bomTableStore = useBOMTableStore()

/**
 * v-model:visible 雙向代理計算屬性
 */
const visibleModel = computed({
  get: () => props.visible,
  set: (val: boolean) => emit('update:visible', val)
})

let nextRowKey = 1
let sourceLoadSeq = 0
let targetLoadSeq = 0

/**
 * 建立一組空白的 Model 對應列
 * @returns {ModelMappingRow} 空白對應列物件
 */
function createEmptyMappingRow(): ModelMappingRow {
  return {
    rowKey: nextRowKey++,
    sourceModelId: null,
    targetModelId: null
  }
}

const copyMatrixSourceId = ref<number | null>(null)
const copyMatrixTargetId = ref<number | null>(null)
const isAdvancedMode = ref(false)
const copyQty = ref(false)
const isSubmitting = ref(false)

const sourceModels = ref<RevisionMatrixModelInfo[]>([])
const targetModels = ref<RevisionMatrixModelInfo[]>([])
const isLoadingSourceModels = ref(false)
const isLoadingTargetModels = ref(false)
const sourceModelsError = ref<string | null>(null)
const targetModelsError = ref<string | null>(null)
const mappingRows = ref<ModelMappingRow[]>([createEmptyMappingRow()])

/**
 * 有效的 Revision 清單（優先使用 props，無傳入則自 projectStore.allRevisionOptions 取得）
 */
const effectiveRevisions = computed<RevisionOption[]>(() => {
  if (props.allRevisions && props.allRevisions.length > 0) {
    return props.allRevisions
  }
  return projectStore.allRevisionOptions
})

/**
 * 當前選取的來源版本物件
 */
const selectedSource = computed(() =>
  effectiveRevisions.value.find(r => r.id === copyMatrixSourceId.value)
)

/**
 * 當前選取的目標版本物件
 */
const selectedTarget = computed(() =>
  effectiveRevisions.value.find(r => r.id === copyMatrixTargetId.value)
)

/**
 * 判斷來源版本與目標版本是否為同一版本
 */
const isSameRevision = computed(
  () => copyMatrixSourceId.value !== null && copyMatrixSourceId.value === copyMatrixTargetId.value
)

/**
 * 將原始 Model 摘要轉換為含顯示文字的下拉選單選項
 * @param {RevisionMatrixModelInfo[]} models - 原始未隱藏 Model 陣列
 * @returns {ModelSelectOption[]} 下拉選單選項陣列
 */
function toModelSelectOptions(models: RevisionMatrixModelInfo[]): ModelSelectOption[] {
  return models.map(m => ({
    ...m,
    displayLabel: `Model ${m.modelAlias} (Qty ${m.qty} · ${m.selectionCount} 筆)`
  }))
}

const sourceModelOptions = computed<ModelSelectOption[]>(() => toModelSelectOptions(sourceModels.value))
const targetModelOptions = computed<ModelSelectOption[]>(() => toModelSelectOptions(targetModels.value))

/**
 * 來源 Model 快速查找 Map（以 Model ID 為鍵）
 */
const sourceModelMap = computed(() => {
  const map = new Map<number, RevisionMatrixModelInfo>()
  for (const m of sourceModels.value) {
    map.set(m.id, m)
  }
  return map
})

/**
 * 統計各目標 Model ID 在映射清單中被指定的次數，用於偵測重複目標
 */
const targetModelUsageCount = computed(() => {
  const counts = new Map<number, number>()
  for (const row of mappingRows.value) {
    if (row.targetModelId !== null) {
      counts.set(row.targetModelId, (counts.get(row.targetModelId) || 0) + 1)
    }
  }
  return counts
})

/**
 * 逐列計算對應清單的驗證狀態（含阻擋錯誤與 0 筆勾選非阻擋提醒）
 */
const rowValidations = computed<RowValidationResult[]>(() => {
  return mappingRows.value.map((row) => {
    const isSameModelInSameRevision =
      isSameRevision.value && row.sourceModelId !== null && row.sourceModelId === row.targetModelId
    const isDuplicateTarget =
      row.targetModelId !== null && (targetModelUsageCount.value.get(row.targetModelId) || 0) > 1

    let errorMessage: string | null = null
    if (isSameModelInSameRevision) {
      errorMessage = '同版本內來源與目標 Model 不可相同'
    } else if (isDuplicateTarget) {
      errorMessage = '目標 Model 不可重複指定'
    }

    const srcModel = row.sourceModelId !== null ? sourceModelMap.value.get(row.sourceModelId) : undefined
    const isEmptySourceWarning = Boolean(
      !errorMessage && srcModel && srcModel.selectionCount === 0
    )

    return {
      isSameModelInSameRevision,
      isDuplicateTarget,
      isEmptySourceWarning,
      errorMessage
    }
  })
})

/**
 * 判斷「新增對應」按鈕是否禁用（目標機種上限為已啟用之目標 Model 數量）
 */
const isAddMappingDisabled = computed(() => {
  if (!copyMatrixSourceId.value || !copyMatrixTargetId.value) return true
  if (isLoadingSourceModels.value || isLoadingTargetModels.value) return true
  if (sourceModels.value.length === 0 || targetModels.value.length === 0) return true
  return mappingRows.value.length >= targetModels.value.length
})

/**
 * 判斷「開始複製」按鈕是否禁用
 */
const isCopyDisabled = computed(() => {
  if (isSubmitting.value) return true
  if (!copyMatrixSourceId.value || !copyMatrixTargetId.value) return true

  if (!isAdvancedMode.value) {
    if (copyMatrixSourceId.value === copyMatrixTargetId.value) return true
    if (!selectedSource.value || selectedSource.value.modelCount === 0) return true
    return false
  }

  if (isLoadingSourceModels.value || isLoadingTargetModels.value) return true
  if (sourceModelsError.value || targetModelsError.value) return true
  if (sourceModels.value.length === 0 || targetModels.value.length === 0) return true
  if (mappingRows.value.length === 0) return true

  for (let i = 0; i < mappingRows.value.length; i++) {
    const row = mappingRows.value[i]
    if (row.sourceModelId === null || row.targetModelId === null) return true
    if (rowValidations.value[i]?.errorMessage) return true
  }

  return false
})

/**
 * 載入指定側別（來源或目標）的未隱藏 Matrix Model 清單，具備防競態請求序號保護
 * @param {'source' | 'target'} side - 指定來源側或目標側
 * @param {number | null} revisionId - 要載入的 BOM Revision ID
 * @returns {Promise<void>}
 */
async function loadRevisionModels(side: 'source' | 'target', revisionId: number | null): Promise<void> {
  const modelsRef = side === 'source' ? sourceModels : targetModels
  const loadingRef = side === 'source' ? isLoadingSourceModels : isLoadingTargetModels
  const errorRef = side === 'source' ? sourceModelsError : targetModelsError

  const seq = side === 'source' ? ++sourceLoadSeq : ++targetLoadSeq
  const isLatest = () => seq === (side === 'source' ? sourceLoadSeq : targetLoadSeq)

  if (!revisionId) {
    modelsRef.value = []
    errorRef.value = null
    loadingRef.value = false
    return
  }

  loadingRef.value = true
  errorRef.value = null

  try {
    const allModels = await GetRevisionMatrixModels(revisionId)
    if (!isLatest()) return
    modelsRef.value = allModels.filter(m => !m.isHidden)
  } catch (error) {
    if (!isLatest()) return
    const msg = error instanceof Error ? error.message : String(error)
    modelsRef.value = []
    errorRef.value = msg
    logStore.addLogEntry('ERROR', `[CopyMatrixDialog] 載入${side === 'source' ? '來源' : '目標'}版本機種失敗：${msg}`)
  } finally {
    if (isLatest()) {
      loadingRef.value = false
    }
  }
}

/**
 * 新增一組 Model 對應列
 */
function handleAddMappingRow(): void {
  if (isAddMappingDisabled.value) return
  mappingRows.value.push(createEmptyMappingRow())
}

/**
 * 移除指定的 Model 對應列（最少保留 1 列）
 * @param {number} rowKey - 要移除的列唯一鍵
 */
function handleRemoveMappingRow(rowKey: number): void {
  if (mappingRows.value.length <= 1) return
  mappingRows.value = mappingRows.value.filter(r => r.rowKey !== rowKey)
}

/**
 * 註冊複製背景任務至 TaskStore 並監聽完成事件以自動重整 BOM 表格與專案列表
 * @param {string} taskId - 後端回傳的任務 ID
 * @param {string} taskName - 顯示於任務面板的任務名稱
 */
function registerCopyTaskWatcher(taskId: string, taskName: string): void {
  taskStore.updateTask(taskId, {
    id: taskId,
    name: taskName,
    type: 'CopyMatrix',
    status: 'queued',
    message: '複製任務已建立',
    progress: 0
  })
  logStore.addLogEntry('INFO', `Matrix 複製任務已提交 (taskID: ${taskId})`)

  const unwatch = watch(
    () => taskStore.getTask(taskId)?.status,
    async (status) => {
      if (status === 'completed') {
        logStore.addLogEntry('DEBUG', `[CopyMatrixDialog] 複製任務 (taskID: ${taskId}) 執行完畢，觸發重新載入 BOMTable 與專案列表`)
        bomTableStore.triggerReload()
        if (appStore.seriesInfo?.id) {
          try {
            await projectStore.loadProjects(appStore.seriesInfo.id)
          } catch (e) {
            console.error('Failed to reload projects after matrix copy:', e)
          }
        }
        unwatch()
      } else if (status === 'failed' || status === 'cancelled') {
        unwatch()
      }
    }
  )
}

/**
 * 監聽來源版本變更：清空已選來源 Model 並於進階模式下重載來源機種清單
 */
watch(copyMatrixSourceId, async (newId) => {
  for (const row of mappingRows.value) {
    row.sourceModelId = null
  }
  if (isAdvancedMode.value) {
    await loadRevisionModels('source', newId)
  } else {
    sourceLoadSeq++
    sourceModels.value = []
    sourceModelsError.value = null
    isLoadingSourceModels.value = false
  }
})

/**
 * 監聽目標版本變更：清空已選目標 Model 並於進階模式下重載目標機種清單
 */
watch(copyMatrixTargetId, async (newId) => {
  for (const row of mappingRows.value) {
    row.targetModelId = null
  }
  if (isAdvancedMode.value) {
    await loadRevisionModels('target', newId)
  } else {
    targetLoadSeq++
    targetModels.value = []
    targetModelsError.value = null
    isLoadingTargetModels.value = false
  }
})

/**
 * 監聽進階模式開關：開啟時確保至少有 1 列對應並載入已選版本之機種清單
 */
watch(isAdvancedMode, async (enabled) => {
  if (enabled) {
    if (mappingRows.value.length === 0) {
      mappingRows.value = [createEmptyMappingRow()]
    }
    await Promise.all([
      loadRevisionModels('source', copyMatrixSourceId.value),
      loadRevisionModels('target', copyMatrixTargetId.value)
    ])
  }
})

/**
 * 對話框開啟時完整重置所有選取與進階狀態，並確保最新專案資料已載入
 */
watch(
  () => props.visible,
  async (newVal) => {
    if (newVal) {
      sourceLoadSeq++
      targetLoadSeq++
      copyMatrixSourceId.value = null
      copyMatrixTargetId.value = null
      isAdvancedMode.value = false
      copyQty.value = false
      isSubmitting.value = false
      sourceModels.value = []
      targetModels.value = []
      isLoadingSourceModels.value = false
      isLoadingTargetModels.value = false
      sourceModelsError.value = null
      targetModelsError.value = null
      mappingRows.value = [createEmptyMappingRow()]

      if (appStore.isOpen && (!projectStore.projects || projectStore.projects.length === 0)) {
        if (appStore.seriesInfo?.id) {
          try {
            await projectStore.loadProjects(appStore.seriesInfo.id)
          } catch (err) {
            const msg = err instanceof Error ? err.message : String(err)
            logStore.addLogEntry('ERROR', `[CopyMatrixDialog] 載入專案資料失敗：${msg}`)
          }
        }
      }
    }
  }
)

/**
 * 執行 Matrix Selection 複製任務（依據當前模式分派整版複製或 Model-to-Model 複製）
 * @returns {Promise<void>}
 */
async function executeCopyMatrix(): Promise<void> {
  if (isCopyDisabled.value) return
  if (!copyMatrixSourceId.value || !copyMatrixTargetId.value) return

  isSubmitting.value = true
  try {
    if (!isAdvancedMode.value) {
      if (copyMatrixSourceId.value === copyMatrixTargetId.value) {
        logStore.addLogEntry('WARN', '來源版本與目標版本不可相同')
        return
      }
      if (!selectedSource.value || selectedSource.value.modelCount === 0) {
        logStore.addLogEntry('WARN', '來源版本無任何機種 (No Model)，無法進行複製')
        return
      }

      const taskId = await CopyMatrixSelections(copyMatrixSourceId.value, copyMatrixTargetId.value)
      if (taskId) {
        registerCopyTaskWatcher(taskId, 'Copy Matrix')
      }
      visibleModel.value = false
      return
    }

    const mappings = mappingRows.value
      .filter(r => r.sourceModelId !== null && r.targetModelId !== null)
      .map(r => ({
        sourceModelId: r.sourceModelId as number,
        targetModelId: r.targetModelId as number
      }))

    const taskId = await CopyMatrixModelSelections({
      sourceRevisionId: copyMatrixSourceId.value,
      targetRevisionId: copyMatrixTargetId.value,
      mappings,
      copyQty: copyQty.value
    })
    if (taskId) {
      registerCopyTaskWatcher(taskId, `Copy Matrix Models (${mappings.length} 組)`)
    }
    visibleModel.value = false
  } catch (error) {
    const msg = error instanceof Error ? error.message : String(error)
    logStore.addLogEntry('ERROR', `Matrix 複製失敗：${msg}`)
  } finally {
    isSubmitting.value = false
  }
}

/**
 * 處理使用者取消 Matrix 複製對話框操作
 */
function handleCancelDialog(): void {
  if (isSubmitting.value) return
  visibleModel.value = false
  logStore.addLogEntry('DEBUG', '使用者已取消 Matrix 複製對話框')
}
</script>

<style scoped>
.copy-matrix-dialog-content {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}
</style>

<style>
/* 確保 PrimeVue Select Teleported 下拉選單項目的 flex 容器填滿寬度並與左側文字保持良好間距 */
.p-select-list-container .p-select-option,
.p-select-overlay .p-select-option {
  display: flex !important;
  align-items: center !important;
  width: 100% !important;
  box-sizing: border-box !important;
}

.revision-option-row {
  display: flex !important;
  align-items: center !important;
  justify-content: space-between !important;
  width: 100% !important;
  min-width: 0 !important;
  gap: 1.5rem !important;
  padding: 0.125rem 0 !important;
  box-sizing: border-box !important;
}

.revision-option-text {
  font-weight: 500 !important;
  font-size: 0.875rem !important;
  white-space: nowrap !important;
  overflow: hidden !important;
  text-overflow: ellipsis !important;
  flex: 1 1 auto !important;
}

.revision-model-badge {
  font-size: 0.75rem !important;
  line-height: 1rem !important;
  font-weight: 500 !important;
  padding: 0.125rem 0.5rem !important;
  border-radius: 9999px !important;
  white-space: nowrap !important;
  flex-shrink: 0 !important;
  margin-left: auto !important;
}
</style>
