<template>
  <button
    type="button"
    class="prompt-chip group flex flex-col items-start p-3.5 rounded-xl border border-surface-200 dark:border-surface-700 bg-surface-0 dark:bg-surface-800/80 hover:border-primary-400 dark:hover:border-primary-500 hover:shadow-md transition-all duration-200 text-left w-full cursor-pointer"
    @click="$emit('click')"
  >
    <div class="flex items-center gap-2 mb-1.5 text-primary-600 dark:text-primary-400 font-medium text-sm">
      <i :class="icon" class="text-base group-hover:scale-110 transition-transform"></i>
      <span>{{ title }}</span>
    </div>
    <div class="text-xs text-surface-500 dark:text-surface-400 line-clamp-2 leading-relaxed">
      {{ description }}
    </div>
  </button>
</template>

<script setup lang="ts">
/**
 * 快捷提問晶片卡片元件
 */
defineProps<{
  icon: string
  title: string
  description: string
}>()

defineEmits<{
  (e: 'click'): void
}>()
</script>

<style scoped>
.prompt-chip {
  outline: none;
}
</style>
