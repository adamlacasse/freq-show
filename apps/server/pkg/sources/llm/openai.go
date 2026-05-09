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
	DefaultOpenAIModel = "gpt-4o-mini"
	openAIBaseURL      = "https://api.openai.com/v1"
)

type OpenAIChatCompleter struct {
	apiKey  string
	model   string
	http    *http.Client
	baseURL string
}

func newOpenAIChatCompleter(cfg Config) (*OpenAIChatCompleter, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("llm: openai api key required")
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = DefaultOpenAIModel
	}
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = openAIBaseURL
	}
	return &OpenAIChatCompleter{
		apiKey:  strings.TrimSpace(cfg.APIKey),
		model:   model,
		http:    httpClient(cfg),
		baseURL: baseURL,
	}, nil
}

func (c *OpenAIChatCompleter) ChatComplete(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	if err := validateChatRequest(req); err != nil {
		return ChatResponse{}, err
	}
	resp, err := c.chatComplete(ctx, c.model, req)
	if err != nil {
		return ChatResponse{}, err
	}
	return resp, nil
}

func (c *OpenAIChatCompleter) Model() string {
	return c.model
}

func (c *OpenAIChatCompleter) chatComplete(ctx context.Context, model string, req ChatRequest) (ChatResponse, error) {
	payload := openAIChatRequest{
		Model:       model,
		Messages:    chatMessages(req),
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("llm: encode openai request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("llm: build openai request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.http.Do(httpReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("llm: call openai: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode == http.StatusTooManyRequests {
		return ChatResponse{}, ErrRateLimit
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return ChatResponse{}, fmt.Errorf("llm: openai returned %s: %s", httpResp.Status, readErrorBody(httpResp.Body))
	}

	var decoded openAIChatResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&decoded); err != nil {
		return ChatResponse{}, fmt.Errorf("llm: decode openai response: %w", err)
	}
	if len(decoded.Choices) == 0 || strings.TrimSpace(decoded.Choices[0].Message.Content) == "" {
		return ChatResponse{}, errors.New("llm: openai returned empty content")
	}
	responseModel := decoded.Model
	if responseModel == "" {
		responseModel = model
	}
	return ChatResponse{Content: decoded.Choices[0].Message.Content, Model: responseModel}, nil
}

type openAIChatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func chatMessages(req ChatRequest) []chatMessage {
	messages := make([]chatMessage, 0, 2)
	if strings.TrimSpace(req.SystemPrompt) != "" {
		messages = append(messages, chatMessage{Role: "system", Content: req.SystemPrompt})
	}
	messages = append(messages, chatMessage{Role: "user", Content: req.UserPrompt})
	return messages
}

type openAIChatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}
