package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIconBytesValidICO 驗證內嵌的 bomx.ico 是否為合法的 ICO 格式
func TestIconBytesValidICO(t *testing.T) {
	data := GetIconBytes()
	if len(data) == 0 {
		t.Skip("在非 Windows 編譯環境下略過圖標資料驗證")
	}

	// ICO 標頭固定為 6 個 bytes：
	// Bytes 0-1: 保留 (0x0000)
	// Bytes 2-3: 類型 (0x0001 表示 ICO)
	// Bytes 4-5: 包含的影像數量 (>= 1)
	assert.GreaterOrEqual(t, len(data), 6, "ICO 檔案大小需大於 6 bytes")
	assert.Equal(t, byte(0), data[0], "Byte 0 需為 0")
	assert.Equal(t, byte(0), data[1], "Byte 1 需為 0")
	assert.Equal(t, byte(1), data[2], "Byte 2 需為 1 (ICO type)")
	assert.Equal(t, byte(0), data[3], "Byte 3 需為 0")

	// 檢查影像數量
	imageCount := int(data[4]) | int(data[5])<<8
	assert.Greater(t, imageCount, 0, "ICO 內應至少包含 1 個圖標解析度")
}
