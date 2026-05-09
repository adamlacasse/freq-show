package embeddings

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
	DefaultOpenAIModel = "text-embedding-3-small"
	openAIBaseURL      = "https://api.openai.com/v1"
)

type OpenAIEmbedder struct {
	apiKey  string
	model   string
	dim     int
	http    *http.Client
	baseURL string
}

func newOpenAIEmbedder(cfg Config) (*OpenAIEmbedder, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("embeddings: openai api key required")
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = DefaultOpenAIModel
	}
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = openAIBaseURL
	}
	return &OpenAIEmbedder{
		apiKey:  strings.TrimSpace(cfg.APIKey),
		model:   model,
		dim:     openAIDim(model),
		http:    httpClient(cfg),
		baseURL: baseURL,
	}, nil
}

func (e *OpenAIEmbedder) Encode(ctx context.Context, text string) ([]float32, error) {
	return one(e.EncodeBatch(ctx, []string{text}))
}

func (e *OpenAIEmbedder) EncodeBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if err := validateTexts(texts); err != nil {
		return nil, err
	}

	payload := openAIEmbeddingRequest{
		Input:          texts,
		Model:          e.model,
		EncodingFormat: "float",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("embeddings: encode openai request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("embeddings: build openai request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embeddings: call openai: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrRateLimit
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("embeddings: openai returned %s: %s", resp.Status, readErrorBody(resp.Body))
	}

	var decoded openAIEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("embeddings: decode openai response: %w", err)
	}
	return vectorsFromEmbeddingData(decoded.Data, len(texts), "openai")
}

func (e *OpenAIEmbedder) Model() string {
	return e.model
}

func (e *OpenAIEmbedder) Dim() int {
	return e.dim
}

type openAIEmbeddingRequest struct {
	Input          []string `json:"input"`
	Model          string   `json:"model"`
	EncodingFormat string   `json:"encoding_format,omitempty"`
}

type openAIEmbeddingResponse struct {
	Data []embeddingDatum `json:"data"`
}

func openAIDim(model string) int {
	switch model {
	case "text-embedding-3-small", "text-embedding-ada-002":
		return 1536
	case "text-embedding-3-large":
		return 3072
	default:
		return 0
	}
}
