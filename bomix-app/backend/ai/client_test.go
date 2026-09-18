package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildModelsURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{
			name:     "標準 v1 結尾",
			baseURL:  "https://api.openai.com/v1",
			expected: "https://api.openai.com/v1/models",
		},
		{
			name:     "帶有斜線結尾",
			baseURL:  "https://api.openai.com/v1/",
			expected: "https://api.openai.com/v1/models",
		},
		{
			name:     "無 v1 結尾",
			baseURL:  "https://api.openai.com",
			expected: "https://api.openai.com/v1/models",
		},
		{
			name:     "帶有 chat/completions 結尾",
			baseURL:  "http://localhost:11434/v1/chat/completions",
			expected: "http://localhost:11434/v1/models",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient(tt.baseURL, "test-key", "gpt-4o", 10)
			actual := c.buildModelsURL()
			if actual != tt.expected {
				t.Errorf("buildModelsURL() = %v, 期望為 %v", actual, tt.expected)
			}
		})
	}
}

func TestListModels(t *testing.T) {
	// 建立 Mock HTTP 伺服器模擬 OpenAI 相容的 /v1/models 端點
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("Authorization") != "Bearer mock-api-key" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		mockResp := ModelListResponse{
			Object: "list",
			Data: []ModelItem{
				{ID: "gpt-4o", Object: "model"},
				{ID: "gpt-4o-mini", Object: "model"},
				{ID: "o3-mini", Object: "model"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	client := NewClient(server.URL+"/v1", "mock-api-key", "", 5)
	models, err := client.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels 失敗: %v", err)
	}

	if len(models) != 3 {
		t.Fatalf("預期取得 3 個模型，實際取得 %d 個", len(models))
	}

	expectedModels := []string{"gpt-4o", "gpt-4o-mini", "o3-mini"}
	for i, m := range models {
		if m != expectedModels[i] {
			t.Errorf("Index %d: 預期為 %s, 實際為 %s", i, expectedModels[i], m)
		}
	}
}
