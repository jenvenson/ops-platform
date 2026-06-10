// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ollamaClient struct {
	baseURL     string
	model       string
	temperature float64
	httpClient  *http.Client
}

type ollamaChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatRequest struct {
	Model    string              `json:"model"`
	Messages []ollamaChatMessage `json:"messages"`
	Stream   bool                `json:"stream"`
	Options  map[string]any      `json:"options,omitempty"`
}

type ollamaChatResponse struct {
	Model   string `json:"model"`
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	PromptEvalCount int `json:"prompt_eval_count"`
	EvalCount       int `json:"eval_count"`
}

type ollamaEmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type ollamaEmbedResponse struct {
	Embeddings [][]float64 `json:"embeddings"`
}

func newOllamaClient(baseURL, model string, temperature float64) *ollamaClient {
	return &ollamaClient{
		baseURL:     strings.TrimRight(baseURL, "/"),
		model:       model,
		temperature: temperature,
		httpClient:  &http.Client{Timeout: 45 * time.Second},
	}
}

func (c *ollamaClient) ModelName() string {
	if c == nil || c.model == "" {
		return "ollama"
	}
	return c.model
}

func (c *ollamaClient) prepareRequest(req *http.Request) {
	if c == nil || req == nil || req.URL == nil {
		return
	}

	if overrideHost := ollamaRequestHost(req.URL.String()); overrideHost != "" {
		req.Host = overrideHost
	}
}

func (c *ollamaClient) Chat(ctx context.Context, system string, history []historyMessage) (string, int, int, error) {
	if c == nil || c.baseURL == "" || c.model == "" {
		return "", 0, 0, fmt.Errorf("ollama client not configured")
	}

	messages := make([]ollamaChatMessage, 0, len(history)+1)
	if strings.TrimSpace(system) != "" {
		messages = append(messages, ollamaChatMessage{Role: "system", Content: system})
	}
	for _, msg := range history {
		messages = append(messages, ollamaChatMessage{Role: msg.Role, Content: msg.Content})
	}

	reqBody := ollamaChatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   false,
		Options: map[string]any{
			"temperature": c.temperature,
		},
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/chat", bytes.NewReader(payload))
	if err != nil {
		return "", 0, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.prepareRequest(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return "", 0, 0, fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var chatResp ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", 0, 0, err
	}

	return strings.TrimSpace(chatResp.Message.Content), chatResp.PromptEvalCount, chatResp.EvalCount, nil
}

func (c *ollamaClient) Embed(ctx context.Context, model string, inputs []string) ([][]float64, error) {
	if c == nil || c.baseURL == "" || model == "" {
		return nil, fmt.Errorf("ollama embed client not configured")
	}
	if len(inputs) == 0 {
		return nil, nil
	}

	reqBody := ollamaEmbedRequest{
		Model: model,
		Input: inputs,
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/embed", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.prepareRequest(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("ollama embed returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var embedResp ollamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, err
	}
	if len(embedResp.Embeddings) == 0 {
		return nil, fmt.Errorf("ollama embed returned no embeddings")
	}
	return embedResp.Embeddings, nil
}

func ollamaRequestHost(baseURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed == nil {
		return ""
	}

	host := strings.ToLower(parsed.Hostname())
	if host != "docker-host" && host != "host.docker.internal" {
		return ""
	}

	overrideHost := "127.0.0.1"
	if port := parsed.Port(); port != "" {
		overrideHost += ":" + port
	}
	return overrideHost
}