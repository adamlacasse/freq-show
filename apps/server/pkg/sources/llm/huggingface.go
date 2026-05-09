package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

const hfChatBaseURL = "https://router.huggingface.co/v1"

var DefaultHFModelCandidates = []string{
	"meta-llama/Llama-3.1-8B-Instruct",
	"Qwen/Qwen2.5-7B-Instruct",
	"HuggingFaceH4/zephyr-7b-beta",
	"microsoft/Phi-3.5-mini-instruct",
}

type HFChatCompleter struct {
	*OpenAIChatCompleter
	mu         sync.RWMutex
	candidates []string
	pinned     string
}

func newHFChatCompleter(cfg Config) (*HFChatCompleter, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("llm: huggingface api key required")
	}
	model := strings.TrimSpace(cfg.Model)
	candidates := append([]string(nil), DefaultHFModelCandidates...)
	if model != "" {
		candidates = []string{model}
	}
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = hfChatBaseURL
	}
	return &HFChatCompleter{
		OpenAIChatCompleter: &OpenAIChatCompleter{
			apiKey:  strings.TrimSpace(cfg.APIKey),
			model:   candidates[0],
			http:    httpClient(cfg),
			baseURL: baseURL,
		},
		candidates: candidates,
	}, nil
}

func (c *HFChatCompleter) ChatComplete(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	if err := validateChatRequest(req); err != nil {
		return ChatResponse{}, err
	}

	if pinned := c.pinnedModel(); pinned != "" {
		return c.chatComplete(ctx, pinned, req)
	}

	var errs []error
	for _, model := range c.candidates {
		resp, err := c.chatComplete(ctx, model, req)
		if err == nil {
			c.setPinnedModel(model)
			if resp.Model == "" {
				resp.Model = model
			}
			return resp, nil
		}
		if errors.Is(err, ErrRateLimit) {
			return ChatResponse{}, err
		}
		errs = append(errs, fmt.Errorf("%s: %w", model, err))
	}
	return ChatResponse{}, fmt.Errorf("llm: huggingface models failed: %w", errors.Join(errs...))
}

func (c *HFChatCompleter) Model() string {
	if pinned := c.pinnedModel(); pinned != "" {
		return pinned
	}
	return c.model
}

func (c *HFChatCompleter) pinnedModel() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.pinned
}

func (c *HFChatCompleter) setPinnedModel(model string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pinned = model
	c.model = model
}
