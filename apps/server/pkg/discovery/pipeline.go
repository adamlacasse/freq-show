package discovery

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/adamlacasse/freq-show/apps/server/pkg/data"
	"github.com/adamlacasse/freq-show/apps/server/pkg/db"
	"github.com/adamlacasse/freq-show/apps/server/pkg/sources/embeddings"
	"github.com/adamlacasse/freq-show/apps/server/pkg/sources/llm"
)

var (
	ErrNoEmbeddedAlbums = errors.New("discovery: no embedded albums available")
	ErrUnconfigured     = errors.New("discovery: service is not configured")
)

type Service struct {
	Embedder   embeddings.Embedder
	LLM        llm.ChatCompleter
	Embeddings db.EmbeddingRepository
	Albums     db.AlbumRepository
}

func (s *Service) Run(ctx context.Context, q data.DiscoveryQuery) (data.DiscoveryResult, error) {
	if err := s.validate(); err != nil {
		return data.DiscoveryResult{}, err
	}
	query := strings.TrimSpace(q.Query)
	if query == "" {
		return data.DiscoveryResult{}, errors.New("discovery: query required")
	}

	interpreted, err := interpretQuery(ctx, s.LLM, query, q.AlreadyKnown)
	if err != nil {
		return data.DiscoveryResult{}, err
	}

	queryText := buildQueryEmbeddingText(interpreted)
	queryVec, err := s.Embedder.Encode(ctx, queryText)
	if err != nil {
		return data.DiscoveryResult{}, fmt.Errorf("discovery: embed interpreted query: %w", err)
	}

	records, err := s.Embeddings.LoadAllForModel(ctx, s.Embedder.Model())
	if err != nil {
		return data.DiscoveryResult{}, fmt.Errorf("discovery: load album embeddings: %w", err)
	}
	if len(records) == 0 {
		return data.DiscoveryResult{}, ErrNoEmbeddedAlbums
	}

	candidates, err := retrieveCandidates(queryVec, records, interpreted, q.AlreadyKnown, func(mbid string) (*data.Album, error) {
		return s.Albums.GetAlbum(ctx, mbid)
	})
	if err != nil {
		return data.DiscoveryResult{}, err
	}
	if len(candidates) == 0 {
		return data.DiscoveryResult{Interpreted: interpreted, Picks: []data.Pick{}}, nil
	}

	picks, err := curate(ctx, s.LLM, interpreted, candidates)
	if err != nil {
		return data.DiscoveryResult{}, err
	}
	return data.DiscoveryResult{Interpreted: interpreted, Picks: picks}, nil
}

func (s *Service) validate() error {
	if s == nil || s.Embedder == nil || s.LLM == nil || s.Embeddings == nil || s.Albums == nil {
		return ErrUnconfigured
	}
	return nil
}
