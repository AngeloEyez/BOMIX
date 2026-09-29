//go:build windows

package platform

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bomix-app/backend/logger"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

//go:embed bomx.ico
var bomxIconBytes []byte

const (
	progID            = "BOMIX.bomx"
	fileExt           = ".bomx"
	progDesc          = "BOMIX Series Database"
	shcneAssocChanged = 0x08000000
	shcnfIdList       = 0x0000
)

// GetIconBytes 提供外部或測試存取內嵌之 ICO 資料
func GetIconBytes() []byte {
	return bomxIconBytes
}

// checkAndSetupFileAssociation 實作 Windows 下的檔案關聯與圖標檢查與修復邏輯
func checkAndSetupFileAssociation(log *logger.Logger) {
	// 取得當前執行檔的絕對路徑
	exePath, err := os.Executable()
	if err != nil {
		if log != nil {
			log.Error("無法取得當前執行檔路徑", "error", err)
		}
		return
	}
	exePath = filepath.Clean(exePath)

	// 圖標存放目錄：%APPDATA%\BOMIX\
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
	}
	targetDir := filepath.Join(appData, "BOMIX")
	targetIconPath := filepath.Join(targetDir, "bomx.ico")

	// 1. 檢查是否需要更新 (比較圖標檔案、副檔名關聯、ProgID、DefaultIcon 與 open\command)
	needsUpdate, reason := checkIfUpdateNeeded(exePath, targetIconPath)
	if !needsUpdate {
		if log != nil {
			log.Debug("檔案關聯與圖標均為最新狀態，無需更新", "exePath", exePath, "iconPath", targetIconPath)
		}
		return
	}

	if log != nil {
		log.Info("偵測到 .bomx 檔案關聯需配置或路徑變更，開始在背景註冊", "reason", reason, "exePath", exePath)
	}

	// 2. 安全寫入圖標檔案 (防崩潰機制：寫入暫存檔後原子替換，即使立即關閉 app 也不會毀損)
	if err := safeWriteIconFile(targetDir, targetIconPath); err != nil {
		if log != nil {
			log.Error("安全寫入 bomx.ico 失敗", "error", err)
		}
		return
	}

	// 3. 寫入 Windows 登錄檔 (HKCU\Software\Classes，免管理員權限、無 UAC 提示)
	if err := registerFileAssociation(exePath, targetIconPath); err != nil {
		if log != nil {
			log.Error("寫入登錄檔檔案關聯失敗", "error", err)
		}
		return
	}

	// 4. 呼叫 Windows Shell API 刷新檔案總管圖標快取
	notifyShell()

	if log != nil {
		log.Info(".bomx 檔案圖標與開啟關聯已成功配置並即時生效", "iconPath", targetIconPath, "exePath", exePath)
	}
}

// checkIfUpdateNeeded 檢查目前檔案關聯、圖標檔案與執行路徑是否需更新
func checkIfUpdateNeeded(exePath, iconPath string) (bool, string) {
	// 檢查圖標檔案是否存在且檔案大小一致
	fi, err := os.Stat(iconPath)
	if err != nil || fi.Size() != int64(len(bomxIconBytes)) {
		return true, "圖標檔案不存在或尺寸不符"
	}

	// 檢查 HKCU\Software\Classes\.bomx
	extKey, err := registry.OpenKey(registry.CURRENT_USER, `Software\Classes\.bomx`, registry.QUERY_VALUE)
	if err != nil {
		return true, "副檔名登錄檔項目不存在"
	}
	defer extKey.Close()

	curProgID, _, err := extKey.GetStringValue("")
	if err != nil || curProgID != progID {
		return true, "ProgID 關聯不吻合"
	}

	// 檢查 HKCU\Software\Classes\BOMIX.bomx\DefaultIcon
	iconKey, err := registry.OpenKey(registry.CURRENT_USER, fmt.Sprintf(`Software\Classes\%s\DefaultIcon`, progID), registry.QUERY_VALUE)
	if err != nil {
		return true, "DefaultIcon 登錄檔項目不存在"
	}
	defer iconKey.Close()

	curIcon, _, err := iconKey.GetStringValue("")
	expectedIcon := fmt.Sprintf("%s,0", iconPath)
	if err != nil || !strings.EqualFold(curIcon, expectedIcon) {
		return true, "DefaultIcon 路徑不吻合"
	}

	// 檢查 HKCU\Software\Classes\BOMIX.bomx\shell\open\command
	cmdKey, err := registry.OpenKey(registry.CURRENT_USER, fmt.Sprintf(`Software\Classes\%s\shell\open\command`, progID), registry.QUERY_VALUE)
	if err != nil {
		return true, "shell open command 登錄檔項目不存在"
	}
	defer cmdKey.Close()

	curCmd, _, err := cmdKey.GetStringValue("")
	expectedCmd := fmt.Sprintf("\"%s\" \"%%1\"", exePath)
	if err != nil || !strings.EqualFold(curCmd, expectedCmd) {
		return true, "執行檔路徑有變動"
	}

	return false, ""
}

// safeWriteIconFile 安全寫入圖標檔案，具備原子性與寫入同步防護
func safeWriteIconFile(targetDir, targetIconPath string) error {
	// 確保目標資料夾存在
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("建立目錄失敗: %w", err)
	}

	// 檢查既有檔案是否已完全相同，若相同則無需重複寫入磁碟
	if fi, err := os.Stat(targetIconPath); err == nil && fi.Size() == int64(len(bomxIconBytes)) {
		existingData, err := os.ReadFile(targetIconPath)
		if err == nil && bytes.Equal(existingData, bomxIconBytes) {
			return nil
		}
	}

	// 寫入暫存檔以確保原子性，防止寫入中途 app 突然被關閉導致檔案損毀
	tmpPath := fmt.Sprintf("%s.tmp.%d", targetIconPath, os.Getpid())
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("建立暫存圖標檔案失敗: %w", err)
	}

	if _, err := f.Write(bomxIconBytes); err != nil {
		f.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("寫入暫存圖標檔案失敗: %w", err)
	}

	// 強制將快取同步刷入磁碟
	if err := f.Sync(); err != nil {
		f.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("同步圖標檔案至磁碟失敗: %w", err)
	}
	f.Close()

	// 刪除可能存在的舊檔案後重命名 (Windows 上替換檔案最佳實踐)
	_ = os.Remove(targetIconPath)
	if err := os.Rename(tmpPath, targetIconPath); err != nil {
		// 若 rename 失敗，作為 fallback 嘗試直接寫入
		if writeErr := os.WriteFile(targetIconPath, bomxIconBytes, 0644); writeErr != nil {
			_ = os.Remove(tmpPath)
			return fmt.Errorf("原子替換圖標檔案失敗: %w (fallback: %v)", err, writeErr)
		}
	}
	_ = os.Remove(tmpPath)
	return nil
}

// registerFileAssociation 寫入 Windows HKCU 登錄檔各項設定
func registerFileAssociation(exePath, iconPath string) error {
	// 1. 建立 / 開啟 HKCU\Software\Classes\.bomx
	extKey, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Classes\.bomx`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("建立 .bomx 註冊項目失敗: %w", err)
	}
	defer extKey.Close()

	if err := extKey.SetStringValue("", progID); err != nil {
		return fmt.Errorf("設置 .bomx ProgID 失敗: %w", err)
	}

	// 2. 建立 / 開啟 HKCU\Software\Classes\BOMIX.bomx
	progKey, _, err := registry.CreateKey(registry.CURRENT_USER, fmt.Sprintf(`Software\Classes\%s`, progID), registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("建立 ProgID 項目失敗: %w", err)
	}
	defer progKey.Close()

	if err := progKey.SetStringValue("", progDesc); err != nil {
		return fmt.Errorf("設置 ProgID 描述失敗: %w", err)
	}

	// 3. 建立 / 開啟 HKCU\Software\Classes\BOMIX.bomx\DefaultIcon
	iconKey, _, err := registry.CreateKey(registry.CURRENT_USER, fmt.Sprintf(`Software\Classes\%s\DefaultIcon`, progID), registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("建立 DefaultIcon 項目失敗: %w", err)
	}
	defer iconKey.Close()

	if err := iconKey.SetStringValue("", fmt.Sprintf("%s,0", iconPath)); err != nil {
		return fmt.Errorf("設置 DefaultIcon 路徑失敗: %w", err)
	}

	// 4. 建立 / 開啟 HKCU\Software\Classes\BOMIX.bomx\shell\open\command
	cmdKey, _, err := registry.CreateKey(registry.CURRENT_USER, fmt.Sprintf(`Software\Classes\%s\shell\open\command`, progID), registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("建立 shell open command 項目失敗: %w", err)
	}
	defer cmdKey.Close()

	cmdValue := fmt.Sprintf("\"%s\" \"%%1\"", exePath)
	if err := cmdKey.SetStringValue("", cmdValue); err != nil {
		return fmt.Errorf("設置 shell open command 數值失敗: %w", err)
	}

	return nil
}

// notifyShell 透過 Win32 API SHChangeNotify 通知 Windows 檔案總管刷新檔案圖標快取
func notifyShell() {
	shell32 := windows.NewLazySystemDLL("shell32.dll")
	procSHChangeNotify := shell32.NewProc("SHChangeNotify")
	procSHChangeNotify.Call(uintptr(shcneAssocChanged), uintptr(shcnfIdList), 0, 0)
}
