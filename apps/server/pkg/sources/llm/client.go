package llm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultProvider = "huggingface"
	HTTPTimeout     = 30 * time.Second
)

var ErrRateLimit = errors.New("llm: rate limit exceeded")

// ChatCompleter returns one text completion for a system/user prompt pair.
type ChatCompleter interface {
	ChatComplete(ctx context.Context, req ChatRequest) (ChatResponse, error)
	Model() string
}

type ChatRequest struct {
	SystemPrompt string
	UserPrompt   string
	Temperature  float64
	MaxTokens    int
}

type ChatResponse struct {
	Content string
	Model   string
}

// Config selects and configures a hosted chat-completion provider.
type Config struct {
	Provider   string
	APIKey     string
	Model      string
	HTTPClient *http.Client
	BaseURL    string
}

func NewFromConfig(cfg Config) (ChatCompleter, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if provider == "" {
		provider = DefaultProvider
	}

	switch provider {
	case "huggingface":
		return newHFChatCompleter(cfg)
	case "openai":
		return newOpenAIChatCompleter(cfg)
	case "anthropic":
		return newAnthropicChatCompleter(cfg)
	default:
		return nil, fmt.Errorf("llm: unsupported provider %q", cfg.Provider)
	}
}

func httpClient(cfg Config) *http.Client {
	if cfg.HTTPClient != nil {
		return cfg.HTTPClient
	}
	return &http.Client{Timeout: HTTPTimeout}
}

func validateChatRequest(req ChatRequest) error {
	if strings.TrimSpace(req.UserPrompt) == "" {
		return errors.New("llm: user prompt required")
	}
	if req.MaxTokens <= 0 {
		return errors.New("llm: max tokens must be positive")
	}
	return nil
}

func readErrorBody(body io.Reader) string {
	data, _ := io.ReadAll(io.LimitReader(body, 4096))
	return strings.TrimSpace(string(data))
}
