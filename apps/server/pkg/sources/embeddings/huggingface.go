package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultHFModel = "sentence-transformers/all-MiniLM-L6-v2"
	DefaultHFDim   = 384
	hfBaseURL      = "https://router.huggingface.co/hf-inference/models"
)

type HFEmbedder struct {
	apiKey  string
	model   string
	dim     int
	http    *http.Client
	baseURL string
}

func newHFEmbedder(cfg Config) (*HFEmbedder, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("embeddings: huggingface api key required")
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = DefaultHFModel
	}
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = hfBaseURL
	}
	return &HFEmbedder{
		apiKey:  strings.TrimSpace(cfg.APIKey),
		model:   model,
		dim:     hfDim(model),
		http:    httpClient(cfg),
		baseURL: baseURL,
	}, nil
}

func (e *HFEmbedder) Encode(ctx context.Context, text string) ([]float32, error) {
	return one(e.EncodeBatch(ctx, []string{text}))
}

func (e *HFEmbedder) EncodeBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if err := validateTexts(texts); err != nil {
		return nil, err
	}

	vecs, retryAfter, err := e.encodeOnce(ctx, texts)
	if err == nil || retryAfter <= 0 {
		return vecs, err
	}

	timer := time.NewTimer(retryAfter)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
	}
	return e.encodeOnceNoRetry(ctx, texts)
}

func (e *HFEmbedder) Model() string {
	return e.model
}

func (e *HFEmbedder) Dim() int {
	return e.dim
}

func (e *HFEmbedder) encodeOnceNoRetry(ctx context.Context, texts []string) ([][]float32, error) {
	vecs, _, err := e.encodeOnce(ctx, texts)
	return vecs, err
}

func (e *HFEmbedder) encodeOnce(ctx context.Context, texts []string) ([][]float32, time.Duration, error) {
	payload := hfEmbeddingRequest{
		Inputs:    texts,
		Normalize: true,
		Truncate:  true,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, fmt.Errorf("embeddings: encode huggingface request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/"+e.model, bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("embeddings: build huggingface request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("embeddings: call huggingface: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, 0, ErrRateLimit
	}
	if resp.StatusCode == http.StatusServiceUnavailable {
		retryAfter := parseHFEstimatedTime(resp)
		if retryAfter > 0 {
			return nil, retryAfter, fmt.Errorf("embeddings: huggingface model loading; retry after %s", retryAfter)
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, 0, fmt.Errorf("embeddings: huggingface returned %s: %s", resp.Status, readErrorBody(resp.Body))
	}

	var decoded [][]float64
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, 0, fmt.Errorf("embeddings: decode huggingface response: %w", err)
	}
	if len(decoded) != len(texts) {
		return nil, 0, fmt.Errorf("embeddings: huggingface returned %d vectors for %d inputs", len(decoded), len(texts))
	}
	vecs := make([][]float32, len(decoded))
	for i, item := range decoded {
		if len(item) == 0 {
			return nil, 0, fmt.Errorf("embeddings: huggingface returned empty vector at index %d", i)
		}
		vec := make([]float32, len(item))
		for j, val := range item {
			vec[j] = float32(val)
		}
		vecs[i] = vec
	}
	return vecs, 0, nil
}

type hfEmbeddingRequest struct {
	Inputs    []string `json:"inputs"`
	Normalize bool     `json:"normalize"`
	Truncate  bool     `json:"truncate"`
}

func parseHFEstimatedTime(resp *http.Response) time.Duration {
	var payload struct {
		EstimatedTime float64 `json:"estimated_time"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0
	}
	if payload.EstimatedTime <= 0 {
		return 0
	}
	return time.Duration(payload.EstimatedTime * float64(time.Second))
}

func hfDim(model string) int {
	switch model {
	case "sentence-transformers/all-MiniLM-L6-v2":
		return DefaultHFDim
	default:
		return 0
	}
}
