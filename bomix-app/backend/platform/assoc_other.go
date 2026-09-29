//go:build !windows

package platform

import (
	"bomix-app/backend/logger"
)

// checkAndSetupFileAssociation 非 Windows 系統之 Stub 實作
func checkAndSetupFileAssociation(log *logger.Logger) {
	// 非 Windows 平臺不進行登錄檔操作
}

// GetIconBytes 非 Windows 系統之 Stub 實作
func GetIconBytes() []byte {
	return nil
}
