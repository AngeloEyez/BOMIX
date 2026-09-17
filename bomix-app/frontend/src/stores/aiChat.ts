import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { Events } from '@wailsio/runtime'
import { AIChatSend, AIChatStop, type AIChatMessage } from '../services/api'

/**
 * 工具呼叫資訊結構
 */
export interface ToolCallItem {
  id: string
  name: string
  arguments: string
  explanation?: string
  result?: string
  status: 'calling' | 'success' | 'error'
}

/**
 * 聊天訊息結構
 */
export interface ChatMessageItem {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  toolCalls?: ToolCallItem[]
  timestamp: number
  isStreaming?: boolean
}

export const useAIChatStore = defineStore('aiChat', () => {
  // 對話歷史訊息
  const messages = ref<ChatMessageItem[]>([])
  // 是否正在等待或生成中
  const isGenerating = ref<boolean>(false)
  // 當前狀態提示文字（如「AI 正在分析...」或「正在執行工具...」）
  const currentStatus = ref<string>('')
  // 當前進行中的工具呼叫
  const activeToolCalls = ref<ToolCallItem[]>([])

  // 是否已有對話訊息
  const hasMessages = computed(() => messages.value.length > 0)

  let unlisteners: Array<() => void> = []

  /**
   * 初始化 Wails 事件監聽
   */
  function initEventListeners(): void {
    if (unlisteners.length > 0) {
      return
    }

    try {
      // 監聽狀態變更
      const u1 = Events.On('ai:status', (e: any) => {
        const data = e?.data !== undefined ? e.data : e
        if (data?.message) {
          currentStatus.value = data.message
        }
      })
      unlisteners.push(u1)

      // 監聽工具呼叫
      const u2 = Events.On('ai:tool_call', (e: any) => {
        const data = e?.data !== undefined ? e.data : e
        if (!data) return
        activeToolCalls.value.push({
          id: data.id,
          name: data.name,
          arguments: data.arguments,
          explanation: data.explanation,
          status: 'calling',
        })
      })
      unlisteners.push(u2)

      // 監聽工具執行結果
      const u3 = Events.On('ai:tool_result', (e: any) => {
        const data = e?.data !== undefined ? e.data : e
        if (!data) return
        const target = activeToolCalls.value.find(t => t.id === data.id)
        if (target) {
          target.result = data.result
          target.status = data.success ? 'success' : 'error'
        }
      })
      unlisteners.push(u3)

      // 監聽文字片段
      const u4 = Events.On('ai:chunk', (e: any) => {
        const data = e?.data !== undefined ? e.data : e
        const chunk = data?.chunk || ''
        if (!chunk) return

        let lastMsg = messages.value[messages.value.length - 1]
        if (!lastMsg || lastMsg.role !== 'assistant' || !lastMsg.isStreaming) {
          lastMsg = {
            id: 'msg-' + Date.now() + '-' + Math.random().toString(36).substring(2, 6),
            role: 'assistant',
            content: '',
            toolCalls: [...activeToolCalls.value],
            timestamp: Date.now(),
            isStreaming: true,
          }
          messages.value.push(lastMsg)
          activeToolCalls.value = []
        }

        lastMsg.content += chunk
      })
      unlisteners.push(u4)

      // 監聽完成事件
      const u5 = Events.On('ai:done', (e: any) => {
        const data = e?.data !== undefined ? e.data : e
        const lastMsg = messages.value[messages.value.length - 1]
        if (lastMsg && lastMsg.role === 'assistant') {
          lastMsg.isStreaming = false
          if (data?.full_response) {
            lastMsg.content = data.full_response
          }
        }
        isGenerating.value = false
        currentStatus.value = ''
        activeToolCalls.value = []
      })
      unlisteners.push(u5)

      // 監聽錯誤事件
      const u6 = Events.On('ai:error', (e: any) => {
        const data = e?.data !== undefined ? e.data : e
        const errMsg = data?.error || '發生未知錯誤'
        isGenerating.value = false
        currentStatus.value = ''

        const lastMsg = messages.value[messages.value.length - 1]
        if (lastMsg && lastMsg.role === 'assistant' && lastMsg.isStreaming) {
          lastMsg.isStreaming = false
          lastMsg.content += `\n\n> ⚠️ **錯誤**：${errMsg}`
        } else {
          messages.value.push({
            id: 'msg-' + Date.now(),
            role: 'assistant',
            content: `> ⚠️ **錯誤**：${errMsg}`,
            toolCalls: [...activeToolCalls.value],
            timestamp: Date.now(),
            isStreaming: false,
          })
          activeToolCalls.value = []
        }
      })
      unlisteners.push(u6)
    } catch (err) {
      console.error('初始化 AI 事件監聽失敗:', err)
    }
  }

  /**
   * 發送使用者訊息
   * @param text 使用者輸入字串
   */
  async function sendMessage(text: string): Promise<void> {
    const trimmed = text.trim()
    if (!trimmed || isGenerating.value) return

    initEventListeners()

    // 1. 新增使用者訊息
    messages.value.push({
      id: 'msg-' + Date.now() + '-user',
      role: 'user',
      content: trimmed,
      timestamp: Date.now(),
    })

    isGenerating.value = true
    currentStatus.value = 'AI 思考中...'
    activeToolCalls.value = []

    // 2. 轉換為後端格式
    const historyPayload: AIChatMessage[] = messages.value.map(m => ({
      role: m.role,
      content: m.content,
    }))

    try {
      await AIChatSend(historyPayload)
    } catch (err: any) {
      isGenerating.value = false
      currentStatus.value = ''
      messages.value.push({
        id: 'msg-' + Date.now() + '-err',
        role: 'assistant',
        content: `> ⚠️ **發送失敗**：${err?.message || err}`,
        timestamp: Date.now(),
        isStreaming: false,
      })
    }
  }

  /**
   * 中斷目前的 AI 生成
   */
  async function stopGeneration(): Promise<void> {
    try {
      await AIChatStop()
    } catch (err) {
      console.error('中斷 AI 生成失敗:', err)
    } finally {
      const lastMsg = messages.value[messages.value.length - 1]
      if (lastMsg && lastMsg.role === 'assistant') {
        lastMsg.isStreaming = false
      }
      isGenerating.value = false
      currentStatus.value = ''
    }
  }

  /**
   * 清空所有對話紀錄
   */
  function clearMessages(): void {
    messages.value = []
    activeToolCalls.value = []
    isGenerating.value = false
    currentStatus.value = ''
  }

  return {
    messages,
    isGenerating,
    currentStatus,
    activeToolCalls,
    hasMessages,
    initEventListeners,
    sendMessage,
    stopGeneration,
    clearMessages,
  }
})
