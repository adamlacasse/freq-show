package reviews

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/adamlacasse/freq-show/apps/server/pkg/data"
)

var (
	ErrNotFound     = errors.New("review not found")
	ErrRateLimit    = errors.New("rate limit exceeded")
	ErrUnauthorized = errors.New("unauthorized access")
)

// Client manages review fetching from multiple sources
type Client struct {
	httpClient *http.Client
	userAgent  string
	discogs    *DiscogsClient
}

// Config holds configuration for review sources
type Config struct {
	UserAgent             string
	Timeout               time.Duration
	DiscogsToken          string // Optional: for higher rate limits with personal token
	DiscogsConsumerKey    string // OAuth consumer key
	DiscogsConsumerSecret string // OAuth consumer secret
}

// NewClient creates a new review aggregation client
func NewClient(cfg Config) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}

	if cfg.UserAgent == "" {
		cfg.UserAgent = "FreqShow/1.0 +https://github.com/adamlacasse/freq-show"
	}

	httpClient := &http.Client{
		Timeout: cfg.Timeout,
	}

	return &Client{
		httpClient: httpClient,
		userAgent:  cfg.UserAgent,
		discogs: &DiscogsClient{
			httpClient:     httpClient,
			userAgent:      cfg.UserAgent,
			token:          cfg.DiscogsToken,
			consumerKey:    cfg.DiscogsConsumerKey,
			consumerSecret: cfg.DiscogsConsumerSecret,
		},
	}
}

// GetAlbumReview fetches and aggregates reviews for an album
// It tries multiple sources and returns the best available review
func (c *Client) GetAlbumReview(ctx context.Context, artistName, albumTitle string) (*data.Review, error) {
	// Try Discogs first (most comprehensive)
	if review, err := c.discogs.GetAlbumReview(ctx, artistName, albumTitle); err == nil && review != nil {
		return review, nil
	}

	// Future: Add other sources here
	// - RateYourMusic (if API becomes available)
	// - AI-generated summaries from AllMusic-style data
	// - MusicBrainz external review links

	// Return empty review if no sources found anything
	return &data.Review{}, nil
}

// DiscogsClient handles Discogs API interactions
type DiscogsClient struct {
	httpClient     *http.Client
	userAgent      string
	token          string
	consumerKey    string
	consumerSecret string
	baseURL        string
}

// DiscogsRelease represents a Discogs release response
type DiscogsRelease struct {
	ID           int                  `json:"id"`
	Title        string               `json:"title"`
	Artists      []DiscogsArtist      `json:"artists"`
	Community    DiscogsCommunityStat `json:"community"`
	Notes        string               `json:"notes"`
	ExtraArtists []DiscogsArtist      `json:"extraartists"`
}

type DiscogsArtist struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

type DiscogsCommunityStat struct {
	Have        int           `json:"have"`
	Want        int           `json:"want"`
	Rating      DiscogsRating `json:"rating"`
	DataQuality string        `json:"data_quality"`
}

type DiscogsRating struct {
	Count   int     `json:"count"`
	Average float64 `json:"average"`
}

type DiscogsDataPoint struct {
	Votes int `json:"votes"`
}

type DiscogsSearchResult struct {
	Results []DiscogsSearchItem `json:"results"`
}

type DiscogsSearchItem struct {
	ID          int                  `json:"id"`
	Type        string               `json:"type"`
	Title       string               `json:"title"`
	MasterID    int                  `json:"master_id"`
	MasterURL   string               `json:"master_url"`
	ResourceURL string               `json:"resource_url"`
	Thumb       string               `json:"thumb"`
	CoverImage  string               `json:"cover_image"`
	Genre       []string             `json:"genre"`
	Style       []string             `json:"style"`
	Country     string               `json:"country"`
	Year        string               `json:"year"`
	Label       []string             `json:"label"`
	Community   DiscogsCommunityStat `json:"community"`
}

func (dc *DiscogsClient) init() {
	if dc.baseURL == "" {
		dc.baseURL = "https://api.discogs.com"
	}
}

// setAuthHeaders sets the appropriate authentication headers for Discogs API requests
// Supports personal token authentication
func (dc *DiscogsClient) setAuthHeaders(req *http.Request) {
	req.Header.Set("User-Agent", dc.userAgent)

	// Use personal token if available
	if dc.token != "" {
		req.Header.Set("Authorization", "Discogs token="+dc.token)
	}
}

// buildAuthURL constructs a URL with authentication parameters
// For OAuth consumer key/secret, adds them as query parameters
func (dc *DiscogsClient) buildAuthURL(baseURL string, params map[string]string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}

	q := u.Query()

	// Add all provided parameters
	for key, value := range params {
		q.Set(key, value)
	}

	// Add OAuth consumer key/secret as query parameters if available (and no token)
	if dc.token == "" && dc.consumerKey != "" && dc.consumerSecret != "" {
		q.Set("key", dc.consumerKey)
		q.Set("secret", dc.consumerSecret)
	}

	u.RawQuery = q.Encode()
	return u.String()
}

// GetAlbumReview searches for and retrieves review data from Discogs
func (dc *DiscogsClient) GetAlbumReview(ctx context.Context, artistName, albumTitle string) (*data.Review, error) {
	dc.init()

	// First, search for the album
	searchResults, err := dc.searchAlbum(ctx, artistName, albumTitle)
	if err != nil {
		return nil, err
	}

	if len(searchResults) == 0 {
		return nil, ErrNotFound
	}

	// Get the first/best match
	bestMatch := searchResults[0]

	// Fetch detailed release information
	release, err := dc.getRelease(ctx, bestMatch.ID)
	if err != nil {
		return nil, err
	}

	// Convert to our Review format
	review := dc.convertToReview(release)
	return review, nil
}

func (dc *DiscogsClient) searchAlbum(ctx context.Context, artistName, albumTitle string) ([]DiscogsSearchItem, error) {
	// Build search query - simple space-separated format works better with Discogs
	query := fmt.Sprintf("%s %s", artistName, albumTitle)

	// Build URL with auth parameters if using OAuth consumer key/secret
	searchURL := dc.buildAuthURL(fmt.Sprintf("%s/database/search", dc.baseURL), map[string]string{
		"q":        query,
		"type":     "release",
		"per_page": "5",
	})

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	dc.setAuthHeaders(req)

	resp, err := dc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// Continue processing
	case http.StatusNotFound:
		return nil, ErrNotFound
	case http.StatusTooManyRequests:
		return nil, ErrRateLimit
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	default:
		return nil, fmt.Errorf("discogs api error: %d", resp.StatusCode)
	}

	var result DiscogsSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	return result.Results, nil
}

func (dc *DiscogsClient) getRelease(ctx context.Context, releaseID int) (*DiscogsRelease, error) {
	releaseURL := dc.buildAuthURL(fmt.Sprintf("%s/releases/%d", dc.baseURL, releaseID), map[string]string{})

	req, err := http.NewRequestWithContext(ctx, "GET", releaseURL, nil)
	if err != nil {
		return nil, err
	}

	dc.setAuthHeaders(req)

	resp, err := dc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// Continue processing
	case http.StatusNotFound:
		return nil, ErrNotFound
	case http.StatusTooManyRequests:
		return nil, ErrRateLimit
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	default:
		return nil, fmt.Errorf("discogs api error: %d", resp.StatusCode)
	}

	var release DiscogsRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode release response: %w", err)
	}

	return &release, nil
}

func (dc *DiscogsClient) convertToReview(release *DiscogsRelease) *data.Review {
	review := &data.Review{
		Source: "Discogs",
		URL:    fmt.Sprintf("https://www.discogs.com/release/%d", release.ID),
	}

	// Use community rating if available
	if release.Community.Rating.Count > 0 {
		review.Rating = release.Community.Rating.Average
		review.Summary = fmt.Sprintf("Community rating based on %d user ratings", release.Community.Rating.Count)
	}

	// Use release notes as review text if available
	if release.Notes != "" {
		review.Text = release.Notes
		review.Author = "Community"
	}

	// If we have very limited data, provide a basic summary
	if review.Summary == "" && review.Text == "" {
		if release.Community.Have > 0 || release.Community.Want > 0 {
			review.Summary = fmt.Sprintf("Collected by %d users, wanted by %d users",
				release.Community.Have, release.Community.Want)
		}
	}

	return review
}
