//go:build production

package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

// isDevMode 標記當前環境是否為開發模式（在 production build tag 時為 false）
const isDevMode = false

/**
 * setupKeyBindings 在生產環境下取消開發者專用快捷鍵
 * 生產模式不提供 F12 開啟 DevTools 快速鍵綁定
 *
 * @returns {map[string]func(window application.Window)} 空的快捷鍵映射表 (nil)
 */
func setupKeyBindings() map[string]func(window application.Window) {
	return nil
}
