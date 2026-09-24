<template>
  <Dialog
    v-model:visible="visibleModel"
    modal
    header="Matrix Selection 版本複製"
    :style="{ width: '540px' }"
  >
    <div class="copy-matrix-dialog-content">
      <div class="copy-matrix-warning flex items-center gap-2 p-3 rounded" style="background: var(--p-yellow-50, #fffbeb); border: 1px solid var(--p-yellow-300, #fcd34d);">
        <i class="pi pi-exclamation-triangle text-amber-500 text-lg flex-shrink-0"></i>
        <span class="text-sm">此操作將<strong>覆蓋</strong>目標版本的現有 Matrix Model 與 Selection！</span>
      </div>

      <div v-if="effectiveRevisions.length === 0" class="text-center py-4 text-sm text-gray-500 dark:text-gray-400">
        目前系列內尚無任何 BOM 版本資料。
      </div>

      <template v-else>
        <div class="form-group">
          <label for="copyMatrixSource" class="font-medium text-sm flex items-center justify-between">
            <span>來源版本 (Source)</span>
            <span v-if="selectedSource" class="text-xs text-gray-500">
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
          >
            <template #option="slotProps">
              <div class="revision-option-row">
                <span class="revision-option-text">{{ slotProps.option.projectCode }} - {{ slotProps.option.phase }} {{ slotProps.option.version }}</span>
                <span
                  class="revision-model-badge"
                  :style="slotProps.option.modelCount > 0 
                    ? 'background: var(--p-primary-50, #eff6ff); color: var(--p-primary-700, #1d4ed8); border: 1px solid var(--p-primary-200, #bfdbfe);' 
                    : 'background: var(--p-surface-100, #f3f4f6); color: var(--p-surface-500, #6b7280); border: 1px solid var(--p-surface-200, #e5e7eb);'"
                >
                  {{ slotProps.option.modelCount > 0 ? `${slotProps.option.modelCount} Models` : 'No Model' }}
                </span>
              </div>
            </template>
          </Select>
          <p v-if="selectedSource && selectedSource.modelCount === 0" class="text-amber-600 dark:text-amber-400 text-xs flex items-center gap-1 mt-1">
            <i class="pi pi-info-circle"></i>
            <span>此來源版本無任何機種 (No Model)，無法作為複製來源。</span>
          </p>
        </div>

        <div class="form-group">
          <label for="copyMatrixTarget" class="font-medium text-sm flex items-center justify-between">
            <span>目標版本 (Target)</span>
            <span v-if="selectedTarget" class="text-xs text-gray-500">
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
          >
            <template #option="slotProps">
              <div class="revision-option-row">
                <span class="revision-option-text">{{ slotProps.option.projectCode }} - {{ slotProps.option.phase }} {{ slotProps.option.version }}</span>
                <span
                  class="revision-model-badge"
                  :style="slotProps.option.modelCount > 0 
                    ? 'background: var(--p-primary-50, #eff6ff); color: var(--p-primary-700, #1d4ed8); border: 1px solid var(--p-primary-200, #bfdbfe);' 
                    : 'background: var(--p-surface-100, #f3f4f6); color: var(--p-surface-500, #6b7280); border: 1px solid var(--p-surface-200, #e5e7eb);'"
                >
                  {{ slotProps.option.modelCount > 0 ? `${slotProps.option.modelCount} Models` : 'No Model' }}
                </span>
              </div>
            </template>
          </Select>
        </div>

        <p v-if="copyMatrixSourceId === copyMatrixTargetId && copyMatrixSourceId !== null" class="text-red-500 text-sm mt-1 flex items-center gap-1">
          <i class="pi pi-times-circle"></i>
          <span>來源版本與目標版本不可相同</span>
        </p>
      </template>
    </div>

    <template #footer>
      <Button
        label="取消"
        icon="pi pi-times"
        text
        severity="secondary"
        @click="visibleModel = false"
      />
      <Button
        label="開始複製"
        icon="pi pi-copy"
        severity="warning"
        @click="executeCopyMatrix"
        :disabled="isCopyDisabled"
      />
    </template>
  </Dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import Select from 'primevue/select'
import { useAppStore, useProjectStore, useLogStore, useTaskStore } from '../../stores'
import { CopyMatrixSelections } from '../../services/api'
import type { RevisionOption } from '../../stores/project'

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

/**
 * v-model:visible 雙向代理計算屬性
 */
const visibleModel = computed({
  get: () => props.visible,
  set: (val: boolean) => emit('update:visible', val)
})

const copyMatrixSourceId = ref<number | null>(null)
const copyMatrixTargetId = ref<number | null>(null)

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
 * 判斷「開始複製」按鈕是否禁用
 */
const isCopyDisabled = computed(() => {
  if (!copyMatrixSourceId.value || !copyMatrixTargetId.value) return true
  if (copyMatrixSourceId.value === copyMatrixTargetId.value) return true
  if (!selectedSource.value || selectedSource.value.modelCount === 0) return true
  return false
})

/**
 * 對話框開啟時重置選取狀態並確保最新專案資料已載入
 */
watch(
  () => props.visible,
  async (newVal) => {
    if (newVal) {
      copyMatrixSourceId.value = null
      copyMatrixTargetId.value = null

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
 * 執行 Matrix Selection 複製任務
 */
async function executeCopyMatrix(): Promise<void> {
  if (!copyMatrixSourceId.value || !copyMatrixTargetId.value) return
  if (copyMatrixSourceId.value === copyMatrixTargetId.value) {
    logStore.addLogEntry('WARN', '來源版本與目標版本不可相同')
    return
  }
  if (!selectedSource.value || selectedSource.value.modelCount === 0) {
    logStore.addLogEntry('WARN', '來源版本無任何機種 (No Model)，無法進行複製')
    return
  }

  try {
    const taskId = await CopyMatrixSelections(copyMatrixSourceId.value, copyMatrixTargetId.value)
    if (taskId) {
      taskStore.updateTask(taskId, {
        id: taskId,
        name: 'Copy Matrix',
        type: 'CopyMatrix',
        status: 'queued',
        message: '複製任務已建立',
        progress: 0,
      })
      logStore.addLogEntry('INFO', `Matrix 複製任務已提交 (taskID: ${taskId})`)
    }
    visibleModel.value = false
  } catch (error) {
    const msg = error instanceof Error ? error.message : String(error)
    logStore.addLogEntry('ERROR', `Matrix 複製失敗：${msg}`)
  }
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

