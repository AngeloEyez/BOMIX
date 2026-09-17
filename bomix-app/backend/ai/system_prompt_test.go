package ai

import (
	"strings"
	"testing"
)

func TestGetSystemPrompt(t *testing.T) {
	tests := []struct {
		lang       string
		wantSubstr string
	}{
		{
			lang:       "zh-TW",
			wantSubstr: "Traditional Chinese (繁體中文)",
		},
		{
			lang:       "zh-CN",
			wantSubstr: "Simplified Chinese (简体中文)",
		},
		{
			lang:       "en",
			wantSubstr: "Always respond in English.",
		},
		{
			lang:       "",
			wantSubstr: "Traditional Chinese (繁體中文)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.lang, func(t *testing.T) {
			prompt := GetSystemPrompt(tc.lang)
			if !strings.Contains(prompt, tc.wantSubstr) {
				t.Errorf("GetSystemPrompt(%q) = %s, want substring %s", tc.lang, prompt, tc.wantSubstr)
			}
			if !strings.Contains(prompt, "You are BOMIX Assistant") {
				t.Errorf("Expected prompt to contain base English instructions")
			}
		})
	}
}
