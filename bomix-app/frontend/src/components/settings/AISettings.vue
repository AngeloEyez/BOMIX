<template>
  <!-- ==================== 5. AI Assistant ==================== -->
  <section id="category-ai" class="category-section">
    <div class="category-header">
      <div class="category-title">
        <i class="pi pi-sparkles category-icon text-primary-500"></i>
        <h2>AI Assistant</h2>
      </div>
      <p class="category-desc">配置大型語言模型（LLM）端點、金鑰憑證、推論參數與對話互動行為。</p>
    </div>

    <!-- Enable AI Assistant -->
    <div class="vscode-setting-item">
      <div class="setting-title-line switch-title-line">
        <ToggleSwitch
          input-id="ai-enabled"
          v-model="ai.enabled"
        />
        <label for="ai-enabled" class="setting-name switch-label">
          Enable Assistant
        </label>
      </div>
      <p class="setting-desc">
        啟用 AI 智慧對話助手，支援自然語言問答、BOM 資料比對分析與多功能對話輔助。
      </p>
    </div>

    <!-- AI 細部設定群組 (當 Enable Assistant 關閉時整體淡化且禁用) -->
    <div :class="['ai-sub-settings-container', { 'is-disabled': !ai.enabled }]">
      <!-- Response Language -->
      <div class="vscode-setting-item">
        <div class="setting-title-line">
          <span class="setting-name">Response Language</span>
        </div>
        <p class="setting-desc">
          設定 AI 助手在回答問題、產生分析報告時優先使用的自然語言。
        </p>
        <div class="setting-control">
          <Select
            id="ai-language"
            v-model="ai.language"
            :options="aiLanguageOptions"
            option-label="label"
            option-value="value"
            :disabled="!ai.enabled"
            size="small"
            class="compact-select-wide"
          />
        </div>
      </div>

      <!-- API Base URL -->
      <div class="vscode-setting-item">
        <div class="setting-title-line">
          <span class="setting-name">API Base URL</span>
        </div>
        <p class="setting-desc">
          相容於 OpenAI 規格的 API 端點基礎網址（如 <code>https://api.openai.com/v1</code> 或本地自架/反向代理伺服器）。
        </p>
        <div class="setting-control">
          <InputText
            id="ai-base-url"
            v-model="ai.baseUrl"
            placeholder="https://api.openai.com/v1"
            :disabled="!ai.enabled"
            size="small"
            class="compact-input-text-wide"
            @blur="onBaseUrlBlur"
          />
        </div>
      </div>

      <!-- API Key -->
      <div class="vscode-setting-item">
        <div class="setting-title-line">
          <span class="setting-name">API Key</span>
        </div>
        <p class="setting-desc">
          用於向 API 伺服器進行身分驗證的金鑰憑證，安全儲存於本機工作站設定檔中。
        </p>
        <div class="setting-control">
          <Password
            id="ai-api-key"
            v-model="ai.apiKey"
            placeholder="sk-..."
            :feedback="false"
            :disabled="!ai.enabled"
            toggle-mask
            size="small"
            class="compact-password"
            input-class="compact-password-input"
          />
        </div>
      </div>

      <!-- Connection Test Action -->
      <div class="vscode-setting-item test-connection-item">
        <div class="setting-title-line">
          <span class="setting-name">Connection Test</span>
        </div>
        <p class="setting-desc">
          即時測試網路連線狀態，並驗證目前的端點網址與 API 金鑰是否有效。
        </p>
        <div class="test-connection-action-row">
          <Button
            label="測試連線"
            icon="pi pi-check-circle"
            size="small"
            severity="secondary"
            outlined
            class="test-btn-neutral"
            :loading="isTestingAI"
            :disabled="!ai.enabled || isTestingAI"
            @click="testConnection"
          />
          <span v-if="testAIResult?.success" class="test-success-msg">
            <i class="pi pi-check-circle"></i>
            {{ testAIResult.message }}
          </span>
          <span v-else-if="testAIResult && !testAIResult.success" class="test-fail-msg">
            <i class="pi pi-exclamation-circle"></i>
            {{ testAIResult.message }}
          </span>
        </div>
      </div>

      <!-- Model Selection -->
      <div class="vscode-setting-item">
        <div class="setting-title-line flex items-center justify-between">
          <div>
            <span class="setting-name">Model</span>
          </div>
          <button
            type="button"
            class="refresh-model-icon-btn"
            title="重新整理伺服器可用模型清單"
            :disabled="!ai.enabled || aiChatStore.isLoadingModels || !ai.baseUrl"
            @click="onRefreshModels"
          >
            <i :class="['pi', aiChatStore.isLoadingModels ? 'pi-spin pi-spinner' : 'pi-refresh']"></i>
          </button>
        </div>
        <p class="setting-desc">
          從伺服器拉取的模型清單中挑選，或直接鍵入任意自訂模型代碼（如 gpt-4o, deepseek-chat）。
        </p>
        <div class="setting-control">
          <Select
            id="ai-model"
            v-model="ai.model"
            :options="modelOptions"
            option-label="label"
            option-value="value"
            editable
            :disabled="!ai.enabled"
            placeholder="選擇或直接輸入模型代碼 (如 gpt-4o, deepseek-chat)..."
            size="small"
            class="compact-select-wide"
            @change="onModelChange"
            @blur="onModelChange"
          />
        </div>
      </div>

      <!-- Timeout -->
      <div class="vscode-setting-item">
        <div class="setting-title-line">
          <span class="setting-name">Timeout</span>
        </div>
        <p class="setting-desc">
          呼叫 AI 模型生成回應的逾時上限（秒），超過設定時間未回應將中止請求。
        </p>
        <div class="setting-control">
          <InputNumber
            id="ai-timeout"
            v-model="ai.timeout"
            :show-buttons="true"
            :min="5"
            :max="300"
            :step="5"
            :disabled="!ai.enabled"
            size="small"
            class="compact-input-number"
          />
        </div>
      </div>

      <!-- Temperature -->
      <div class="vscode-setting-item">
        <div class="setting-title-line">
          <span class="setting-name">Temperature</span>
        </div>
        <p class="setting-desc">
          模型採樣隨機性係數（0.0 - 2.0）。數值越低回答越聚焦嚴謹；數值越高回答越具多樣性。
        </p>
        <div class="setting-control">
          <InputNumber
            id="ai-temperature"
            v-model="ai.temperature"
            :show-buttons="true"
            :min="0"
            :max="2"
            :step="0.1"
            :max-fraction-digits="2"
            :disabled="!ai.enabled"
            size="small"
            class="compact-input-number"
          />
        </div>
      </div>

      <!-- Max Tokens -->
      <div class="vscode-setting-item">
        <div class="setting-title-line">
          <span class="setting-name">Max Output Tokens</span>
        </div>
        <p class="setting-desc">
          單次呼叫模型生成回覆時允許輸出的最大 Token 長度上限。
        </p>
        <div class="setting-control">
          <InputNumber
            id="ai-max-tokens"
            v-model="ai.maxTokens"
            :show-buttons="true"
            :min="256"
            :max="32768"
            :step="512"
            :disabled="!ai.enabled"
            size="small"
            class="compact-input-number"
          />
        </div>
      </div>

      <!-- Max Iterations (推論輪次上限) -->
      <div class="vscode-setting-item">
        <div class="setting-title-line">
          <span class="setting-name">Max Reasoning Iterations</span>
        </div>
        <p class="setting-desc">
          AI 助手推論思考與工具調用的最大迭代輪次上限（預設 8 輪，範圍 1 - 30 輪）。
        </p>
        <div class="setting-control">
          <InputNumber
            id="ai-max-iterations"
            v-model="ai.maxIterations"
            :show-buttons="true"
            :min="1"
            :max="30"
            :step="1"
            :disabled="!ai.enabled"
            size="small"
            class="compact-input-number"
          />
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
/**
 * @file AISettings.vue
 * @description 大型語言模型 (LLM) 端點、連線測試、金鑰與推論參數設定子元件
 */
import { ref, computed } from 'vue'
import ToggleSwitch from 'primevue/toggleswitch'
import Select from 'primevue/select'
import InputNumber from 'primevue/inputnumber'
import InputText from 'primevue/inputtext'
import Password from 'primevue/password'
import Button from 'primevue/button'
import { AIChatTestConnection, type AISettings } from '../../services/api'
import { useAIChatStore } from '../../stores/aiChat'

/**
 * AI 設定物件雙向綁定
 */
const ai = defineModel<AISettings>({ required: true })

/**
 * 宣告自訂事件：通知父層立即儲存當前設定（用於連線測試前先同步設定至後端）
 */
const emit = defineEmits<{
  (e: 'save-settings'): Promise<void> | void
}>()

const aiChatStore = useAIChatStore()

// 可用模型選項 (與 aiChatStore.availableModels 雙向同步)
const modelOptions = computed(() => {
  const list = aiChatStore.availableModels.map(m => ({ label: m, value: m }))
  const current = ai.value?.model?.trim()
  if (current && !list.some(item => item.value === current)) {
    return [{ label: current, value: current }, ...list]
  }
  return list
})

/**
 * Base URL 失焦時自動拉取伺服器可用模型
 */
async function onBaseUrlBlur(): Promise<void> {
  const url = ai.value?.baseUrl?.trim()
  if (url) {
    await aiChatStore.fetchAvailableModels(url, ai.value?.apiKey)
  }
}

/**
 * 點擊手動重新整理模型
 */
async function onRefreshModels(): Promise<void> {
  const url = ai.value?.baseUrl?.trim()
  if (url) {
    await aiChatStore.fetchAvailableModels(url, ai.value?.apiKey)
  }
}

/**
 * 模型變更時同步目前選擇給 aiChatStore
 */
function onModelChange(): void {
  const model = ai.value?.model?.trim()
  if (model) {
    aiChatStore.currentModel = model
  }
}

// AI 回應語言選項
const aiLanguageOptions = [
  { label: '繁體中文 (Traditional Chinese)', value: 'zh-TW' },
  { label: '簡體中文 (Simplified Chinese)', value: 'zh-CN' },
  { label: 'English', value: 'en' },
]

const isTestingAI = ref(false)
const testAIResult = ref<{ success: boolean; message: string } | null>(null)

/**
 * 測試與 AI 伺服器的連線
 */
async function testConnection(): Promise<void> {
  isTestingAI.value = true
  testAIResult.value = null
  try {
    // 透過父層通知或直接更新設定至後端，確保後端取得最新端點與金鑰
    await emit('save-settings')
    await AIChatTestConnection()
    const models = await aiChatStore.fetchAvailableModels(ai.value.baseUrl, ai.value.apiKey)
    if (models.length > 0) {
      testAIResult.value = { success: true, message: `連線成功！已同步 ${models.length} 個可用伺服器模型。` }
    } else {
      testAIResult.value = { success: true, message: '連線成功！API 端點與金鑰有效。' }
    }
  } catch (err: any) {
    testAIResult.value = { success: false, message: err?.message || '連線測試失敗' }
  } finally {
    isTestingAI.value = false
  }
}
</script>

<style scoped>
@import './styles/settings.css';

.compact-select-wide {
  width: 280px !important;
  height: 28px !important;
  font-size: 0.8rem;
}

:deep(.compact-select-wide .p-select-label) {
  padding: 0.2rem 0.55rem !important;
  font-size: 0.8rem !important;
}

.compact-input-text-wide {
  width: 380px !important;
  max-width: 100% !important;
  height: 28px !important;
  padding: 0.15rem 0.55rem !important;
  font-size: 0.8rem !important;
}

.compact-password {
  width: 380px !important;
  max-width: 100% !important;
  height: 28px !important;
}

:deep(.compact-password .p-password-input) {
  width: 100% !important;
  height: 28px !important;
  padding: 0.15rem 2rem 0.15rem 0.55rem !important;
  font-size: 0.8rem !important;
}

.refresh-model-icon-btn {
  font-size: 0.85rem;
  color: var(--text-color-secondary);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  cursor: pointer;
  background: transparent;
  border: none;
  border-radius: 4px;
  transition: color 0.15s ease, background-color 0.15s ease;
}

.refresh-model-icon-btn:hover:not(:disabled) {
  color: var(--text-color);
  background: var(--surface-hover);
}

.refresh-model-icon-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

.test-connection-action-row {
  display: flex;
  align-items: center;
  gap: 1.5rem;
  margin-top: 0.35rem;
}

.test-btn-neutral {
  height: 28px !important;
  background: var(--surface-card) !important;
  border: 1px solid var(--surface-border) !important;
  color: var(--text-color) !important;
  font-size: 0.8rem !important;
  padding: 0 0.75rem !important;
  border-radius: 4px !important;
  transition: all 0.15s ease !important;
}

.test-btn-neutral:hover:not(:disabled) {
  background: var(--surface-hover) !important;
  border-color: var(--surface-border-hover, #94a3b8) !important;
  color: var(--text-color) !important;
}

:deep(.test-btn-neutral .p-button-icon) {
  color: var(--text-color-secondary) !important;
  font-size: 0.82rem !important;
}

:deep(.test-btn-neutral:hover:not(:disabled) .p-button-icon) {
  color: var(--text-color) !important;
}

.test-success-msg {
  color: #10b981;
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.8rem;
  font-weight: 500;
}

.test-fail-msg {
  color: #f43f5e;
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.8rem;
  font-weight: 500;
}

/* AI Assistant 子設定項淡化與禁用樣式 */
.ai-sub-settings-container {
  display: flex;
  flex-direction: column;
  transition: opacity 0.2s ease, filter 0.2s ease;
}

.ai-sub-settings-container.is-disabled {
  opacity: 0.45;
  pointer-events: none;
  filter: grayscale(0.5);
  user-select: none;
}
</style>
