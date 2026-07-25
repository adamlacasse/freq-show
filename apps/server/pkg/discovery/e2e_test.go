package discovery

// Real-key end-to-end smoke test for the discovery pipeline.
//
// Run with:
//
//	DISCOVERY_E2E=1 \
//	  DISCOVERY_EMBEDDINGS_PROVIDER=voyage \
//	  DISCOVERY_EMBEDDINGS_API_KEY=<key> \
//	  DISCOVERY_LLM_PROVIDER=huggingface \
//	  DISCOVERY_LLM_API_KEY=<key> \
//	  go test ./pkg/discovery/ -run TestDiscoveryE2E -v -timeout 120s
//
// The test seeds an in-memory store with 10 well-known albums, embeds them
// with the live embedding provider, then runs a natural-language query through
// the full pipeline and validates the response shape.

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/adamlacasse/freq-show/apps/server/pkg/data"
	"github.com/adamlacasse/freq-show/apps/server/pkg/db"
	"github.com/adamlacasse/freq-show/apps/server/pkg/sources/embeddings"
	"github.com/adamlacasse/freq-show/apps/server/pkg/sources/llm"
)

func TestDiscoveryE2E(t *testing.T) {
	if os.Getenv("DISCOVERY_E2E") != "1" {
		t.Skip("set DISCOVERY_E2E=1 to run real-key smoke test")
	}

	embProvider := os.Getenv("DISCOVERY_EMBEDDINGS_PROVIDER")
	embKey := os.Getenv("DISCOVERY_EMBEDDINGS_API_KEY")
	embModel := os.Getenv("DISCOVERY_EMBEDDINGS_MODEL")
	llmProvider := os.Getenv("DISCOVERY_LLM_PROVIDER")
	llmKey := os.Getenv("DISCOVERY_LLM_API_KEY")
	llmModel := os.Getenv("DISCOVERY_LLM_MODEL")

	if embProvider == "" {
		embProvider = "voyage"
	}
	if llmProvider == "" {
		llmProvider = "huggingface"
	}
	if embKey == "" {
		t.Fatal("DISCOVERY_EMBEDDINGS_API_KEY required")
	}
	if llmKey == "" {
		t.Fatal("DISCOVERY_LLM_API_KEY required")
	}

	embedder, err := embeddings.NewFromConfig(embeddings.Config{
		Provider: embProvider,
		APIKey:   embKey,
		Model:    embModel,
	})
	if err != nil {
		t.Fatalf("embedder init: %v", err)
	}

	chatClient, err := llm.NewFromConfig(llm.Config{
		Provider: llmProvider,
		APIKey:   llmKey,
		Model:    llmModel,
	})
	if err != nil {
		t.Fatalf("llm init: %v", err)
	}

	ctx := context.Background()
	store, err := db.NewMemoryStore(ctx)
	if err != nil {
		t.Fatalf("store init: %v", err)
	}

	// Seed 10 albums covering a range of styles so the pipeline has real
	// candidates. Album IDs are fake MBIDs for test purposes.
	seed := []data.Album{
		{
			ID: "seed-001", Title: "In Rainbows", ArtistName: "Radiohead", Year: 2007, Genre: "art rock",
			Review: data.Review{Source: "Pitchfork", Rating: 9.3, Summary: "A pivot toward warmth and emotional directness after years of electronic experiments. Inventive arrangements, intimate songwriting."},
			Tracks: []data.Track{{Title: "15 Step"}, {Title: "Bodysnatchers"}, {Title: "Nude"}, {Title: "Weird Fishes/Arpeggi"}, {Title: "All I Need"}, {Title: "Reckoner"}},
		},
		{
			ID: "seed-002", Title: "Kind of Blue", ArtistName: "Miles Davis", Year: 1959, Genre: "modal jazz",
			Review: data.Review{Source: "AllMusic", Rating: 5.0, Summary: "The best-selling jazz album of all time. Modal improvisation at its most serene and precise, a perfect entry point and a lifetime companion."},
			Tracks: []data.Track{{Title: "So What"}, {Title: "Freddie Freeloader"}, {Title: "Blue in Green"}, {Title: "All Blues"}, {Title: "Flamenco Sketches"}},
		},
		{
			ID: "seed-003", Title: "Blue", ArtistName: "Joni Mitchell", Year: 1971, Genre: "folk",
			Review: data.Review{Source: "Rolling Stone", Rating: 5.0, Summary: "Devastatingly intimate confessional songwriting. One of the most emotionally raw and lyrically vivid albums ever made."},
			Tracks: []data.Track{{Title: "All I Want"}, {Title: "My Old Man"}, {Title: "River"}, {Title: "A Case of You"}, {Title: "The Last Time I Saw Richard"}},
		},
		{
			ID: "seed-004", Title: "Illmatic", ArtistName: "Nas", Year: 1994, Genre: "hip-hop",
			Review: data.Review{Source: "Pitchfork", Rating: 10.0, Summary: "Dense, cinematic New York street poetry over jazz-inflected beats. A debut that set the standard for lyrical rap."},
			Tracks: []data.Track{{Title: "N.Y. State of Mind"}, {Title: "The World Is Yours"}, {Title: "One Love"}, {Title: "Memory Lane"}},
		},
		{
			ID: "seed-005", Title: "Vespertine", ArtistName: "Björk", Year: 2001, Genre: "art pop",
			Review: data.Review{Source: "Pitchfork", Rating: 9.4, Summary: "Microscopic and otherworldly. Harp, harpsichord, micro-beats, and choir weave a domestic interior world of crystalline precision."},
			Tracks: []data.Track{{Title: "Hidden Place"}, {Title: "Cocoon"}, {Title: "It's Not Up to You"}, {Title: "Undo"}, {Title: "Aurora"}},
		},
		{
			ID: "seed-006", Title: "Endtroducing.....", ArtistName: "DJ Shadow", Year: 1996, Genre: "instrumental hip-hop",
			Review: data.Review{Source: "AllMusic", Rating: 5.0, Summary: "A landmark of sample-based composition. Entirely constructed from vinyl, it moves from melancholy to menace across deep, dusty textures."},
			Tracks: []data.Track{{Title: "Building Steam with a Grain of Salt"}, {Title: "The Number Song"}, {Title: "Stem/Long Stem"}, {Title: "Mutual Slump"}, {Title: "Napalm Brain/Scatter Brain"}},
		},
		{
			ID: "seed-007", Title: "Careless Love", ArtistName: "Madeleine Peyroux", Year: 2004, Genre: "jazz",
			Review: data.Review{Source: "AllMusic", Rating: 4.0, Summary: "Warm, low-key, and charming. A Sunday-morning record par excellence: Peyroux's husky voice glides over acoustic jazz arrangements."},
			Tracks: []data.Track{{Title: "Dance Me to the End of Love"}, {Title: "Between the Bars"}, {Title: "Don't Wait Too Long"}, {Title: "Getting Some Fun Out of Life"}},
		},
		{
			ID: "seed-008", Title: "Neon Bible", ArtistName: "Arcade Fire", Year: 2007, Genre: "indie rock",
			Review: data.Review{Source: "NME", Rating: 9.0, Summary: "Epic and grandiose, preoccupied with mortality and American mythology. Orchestral arrangements swell under urgent, anthemic songwriting."},
			Tracks: []data.Track{{Title: "Black Mirror"}, {Title: "Keep the Car Running"}, {Title: "Intervention"}, {Title: "No Cars Go"}, {Title: "My Body Is a Cage"}},
		},
		{
			ID: "seed-009", Title: "Blackstar", ArtistName: "David Bowie", Year: 2016, Genre: "art rock",
			Review: data.Review{Source: "Pitchfork", Rating: 9.2, Summary: "A farewell that transcends the genre. Jazz-inflected, lyrically cryptic, and emotionally devastating. Bowie fusing avant-garde with pop one last time."},
			Tracks: []data.Track{{Title: "Blackstar"}, {Title: "Tis a Pity She Was a Whore"}, {Title: "Lazarus"}, {Title: "Girl Loves Me"}, {Title: "Dollar Days"}, {Title: "I Can't Give Everything Away"}},
		},
		{
			ID: "seed-010", Title: "The Miseducation of Lauryn Hill", ArtistName: "Lauryn Hill", Year: 1998, Genre: "neo soul",
			Review: data.Review{Source: "Rolling Stone", Rating: 5.0, Summary: "Effortlessly fuses soul, R&B, hip-hop, and reggae into a masterpiece of personal conviction. Warm production, towering vocals."},
			Tracks: []data.Track{{Title: "Lost Ones"}, {Title: "Ex-Factor"}, {Title: "To Zion"}, {Title: "Doo Wop (That Thing)"}, {Title: "Everything Is Everything"}},
		},
	}

	t.Log("seeding store and embedding albums...")
	for i := range seed {
		album := seed[i]
		if err := store.SaveAlbum(ctx, &album); err != nil {
			t.Fatalf("SaveAlbum %s: %v", album.ID, err)
		}
		text := BuildAlbumEmbeddingText(&album, nil)
		if text == "" {
			t.Fatalf("album %s produced empty embedding text", album.ID)
		}
		vec, err := embedder.Encode(ctx, text)
		if err != nil {
			t.Fatalf("Encode album %s: %v", album.ID, err)
		}
		if err := store.SaveEmbedding(ctx, album.ID, embedder.Model(), vec); err != nil {
			t.Fatalf("SaveEmbedding %s: %v", album.ID, err)
		}
		t.Logf("  embedded %s — %s (%d dims)", album.ArtistName, album.Title, len(vec))
	}

	svc := &Service{
		Embedder:   embedder,
		LLM:        chatClient,
		Embeddings: store,
		Albums:     store,
	}

	query := data.DiscoveryQuery{
		Query: "saturday morning coffee, jazzy but modern, nothing harsh",
	}
	t.Logf("running query: %q", query.Query)

	result, err := svc.Run(ctx, query)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	// Validate interpretation shape.
	interp := result.Interpreted
	if interp.Mood == "" {
		t.Error("interpreted mood is empty")
	}
	validAppetites := map[string]bool{"low": true, "medium": true, "high": true}
	if !validAppetites[interp.DiscoveryAppetite] {
		t.Errorf("unexpected discoveryAppetite %q", interp.DiscoveryAppetite)
	}
	t.Logf("interpreted: mood=%q appetite=%s sonicQualities=%v",
		interp.Mood, interp.DiscoveryAppetite, interp.SonicQualities)

	// Validate picks.
	if len(result.Picks) != 5 {
		t.Errorf("expected 5 picks, got %d", len(result.Picks))
	}
	for _, pick := range result.Picks {
		if pick.AlbumID == "" {
			t.Error("pick has empty albumId")
		}
		if strings.TrimSpace(pick.WhyItFits) == "" {
			t.Errorf("pick %d has empty whyItFits", pick.Rank)
		}
		if strings.TrimSpace(pick.WhatToListenFor) == "" {
			t.Errorf("pick %d has empty whatToListenFor", pick.Rank)
		}
		if pick.SpotifySearchURL == "" {
			t.Errorf("pick %d has empty spotifySearchUrl", pick.Rank)
		}
		t.Logf("  pick %d: %s — %s", pick.Rank, pick.ArtistName, pick.Title)
		t.Logf("    why: %s", pick.WhyItFits)
	}
}
