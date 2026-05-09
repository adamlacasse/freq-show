package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

const (
	DefaultVoyageModel = "voyage-3-lite"
	DefaultVoyageDim   = 512
	voyageBaseURL      = "https://api.voyageai.com/v1"
)

type VoyageEmbedder struct {
	apiKey  string
	model   string
	dim     int
	http    *http.Client
	baseURL string
}

func newVoyageEmbedder(cfg Config) (*VoyageEmbedder, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("embeddings: voyage api key required")
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = DefaultVoyageModel
	}
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = voyageBaseURL
	}
	return &VoyageEmbedder{
		apiKey:  strings.TrimSpace(cfg.APIKey),
		model:   model,
		dim:     voyageDim(model),
		http:    httpClient(cfg),
		baseURL: baseURL,
	}, nil
}

func (e *VoyageEmbedder) Encode(ctx context.Context, text string) ([]float32, error) {
	return one(e.encode(ctx, []string{text}, "query"))
}

func (e *VoyageEmbedder) EncodeBatch(ctx context.Context, texts []string) ([][]float32, error) {
	return e.encode(ctx, texts, "document")
}

func (e *VoyageEmbedder) Model() string {
	return e.model
}

func (e *VoyageEmbedder) Dim() int {
	return e.dim
}

func (e *VoyageEmbedder) encode(ctx context.Context, texts []string, inputType string) ([][]float32, error) {
	if err := validateTexts(texts); err != nil {
		return nil, err
	}

	payload := voyageEmbeddingRequest{
		Input:     texts,
		Model:     e.model,
		InputType: inputType,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("embeddings: encode voyage request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("embeddings: build voyage request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embeddings: call voyage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrRateLimit
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("embeddings: voyage returned %s: %s", resp.Status, readErrorBody(resp.Body))
	}

	var decoded voyageEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("embeddings: decode voyage response: %w", err)
	}
	return vectorsFromEmbeddingData(decoded.Data, len(texts), "voyage")
}

type voyageEmbeddingRequest struct {
	Input     []string `json:"input"`
	Model     string   `json:"model"`
	InputType string   `json:"input_type,omitempty"`
}

type voyageEmbeddingResponse struct {
	Data []embeddingDatum `json:"data"`
}

func voyageDim(model string) int {
	switch model {
	case "voyage-3-lite":
		return 512
	case "voyage-3-large", "voyage-3.5", "voyage-3.5-lite", "voyage-4", "voyage-4-lite":
		return 1024
	default:
		return 0
	}
}

type embeddingDatum struct {
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}

func vectorsFromEmbeddingData(data []embeddingDatum, want int, provider string) ([][]float32, error) {
	if len(data) != want {
		return nil, fmt.Errorf("embeddings: %s returned %d vectors for %d inputs", provider, len(data), want)
	}
	sort.Slice(data, func(i, j int) bool {
		return data[i].Index < data[j].Index
	})

	vecs := make([][]float32, len(data))
	for i, item := range data {
		if item.Index != i {
			return nil, fmt.Errorf("embeddings: %s response missing index %d", provider, i)
		}
		if len(item.Embedding) == 0 {
			return nil, fmt.Errorf("embeddings: %s returned empty vector at index %d", provider, i)
		}
		vec := make([]float32, len(item.Embedding))
		for j, val := range item.Embedding {
			vec[j] = float32(val)
		}
		vecs[i] = vec
	}
	return vecs, nil
}

func readErrorBody(body io.Reader) string {
	data, _ := io.ReadAll(io.LimitReader(body, 4096))
	return strings.TrimSpace(string(data))
}
