package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const (
	DefaultAnthropicModel = "claude-3-5-haiku-latest"
	anthropicBaseURL      = "https://api.anthropic.com/v1"
	anthropicVersion      = "2023-06-01"
)

type AnthropicChatCompleter struct {
	apiKey  string
	model   string
	http    *http.Client
	baseURL string
}

func newAnthropicChatCompleter(cfg Config) (*AnthropicChatCompleter, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("llm: anthropic api key required")
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = DefaultAnthropicModel
	}
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = anthropicBaseURL
	}
	return &AnthropicChatCompleter{
		apiKey:  strings.TrimSpace(cfg.APIKey),
		model:   model,
		http:    httpClient(cfg),
		baseURL: baseURL,
	}, nil
}

func (c *AnthropicChatCompleter) ChatComplete(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	if err := validateChatRequest(req); err != nil {
		return ChatResponse{}, err
	}

	payload := anthropicRequest{
		Model:       c.model,
		System:      req.SystemPrompt,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Messages: []anthropicMessage{
			{Role: "user", Content: req.UserPrompt},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("llm: encode anthropic request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("llm: build anthropic request: %w", err)
	}
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.http.Do(httpReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("llm: call anthropic: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode == http.StatusTooManyRequests {
		return ChatResponse{}, ErrRateLimit
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return ChatResponse{}, fmt.Errorf("llm: anthropic returned %s: %s", httpResp.Status, readErrorBody(httpResp.Body))
	}

	var decoded anthropicResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&decoded); err != nil {
		return ChatResponse{}, fmt.Errorf("llm: decode anthropic response: %w", err)
	}

	var content strings.Builder
	for _, block := range decoded.Content {
		if block.Type == "text" {
			content.WriteString(block.Text)
		}
	}
	if strings.TrimSpace(content.String()) == "" {
		return ChatResponse{}, errors.New("llm: anthropic returned empty content")
	}
	model := decoded.Model
	if model == "" {
		model = c.model
	}
	return ChatResponse{Content: content.String(), Model: model}, nil
}

func (c *AnthropicChatCompleter) Model() string {
	return c.model
}

type anthropicRequest struct {
	Model       string             `json:"model"`
	System      string             `json:"system,omitempty"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature,omitempty"`
	Messages    []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Model   string `json:"model"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}
