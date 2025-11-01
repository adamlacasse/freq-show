package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/adamlacasse/freq-show/apps/server/pkg/data"
	"github.com/adamlacasse/freq-show/apps/server/pkg/db"
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
	GetArtistBiography(ctx context.Context, artistName string) (string, error)
}

// RouterConfig captures dependencies required by the HTTP router.
type RouterConfig struct {
	MusicBrainz MusicBrainzClient
	Wikipedia   WikipediaClient
	Artists     db.ArtistRepository
	Albums      db.AlbumRepository
}

// NewRouter wires the top-level HTTP routes for the backend.
func NewRouter(cfg RouterConfig) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler)
	mux.Handle("/artists/", artistLookupHandler(cfg.Artists, cfg.MusicBrainz, cfg.Wikipedia))
	mux.Handle("/albums/", albumLookupHandler(cfg.Albums, cfg.MusicBrainz))
	mux.HandleFunc("/search", searchHandler(cfg.MusicBrainz))
	return corsMiddleware(mux)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
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

		writeJSON(w, http.StatusOK, artist)
	})
}

func albumLookupHandler(repo db.AlbumRepository, client MusicBrainzClient) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assertMethod(w, r, http.MethodGet) {
			return
		}

		id, err := parseAlbumID(r.URL.Path)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{err.Error()})
			return
		}

		album, err := getOrFetchAlbum(r.Context(), repo, client, id)
		if err != nil {
			handleAPIError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, album)
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
			// If cached artist has no albums, fetch them
			if artist.Albums == nil || len(artist.Albums) == 0 {
				if mbClient != nil {
					releaseGroups, err := mbClient.GetArtistReleaseGroups(ctx, id, 50, 0)
					if err == nil {
						artist.Albums = transformReleaseGroupsToAlbums(releaseGroups.ReleaseGroups)
						// Update the cached artist with albums
						_ = repo.SaveArtist(ctx, artist)
					}
				}
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
		default:
			return nil, newAPIError(http.StatusBadGateway, "musicbrainz lookup failed")
		}
	}

	domainArtist := transformArtist(remote)

	// Fetch biography from Wikipedia
	if wikiClient != nil {
		biography, err := wikiClient.GetArtistBiography(ctx, remote.Name)
		if err == nil {
			domainArtist.Biography = biography
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

func getOrFetchAlbum(ctx context.Context, repo db.AlbumRepository, client MusicBrainzClient, id string) (*data.Album, error) {
	if repo != nil {
		album, err := repo.GetAlbum(ctx, id)
		if err != nil {
			return nil, newAPIError(http.StatusInternalServerError, "album lookup failed")
		}
		if album != nil {
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

	if repo != nil {
		if err := repo.SaveAlbum(ctx, domainAlbum); err != nil {
			return nil, newAPIError(http.StatusInternalServerError, "album cache failed")
		}
	}

	return domainAlbum, nil
}

func transformArtist(src *musicbrainz.Artist) *data.Artist {
	if src == nil {
		return nil
	}
	return &data.Artist{
		ID:             src.ID,
		Name:           src.Name,
		Biography:      "",
		Genres:         append([]string(nil), src.Tags...),
		Albums:         nil,
		Related:        nil,
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

func searchHandler(client MusicBrainzClient) http.HandlerFunc {
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

		result, err := client.SearchArtists(r.Context(), query, limit, offset)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "search failed"})
			return
		}

		writeJSON(w, http.StatusOK, result)
	}
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
