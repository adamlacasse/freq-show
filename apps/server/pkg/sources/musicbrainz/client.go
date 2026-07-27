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
	"sync"
	"time"
)

// ErrNotFound indicates the requested resource was not present in MusicBrainz.
var ErrNotFound = errors.New("musicbrainz: resource not found")

// ErrRateLimit indicates the MusicBrainz API has throttled the request.
var ErrRateLimit = errors.New("musicbrainz: rate limit exceeded")

const (
	errRequestBuildFailed = "musicbrainz: request build failed: %w"
	errRequestFailed      = "musicbrainz: request failed: %w"
	errDecodeFailed       = "musicbrainz: decode failed: %w"
	errUnexpectedStatus   = "musicbrainz: unexpected status %d: %s"
	headerUserAgent       = "User-Agent"
	headerAccept          = "Accept"
	headerRetryAfter      = "Retry-After"
	contentTypeJSON       = "application/json"
	defaultMinInterval    = 1100 * time.Millisecond
	maxAttempts           = 2
)

// Config describes how to connect to the MusicBrainz API.
type Config struct {
	BaseURL     string
	AppName     string
	AppVersion  string
	Contact     string
	Timeout     time.Duration
	MinInterval time.Duration
}

// Client issues requests against the MusicBrainz API.
type Client struct {
	baseURL       string
	userAgent     string
	httpClient    *http.Client
	minInterval   time.Duration
	mu            sync.Mutex
	nextRequestAt time.Time
}

// New constructs a MusicBrainz API client using the supplied configuration.
func New(_ context.Context, cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("musicbrainz: base URL is required")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	if cfg.MinInterval <= 0 {
		cfg.MinInterval = defaultMinInterval
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
		minInterval: cfg.MinInterval,
	}, nil
}

// Artist models a subset of the MusicBrainz artist payload.
type Artist struct {
	ID             string           `json:"id"`
	Name           string           `json:"name"`
	Country        string           `json:"country,omitempty"`
	Type           string           `json:"type,omitempty"`
	Disambiguation string           `json:"disambiguation,omitempty"`
	Aliases        []string         `json:"aliases"`
	Tags           []string         `json:"tags,omitempty"`
	LifeSpan       LifeSpan         `json:"lifeSpan"`
	Relations      []ArtistRelation `json:"relations,omitempty"`
}

// ArtistRelation represents a relationship from one artist to another.
type ArtistRelation struct {
	Type       string         `json:"type"`
	Direction  string         `json:"direction"`
	TargetType string         `json:"targetType"`
	Artist     RelationArtist `json:"artist"`
}

// RelationArtist represents the target artist in an artist relation.
type RelationArtist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
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

// Release represents a specific release of an album with track information.
type Release struct {
	ID     string  `json:"id"`
	Title  string  `json:"title"`
	Status string  `json:"status"`
	Date   string  `json:"date"`
	Tracks []Track `json:"tracks"`
}

// Track represents a single track/recording within a release.
type Track struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Length    string `json:"length"`
	ID        string `json:"id"`
	Recording struct {
		ID     string `json:"id"`
		Title  string `json:"title"`
		Length int    `json:"length"`
	} `json:"recording"`
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
	Tags []struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	} `json:"tags"`
	LifeSpan  LifeSpan `json:"life-span"`
	Relations []struct {
		Type       string `json:"type"`
		Direction  string `json:"direction"`
		TargetType string `json:"target-type"`
		Artist     struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"artist"`
	} `json:"relations"`
}

type releaseGroupResponse struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	PrimaryType      string   `json:"primary-type"`
	SecondaryTypes   []string `json:"secondary-types"`
	FirstReleaseDate string   `json:"first-release-date"`
	Releases         []struct {
		ID     string `json:"id"`
		Title  string `json:"title"`
		Status string `json:"status"`
		Date   string `json:"date"`
	} `json:"releases"`
	ArtistCredit []struct {
		Name   string `json:"name"`
		Artist struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"artist"`
	} `json:"artist-credit"`
}

type releaseResponse struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Date   string `json:"date"`
	Media  []struct {
		Position int `json:"position"`
		Tracks   []struct {
			Position  int    `json:"position"`
			Number    string `json:"number"`
			Title     string `json:"title"`
			Length    int    `json:"length"`
			ID        string `json:"id"`
			Recording struct {
				ID     string `json:"id"`
				Title  string `json:"title"`
				Length int    `json:"length"`
			} `json:"recording"`
		} `json:"tracks"`
	} `json:"media"`
}

// LookupArtist retrieves a single artist record by MusicBrainz ID.
func (c *Client) LookupArtist(ctx context.Context, id string) (*Artist, error) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return nil, errors.New("musicbrainz: artist id is required")
	}

	endpoint := fmt.Sprintf("%s/artist/%s?fmt=json&inc=tags+artist-rels", c.baseURL, url.PathEscape(trimmed))
	var payload artistResponse
	if err := c.getJSON(ctx, endpoint, &payload); err != nil {
		return nil, err
	}
	return transformArtist(payload), nil
}

func transformArtist(payload artistResponse) *Artist {
	aliases := make([]string, 0, len(payload.Aliases))
	for _, alias := range payload.Aliases {
		if alias.Name != "" {
			aliases = append(aliases, alias.Name)
		}
	}

	// Extract tags and convert them to genres, filtering out common non-genre tags
	var tags []string
	for _, tag := range payload.Tags {
		if tag.Name != "" && isGenreTag(tag.Name) {
			tags = append(tags, tag.Name)
		}
	}

	return &Artist{
		ID:             payload.ID,
		Name:           payload.Name,
		Country:        payload.Country,
		Type:           payload.Type,
		Disambiguation: payload.Disambiguation,
		Aliases:        aliases,
		Tags:           tags,
		LifeSpan:       payload.LifeSpan,
		Relations:      transformArtistRelations(payload.Relations),
	}
}

func transformArtistRelations(payload []struct {
	Type       string `json:"type"`
	Direction  string `json:"direction"`
	TargetType string `json:"target-type"`
	Artist     struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"artist"`
}) []ArtistRelation {
	if len(payload) == 0 {
		return nil
	}

	relations := make([]ArtistRelation, 0, len(payload))
	for _, relation := range payload {
		if strings.TrimSpace(relation.Artist.ID) == "" || strings.TrimSpace(relation.Artist.Name) == "" {
			continue
		}
		relations = append(relations, ArtistRelation{
			Type:       relation.Type,
			Direction:  relation.Direction,
			TargetType: relation.TargetType,
			Artist: RelationArtist{
				ID:   relation.Artist.ID,
				Name: relation.Artist.Name,
			},
		})
	}

	if len(relations) == 0 {
		return nil
	}

	return relations
}

// isGenreTag filters out non-genre tags like years, places, etc.
func isGenreTag(tag string) bool {
	// Filter out common non-genre tags
	excludedTags := map[string]bool{
		"american":          true,
		"british":           true,
		"english":           true,
		"canadian":          true,
		"german":            true,
		"french":            true,
		"japanese":          true,
		"australian":        true,
		"swedish":           true,
		"norwegian":         true,
		"danish":            true,
		"dutch":             true,
		"italian":           true,
		"spanish":           true,
		"male":              true,
		"female":            true,
		"vocalist":          true,
		"singer":            true,
		"composer":          true,
		"songwriter":        true,
		"producer":          true,
		"guitarist":         true,
		"bassist":           true,
		"drummer":           true,
		"pianist":           true,
		"keyboardist":       true,
		"violinist":         true,
		"saxophonist":       true,
		"trumpeter":         true,
		"born in the 1950s": true,
		"born in the 1960s": true,
		"born in the 1970s": true,
		"born in the 1980s": true,
		"born in the 1990s": true,
		"died in the 1990s": true,
		"died in the 2000s": true,
		"died in the 2010s": true,
		"died in the 2020s": true,
		"1990s":             true,
		"2000s":             true,
		"2010s":             true,
		"2020s":             true,
		"active":            true,
		"inactive":          true,
		"disbanded":         true,
		"solo":              true,
		"duo":               true,
		"trio":              true,
		"quartet":           true,
		"band":              true,
		"group":             true,
		"orchestra":         true,
		"ensemble":          true,
	}

	tagLower := strings.ToLower(tag)
	return !excludedTags[tagLower]
}

// LookupReleaseGroup retrieves an album (release group) by ID.
func (c *Client) LookupReleaseGroup(ctx context.Context, id string) (*ReleaseGroup, error) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return nil, errors.New("musicbrainz: release group id is required")
	}

	endpoint := fmt.Sprintf("%s/release-group/%s?fmt=json&inc=artists+releases", c.baseURL, url.PathEscape(trimmed))
	var payload releaseGroupResponse
	if err := c.getJSON(ctx, endpoint, &payload); err != nil {
		return nil, err
	}
	return transformReleaseGroup(payload), nil
}

// GetReleaseGroupTracks retrieves track listings for a release group by finding a representative release.
func (c *Client) GetReleaseGroupTracks(ctx context.Context, releaseGroupID string) ([]Track, error) {
	trimmed := strings.TrimSpace(releaseGroupID)
	if trimmed == "" {
		return nil, errors.New("musicbrainz: release group id is required")
	}

	// Find a good representative release (prefer official releases)
	releaseID, err := c.findRepresentativeRelease(ctx, trimmed)
	if err != nil {
		return nil, fmt.Errorf("musicbrainz: failed to find representative release: %w", err)
	}

	// Get the release with recordings
	return c.getReleaseRecordings(ctx, releaseID)
}

// findRepresentativeRelease finds the best release to use for track listings.
func (c *Client) findRepresentativeRelease(ctx context.Context, releaseGroupID string) (string, error) {
	payload, err := c.fetchReleaseGroupWithReleases(ctx, releaseGroupID)
	if err != nil {
		return "", err
	}

	return c.selectBestRelease(payload.Releases), nil
}

func (c *Client) fetchReleaseGroupWithReleases(ctx context.Context, releaseGroupID string) (*releaseGroupResponse, error) {
	endpoint := fmt.Sprintf("%s/release-group/%s?fmt=json&inc=releases", c.baseURL, url.PathEscape(releaseGroupID))
	var payload releaseGroupResponse
	if err := c.getJSON(ctx, endpoint, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

func (c *Client) selectBestRelease(releases []struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Date   string `json:"date"`
}) string {
	// Find the best release (prefer official releases)
	for _, release := range releases {
		if release.Status == "Official" {
			return release.ID
		}
	}

	// If no official release found, use the first release
	if len(releases) > 0 {
		return releases[0].ID
	}

	return ""
}

// getReleaseRecordings gets the track/recording data for a specific release.
func (c *Client) getReleaseRecordings(ctx context.Context, releaseID string) ([]Track, error) {
	endpoint := fmt.Sprintf("%s/release/%s?fmt=json&inc=recordings", c.baseURL, url.PathEscape(releaseID))
	var payload releaseResponse
	if err := c.getJSON(ctx, endpoint, &payload); err != nil {
		return nil, err
	}
	return transformReleaseTracks(payload), nil
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

func transformReleaseTracks(payload releaseResponse) []Track {
	var allTracks []Track
	for _, medium := range payload.Media {
		for _, track := range medium.Tracks {
			// Convert track length from milliseconds to MM:SS format
			length := ""
			if track.Length > 0 {
				seconds := track.Length / 1000
				minutes := seconds / 60
				remainingSeconds := seconds % 60
				length = fmt.Sprintf("%d:%02d", minutes, remainingSeconds)
			}

			// Parse track number (handle string to int conversion)
			trackNumber := track.Position
			if trackNumber == 0 {
				// Try to parse the number field if position is not available
				if num, err := strconv.Atoi(track.Number); err == nil {
					trackNumber = num
				}
			}

			allTracks = append(allTracks, Track{
				Number: trackNumber,
				Title:  track.Title,
				Length: length,
				ID:     track.ID,
				Recording: struct {
					ID     string `json:"id"`
					Title  string `json:"title"`
					Length int    `json:"length"`
				}{
					ID:     track.Recording.ID,
					Title:  track.Recording.Title,
					Length: track.Recording.Length,
				},
			})
		}
	}
	return allTracks
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
	var payload searchResponse
	if err := c.getJSON(ctx, endpoint, &payload); err != nil {
		return nil, err
	}
	return transformSearchResult(payload), nil
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
	var payload releaseGroupSearchResponse
	if err := c.getJSON(ctx, endpoint, &payload); err != nil {
		return nil, err
	}
	return transformReleaseGroupSearchResult(payload, artistID), nil
}

// releaseGroupPageSize is the MusicBrainz maximum page size for browse requests.
const releaseGroupPageSize = 100

// maxArtistReleaseGroups caps how many release groups we will accumulate for a
// single artist. This bounds both memory and the number of upstream requests
// for pathological cases; MusicBrainz's most prolific artists sit well under it.
const maxArtistReleaseGroups = 1000

// GetAllArtistReleaseGroups retrieves every release group for an artist by
// paging through the browse endpoint until the reported total is reached.
//
// The previous single-call approach capped results at the first page, which
// silently truncated prolific artists (Frank Zappa, David Bowie) to an
// arbitrary subset — MusicBrainz's browse endpoint returns no meaningful
// ordering, so the retained subset was effectively random.
//
// A failure on any page returns an error rather than the partial results
// gathered so far. Callers cache what they receive, so returning a partial
// discography would persist the truncation we are trying to fix.
func (c *Client) GetAllArtistReleaseGroups(ctx context.Context, artistID string) (*ReleaseGroupSearchResult, error) {
	trimmed := strings.TrimSpace(artistID)
	if trimmed == "" {
		return nil, errors.New("musicbrainz: artist id is required")
	}

	var (
		all   []ReleaseGroup
		seen  = make(map[string]struct{})
		total int
	)

	for offset := 0; offset < maxArtistReleaseGroups; offset += releaseGroupPageSize {
		page, err := c.GetArtistReleaseGroups(ctx, trimmed, releaseGroupPageSize, offset)
		if err != nil {
			return nil, err
		}
		if page == nil {
			break
		}

		total = page.Count

		// An empty page means we have run past the end of the collection.
		// Guard on this as well as the count so a stale or inconsistent
		// count cannot spin the loop.
		if len(page.ReleaseGroups) == 0 {
			break
		}

		// Deduplicate by ID: paging is not a consistent snapshot, so an
		// edit upstream mid-crawl can surface the same group on two pages.
		for _, rg := range page.ReleaseGroups {
			if _, dup := seen[rg.ID]; dup {
				continue
			}
			seen[rg.ID] = struct{}{}
			all = append(all, rg)
		}

		if offset+len(page.ReleaseGroups) >= total {
			break
		}
	}

	return &ReleaseGroupSearchResult{
		ReleaseGroups: all,
		Count:         total,
		Offset:        0,
	}, nil
}

func (c *Client) getJSON(ctx context.Context, endpoint string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf(errRequestBuildFailed, err)
	}
	req.Header.Set(headerUserAgent, c.userAgent)
	req.Header.Set(headerAccept, contentTypeJSON)

	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf(errDecodeFailed, err)
		}
		return nil
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusTooManyRequests, http.StatusServiceUnavailable:
		return ErrRateLimit
	default:
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf(errUnexpectedStatus, resp.StatusCode, strings.TrimSpace(string(snippet)))
	}
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := c.waitForTurn(req.Context()); err != nil {
			return nil, fmt.Errorf(errRequestFailed, err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf(errRequestFailed, err)
		}
		if !isRetryableStatus(resp.StatusCode) || attempt == maxAttempts {
			return resp, nil
		}

		delay := c.retryDelay(resp.Header.Get(headerRetryAfter))
		resp.Body.Close()
		if err := sleepWithContext(req.Context(), delay); err != nil {
			return nil, fmt.Errorf(errRequestFailed, err)
		}
	}

	return nil, fmt.Errorf(errRequestFailed, errors.New("exhausted retry attempts"))
}

func (c *Client) waitForTurn(ctx context.Context) error {
	c.mu.Lock()
	waitUntil := c.nextRequestAt
	now := time.Now()
	if waitUntil.Before(now) {
		waitUntil = now
	}
	c.nextRequestAt = waitUntil.Add(c.minInterval)
	c.mu.Unlock()

	wait := time.Until(waitUntil)
	if wait <= 0 {
		return nil
	}

	return sleepWithContext(ctx, wait)
}

func (c *Client) retryDelay(raw string) time.Duration {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return c.minInterval
	}

	if seconds, err := strconv.Atoi(trimmed); err == nil {
		if seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
		return c.minInterval
	}

	if retryAt, err := http.ParseTime(trimmed); err == nil {
		delay := time.Until(retryAt)
		if delay > 0 {
			return delay
		}
	}

	return c.minInterval
}

func isRetryableStatus(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode == http.StatusServiceUnavailable
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
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
