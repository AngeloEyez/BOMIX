<template>
  <Dialog
    v-model:visible="visibleModel"
    modal
    header="Matrix Selection 版本複製"
    :style="{ width: '520px' }"
  >
    <div class="copy-matrix-dialog-content">
      <div class="copy-matrix-warning mb-4 flex items-center gap-2 p-3 rounded" style="background: var(--p-yellow-50, #fffbeb); border: 1px solid var(--p-yellow-300, #fcd34d);">
        <i class="pi pi-exclamation-triangle text-amber-500"></i>
        <span>此操作將<strong>覆蓋</strong>目標版本的現有 Matrix Model 與 Selection！</span>
      </div>

      <div class="form-group">
        <label for="copyMatrixSource">來源版本 (Source)</label>
        <Select
          v-model="copyMatrixSourceId"
          :options="allRevisions"
          option-label="label"
          option-value="id"
          placeholder="選擇來源版本..."
          id="copyMatrixSource"
          class="w-full"
        />
      </div>

      <div class="form-group">
        <label for="copyMatrixTarget">目標版本 (Target)</label>
        <Select
          v-model="copyMatrixTargetId"
          :options="allRevisions"
          option-label="label"
          option-value="id"
          placeholder="選擇目標版本..."
          id="copyMatrixTarget"
          class="w-full"
        />
      </div>

      <p v-if="copyMatrixSourceId === copyMatrixTargetId && copyMatrixSourceId !== null" class="text-red-500 text-sm mt-2">
        來源版本與目標版本不可相同
      </p>
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
        :disabled="!copyMatrixSourceId || !copyMatrixTargetId || copyMatrixSourceId === copyMatrixTargetId"
      />
    </template>
  </Dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import Select from 'primevue/select'
import { useLogStore, useTaskStore } from '../../stores'
import { CopyMatrixSelections } from '../../services/api'
import type { RevisionOption } from './ExportDialog.vue'

/**
 * Component Props 定義
 */
const props = defineProps<{
  visible: boolean
  allRevisions: RevisionOption[]
}>()

/**
 * Component Emits 定義
 */
const emit = defineEmits<{
  (e: 'update:visible', val: boolean): void
}>()

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
 * 對話框開啟時清空狀態
 */
watch(
  () => props.visible,
  (newVal) => {
    if (newVal) {
      copyMatrixSourceId.value = null
      copyMatrixTargetId.value = null
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
  gap: 0.5rem;
}

.form-group label {
  font-weight: 500;
  font-size: 0.875rem;
}
</style>
