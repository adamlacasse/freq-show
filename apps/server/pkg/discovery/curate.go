package discovery

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/adamlacasse/freq-show/apps/server/pkg/data"
	"github.com/adamlacasse/freq-show/apps/server/pkg/sources/llm"
)

const (
	FinalPicks          = 5
	CurationTemperature = 0.7
	CurationMaxTokens   = 900
)

func curate(ctx context.Context, client llm.ChatCompleter, interpreted data.InterpretedQuery, candidates []*data.Album) ([]data.Pick, error) {
	if len(candidates) == 0 {
		return []data.Pick{}, nil
	}
	userPrompt, err := renderCurateUserPrompt(interpreted, candidates)
	if err != nil {
		return nil, err
	}
	return curateWithRetry(ctx, client, userPrompt, candidates, "")
}

func curateWithRetry(ctx context.Context, client llm.ChatCompleter, userPrompt string, candidates []*data.Album, previousError string) ([]data.Pick, error) {
	if previousError != "" {
		userPrompt += "\n\n" + invalidJSONReminder + "\nValidation error: " + previousError
	}
	resp, err := client.ChatComplete(ctx, llm.ChatRequest{
		SystemPrompt: curateSystemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  CurationTemperature,
		MaxTokens:    CurationMaxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("discovery: curate candidates: %w", err)
	}

	picks, err := parseCuration(resp.Content, candidates)
	if err != nil {
		if previousError != "" {
			return nil, err
		}
		return curateWithRetry(ctx, client, userPrompt, candidates, err.Error())
	}
	return picks, nil
}

type curationResponse struct {
	Picks []curationPick `json:"picks"`
}

type curationPick struct {
	Rank            int    `json:"rank"`
	AlbumID         string `json:"albumId"`
	WhyItFits       string `json:"whyItFits"`
	WhatToListenFor string `json:"whatToListenFor"`
}

func parseCuration(raw string, candidates []*data.Album) ([]data.Pick, error) {
	var parsed curationResponse
	fields, err := decodeJSONObject(raw, &parsed)
	if err != nil {
		return nil, err
	}
	if err := requireKeys(fields, "picks"); err != nil {
		return nil, err
	}
	if err := validateCuration(parsed.Picks, candidates); err != nil {
		return nil, err
	}
	return enrichPicks(parsed.Picks, candidates), nil
}

func validateCuration(picks []curationPick, candidates []*data.Album) error {
	expected := pickCount(candidates)
	if len(picks) != expected {
		return fmt.Errorf("discovery: expected %d picks, got %d", expected, len(picks))
	}
	knownAlbums := make(map[string]struct{}, len(candidates))
	for _, album := range candidates {
		knownAlbums[album.ID] = struct{}{}
	}
	seenRanks := make(map[int]struct{}, FinalPicks)
	for _, pick := range picks {
		if pick.Rank < 1 || pick.Rank > expected {
			return fmt.Errorf("discovery: invalid pick rank %d", pick.Rank)
		}
		if _, ok := seenRanks[pick.Rank]; ok {
			return fmt.Errorf("discovery: duplicate pick rank %d", pick.Rank)
		}
		seenRanks[pick.Rank] = struct{}{}
		if _, ok := knownAlbums[pick.AlbumID]; !ok {
			return fmt.Errorf("discovery: pick albumId %q was not a candidate", pick.AlbumID)
		}
		if strings.TrimSpace(pick.WhyItFits) == "" {
			return fmt.Errorf("discovery: pick %d missing whyItFits", pick.Rank)
		}
		if strings.TrimSpace(pick.WhatToListenFor) == "" {
			return fmt.Errorf("discovery: pick %d missing whatToListenFor", pick.Rank)
		}
	}
	for rank := 1; rank <= expected; rank++ {
		if _, ok := seenRanks[rank]; !ok {
			return fmt.Errorf("discovery: missing rank %d", rank)
		}
	}
	return nil
}

func pickCount(candidates []*data.Album) int {
	if len(candidates) < FinalPicks {
		return len(candidates)
	}
	return FinalPicks
}

func enrichPicks(picks []curationPick, candidates []*data.Album) []data.Pick {
	byID := make(map[string]*data.Album, len(candidates))
	for _, album := range candidates {
		byID[album.ID] = album
	}
	sort.Slice(picks, func(i, j int) bool {
		return picks[i].Rank < picks[j].Rank
	})
	out := make([]data.Pick, 0, len(picks))
	for _, pick := range picks {
		album := byID[pick.AlbumID]
		out = append(out, data.Pick{
			Rank:             pick.Rank,
			AlbumID:          pick.AlbumID,
			Title:            album.Title,
			ArtistName:       album.ArtistName,
			Year:             album.Year,
			WhyItFits:        pick.WhyItFits,
			WhatToListenFor:  pick.WhatToListenFor,
			SpotifySearchURL: spotifySearchURL(album),
		})
	}
	return out
}
