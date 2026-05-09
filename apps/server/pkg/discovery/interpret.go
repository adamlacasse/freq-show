package discovery

import (
	"context"
	"fmt"
	"strings"

	"github.com/adamlacasse/freq-show/apps/server/pkg/data"
	"github.com/adamlacasse/freq-show/apps/server/pkg/sources/llm"
)

const (
	InterpretationTemperature = 0.3
	InterpretationMaxTokens   = 400
)

func interpretQuery(ctx context.Context, client llm.ChatCompleter, raw string, alreadyKnown []string) (data.InterpretedQuery, error) {
	userPrompt, err := renderInterpretUserPrompt(raw, alreadyKnown)
	if err != nil {
		return data.InterpretedQuery{}, err
	}
	return interpretWithRetry(ctx, client, userPrompt, "")
}

func interpretWithRetry(ctx context.Context, client llm.ChatCompleter, userPrompt, previousError string) (data.InterpretedQuery, error) {
	if previousError != "" {
		userPrompt += "\n\n" + invalidJSONReminder + "\nValidation error: " + previousError
	}
	resp, err := client.ChatComplete(ctx, llm.ChatRequest{
		SystemPrompt: interpretSystemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  InterpretationTemperature,
		MaxTokens:    InterpretationMaxTokens,
	})
	if err != nil {
		return data.InterpretedQuery{}, fmt.Errorf("discovery: interpret query: %w", err)
	}

	parsed, err := parseInterpretation(resp.Content)
	if err != nil {
		if previousError != "" {
			return data.InterpretedQuery{}, err
		}
		return interpretWithRetry(ctx, client, userPrompt, err.Error())
	}
	return parsed, nil
}

func parseInterpretation(raw string) (data.InterpretedQuery, error) {
	var parsed data.InterpretedQuery
	fields, err := decodeJSONObject(raw, &parsed)
	if err != nil {
		return data.InterpretedQuery{}, err
	}
	if err := requireKeys(fields, "mood", "eraHints", "sonicQualities", "referenceArtists", "avoid", "discoveryAppetite"); err != nil {
		return data.InterpretedQuery{}, err
	}
	if err := validateInterpretation(parsed); err != nil {
		return data.InterpretedQuery{}, err
	}
	return parsed, nil
}

func validateInterpretation(parsed data.InterpretedQuery) error {
	if strings.TrimSpace(parsed.Mood) == "" {
		return fmt.Errorf("discovery: interpretation mood required")
	}
	switch parsed.DiscoveryAppetite {
	case "low", "medium", "high":
		return nil
	default:
		return fmt.Errorf("discovery: invalid discovery appetite %q", parsed.DiscoveryAppetite)
	}
}
