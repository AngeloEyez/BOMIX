package version

import (
	"testing"
)

// TestGetVersion 測試取得版本號字串函式
func TestGetVersion(t *testing.T) {
	orig := Version
	defer func() { Version = orig }()

	Version = "2.1.0"
	if got := GetVersion(); got != "2.1.0" {
		t.Errorf("GetVersion() = %v, 預期 %v", got, "2.1.0")
	}
}

// TestGet 測試取得完整版本資訊結構體函式
func TestGet(t *testing.T) {
	origVer := Version
	origCommit := GitCommit
	origTime := BuildTime
	defer func() {
		Version = origVer
		GitCommit = origCommit
		BuildTime = origTime
	}()

	Version = "1.5.0"
	GitCommit = "abc1234"
	BuildTime = "2026-09-21T00:00:00Z"

	info := Get()
	if info.Version != "1.5.0" {
		t.Errorf("info.Version = %v, 預期 %v", info.Version, "1.5.0")
	}
	if info.GitCommit != "abc1234" {
		t.Errorf("info.GitCommit = %v, 預期 %v", info.GitCommit, "abc1234")
	}
	if info.BuildTime != "2026-09-21T00:00:00Z" {
		t.Errorf("info.BuildTime = %v, 預期 %v", info.BuildTime, "2026-09-21T00:00:00Z")
	}
}

// TestLdflagsDefault 測試當前注入值之有效性
func TestLdflagsDefault(t *testing.T) {
	t.Logf("當前注入狀態: Version=%s, GitCommit=%s, BuildTime=%s", Version, GitCommit, BuildTime)
	if Version == "" {
		t.Error("Version 不可為空字串")
	}
}

