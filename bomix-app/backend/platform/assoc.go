package platform

import (
	"bomix-app/backend/logger"
)

// CheckAndSetupFileAssociation 於應用程式啟動時於背景執行檢查與配置
// 檢測 .bomx 檔案圖標與開啟關聯是否正確，若未配置或路徑變更則安全寫入
func CheckAndSetupFileAssociation(log *logger.Logger) {
	checkAndSetupFileAssociation(log)
}
