package musicbrainz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ErrNotFound indicates the requested resource was not present in MusicBrainz.
var ErrNotFound = errors.New("musicbrainz: resource not found")

const (
	errRequestBuildFailed = "musicbrainz: request build failed: %w"
	errRequestFailed      = "musicbrainz: request failed: %w"
	errDecodeFailed       = "musicbrainz: decode failed: %w"
	errUnexpectedStatus   = "musicbrainz: unexpected status %d: %s"
	headerUserAgent       = "User-Agent"
	headerAccept          = "Accept"
	contentTypeJSON       = "application/json"
)

// Config describes how to connect to the MusicBrainz API.
type Config struct {
	BaseURL    string
	AppName    string
	AppVersion string
	Contact    string
	Timeout    time.Duration
}

// Client issues requests against the MusicBrainz API.
type Client struct {
	baseURL    string
	userAgent  string
	httpClient *http.Client
}

// New constructs a MusicBrainz API client using the supplied configuration.
func New(_ context.Context, cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("musicbrainz: base URL is required")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}

	contact := strings.TrimSpace(cfg.Contact)
	if contact == "" {
		return nil, errors.New("musicbrainz: contact information is required")
	}

	name := strings.TrimSpace(cfg.AppName)
	if name == "" {
		name = "freq-show"
	}
	version := strings.TrimSpace(cfg.AppVersion)
	if version == "" {
		version = "dev"
	}

	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if _, err := url.Parse(baseURL); err != nil {
		return nil, fmt.Errorf("musicbrainz: invalid base URL %q: %w", cfg.BaseURL, err)
	}

	userAgent := fmt.Sprintf("%s/%s (%s)", name, version, contact)

	return &Client{
		baseURL:   baseURL,
		userAgent: userAgent,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}, nil
}

// Artist models a subset of the MusicBrainz artist payload.
type Artist struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Country        string   `json:"country,omitempty"`
	Type           string   `json:"type,omitempty"`
	Disambiguation string   `json:"disambiguation,omitempty"`
	Aliases        []string `json:"aliases,omitempty"`
	LifeSpan       LifeSpan `json:"lifeSpan"`
}

// ReleaseGroup models an album (release group) payload from MusicBrainz.
type ReleaseGroup struct {
	ID               string         `json:"id"`
	Title            string         `json:"title"`
	PrimaryType      string         `json:"primaryType"`
	SecondaryTypes   []string       `json:"secondaryTypes"`
	FirstReleaseDate string         `json:"firstReleaseDate"`
	ArtistCredit     []ArtistCredit `json:"artistCredit"`
}

// ArtistCredit represents a contributing artist on a release group.
type ArtistCredit struct {
	Name   string             `json:"name"`
	Artist ReleaseGroupArtist `json:"artist"`
}

// ReleaseGroupArtist represents artist details within a credit block.
type ReleaseGroupArtist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// LifeSpan represents the active period of an artist.
type LifeSpan struct {
	Begin string `json:"begin,omitempty"`
	End   string `json:"end,omitempty"`
	Ended bool   `json:"ended,omitempty"`
}

type artistResponse struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Country        string `json:"country"`
	Type           string `json:"type"`
	Disambiguation string `json:"disambiguation"`
	Aliases        []struct {
		Name string `json:"name"`
	} `json:"aliases"`
	LifeSpan LifeSpan `json:"life-span"`
}

type releaseGroupResponse struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	PrimaryType      string   `json:"primary-type"`
	SecondaryTypes   []string `json:"secondary-types"`
	FirstReleaseDate string   `json:"first-release-date"`
	ArtistCredit     []struct {
		Name   string `json:"name"`
		Artist struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"artist"`
	} `json:"artist-credit"`
}

// LookupArtist retrieves a single artist record by MusicBrainz ID.
func (c *Client) LookupArtist(ctx context.Context, id string) (*Artist, error) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return nil, errors.New("musicbrainz: artist id is required")
	}

	endpoint := fmt.Sprintf("%s/artist/%s?fmt=json", c.baseURL, url.PathEscape(trimmed))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf(errRequestBuildFailed, err)
	}
	req.Header.Set(headerUserAgent, c.userAgent)
	req.Header.Set(headerAccept, contentTypeJSON)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(errRequestFailed, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var payload artistResponse
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			return nil, fmt.Errorf(errDecodeFailed, err)
		}
		return transformArtist(payload), nil
	case http.StatusNotFound:
		return nil, ErrNotFound
	default:
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf(errUnexpectedStatus, resp.StatusCode, strings.TrimSpace(string(snippet)))
	}
}

func transformArtist(payload artistResponse) *Artist {
	aliases := make([]string, 0, len(payload.Aliases))
	for _, alias := range payload.Aliases {
		if alias.Name != "" {
			aliases = append(aliases, alias.Name)
		}
	}

	return &Artist{
		ID:             payload.ID,
		Name:           payload.Name,
		Country:        payload.Country,
		Type:           payload.Type,
		Disambiguation: payload.Disambiguation,
		Aliases:        aliases,
		LifeSpan:       payload.LifeSpan,
	}
}

// LookupReleaseGroup retrieves an album (release group) by ID.
func (c *Client) LookupReleaseGroup(ctx context.Context, id string) (*ReleaseGroup, error) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return nil, errors.New("musicbrainz: release group id is required")
	}

	endpoint := fmt.Sprintf("%s/release-group/%s?fmt=json&inc=artists", c.baseURL, url.PathEscape(trimmed))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf(errRequestBuildFailed, err)
	}
	req.Header.Set(headerUserAgent, c.userAgent)
	req.Header.Set(headerAccept, contentTypeJSON)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(errRequestFailed, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var payload releaseGroupResponse
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			return nil, fmt.Errorf(errDecodeFailed, err)
		}
		return transformReleaseGroup(payload), nil
	case http.StatusNotFound:
		return nil, ErrNotFound
	default:
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf(errUnexpectedStatus, resp.StatusCode, strings.TrimSpace(string(snippet)))
	}
}

func transformReleaseGroup(payload releaseGroupResponse) *ReleaseGroup {
	credits := make([]ArtistCredit, 0, len(payload.ArtistCredit))
	for _, credit := range payload.ArtistCredit {
		credits = append(credits, ArtistCredit{
			Name: credit.Name,
			Artist: ReleaseGroupArtist{
				ID:   credit.Artist.ID,
				Name: credit.Artist.Name,
			},
		})
	}

	return &ReleaseGroup{
		ID:               payload.ID,
		Title:            payload.Title,
		PrimaryType:      payload.PrimaryType,
		SecondaryTypes:   append([]string(nil), payload.SecondaryTypes...),
		FirstReleaseDate: payload.FirstReleaseDate,
		ArtistCredit:     credits,
	}
}

// PrimaryArtistID returns the ID of the first credited artist, if present.
func (r *ReleaseGroup) PrimaryArtistID() string {
	for _, credit := range r.ArtistCredit {
		if credit.Artist.ID != "" {
			return credit.Artist.ID
		}
	}
	return ""
}

// PrimaryArtistName returns the display name of the first credited artist, if present.
func (r *ReleaseGroup) PrimaryArtistName() string {
	for _, credit := range r.ArtistCredit {
		if credit.Artist.Name != "" {
			return credit.Artist.Name
		}
		if credit.Name != "" {
			return credit.Name
		}
	}
	return ""
}

// ReleaseYear attempts to parse the release year from the first release date.
func (r *ReleaseGroup) ReleaseYear() int {
	if len(r.FirstReleaseDate) < 4 {
		return 0
	}
	year, err := strconv.Atoi(r.FirstReleaseDate[:4])
	if err != nil {
		return 0
	}
	return year
}

// SearchResult represents a search result container from MusicBrainz.
type SearchResult struct {
	Artists []Artist `json:"artists"`
	Offset  int      `json:"offset"`
	Count   int      `json:"count"`
}

type searchResponse struct {
	Artists []struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		Country        string `json:"country"`
		Type           string `json:"type"`
		Disambiguation string `json:"disambiguation"`
		Aliases        []struct {
			Name string `json:"name"`
		} `json:"aliases"`
		LifeSpan LifeSpan `json:"life-span"`
		Score    int      `json:"score"`
	} `json:"artists"`
	Offset int `json:"offset"`
	Count  int `json:"count"`
}

// SearchArtists searches for artists by name or other criteria.
func (c *Client) SearchArtists(ctx context.Context, query string, limit int, offset int) (*SearchResult, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return nil, errors.New("musicbrainz: search query is required")
	}

	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	params := url.Values{}
	params.Set("query", trimmed)
	params.Set("fmt", "json")
	params.Set("limit", strconv.Itoa(limit))
	params.Set("offset", strconv.Itoa(offset))

	endpoint := fmt.Sprintf("%s/artist/?%s", c.baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf(errRequestBuildFailed, err)
	}
	req.Header.Set(headerUserAgent, c.userAgent)
	req.Header.Set(headerAccept, contentTypeJSON)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(errRequestFailed, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var payload searchResponse
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			return nil, fmt.Errorf(errDecodeFailed, err)
		}
		return transformSearchResult(payload), nil
	default:
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf(errUnexpectedStatus, resp.StatusCode, strings.TrimSpace(string(snippet)))
	}
}

func transformSearchResult(payload searchResponse) *SearchResult {
	artists := make([]Artist, 0, len(payload.Artists))
	for _, item := range payload.Artists {
		aliases := make([]string, 0, len(item.Aliases))
		for _, alias := range item.Aliases {
			if alias.Name != "" {
				aliases = append(aliases, alias.Name)
			}
		}

		artists = append(artists, Artist{
			ID:             item.ID,
			Name:           item.Name,
			Country:        item.Country,
			Type:           item.Type,
			Disambiguation: item.Disambiguation,
			Aliases:        aliases,
			LifeSpan:       item.LifeSpan,
		})
	}

	return &SearchResult{
		Artists: artists,
		Offset:  payload.Offset,
		Count:   payload.Count,
	}
}

// ReleaseGroupSearchResult represents the response from a release group search for an artist.
type ReleaseGroupSearchResult struct {
	ReleaseGroups []ReleaseGroup `json:"release-groups"`
	Count         int            `json:"release-group-count"`
	Offset        int            `json:"release-group-offset"`
}

type releaseGroupSearchResponse struct {
	ReleaseGroups []struct {
		ID               string   `json:"id"`
		Title            string   `json:"title"`
		PrimaryType      string   `json:"primary-type"`
		SecondaryTypes   []string `json:"secondary-types"`
		FirstReleaseDate string   `json:"first-release-date"`
	} `json:"release-groups"`
	Count  int `json:"release-group-count"`
	Offset int `json:"release-group-offset"`
}

// GetArtistReleaseGroups retrieves the release groups (albums) for a given artist.
func (c *Client) GetArtistReleaseGroups(ctx context.Context, artistID string, limit int, offset int) (*ReleaseGroupSearchResult, error) {
	trimmed := strings.TrimSpace(artistID)
	if trimmed == "" {
		return nil, errors.New("musicbrainz: artist id is required")
	}

	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	params := url.Values{}
	params.Set("fmt", "json")
	params.Set("limit", strconv.Itoa(limit))
	params.Set("offset", strconv.Itoa(offset))
	params.Set("type", "album|ep") // Focus on main releases

	endpoint := fmt.Sprintf("%s/release-group?artist=%s&%s", c.baseURL, url.QueryEscape(trimmed), params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf(errRequestBuildFailed, err)
	}
	req.Header.Set(headerUserAgent, c.userAgent)
	req.Header.Set(headerAccept, contentTypeJSON)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(errRequestFailed, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var payload releaseGroupSearchResponse
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			return nil, fmt.Errorf(errDecodeFailed, err)
		}
		return transformReleaseGroupSearchResult(payload, artistID), nil
	default:
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf(errUnexpectedStatus, resp.StatusCode, strings.TrimSpace(string(snippet)))
	}
}

func transformReleaseGroupSearchResult(payload releaseGroupSearchResponse, artistID string) *ReleaseGroupSearchResult {
	releaseGroups := make([]ReleaseGroup, 0, len(payload.ReleaseGroups))
	for _, item := range payload.ReleaseGroups {
		// Create a basic artist credit for the known artist
		artistCredit := []ArtistCredit{
			{
				Name: "", // We don't have the artist name in this response
				Artist: ReleaseGroupArtist{
					ID:   artistID,
					Name: "",
				},
			},
		}

		releaseGroups = append(releaseGroups, ReleaseGroup{
			ID:               item.ID,
			Title:            item.Title,
			PrimaryType:      item.PrimaryType,
			SecondaryTypes:   append([]string(nil), item.SecondaryTypes...),
			FirstReleaseDate: item.FirstReleaseDate,
			ArtistCredit:     artistCredit,
		})
	}

	return &ReleaseGroupSearchResult{
		ReleaseGroups: releaseGroups,
		Count:         payload.Count,
		Offset:        payload.Offset,
	}
}
