package discovery

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/adamlacasse/freq-show/apps/server/pkg/data"
	"github.com/adamlacasse/freq-show/apps/server/pkg/db"
	"github.com/adamlacasse/freq-show/apps/server/pkg/sources/llm"
)

func TestBuildAlbumEmbeddingTextUsesMetadataAndSkipsThinRecords(t *testing.T) {
	thin := BuildAlbumEmbeddingText(&data.Album{Title: "Bare"}, nil)
	if thin != "" {
		t.Fatalf("expected thin record to be skipped, got %q", thin)
	}

	text := BuildAlbumEmbeddingText(&data.Album{
		Title:      "In Rainbows",
		ArtistName: "Radiohead",
		Year:       2007,
		Genre:      "art rock",
		Review: data.Review{
			Source:  "Pitchfork",
			Rating:  9.3,
			Summary: strings.Repeat("warm electronic textures with human songwriting ", 4),
		},
		Tracks: []data.Track{{Title: "15 Step"}, {Title: "Weird Fishes/Arpeggi"}},
	}, nil)
	for _, want := range []string{`"In Rainbows" by Radiohead`, "Genre: art rock.", "Pitchfork (9.3/10)", "Tracks include: 15 Step"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected embedding text to contain %q, got %q", want, text)
		}
	}
}

func TestServiceRunHappyPath(t *testing.T) {
	ctx := context.Background()
	store, err := db.NewMemoryStore(ctx)
	if err != nil {
		t.Fatalf("NewMemoryStore returned error: %v", err)
	}
	for i := 1; i <= 5; i++ {
		album := &data.Album{
			ID:         fmt.Sprintf("album-%d", i),
			Title:      fmt.Sprintf("Album %d", i),
			ArtistName: fmt.Sprintf("Artist %d", i),
			Year:       2000 + i,
			Genre:      "jazz",
		}
		if err := store.SaveAlbum(ctx, album); err != nil {
			t.Fatalf("SaveAlbum returned error: %v", err)
		}
		if err := store.SaveEmbedding(ctx, album.ID, "test-model", []float32{float32(6 - i), 1}); err != nil {
			t.Fatalf("SaveEmbedding returned error: %v", err)
		}
	}

	service := &Service{
		Embedder:   fakeEmbedder{model: "test-model", vec: []float32{1, 0}},
		LLM:        &fakeLLM{responses: []string{interpretationJSON(), curationJSON()}},
		Embeddings: store,
		Albums:     store,
	}

	result, err := service.Run(ctx, data.DiscoveryQuery{Query: "modern jazz for a quiet morning"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Interpreted.DiscoveryAppetite != "medium" {
		t.Fatalf("unexpected interpretation: %#v", result.Interpreted)
	}
	if len(result.Picks) != FinalPicks {
		t.Fatalf("expected %d picks, got %d", FinalPicks, len(result.Picks))
	}
	if result.Picks[0].AlbumID != "album-1" || result.Picks[0].Title != "Album 1" {
		t.Fatalf("unexpected first pick: %#v", result.Picks[0])
	}
	if result.Picks[0].SpotifySearchURL == "" {
		t.Fatalf("expected spotify search URL")
	}
}

func TestInterpretationRetriesInvalidShape(t *testing.T) {
	ctx := context.Background()
	client := &fakeLLM{responses: []string{`{"mood":"missing fields"}`, interpretationJSON()}}

	interpreted, err := interpretQuery(ctx, client, "give me something warm", nil)
	if err != nil {
		t.Fatalf("interpretQuery returned error: %v", err)
	}
	if interpreted.Mood != "warm modern jazz" {
		t.Fatalf("unexpected interpretation: %#v", interpreted)
	}
	if client.calls != 2 {
		t.Fatalf("expected 2 calls, got %d", client.calls)
	}
	if !strings.Contains(client.requests[1].UserPrompt, invalidJSONReminder) {
		t.Fatalf("expected retry prompt reminder, got %q", client.requests[1].UserPrompt)
	}
}

func TestCurateAllowsSparseCandidateSet(t *testing.T) {
	ctx := context.Background()
	candidates := []*data.Album{
		{ID: "album-1", Title: "One", ArtistName: "Artist One"},
		{ID: "album-2", Title: "Two", ArtistName: "Artist Two"},
	}
	client := &fakeLLM{responses: []string{`{
		"picks":[
			{"rank":1,"albumId":"album-1","whyItFits":"Warm and textural.","whatToListenFor":"Listen for the brushed drums."},
			{"rank":2,"albumId":"album-2","whyItFits":"Modern but relaxed.","whatToListenFor":"Notice the piano voicings."}
		]
	}`}}

	picks, err := curate(ctx, client, data.InterpretedQuery{Mood: "warm", DiscoveryAppetite: "medium"}, candidates)
	if err != nil {
		t.Fatalf("curate returned error: %v", err)
	}
	if len(picks) != 2 {
		t.Fatalf("expected 2 picks, got %d", len(picks))
	}
	if !strings.Contains(client.requests[0].UserPrompt, "Pick 2.") {
		t.Fatalf("expected sparse prompt to ask for 2 picks, got %q", client.requests[0].UserPrompt)
	}
}

func TestServiceRunReturnsNoEmbeddedAlbums(t *testing.T) {
	ctx := context.Background()
	store, err := db.NewMemoryStore(ctx)
	if err != nil {
		t.Fatalf("NewMemoryStore returned error: %v", err)
	}
	service := &Service{
		Embedder:   fakeEmbedder{model: "test-model", vec: []float32{1, 0}},
		LLM:        &fakeLLM{responses: []string{interpretationJSON()}},
		Embeddings: store,
		Albums:     store,
	}

	_, err = service.Run(ctx, data.DiscoveryQuery{Query: "modern jazz"})
	if !errors.Is(err, ErrNoEmbeddedAlbums) {
		t.Fatalf("expected ErrNoEmbeddedAlbums, got %v", err)
	}
}

type fakeEmbedder struct {
	model string
	vec   []float32
}

func (e fakeEmbedder) Encode(ctx context.Context, text string) ([]float32, error) {
	_ = ctx
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("empty query text")
	}
	return append([]float32(nil), e.vec...), nil
}

func (e fakeEmbedder) EncodeBatch(ctx context.Context, texts []string) ([][]float32, error) {
	_ = ctx
	vecs := make([][]float32, len(texts))
	for i := range texts {
		vecs[i] = append([]float32(nil), e.vec...)
	}
	return vecs, nil
}

func (e fakeEmbedder) Model() string { return e.model }
func (e fakeEmbedder) Dim() int      { return len(e.vec) }

type fakeLLM struct {
	responses []string
	requests  []llm.ChatRequest
	calls     int
}

func (f *fakeLLM) ChatComplete(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
	_ = ctx
	f.requests = append(f.requests, req)
	if f.calls >= len(f.responses) {
		return llm.ChatResponse{}, errors.New("no fake response configured")
	}
	response := f.responses[f.calls]
	f.calls++
	return llm.ChatResponse{Content: response, Model: "fake"}, nil
}

func (f *fakeLLM) Model() string { return "fake" }

func interpretationJSON() string {
	return `{
		"mood":"warm modern jazz",
		"eraHints":["2010s"],
		"sonicQualities":["acoustic", "textural"],
		"referenceArtists":[],
		"avoid":[],
		"discoveryAppetite":"medium"
	}`
}

func curationJSON() string {
	return `{
		"picks":[
			{"rank":1,"albumId":"album-1","whyItFits":"Warm and textural.","whatToListenFor":"Listen for the brushed drums."},
			{"rank":2,"albumId":"album-2","whyItFits":"Modern but relaxed.","whatToListenFor":"Notice the piano voicings."},
			{"rank":3,"albumId":"album-3","whyItFits":"Acoustic and spacious.","whatToListenFor":"Follow the bass movement."},
			{"rank":4,"albumId":"album-4","whyItFits":"Softly exploratory.","whatToListenFor":"Hear the horn phrasing."},
			{"rank":5,"albumId":"album-5","whyItFits":"Adjacent and approachable.","whatToListenFor":"Focus on the cymbal texture."}
		]
	}`
}
