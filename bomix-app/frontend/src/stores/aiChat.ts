import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { Events } from '@wailsio/runtime'
import {
  AIChatSend,
  AIChatStop,
  AIChatGetAvailableModels,
  AIChatFetchModelsWithConfig,
  GetSettings,
  UpdateSettings,
  type AIChatMessage,
} from '../services/api'

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

  // 當前選擇模型 (例如 gpt-4o-mini, o3-mini 等)
  const currentModel = ref<string>('')
  // 伺服器提供的可用模型 ID 清單（遵照使用者指示：無模型時為空，不自動補假清單）
  const availableModels = ref<string[]>([])
  // 是否正在取得可用模型
  const isLoadingModels = ref<boolean>(false)
  // 本輪生成的 Token 數量
  const currentRoundTokens = ref<number>(0)
  // 全局累計消耗的 Token 數量
  const totalTokens = ref<number>(0)
  // 最近一次發送內容字元長度 (供備援預估 Token 使用)
  let lastPromptLength = 0

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

        // 統計本輪與累計 Token 數量
        if (data?.usage?.total_tokens && data.usage.total_tokens > 0) {
          currentRoundTokens.value = data.usage.total_tokens
          totalTokens.value += data.usage.total_tokens
        } else {
          // 備援方案：依輸入與回傳字元長度加權估算 Token (約 1.3-1.5 chars/token)
          const replyLen = data?.full_response?.length || (lastMsg?.content?.length || 0)
          const est = Math.max(1, Math.round((lastPromptLength + replyLen) * 0.75))
          currentRoundTokens.value = est
          totalTokens.value += est
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
   * 取得伺服器提供的可用模型清單
   * @param customBaseUrl 可選自訂 Base URL (例如設定頁輸入)
   * @param customApiKey 可選自訂 API Key
   */
  async function fetchAvailableModels(customBaseUrl?: string, customApiKey?: string): Promise<string[]> {
    isLoadingModels.value = true
    try {
      let list: string[] = []
      if (customBaseUrl !== undefined && customBaseUrl.trim() !== '') {
        list = await AIChatFetchModelsWithConfig(customBaseUrl, customApiKey || '')
      } else {
        list = await AIChatGetAvailableModels()
      }

      availableModels.value = Array.isArray(list) ? list : []

      // 檢查目前選取的 model
      if (availableModels.value.length > 0) {
        if (!currentModel.value) {
          const s = await GetSettings()
          if (s?.ai?.model && availableModels.value.includes(s.ai.model)) {
            currentModel.value = s.ai.model
          } else {
            currentModel.value = availableModels.value[0]
          }
        }
      }
      return availableModels.value
    } catch (err) {
      console.warn('拉取可用模型清單失敗或無可用模型:', err)
      // 使用者明確交代：不要自動推薦清單，若真的沒有任何模型，選單可以空白
      availableModels.value = []
      return []
    } finally {
      isLoadingModels.value = false
    }
  }

  /**
   * 取得並載入當前設定的模型名稱 (僅讀取持久化中記錄的選中 model)
   */
  async function fetchCurrentModel(): Promise<string> {
    try {
      const s = await GetSettings()
      if (s?.ai?.model) {
        currentModel.value = s.ai.model
      }
    } catch (err) {
      console.error('載入設定模型失敗:', err)
    }
    return currentModel.value
  }

  /**
   * 即時切換 AI 模型並同步儲存設定
   * @param newModel 模型名稱 (例如 'Gemini 3.8 Flash High', 'gpt-4o', 'deepseek-chat')
   */
  async function switchModel(newModel: string): Promise<void> {
    const trimmed = newModel.trim()
    if (!trimmed) return
    currentModel.value = trimmed
    try {
      const s = await GetSettings()
      if (s) {
        if (!s.ai) {
          s.ai = {
            enabled: true,
            baseUrl: '',
            apiKey: '',
            model: trimmed,
            temperature: 0.2,
            maxTokens: 4096,
            timeout: 60,
            language: 'zh-TW',
          }
        } else {
          s.ai.model = trimmed
        }
        await UpdateSettings(s)
      }
    } catch (err) {
      console.error('儲存切換模型失敗:', err)
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

    // 記錄字元長度
    lastPromptLength = trimmed.length

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
    currentRoundTokens.value = 0
  }

  return {
    messages,
    isGenerating,
    currentStatus,
    activeToolCalls,
    hasMessages,
    currentModel,
    availableModels,
    isLoadingModels,
    currentRoundTokens,
    totalTokens,
    initEventListeners,
    fetchCurrentModel,
    fetchAvailableModels,
    switchModel,
    sendMessage,
    stopGeneration,
    clearMessages,
  }
})
