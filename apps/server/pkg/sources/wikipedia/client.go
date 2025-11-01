package wikipedia

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// ErrNotFound indicates the requested Wikipedia page was not found.
var ErrNotFound = errors.New("wikipedia: page not found")

// Config describes how to connect to the Wikipedia API.
type Config struct {
	BaseURL   string
	UserAgent string
	Timeout   time.Duration
}

// Client issues requests against the Wikipedia API.
type Client struct {
	baseURL    string
	userAgent  string
	httpClient *http.Client
}

// New constructs a Wikipedia API client.
func New(_ context.Context, cfg Config) (*Client, error) {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = "https://en.wikipedia.org/api/rest_v1"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	userAgent := strings.TrimSpace(cfg.UserAgent)
	if userAgent == "" {
		userAgent = "FreqShow/1.0 (https://github.com/adamlacasse/freq-show)"
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	return &Client{
		baseURL:   baseURL,
		userAgent: userAgent,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// Summary represents a Wikipedia page summary.
type Summary struct {
	Title   string `json:"title"`
	Extract string `json:"extract"`
	Type    string `json:"type"`
}

type summaryResponse struct {
	Type         string `json:"type"`
	Title        string `json:"title"`
	Displaytitle string `json:"displaytitle"`
	Extract      string `json:"extract"`
	ExtractHTML  string `json:"extract_html"`
}

// GetArtistBiography attempts to fetch a biography for an artist by searching Wikipedia.
func (c *Client) GetArtistBiography(ctx context.Context, artistName string) (string, error) {
	if strings.TrimSpace(artistName) == "" {
		return "", errors.New("wikipedia: artist name is required")
	}

	// First, try to get the page summary directly
	summary, err := c.getPageSummary(ctx, artistName)
	if err == nil && summary.Extract != "" {
		return c.cleanExtract(summary.Extract), nil
	}

	// If direct lookup fails, try with "band" suffix for groups
	if err == ErrNotFound {
		bandName := artistName + " (band)"
		summary, err = c.getPageSummary(ctx, bandName)
		if err == nil && summary.Extract != "" {
			return c.cleanExtract(summary.Extract), nil
		}
	}

	// Try with "musician" suffix
	if err == ErrNotFound {
		musicianName := artistName + " (musician)"
		summary, err = c.getPageSummary(ctx, musicianName)
		if err == nil && summary.Extract != "" {
			return c.cleanExtract(summary.Extract), nil
		}
	}

	// Try with "singer" suffix
	if err == ErrNotFound {
		singerName := artistName + " (singer)"
		summary, err = c.getPageSummary(ctx, singerName)
		if err == nil && summary.Extract != "" {
			return c.cleanExtract(summary.Extract), nil
		}
	}

	return "", ErrNotFound
}

func (c *Client) getPageSummary(ctx context.Context, title string) (*Summary, error) {
	encodedTitle := url.PathEscape(title)
	endpoint := fmt.Sprintf("%s/page/summary/%s", c.baseURL, encodedTitle)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("wikipedia: request build failed: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("wikipedia: request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var payload summaryResponse
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			return nil, fmt.Errorf("wikipedia: decode failed: %w", err)
		}

		// Check if this is a disambiguation page or has no useful content
		if payload.Type == "disambiguation" || strings.Contains(strings.ToLower(payload.Extract), "may refer to") {
			return nil, ErrNotFound
		}

		return &Summary{
			Title:   payload.Title,
			Extract: payload.Extract,
			Type:    payload.Type,
		}, nil
	case http.StatusNotFound:
		return nil, ErrNotFound
	default:
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("wikipedia: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(snippet)))
	}
}

// cleanExtract processes the Wikipedia extract to make it more suitable for display.
func (c *Client) cleanExtract(extract string) string {
	if extract == "" {
		return ""
	}

	// Remove common Wikipedia artifacts
	cleaned := extract

	// Remove pronunciation guides in parentheses at the start
	pronounceRegex := regexp.MustCompile(`^[^(]*\([^)]*pronunciation[^)]*\)\s*`)
	cleaned = pronounceRegex.ReplaceAllString(cleaned, "")

	// Remove "listen" audio links
	listenRegex := regexp.MustCompile(`\s*\([^)]*listen[^)]*\)\s*`)
	cleaned = listenRegex.ReplaceAllString(cleaned, " ")

	// Clean up multiple spaces
	spaceRegex := regexp.MustCompile(`\s+`)
	cleaned = spaceRegex.ReplaceAllString(cleaned, " ")

	// Trim whitespace
	cleaned = strings.TrimSpace(cleaned)

	// Limit length to first 3 sentences or 500 characters, whichever comes first
	sentences := strings.Split(cleaned, ". ")
	if len(sentences) > 3 {
		cleaned = strings.Join(sentences[:3], ". ") + "."
	}

	if len(cleaned) > 500 {
		// Find the last sentence that fits within 500 characters
		parts := strings.Split(cleaned, ". ")
		result := ""
		for _, part := range parts {
			if len(result+part+". ") <= 500 {
				if result == "" {
					result = part
				} else {
					result += ". " + part
				}
			} else {
				break
			}
		}
		if result != "" && !strings.HasSuffix(result, ".") {
			result += "."
		}
		cleaned = result
	}

	return cleaned
}
