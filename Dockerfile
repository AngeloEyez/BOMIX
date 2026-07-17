# 使用 Node.js 官方映像檔作為基底 (對應前端 Vue 開發需求)
FROM node:22-bullseye

# 安裝 Claude Code 與 Wails v3 CLI
RUN npm install -g @anthropic-ai/claude-code

# === 寫入 CCR 環境變數 ===
# 讓容器內的 Claude Code 直接預設連線到 Host 的 3456 端口
ENV ANTHROPIC_AUTH_TOKEN="ccr-local-key"
ENV ANTHROPIC_BASE_URL="http://127.0.0.1:3456"
ENV NO_PROXY="127.0.0.1"
ENV DISABLE_TELEMETRY="true"
ENV DISABLE_COST_WARNINGS="true"
ENV API_TIMEOUT_MS="600000"
ENV CLAUDE_CODE_ATTRIBUTION_HEADER="0"

# 設定容器啟動時的預設工作目錄
WORKDIR /project
