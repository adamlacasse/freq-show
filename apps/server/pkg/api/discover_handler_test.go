package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/adamlacasse/freq-show/apps/server/pkg/data"
	"github.com/adamlacasse/freq-show/apps/server/pkg/db"
	"github.com/adamlacasse/freq-show/apps/server/pkg/discovery"
	"github.com/adamlacasse/freq-show/apps/server/pkg/sources/llm"
)

func TestDiscoverHandlerRequiresConfiguredService(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/discover", strings.NewReader(`{"query":"jazz"}`))
	res := httptest.NewRecorder()

	discoverHandler(nil).ServeHTTP(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", res.Code)
	}
}

func TestDiscoverHandlerValidatesQuery(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/discover", strings.NewReader(`{"query":"   "}`))
	res := httptest.NewRecorder()

	discoverHandler(&discovery.Service{}).ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", res.Code)
	}
}

func TestDiscoverHandlerReturnsDiscoveryResult(t *testing.T) {
	ctx := context.Background()
	store, err := db.NewMemoryStore(ctx)
	if err != nil {
		t.Fatalf("NewMemoryStore returned error: %v", err)
	}
	for i := 1; i <= 5; i++ {
		album := &data.Album{ID: "album-" + string(rune('0'+i)), Title: "Album", ArtistName: "Artist", Genre: "jazz"}
		if err := store.SaveAlbum(ctx, album); err != nil {
			t.Fatalf("SaveAlbum returned error: %v", err)
		}
		if err := store.SaveEmbedding(ctx, album.ID, "test-model", []float32{float32(i), 1}); err != nil {
			t.Fatalf("SaveEmbedding returned error: %v", err)
		}
	}
	svc := &discovery.Service{
		Embedder:   apiFakeEmbedder{model: "test-model", vec: []float32{1, 0}},
		LLM:        &apiFakeLLM{responses: []string{apiInterpretationJSON(), apiCurationJSON()}},
		Embeddings: store,
		Albums:     store,
	}

	req := httptest.NewRequest(http.MethodPost, "/discover", strings.NewReader(`{"query":"modern jazz"}`))
	res := httptest.NewRecorder()

	discoverHandler(svc).ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	var payload data.DiscoveryResult
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Picks) != discovery.FinalPicks {
		t.Fatalf("expected %d picks, got %d", discovery.FinalPicks, len(payload.Picks))
	}
}

func TestDiscoveryUserErrorIdentifiesStage(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want string
	}{
		{fmt.Errorf("discovery: interpret query: upstream failed"), "discovery interpretation failed"},
		{fmt.Errorf("discovery: embed interpreted query: upstream failed"), "discovery query embedding failed"},
		{fmt.Errorf("discovery: curate candidates: upstream failed"), "discovery curation failed"},
		{errors.New("other failure"), "discovery request failed"},
	} {
		if got := discoveryUserError(tc.err); got != tc.want {
			t.Fatalf("discoveryUserError(%v) = %q, want %q", tc.err, got, tc.want)
		}
	}
}

type apiFakeEmbedder struct {
	model string
	vec   []float32
}

func (e apiFakeEmbedder) Encode(ctx context.Context, text string) ([]float32, error) {
	_ = ctx
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("empty query text")
	}
	return append([]float32(nil), e.vec...), nil
}

func (e apiFakeEmbedder) EncodeBatch(ctx context.Context, texts []string) ([][]float32, error) {
	_ = ctx
	vecs := make([][]float32, len(texts))
	for i := range texts {
		vecs[i] = append([]float32(nil), e.vec...)
	}
	return vecs, nil
}

func (e apiFakeEmbedder) Model() string { return e.model }
func (e apiFakeEmbedder) Dim() int      { return len(e.vec) }

type apiFakeLLM struct {
	responses []string
	calls     int
}

func (f *apiFakeLLM) ChatComplete(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
	_ = ctx
	_ = req
	if f.calls >= len(f.responses) {
		return llm.ChatResponse{}, errors.New("no fake response configured")
	}
	response := f.responses[f.calls]
	f.calls++
	return llm.ChatResponse{Content: response, Model: "fake"}, nil
}

func (f *apiFakeLLM) Model() string { return "fake" }

func apiInterpretationJSON() string {
	return `{
		"mood":"warm modern jazz",
		"eraHints":[],
		"sonicQualities":["acoustic"],
		"referenceArtists":[],
		"avoid":[],
		"discoveryAppetite":"medium"
	}`
}

func apiCurationJSON() string {
	return `{
		"picks":[
			{"rank":1,"albumId":"album-5","whyItFits":"Warm and textural.","whatToListenFor":"Listen for the brushed drums."},
			{"rank":2,"albumId":"album-4","whyItFits":"Modern but relaxed.","whatToListenFor":"Notice the piano voicings."},
			{"rank":3,"albumId":"album-3","whyItFits":"Acoustic and spacious.","whatToListenFor":"Follow the bass movement."},
			{"rank":4,"albumId":"album-2","whyItFits":"Softly exploratory.","whatToListenFor":"Hear the horn phrasing."},
			{"rank":5,"albumId":"album-1","whyItFits":"Adjacent and approachable.","whatToListenFor":"Focus on the cymbal texture."}
		]
	}`
}
