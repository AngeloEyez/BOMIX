package version

// 應用程式版本資訊全域變數
// 這些變數會在編譯時期透過 Go 編譯旗標 (-ldflags) 動態注入，作為整個專案的單一真實來源 (SSOT)。
var (
	// Version 應用程式語意化版本號 (例如 "1.0.0" 或預設 "dev")
	Version = "dev"
	// GitCommit 當前編譯之 Git Commit 短碼 (例如 "7a1b2c3" 或預設 "none")
	GitCommit = "none"
	// BuildTime 編譯完成之 UTC 時間戳記 (例如 "2026-09-21T03:55:00Z" 或預設 "unknown")
	BuildTime = "unknown"
)

// Info 代表應用程式完整的版本與建置元資料結構
type Info struct {
	// Version 應用程式版本號
	Version string `json:"version"`
	// GitCommit Git Commit 雜湊碼
	GitCommit string `json:"gitCommit"`
	// BuildTime 二進位檔建置時間戳
	BuildTime string `json:"buildTime"`
}

// Get 取得當前應用程式完整的版本與建置資訊結構體
//
// 回傳：
//   - Info: 包含 Version、GitCommit 與 BuildTime 的資訊
func Get() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildTime: BuildTime,
	}
}

// GetVersion 取得簡短的應用程式版本字串
//
// 回傳：
//   - string: 當前版本號字串 (如 "1.0.0" 或 "dev")
func GetVersion() string {
	return Version
}
