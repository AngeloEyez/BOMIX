<template>
  <template v-if="!query || !text">
    <span>{{ text ?? '' }}</span>
  </template>
  <template v-else>
    <template v-for="(part, idx) in parts" :key="idx">
      <mark v-if="part.isMatch" class="highlight-text">{{ part.text }}</mark>
      <span v-else>{{ part.text }}</span>
    </template>
  </template>
</template>

<script setup lang="ts">
/**
 * @file BOMHighlightText.vue
 * @description 關鍵字搜尋高亮文字渲染元件
 * 
 * 依據傳入的搜尋字串，自動將目標文字分割並標註高亮顏色 (<mark>)，
 * 支援無關鍵字時之快速純文字渲染退化機制。
 */

import { computed } from 'vue'
import { getHighlightedParts } from '../utils/textHighlight'

interface Props {
  /** 欲顯示之完整文字內容 */
  text?: string | number | null
  /** 搜尋關鍵字 */
  query?: string
}

const props = withDefaults(defineProps<Props>(), {
  text: '',
  query: ''
})

/** 計算高亮切割後之片段清單 */
const parts = computed(() => {
  const str = props.text !== null && props.text !== undefined ? String(props.text) : ''
  if (!props.query) {
    return [{ text: str, isMatch: false }]
  }
  return getHighlightedParts(str, props.query)
})
</script>

<style scoped>
.highlight-text {
  background-color: #fef08a;
  color: #854d0e;
  font-weight: 700;
  padding: 0 1px;
  border-radius: 2px;
}
</style>
