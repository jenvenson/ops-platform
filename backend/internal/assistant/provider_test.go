// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package assistant

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jenvenson/ops-platform/pkg/config"
)

func TestProviderURLResolution(t *testing.T) {
	providers := map[string]string{
		"openai":    "https://api.openai.com/v1",
		"deepseek":  "https://api.deepseek.com/v1",
		"qwen":      "https://dashscope.aliyuncs.com/compatible-mode/v1",
		"tongyi":    "https://dashscope.aliyuncs.com/compatible-mode/v1",
		"zhipu":     "https://open.bigmodel.cn/api/paas/v4",
		"glm":       "https://open.bigmodel.cn/api/paas/v4",
		"moonshot":  "https://api.moonshot.cn/v1",
		"kimi":      "https://api.moonshot.cn/v1",
		"minimax":   "https://api.minimax.chat/v1",
		"doubao":    "https://ark.cn-beijing.volces.com/api/v3",
		"volcano":   "https://ark.cn-beijing.volces.com/api/v3",
		"baichuan":  "https://api.baichuan-ai.com/v1",
		"hunyuan":   "https://api.hunyuan.cloud.tencent.com/v1",
		"ernie":     "https://qianfan.baidubce.com/v2",
		"qianfan":   "https://qianfan.baidubce.com/v2",
	}

	baseOnly := func(url string) string {
		url = strings.TrimSuffix(url, "/v1")
		url = strings.TrimSuffix(url, "/v4")
		url = strings.TrimSuffix(url, "/v2")
		url = strings.TrimSuffix(url, "/v3")
		url = strings.TrimSuffix(url, "/compatible-mode/v1")
		return url
	}

	for provider, expected := range providers {
		t.Run(provider, func(t *testing.T) {
			cfg := config.AssistantConfig{
				Provider: provider,
			}
			got := resolveOpenAIBaseURL(cfg)
			if got != expected {
				t.Errorf("resolveOpenAIBaseURL(%q) = %q, want %q", provider, got, expected)
			}

			// Check default model is not empty
			model := defaultModelForProvider(provider)
			if model == "" {
				t.Errorf("defaultModelForProvider(%q) returned empty string", provider)
			}

			// Check connectivity (expect 401/404 — endpoint is alive)
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Head(baseOnly(got))
			if err != nil {
				t.Logf("WARN: %s unreachable: %v", provider, err)
				return
			}
			resp.Body.Close()
			t.Logf("OK: %s (%s) → HTTP %d", provider, got, resp.StatusCode)
		})
	}
}

func TestProviderCustomBaseURL(t *testing.T) {
	cfg := config.AssistantConfig{
		Provider: "custom",
		BaseURL:  "https://my-proxy.example.com/v1",
		ChatModel: "custom-model",
	}

	client := newOpenAIClient(cfg)
	if client == nil {
		t.Fatal("newOpenAIClient returned nil for custom provider")
	}
	if client.baseURL != "https://my-proxy.example.com/v1" {
		t.Errorf("baseURL = %q, want custom URL", client.baseURL)
	}
	if client.model != "custom-model" {
		t.Errorf("model = %q, want custom-model", client.model)
	}
}

func TestNewChatProviderRouting(t *testing.T) {
	tests := []struct {
		provider string
		isOllama bool
	}{
		{"ollama", true},
		{"", true},
		{"unknown", true},
		{"deepseek", false},
		{"qwen", false},
		{"zhipu", false},
		{"minimax", false},
		{"doubao", false},
		{"openai", false},
		{"custom", false}, // has baseURL set below
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			cfg := config.AssistantConfig{
				Provider: tt.provider,
				ChatModel: "test-model",
			}
			if tt.provider == "custom" {
				cfg.BaseURL = "https://custom.example.com/v1"
			}
			p := newChatProvider(cfg)
			if p == nil {
				t.Fatal("newChatProvider returned nil")
			}
			_, isOllama := p.(*ollamaClient)
			if isOllama != tt.isOllama {
				t.Errorf("isOllama = %v, want %v", isOllama, tt.isOllama)
			}
		})
	}
}

func TestDefaultModelForProvider(t *testing.T) {
	tests := map[string]string{
		"deepseek": "deepseek-chat",
		"qwen":     "qwen-plus",
		"zhipu":    "glm-4-flash",
		"kimi":     "moonshot-v1-8k",
		"minimax":  "abab6.5s-chat",
		"doubao":   "doubao-pro-32k",
		"baichuan": "Baichuan4",
	}

	for provider, expected := range tests {
		t.Run(provider, func(t *testing.T) {
			got := defaultModelForProvider(provider)
			if got != expected {
				t.Errorf("defaultModelForProvider(%q) = %q, want %q", provider, got, expected)
			}
		})
	}
}