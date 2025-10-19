package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/adamlacasse/freq-show/apps/server/pkg/data"
	"github.com/adamlacasse/freq-show/apps/server/pkg/db"
	"github.com/adamlacasse/freq-show/apps/server/pkg/sources/musicbrainz"
)

// MusicBrainzClient captures the MusicBrainz operations the router relies on.
type MusicBrainzClient interface {
	LookupArtist(ctx context.Context, id string) (*musicbrainz.Artist, error)
	LookupReleaseGroup(ctx context.Context, id string) (*musicbrainz.ReleaseGroup, error)
}

// RouterConfig captures dependencies required by the HTTP router.
type RouterConfig struct {
	MusicBrainz MusicBrainzClient
	Artists     db.ArtistRepository
	Albums      db.AlbumRepository
}

// NewRouter wires the top-level HTTP routes for the backend.
func NewRouter(cfg RouterConfig) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler)
	mux.Handle("/artists/", artistLookupHandler(cfg.Artists, cfg.MusicBrainz))
	mux.Handle("/albums/", albumLookupHandler(cfg.Albums, cfg.MusicBrainz))
	return mux
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func artistLookupHandler(repo db.ArtistRepository, client MusicBrainzClient) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assertMethod(w, r, http.MethodGet) {
			return
		}

		id, err := parseArtistID(r.URL.Path)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{err.Error()})
			return
		}

		artist, err := getOrFetchArtist(r.Context(), repo, client, id)
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

func getOrFetchArtist(ctx context.Context, repo db.ArtistRepository, client MusicBrainzClient, id string) (*data.Artist, error) {
	if repo != nil {
		artist, err := repo.GetArtist(ctx, id)
		if err != nil {
			return nil, newAPIError(http.StatusInternalServerError, "artist lookup failed")
		}
		if artist != nil {
			return artist, nil
		}
	}

	if client == nil {
		return nil, newAPIError(http.StatusServiceUnavailable, "musicbrainz client unavailable")
	}

	remote, err := client.LookupArtist(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, musicbrainz.ErrNotFound):
			return nil, newAPIError(http.StatusNotFound, "artist not found")
		default:
			return nil, newAPIError(http.StatusBadGateway, "musicbrainz lookup failed")
		}
	}

	domainArtist := transformArtist(remote)
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
		Genres:         nil,
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
