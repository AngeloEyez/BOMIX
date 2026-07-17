#!/bin/bash

IMAGE_NAME="bomix-dev-env"
CONTAINER_NAME="claude-sandbox"

# 1. 確保基礎映像檔存在 (只在第一次或手動刪除 image 後執行)
if [[ "$(docker images -q $IMAGE_NAME 2> /dev/null)" == "" ]]; then
  echo "🔨 首次執行，建立基礎 Docker 映像檔..."
  docker build -t $IMAGE_NAME .
fi

# 確保 Host 端的 .claude 資料夾存在
mkdir -p "$HOME/.claude"

# 2. 檢查容器是否已經存在
if [ "$(docker ps -aq -f name=^/${CONTAINER_NAME}$)" ]; then
    echo "🚀 發現既存的開發環境，正在喚醒並進入..."
    # 使用 start -ai 喚醒並進入既有容器，保留你在裡面做的所有升級與安裝
    docker start -ai $CONTAINER_NAME
else
    echo "🌱 建立並啟動全新的持久化開發環境..."
    # 拿掉 --rm，這樣退出後容器不會被刪除
    docker run -it \
      --name $CONTAINER_NAME \
      --network host \
      -v "$PWD:/project" \
      -v "$HOME/.claude:/root/.claude" \
      -w /project \
      $IMAGE_NAME bash
fi
