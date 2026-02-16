import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    // 排除測試資料夾與 AI 工作區
    exclude: [
      '**/node_modules/**',
      '**/dist/**',
      'tests/**',
      'agent-workspace/**',
    ],
    // 支援環境
    environment: 'node',
  },
})
