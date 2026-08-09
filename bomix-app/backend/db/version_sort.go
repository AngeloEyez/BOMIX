package db

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"gorm.io/gorm"
)

// ErrNonNumericVersion 表示版本號包含非數字字元，無法進行數值自動排序
var ErrNonNumericVersion = errors.New("版本號包含非數字字元，無法進行數值自動排序")

// HasNonNumericText 檢查版本字串是否包含數字 (0-9) 與小數點 (.) 以外的字元。
//
// 例如：
//   - "0.1"  → false（純數字版本，可排序）
//   - "0.10" → false（純數字版本，可排序）
//   - "v1.0" → true（含文字，無法自動排序）
//   - "RevA" → true（含文字，無法自動排序）
//
// 參數：
//   - v: 要檢查的版本字串
//
// 回傳：
//   - bool: true 表示含有非數字字元
func HasNonNumericText(v string) bool {
	for _, r := range v {
		if !unicode.IsDigit(r) && r != '.' {
			return true
		}
	}
	return false
}

// parseVersionParts 將版本字串解析為整數切片，供數值比較使用。
// 以小數點分割後，逐段轉換為 int。
//
// 例如：
//   - "0.1"  → [0, 1]
//   - "0.10" → [0, 10]
//   - "1.2.3" → [1, 2, 3]
//
// 參數：
//   - v: 純數字版本字串（不含文字）
//
// 回傳：
//   - []int: 解析後的整數切片
//   - error: 若解析失敗則回傳錯誤
func parseVersionParts(v string) ([]int, error) {
	parts := strings.Split(v, ".")
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("無法將版本段 %q 轉換為整數: %w", p, err)
		}
		result = append(result, n)
	}
	return result, nil
}

// CompareNumericVersions 以數值方式比較兩個純數字版本字串（支援多段小數點）。
//
// 比較規則：
//   - 逐段（以小數點分割）比較整數值
//   - 較短的版本在不足的段位補零（例如 "1.0" vs "1.0.3" 視為 "1.0.0" vs "1.0.3"）
//
// 例如：
//   - CompareNumericVersions("0.2", "0.10") → -1（0.2 < 0.10）
//   - CompareNumericVersions("0.10", "0.2") → 1（0.10 > 0.2）
//   - CompareNumericVersions("1.0", "1.0")  → 0（相等）
//
// 參數：
//   - v1: 第一個版本字串
//   - v2: 第二個版本字串
//
// 回傳：
//   - int: -1 表示 v1 < v2, 0 表示相等, 1 表示 v1 > v2
//   - error: 若版本字串格式不合法則回傳錯誤
func CompareNumericVersions(v1, v2 string) (int, error) {
	parts1, err := parseVersionParts(v1)
	if err != nil {
		return 0, fmt.Errorf("解析 v1 %q 失敗: %w", v1, err)
	}
	parts2, err := parseVersionParts(v2)
	if err != nil {
		return 0, fmt.Errorf("解析 v2 %q 失敗: %w", v2, err)
	}

	// 取最大長度進行比較
	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var a, b int
		if i < len(parts1) {
			a = parts1[i]
		}
		if i < len(parts2) {
			b = parts2[i]
		}
		if a < b {
			return -1, nil
		}
		if a > b {
			return 1, nil
		}
	}
	return 0, nil
}

// FindPreviousRevisionSmart 以智慧數值排序，搜尋同 project & phase 下版本號最接近且小於
// currentVersion 的前一版 BomRevision。
//
// 版本字串處理規則：
//  1. 先查詢同 project + phase 的所有 revision（不含 currentVersion 自身）。
//  2. 若任一 revision 的版本號（或 currentVersion）包含非數字字元：
//     - 回傳 (nil, true, nil)，hasNonNumericVersion=true，
//       呼叫端應輸出 Warning Log 並跳過自動匯入。
//  3. 若全部為純數字版本，以數值排序找出最高且 < currentVersion 的版本並回傳。
//
// 參數：
//   - db: GORM 資料庫連線
//   - projectID: 專案 ID
//   - phase: 階段名稱（如 "PV"）
//   - currentVersion: 當前版本字串（如 "0.3"）
//
// 回傳：
//   - *BomRevision: 找到的前一版，nil 表示無前一版
//   - bool: hasNonNumericVersion，true 表示版本號含文字，呼叫端需警告
//   - error: 資料庫查詢錯誤
func FindPreviousRevisionSmart(db *gorm.DB, projectID int64, phase, currentVersion string) (*BomRevision, bool, error) {
	// 步驟 1：檢查 currentVersion 本身是否含文字
	if HasNonNumericText(currentVersion) {
		return nil, true, nil
	}

	// 步驟 2：查詢同 project + phase 下所有其他 revision
	var candidates []BomRevision
	if err := db.Where("project_id = ? AND phase = ? AND version != ?",
		projectID, phase, currentVersion).Find(&candidates).Error; err != nil {
		return nil, false, fmt.Errorf("查詢候選 revision 失敗: %w", err)
	}

	if len(candidates) == 0 {
		// 無其他版本，表示此為第一版
		return nil, false, nil
	}

	// 步驟 3：檢查所有候選版本是否含文字
	for _, c := range candidates {
		if HasNonNumericText(c.Version) {
			return nil, true, nil
		}
	}

	// 步驟 4：依數值排序，找出最高且 < currentVersion 的版本
	var best *BomRevision
	for i := range candidates {
		c := &candidates[i]
		cmp, err := CompareNumericVersions(c.Version, currentVersion)
		if err != nil {
			// 解析失敗，視同含文字
			return nil, true, nil
		}
		if cmp >= 0 {
			// 候選版本 >= currentVersion，不符合「前一版」條件，略過
			continue
		}
		// 候選版本 < currentVersion，找最大值
		if best == nil {
			best = c
		} else {
			cmpBest, err := CompareNumericVersions(c.Version, best.Version)
			if err != nil {
				return nil, true, nil
			}
			if cmpBest > 0 {
				// 新候選比目前 best 更大（更接近 currentVersion）
				best = c
			}
		}
	}

	return best, false, nil
}
