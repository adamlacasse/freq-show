package embeddings

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultProvider = "voyage"
	HTTPTimeout     = 10 * time.Second
)

var ErrRateLimit = errors.New("embeddings: rate limit exceeded")

// Embedder converts text into dense vectors suitable for semantic retrieval.
type Embedder interface {
	Encode(ctx context.Context, text string) ([]float32, error)
	EncodeBatch(ctx context.Context, texts []string) ([][]float32, error)
	Model() string
	Dim() int
}

// Config selects and configures a hosted embedding provider.
type Config struct {
	Provider   string
	APIKey     string
	Model      string
	HTTPClient *http.Client
	BaseURL    string
}

// NewFromConfig returns an embedding client for the configured provider.
func NewFromConfig(cfg Config) (Embedder, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if provider == "" {
		provider = DefaultProvider
	}

	switch provider {
	case "voyage":
		return newVoyageEmbedder(cfg)
	case "openai":
		return newOpenAIEmbedder(cfg)
	case "huggingface":
		return newHFEmbedder(cfg)
	default:
		return nil, fmt.Errorf("embeddings: unsupported provider %q", cfg.Provider)
	}
}

func httpClient(cfg Config) *http.Client {
	if cfg.HTTPClient != nil {
		return cfg.HTTPClient
	}
	return &http.Client{Timeout: HTTPTimeout}
}

func validateTexts(texts []string) error {
	if len(texts) == 0 {
		return errors.New("embeddings: at least one input is required")
	}
	for i, text := range texts {
		if strings.TrimSpace(text) == "" {
			return fmt.Errorf("embeddings: input %d is empty", i)
		}
	}
	return nil
}

func one(vecs [][]float32, err error) ([]float32, error) {
	if err != nil {
		return nil, err
	}
	if len(vecs) != 1 {
		return nil, fmt.Errorf("embeddings: expected 1 vector, got %d", len(vecs))
	}
	return vecs[0], nil
}
