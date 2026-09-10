//go:build !production

package main

import (
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
