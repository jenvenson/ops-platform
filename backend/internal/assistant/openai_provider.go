package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/edy/ops-platform/pkg/config"
)

type openaiClient struct {
	baseURL     string
	apiKey      string
	model       string
	embedModel  string
	temperature float64
	httpClient  *http.Client
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openaiMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	Stream      bool            `json:"stream"`
}

type openaiChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

type openaiEmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type openaiEmbedResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

func newOpenAIClient(cfg config.AssistantConfig) *openaiClient {
	baseURL := resolveOpenAIBaseURL(cfg)
	model := cfg.ChatModel
	if model == "" {
		model = defaultModelForProvider(cfg.Provider)
	}
	embedModel := cfg.EmbedModel
	if embedModel == "" && cfg.OllamaEmbedModel != "" {
		embedModel = cfg.OllamaEmbedModel
	}

	timeout := time.Duration(cfg.RequestTimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	return &openaiClient{
		baseURL:     strings.TrimRight(baseURL, "/"),
		apiKey:      cfg.APIKey,
		model:       model,
		embedModel:  embedModel,
		temperature: cfg.Temperature,
		httpClient:  &http.Client{Timeout: timeout},
	}
}

func resolveOpenAIBaseURL(cfg config.AssistantConfig) string {
	if cfg.BaseURL != "" {
		return cfg.BaseURL
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	// International
	case "openai":
		return "https://api.openai.com/v1"
	// Chinese mainstream models (all OpenAI-compatible)
	case "deepseek":
		return "https://api.deepseek.com/v1"
	case "qwen", "tongyi":
		return "https://dashscope.aliyuncs.com/compatible-mode/v1"
	case "zhipu", "glm":
		return "https://open.bigmodel.cn/api/paas/v4"
	case "moonshot", "kimi":
		return "https://api.moonshot.cn/v1"
	case "minimax":
		return "https://api.minimax.chat/v1"
	case "doubao", "volcano":
		return "https://ark.cn-beijing.volces.com/api/v3"
	case "baichuan":
		return "https://api.baichuan-ai.com/v1"
	case "hunyuan":
		return "https://api.hunyuan.cloud.tencent.com/v1"
	case "ernie", "qianfan":
		return "https://qianfan.baidubce.com/v2"
	default:
		return ""
	}
}

func defaultModelForProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "deepseek":
		return "deepseek-chat"
	case "qwen", "tongyi":
		return "qwen-plus"
	case "zhipu", "glm":
		return "glm-4-flash"
	case "moonshot", "kimi":
		return "moonshot-v1-8k"
	case "minimax":
		return "abab6.5s-chat"
	case "doubao", "volcano":
		return "doubao-pro-32k"
	case "baichuan":
		return "Baichuan4"
	case "hunyuan":
		return "hunyuan-lite"
	case "ernie", "qianfan":
		return "ernie-speed-128k"
	case "openai":
		return "gpt-4o"
	default:
		return "gpt-4o"
	}
}

func (c *openaiClient) Chat(ctx context.Context, system string, history []historyMessage) (string, int, int, error) {
	if c == nil || c.baseURL == "" || c.model == "" {
		return "", 0, 0, fmt.Errorf("openai client not configured: base_url=%q model=%q", c.baseURL, c.model)
	}

	messages := make([]openaiMessage, 0, len(history)+1)
	if strings.TrimSpace(system) != "" {
		messages = append(messages, openaiMessage{Role: "system", Content: system})
	}
	for _, msg := range history {
		messages = append(messages, openaiMessage{Role: msg.Role, Content: msg.Content})
	}

	reqBody := openaiChatRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: c.temperature,
		Stream:      false,
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", 0, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", 0, 0, fmt.Errorf("openai returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var chatResp openaiChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", 0, 0, err
	}

	if len(chatResp.Choices) == 0 {
		return "", 0, 0, fmt.Errorf("openai returned no choices")
	}

	return strings.TrimSpace(chatResp.Choices[0].Message.Content),
		chatResp.Usage.PromptTokens,
		chatResp.Usage.CompletionTokens,
		nil
}

func (c *openaiClient) ModelName() string {
	if c == nil || c.model == "" {
		return "openai"
	}
	return c.model
}

func (c *openaiClient) Embed(ctx context.Context, model string, inputs []string) ([][]float64, error) {
	if c == nil || c.baseURL == "" {
		return nil, fmt.Errorf("openai embed client not configured")
	}
	embedModel := model
	if embedModel == "" {
		embedModel = c.embedModel
	}
	if embedModel == "" {
		return nil, fmt.Errorf("embed model not configured")
	}
	if len(inputs) == 0 {
		return nil, nil
	}

	reqBody := openaiEmbedRequest{
		Model: embedModel,
		Input: inputs,
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/embeddings", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("openai embed returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var embedResp openaiEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, err
	}
	if len(embedResp.Data) == 0 {
		return nil, fmt.Errorf("openai embed returned no embeddings")
	}

	embeddings := make([][]float64, len(embedResp.Data))
	for i, d := range embedResp.Data {
		embeddings[i] = d.Embedding
	}
	return embeddings, nil
}
