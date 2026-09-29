//go:build !production

package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// isDevMode 標記當前環境是否為開發模式（在非 production build tag 時為 true）
const isDevMode = true

/**
 * setupKeyBindings 在開發模式下註冊快捷鍵綁定
 * 僅在開發模式啟用 F12 開啟 WebView2 開發者工具 (DevTools)
 *
 * @returns {map[string]func(window application.Window)} 快捷鍵映射表
 */
func setupKeyBindings() map[string]func(window application.Window) {
	return map[string]func(window application.Window){
		"F12": func(window application.Window) {
			window.OpenDevTools()
		},
	}
}

/**
 * waitForDevServer 在開發模式下等待前端 Vite 伺服器就緒
 *
 * 緣由：Wails v3 內部 preRun() 僅固定重試 10 次 (每次間隔 500ms，共 5 秒)。
 * 在 Windows 系統冷啟動或依賴重新預編譯時，Vite 啟動耗時可能稍大於 5 秒，
 * 容易導致 Go 端因提早到達上限而報 FATAL 錯誤崩潰退出。
 *
 * 此函式在主程式調用 wailsApp.Run() 之前，主動輪詢 FRONTEND_DEVSERVER_URL（最多等待 30 秒），
 * 確認前端伺服器已就緒並可回應請求後才繼續交由 Wails 啟動視窗，從根源避免啟動競爭 (Race Condition)。
 */
func waitForDevServer() {
	frontendURL := os.Getenv("FRONTEND_DEVSERVER_URL")
	if frontendURL == "" {
		return
	}

	log.Printf("[BOMIX Dev] 正在等待前端開發伺服器 (%s) 就緒...", frontendURL)
	client := http.Client{
		Timeout: 1 * time.Second,
	}

	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			log.Printf("[BOMIX Dev] 等待前端開發伺服器逾時 (30s)，嘗試直接繼續...")
			return
		case <-ticker.C:
			resp, err := client.Get(frontendURL)
			if err == nil {
				_ = resp.Body.Close()
				log.Printf("[BOMIX Dev] 前端開發伺服器已就緒！繼續啟動主應用視窗...")
				return
			}
		}
	}
}
