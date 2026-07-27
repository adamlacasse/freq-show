package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/adamlacasse/freq-show/apps/server/pkg/data"
	"github.com/adamlacasse/freq-show/apps/server/pkg/db"
	"github.com/adamlacasse/freq-show/apps/server/pkg/discovery"
	"github.com/adamlacasse/freq-show/apps/server/pkg/sources/embeddings"
	"github.com/adamlacasse/freq-show/apps/server/pkg/sources/musicbrainz"
)

// MusicBrainzClient captures the MusicBrainz operations the router relies on.
type MusicBrainzClient interface {
	LookupArtist(ctx context.Context, id string) (*musicbrainz.Artist, error)
	LookupReleaseGroup(ctx context.Context, id string) (*musicbrainz.ReleaseGroup, error)
	SearchArtists(ctx context.Context, query string, limit int, offset int) (*musicbrainz.SearchResult, error)
	GetArtistReleaseGroups(ctx context.Context, artistID string, limit int, offset int) (*musicbrainz.ReleaseGroupSearchResult, error)
	GetReleaseGroupTracks(ctx context.Context, releaseGroupID string) ([]musicbrainz.Track, error)
}

// WikipediaClient captures the Wikipedia operations the router relies on.
type WikipediaClient interface {
	GetArtistBiography(ctx context.Context, artistName string) (string, string, error)
}

// ReviewsClient captures the reviews operations the router relies on.
type ReviewsClient interface {
	GetAlbumReview(ctx context.Context, artistName, albumTitle string) (*data.Review, error)
}

// RouterConfig captures dependencies required by the HTTP router.
type RouterConfig struct {
	MusicBrainz MusicBrainzClient
	Wikipedia   WikipediaClient
	Reviews     ReviewsClient
	Artists     db.ArtistRepository
	Albums      db.AlbumRepository
	Embeddings  db.EmbeddingRepository
	Embedder    embeddings.Embedder
	Discovery   *discovery.Service
	Collections db.CollectionRepository
}

// NewRouter wires the top-level HTTP routes for the backend.
func NewRouter(cfg RouterConfig) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler)
	mux.Handle("/artists/", artistLookupHandler(cfg.Artists, cfg.MusicBrainz, cfg.Wikipedia))
	mux.Handle("/albums/", albumLookupHandler(cfg.Albums, cfg.Embeddings, cfg.Embedder, cfg.MusicBrainz, cfg.Reviews))
	mux.HandleFunc("/search", searchHandler(cfg.MusicBrainz))
	mux.Handle("/discover", discoverRateLimit(newDiscoverLimiter(), discoverHandler(cfg.Discovery)))
	mux.Handle("/collections/", collectionHandler(cfg.Collections))
	return corsMiddleware(mux)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if delay := r.URL.Query().Get("delay"); delay != "" {
		if ms, err := strconv.Atoi(delay); err == nil {
			time.Sleep(time.Duration(ms) * time.Millisecond)
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func artistLookupHandler(repo db.ArtistRepository, mbClient MusicBrainzClient, wikiClient WikipediaClient) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assertMethod(w, r, http.MethodGet) {
			return
		}

		id, err := parseArtistID(r.URL.Path)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{err.Error()})
			return
		}

		artist, err := getOrFetchArtist(r.Context(), repo, mbClient, wikiClient, id)
		if err != nil {
			handleAPIError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, normalizeArtistForJSON(artist))
	})
}

func albumLookupHandler(repo db.AlbumRepository, embeddingsRepo db.EmbeddingRepository, embedder embeddings.Embedder, client MusicBrainzClient, reviewsClient ReviewsClient) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assertMethod(w, r, http.MethodGet) {
			return
		}

		id, err := parseAlbumID(r.URL.Path)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{err.Error()})
			return
		}

		album, err := getOrFetchAlbum(r.Context(), repo, embeddingsRepo, embedder, client, reviewsClient, id)
		if err != nil {
			handleAPIError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, normalizeAlbumForJSON(album))
	})
}

func collectionHandler(repo db.CollectionRepository) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/collections/")
		if path == "" || path == r.URL.Path {
			writeJSON(w, http.StatusBadRequest, errorResponse{"user id required"})
			return
		}

		parts := strings.Split(path, "/")
		userID := parts[0]

		if len(parts) == 1 && r.Method == http.MethodGet {
			items, err := repo.GetUserCollection(r.Context(), userID)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{"failed to fetch collection"})
				return
			}
			if items == nil {
				items = []data.CollectionItem{}
			}
			writeJSON(w, http.StatusOK, items)
			return
		}

		if len(parts) == 3 && parts[1] == "albums" {
			albumID := parts[2]
			if r.Method == http.MethodPost {
				var req struct {
					Format string `json:"format"`
				}
				_ = json.NewDecoder(r.Body).Decode(&req)
				
				err := repo.AddAlbumToCollection(r.Context(), userID, albumID, req.Format)
				if err != nil {
					writeJSON(w, http.StatusInternalServerError, errorResponse{"failed to add to collection"})
					return
				}
				writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
				return
			}
			
			if r.Method == http.MethodPut || r.Method == http.MethodPatch {
				var req struct {
					Format           string `json:"format"`
					CustomArtistName string `json:"customArtistName"`
				}
				_ = json.NewDecoder(r.Body).Decode(&req)
				
				err := repo.UpdateCollectionItem(r.Context(), userID, albumID, req.Format, req.CustomArtistName)
				if err != nil {
					writeJSON(w, http.StatusInternalServerError, errorResponse{"failed to update collection item"})
					return
				}
				writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
				return
			}
			
			if r.Method == http.MethodDelete {
				err := repo.RemoveAlbumFromCollection(r.Context(), userID, albumID)
				if err != nil {
					writeJSON(w, http.StatusInternalServerError, errorResponse{"failed to remove from collection"})
					return
				}
				writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
				return
			}
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	})
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func parseArtistID(path string) (string, error) {
	return parseResourceID(path, "/artists/", "artist id required")
}

func parseAlbumID(path string) (string, error) {
	return parseResourceID(path, "/albums/", "album id required")
}

func parseResourceID(path, prefix, errMsg string) (string, error) {
	trimmed := strings.TrimPrefix(path, prefix)
	if trimmed == path {
		return "", errors.New(errMsg)
	}
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return "", errors.New(errMsg)
	}
	if idx := strings.Index(trimmed, "/"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	return trimmed, nil
}

func assertMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return false
	}
	return true
}

type apiError struct {
	status int
	msg    string
}

func (e apiError) Error() string {
	return e.msg
}

func newAPIError(status int, msg string) error {
	return apiError{status: status, msg: msg}
}

func handleAPIError(w http.ResponseWriter, err error) {
	var apiErr apiError
	if errors.As(err, &apiErr) {
		writeJSON(w, apiErr.status, errorResponse{apiErr.msg})
		return
	}
	writeJSON(w, http.StatusInternalServerError, errorResponse{"request failed"})
}

func getOrFetchArtist(ctx context.Context, repo db.ArtistRepository, mbClient MusicBrainzClient, wikiClient WikipediaClient, id string) (*data.Artist, error) {
	if repo != nil {
		artist, err := repo.GetArtist(ctx, id)
		if err != nil {
			return nil, newAPIError(http.StatusInternalServerError, "artist lookup failed")
		}
		if artist != nil {
			updated := false

			if artist.BiographyURL == "" && wikiClient != nil && strings.TrimSpace(artist.Name) != "" {
				if biography, sourceURL, err := wikiClient.GetArtistBiography(ctx, artist.Name); err == nil {
					if artist.Biography == "" && biography != "" {
						artist.Biography = biography
						updated = true
					}
					if sourceURL != "" {
						artist.BiographyURL = sourceURL
						updated = true
					}
				}
			}

			if len(artist.Related) == 0 && mbClient != nil {
				if remoteArtist, err := mbClient.LookupArtist(ctx, id); err == nil {
					related := transformRelatedArtists(remoteArtist.Relations)
					if len(related) > 0 {
						artist.Related = related
						updated = true
					}
				}
			}

			// If cached artist has no albums, fetch them.
			if len(artist.Albums) == 0 {
				if mbClient != nil {
					releaseGroups, err := mbClient.GetArtistReleaseGroups(ctx, id, 50, 0)
					if err == nil {
						artist.Albums = transformReleaseGroupsToAlbums(releaseGroups.ReleaseGroups)
						updated = true
					}
				}
			}
			if updated {
				_ = repo.SaveArtist(ctx, artist)
			}
			return artist, nil
		}
	}

	if mbClient == nil {
		return nil, newAPIError(http.StatusServiceUnavailable, "musicbrainz client unavailable")
	}

	remote, err := mbClient.LookupArtist(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, musicbrainz.ErrNotFound):
			return nil, newAPIError(http.StatusNotFound, "artist not found")
		case errors.Is(err, musicbrainz.ErrRateLimit):
			return nil, newAPIError(http.StatusTooManyRequests, "musicbrainz rate limit exceeded, please try again shortly")
		default:
			return nil, newAPIError(http.StatusBadGateway, "musicbrainz lookup failed")
		}
	}

	domainArtist := transformArtist(remote)

	// Fetch biography from Wikipedia
	if wikiClient != nil {
		biography, sourceURL, err := wikiClient.GetArtistBiography(ctx, remote.Name)
		if err == nil {
			domainArtist.Biography = biography
			domainArtist.BiographyURL = sourceURL
		}
		// Continue even if biography fetch fails
	}

	// Fetch artist's albums/release groups
	releaseGroups, err := mbClient.GetArtistReleaseGroups(ctx, id, 50, 0)
	if err != nil {
		// Don't fail the artist lookup if albums can't be fetched
		// Just log and continue with empty albums
		domainArtist.Albums = nil
	} else {
		domainArtist.Albums = transformReleaseGroupsToAlbums(releaseGroups.ReleaseGroups)
	}

	if repo != nil {
		if err := repo.SaveArtist(ctx, domainArtist); err != nil {
			return nil, newAPIError(http.StatusInternalServerError, "artist cache failed")
		}
	}

	return domainArtist, nil
}

func getOrFetchAlbum(ctx context.Context, repo db.AlbumRepository, embeddingsRepo db.EmbeddingRepository, embedder embeddings.Embedder, client MusicBrainzClient, reviewsClient ReviewsClient, id string) (*data.Album, error) {
	if repo != nil {
		album, err := repo.GetAlbum(ctx, id)
		if err != nil {
			return nil, newAPIError(http.StatusInternalServerError, "album lookup failed")
		}
		if album != nil {
			saveAlbumEmbeddingBestEffort(ctx, album, embeddingsRepo, embedder)
			return album, nil
		}
	}

	if client == nil {
		return nil, newAPIError(http.StatusServiceUnavailable, "musicbrainz client unavailable")
	}

	remote, err := client.LookupReleaseGroup(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, musicbrainz.ErrNotFound):
			return nil, newAPIError(http.StatusNotFound, "album not found")
		case errors.Is(err, musicbrainz.ErrRateLimit):
			return nil, newAPIError(http.StatusTooManyRequests, "musicbrainz rate limit exceeded, please try again shortly")
		default:
			return nil, newAPIError(http.StatusBadGateway, "musicbrainz lookup failed")
		}
	}

	domainAlbum := transformAlbum(remote)

	// Fetch track listings
	tracks, err := client.GetReleaseGroupTracks(ctx, id)
	if err == nil {
		domainAlbum.Tracks = transformTracks(tracks)
	}
	// If track fetching fails, we continue without tracks rather than failing the whole request

	// Fetch review data
	if reviewsClient != nil {
		review, err := reviewsClient.GetAlbumReview(ctx, domainAlbum.ArtistName, domainAlbum.Title)
		if err == nil && review != nil {
			domainAlbum.Review = *review
		}
	}
	// If review fetching fails, we continue without reviews rather than failing the whole request

	if repo != nil {
		if err := repo.SaveAlbum(ctx, domainAlbum); err != nil {
			return nil, newAPIError(http.StatusInternalServerError, "album cache failed")
		}
	}
	saveAlbumEmbeddingBestEffort(ctx, domainAlbum, embeddingsRepo, embedder)

	return domainAlbum, nil
}

func saveAlbumEmbeddingBestEffort(ctx context.Context, album *data.Album, repo db.EmbeddingRepository, embedder embeddings.Embedder) {
	if album == nil || repo == nil || embedder == nil {
		return
	}
	existing, err := repo.GetEmbedding(ctx, album.ID, embedder.Model())
	if err != nil || existing != nil {
		return
	}
	text := discovery.BuildAlbumEmbeddingText(album, nil)
	if text == "" {
		return
	}
	vec, err := embedder.EncodeBatch(ctx, []string{text})
	if err != nil || len(vec) != 1 || len(vec[0]) == 0 {
		return
	}
	_ = repo.SaveEmbedding(ctx, album.ID, embedder.Model(), vec[0])
}

func transformArtist(src *musicbrainz.Artist) *data.Artist {
	if src == nil {
		return nil
	}
	return &data.Artist{
		ID:             src.ID,
		Name:           src.Name,
		Biography:      "",
		BiographyURL:   "",
		Genres:         append([]string(nil), src.Tags...),
		Albums:         nil,
		Related:        transformRelatedArtists(src.Relations),
		ImageURL:       "",
		Country:        src.Country,
		Type:           src.Type,
		Disambiguation: src.Disambiguation,
		Aliases:        append([]string(nil), src.Aliases...),
		LifeSpan: data.LifeSpan{
			Begin: src.LifeSpan.Begin,
			End:   src.LifeSpan.End,
			Ended: src.LifeSpan.Ended,
		},
	}
}

func transformRelatedArtists(relations []musicbrainz.ArtistRelation) []data.RelatedArtist {
	if len(relations) == 0 {
		return nil
	}

	ordered := make([]data.RelatedArtist, 0, len(relations))
	index := make(map[string]int, len(relations))

	for _, relation := range relations {
		targetType := strings.TrimSpace(relation.TargetType)
		if targetType != "" && targetType != "artist" {
			continue
		}

		id := strings.TrimSpace(relation.Artist.ID)
		name := strings.TrimSpace(relation.Artist.Name)
		if id == "" || name == "" {
			continue
		}

		if pos, ok := index[id]; ok {
			if relation.Type != "" && relation.Type != ordered[pos].RelationshipType {
				ordered[pos].RelationshipType = mergeRelationshipTypes(ordered[pos].RelationshipType, relation.Type)
			}
			continue
		}

		ordered = append(ordered, data.RelatedArtist{
			ID:               id,
			Name:             name,
			RelationshipType: relation.Type,
		})
		index[id] = len(ordered) - 1
	}

	if len(ordered) == 0 {
		return nil
	}

	return ordered
}

func mergeRelationshipTypes(existing, next string) string {
	existing = strings.TrimSpace(existing)
	next = strings.TrimSpace(next)
	if existing == "" {
		return next
	}
	if next == "" {
		return existing
	}
	for _, value := range strings.Split(existing, ", ") {
		if value == next {
			return existing
		}
	}
	return existing + ", " + next
}

func transformAlbum(src *musicbrainz.ReleaseGroup) *data.Album {
	if src == nil {
		return nil
	}

	album := &data.Album{
		ID:               src.ID,
		Title:            src.Title,
		ArtistID:         src.PrimaryArtistID(),
		ArtistName:       src.PrimaryArtistName(),
		PrimaryType:      src.PrimaryType,
		SecondaryTypes:   append([]string(nil), src.SecondaryTypes...),
		FirstReleaseDate: src.FirstReleaseDate,
		Year:             src.ReleaseYear(),
		Genre:            "",
		Label:            "",
		Tracks:           nil,
		Review:           data.Review{},
		CoverURL:         "",
	}
	return album
}

func transformTracks(mbTracks []musicbrainz.Track) []data.Track {
	if len(mbTracks) == 0 {
		return nil
	}

	tracks := make([]data.Track, 0, len(mbTracks))
	for _, mbTrack := range mbTracks {
		track := data.Track{
			Number: mbTrack.Number,
			Title:  mbTrack.Title,
			Length: mbTrack.Length,
		}
		tracks = append(tracks, track)
	}
	return tracks
}

func transformReleaseGroupsToAlbums(releaseGroups []musicbrainz.ReleaseGroup) []data.Album {
	if len(releaseGroups) == 0 {
		return nil
	}

	albums := make([]data.Album, 0, len(releaseGroups))
	for _, rg := range releaseGroups {
		album := data.Album{
			ID:               rg.ID,
			Title:            rg.Title,
			ArtistID:         rg.PrimaryArtistID(),
			ArtistName:       rg.PrimaryArtistName(),
			PrimaryType:      rg.PrimaryType,
			SecondaryTypes:   append([]string(nil), rg.SecondaryTypes...),
			FirstReleaseDate: rg.FirstReleaseDate,
			Year:             rg.ReleaseYear(),
			Genre:            "",
			Label:            "",
			Tracks:           nil,
			Review:           data.Review{},
			CoverURL:         "",
		}
		albums = append(albums, album)
	}
	return albums
}

// searchCacheEntry holds a cached search result with an expiry timestamp.
type searchCacheEntry struct {
	result    interface{}
	expiresAt time.Time
}

// searchCache is a simple in-memory TTL cache for search results.
type searchCache struct {
	mu      sync.Mutex
	entries map[string]searchCacheEntry
	ttl     time.Duration
}

func newSearchCache(ttl time.Duration) *searchCache {
	return &searchCache{
		entries: make(map[string]searchCacheEntry),
		ttl:     ttl,
	}
}

func (c *searchCache) get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.expiresAt) {
		delete(c.entries, key)
		return nil, false
	}
	return entry.result, true
}

func (c *searchCache) set(key string, result interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = searchCacheEntry{
		result:    result,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func searchHandler(client MusicBrainzClient) http.HandlerFunc {
	cache := newSearchCache(5 * time.Minute)

	return func(w http.ResponseWriter, r *http.Request) {
		if !assertMethod(w, r, http.MethodGet) {
			return
		}

		query := r.URL.Query().Get("q")
		if strings.TrimSpace(query) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "search query parameter 'q' is required"})
			return
		}

		limit := parseSearchLimit(r.URL.Query().Get("limit"))
		offset := parseSearchOffset(r.URL.Query().Get("offset"))

		cacheKey := strings.ToLower(strings.TrimSpace(query)) + "|" + strconv.Itoa(limit) + "|" + strconv.Itoa(offset)
		if cached, ok := cache.get(cacheKey); ok {
			writeJSON(w, http.StatusOK, cached)
			return
		}

		result, err := client.SearchArtists(r.Context(), query, limit, offset)
		if err != nil {
			if errors.Is(err, musicbrainz.ErrRateLimit) {
				writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "search rate limit exceeded, please try again shortly"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "search failed"})
			return
		}

		normalized := normalizeSearchResultForJSON(result)
		cache.set(cacheKey, normalized)
		writeJSON(w, http.StatusOK, normalized)
	}
}

func normalizeArtistForJSON(artist *data.Artist) *data.Artist {
	if artist == nil {
		return nil
	}

	if artist.Genres == nil {
		artist.Genres = []string{}
	}
	if artist.Albums == nil {
		artist.Albums = []data.Album{}
	}
	if artist.Related == nil {
		artist.Related = []data.RelatedArtist{}
	}
	if artist.Aliases == nil {
		artist.Aliases = []string{}
	}

	for i := range artist.Albums {
		normalizeAlbumFields(&artist.Albums[i])
	}

	return artist
}

func normalizeAlbumForJSON(album *data.Album) *data.Album {
	if album == nil {
		return nil
	}

	normalizeAlbumFields(album)
	return album
}

func normalizeAlbumFields(album *data.Album) {
	if album.SecondaryTypes == nil {
		album.SecondaryTypes = []string{}
	}
	if album.Tracks == nil {
		album.Tracks = []data.Track{}
	}
}

func normalizeSearchResultForJSON(result *musicbrainz.SearchResult) *musicbrainz.SearchResult {
	if result == nil {
		return nil
	}

	if result.Artists == nil {
		result.Artists = []musicbrainz.Artist{}
	}

	for i := range result.Artists {
		if result.Artists[i].Aliases == nil {
			result.Artists[i].Aliases = []string{}
		}
	}

	return result
}

func parseSearchLimit(limitStr string) int {
	if limitStr == "" {
		return 25
	}
	if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 100 {
		return parsed
	}
	return 25
}

func parseSearchOffset(offsetStr string) int {
	if offsetStr == "" {
		return 0
	}
	if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
		return parsed
	}
	return 0
}

// corsMiddleware adds CORS headers for local development
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow requests from Angular dev server
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:4200")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
