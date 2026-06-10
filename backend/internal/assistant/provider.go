// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package assistant

import (
	"context"
	"strings"

	"github.com/jenvenson/ops-platform/pkg/config"
)

// ChatProvider defines the interface for AI chat model backends.
type ChatProvider interface {
	Chat(ctx context.Context, system string, history []historyMessage) (string, int, int, error)
	ModelName() string
}

// EmbedProvider defines the interface for embedding model backends (optional capability).
type EmbedProvider interface {
	Embed(ctx context.Context, model string, inputs []string) ([][]float64, error)
}

func newChatProvider(cfg config.AssistantConfig) ChatProvider {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "openai",
		"deepseek",
		"qwen", "tongyi",
		"zhipu", "glm",
		"moonshot", "kimi",
		"minimax",
		"doubao", "volcano",
		"baichuan",
		"hunyuan",
		"ernie", "qianfan":
		return newOpenAIClient(cfg)
	case "custom":
		if cfg.BaseURL != "" {
			return newOpenAIClient(cfg)
		}
		return nil
	default:
		return newOllamaClient(cfg.OllamaBaseURL, cfg.OllamaChatModel, cfg.Temperature)
	}
}

func newEmbedProvider(cfg config.AssistantConfig, chatProvider ChatProvider) EmbedProvider {
	if ep, ok := chatProvider.(EmbedProvider); ok {
		return ep
	}
	return nil
}