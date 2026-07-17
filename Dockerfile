# 使用 Node.js 官方映像檔作為基底 (對應前端 Vue 開發需求)
FROM node:22-bullseye

# 安裝編譯 Wails 與 SQLite 所需的 Linux 系統相依套件
RUN apt-get update && apt-get install -y \
    build-essential \
    libgtk-3-dev \
    libwebkit2gtk-4.0-dev \
    pkg-config \
    wget \
    && rm -rf /var/lib/apt/lists/*

# 下載並安裝 Go (使用 1.22 版本)
ENV GO_VERSION=1.22.5
RUN wget https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz && \
    rm go${GO_VERSION}.linux-amd64.tar.gz

# 設定 Go 環境變數
ENV PATH=$PATH:/usr/local/go/bin
ENV GOPATH=/root/go
ENV PATH=$PATH:/root/go/bin

# 安裝 Claude Code 與 Wails v3 CLI
RUN npm install -g @anthropic-ai/claude-code && \
    go install github.com/wailsapp/wails/v3/cmd/wails3@latest

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
