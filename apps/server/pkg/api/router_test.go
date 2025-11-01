package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/adamlacasse/freq-show/apps/server/pkg/data"
	"github.com/adamlacasse/freq-show/apps/server/pkg/sources/musicbrainz"
)

const (
	testArtistID   = "artist-id"
	artistPath     = "/artists/" + testArtistID
	missingPath    = "/artists/missing"
	baseArtistPath = "/artists/"
	testAlbumID    = "album-id"
	albumPath      = "/albums/" + testAlbumID
	missingAlbum   = "/albums/missing"
	baseAlbumPath  = "/albums/"
	status200Fmt   = "expected status 200, got %d"
	status400Fmt   = "expected status 400, got %d"
	decodeErrFmt   = "failed to decode response: %v"
	remoteArtist   = "Remote Artist"
	unexpectedCall = "unexpected call"
)

type stubArtistRepo struct {
	getFunc  func(ctx context.Context, id string) (*data.Artist, error)
	saveFunc func(ctx context.Context, artist *data.Artist) error
}

func (s *stubArtistRepo) GetArtist(ctx context.Context, id string) (*data.Artist, error) {
	if s.getFunc != nil {
		return s.getFunc(ctx, id)
	}
	return nil, nil
}

func (s *stubArtistRepo) SaveArtist(ctx context.Context, artist *data.Artist) error {
	if s.saveFunc != nil {
		return s.saveFunc(ctx, artist)
	}
	return nil
}

type stubMusicBrainz struct {
	lookupArtistFunc           func(ctx context.Context, id string) (*musicbrainz.Artist, error)
	lookupReleaseGroupFunc     func(ctx context.Context, id string) (*musicbrainz.ReleaseGroup, error)
	searchArtistsFunc          func(ctx context.Context, query string, limit int, offset int) (*musicbrainz.SearchResult, error)
	getArtistReleaseGroupsFunc func(ctx context.Context, artistID string, limit int, offset int) (*musicbrainz.ReleaseGroupSearchResult, error)
	getReleaseGroupTracksFunc  func(ctx context.Context, releaseGroupID string) ([]musicbrainz.Track, error)
}

func (s *stubMusicBrainz) LookupArtist(ctx context.Context, id string) (*musicbrainz.Artist, error) {
	if s.lookupArtistFunc != nil {
		return s.lookupArtistFunc(ctx, id)
	}
	return nil, errors.New(unexpectedCall)
}

func (s *stubMusicBrainz) LookupReleaseGroup(ctx context.Context, id string) (*musicbrainz.ReleaseGroup, error) {
	if s.lookupReleaseGroupFunc != nil {
		return s.lookupReleaseGroupFunc(ctx, id)
	}
	return nil, errors.New(unexpectedCall)
}

func (s *stubMusicBrainz) SearchArtists(ctx context.Context, query string, limit int, offset int) (*musicbrainz.SearchResult, error) {
	if s.searchArtistsFunc != nil {
		return s.searchArtistsFunc(ctx, query, limit, offset)
	}
	return nil, errors.New(unexpectedCall)
}

func (s *stubMusicBrainz) GetArtistReleaseGroups(ctx context.Context, artistID string, limit int, offset int) (*musicbrainz.ReleaseGroupSearchResult, error) {
	if s.getArtistReleaseGroupsFunc != nil {
		return s.getArtistReleaseGroupsFunc(ctx, artistID, limit, offset)
	}
	return nil, errors.New(unexpectedCall)
}

func (s *stubMusicBrainz) GetReleaseGroupTracks(ctx context.Context, releaseGroupID string) ([]musicbrainz.Track, error) {
	if s.getReleaseGroupTracksFunc != nil {
		return s.getReleaseGroupTracksFunc(ctx, releaseGroupID)
	}
	return nil, nil // Return empty tracks by default for tests
}

type stubWikipedia struct {
	getArtistBiographyFunc func(ctx context.Context, artistName string) (string, error)
}

func (s *stubWikipedia) GetArtistBiography(ctx context.Context, artistName string) (string, error) {
	if s.getArtistBiographyFunc != nil {
		return s.getArtistBiographyFunc(ctx, artistName)
	}
	return "", nil // Return empty biography by default for tests
}

type stubAlbumRepo struct {
	getFunc  func(ctx context.Context, id string) (*data.Album, error)
	saveFunc func(ctx context.Context, album *data.Album) error
}

func (s *stubAlbumRepo) GetAlbum(ctx context.Context, id string) (*data.Album, error) {
	if s.getFunc != nil {
		return s.getFunc(ctx, id)
	}
	return nil, nil
}

func (s *stubAlbumRepo) SaveAlbum(ctx context.Context, album *data.Album) error {
	if s.saveFunc != nil {
		return s.saveFunc(ctx, album)
	}
	return nil
}

func TestArtistLookupHandlerReturnsCachedArtist(t *testing.T) {
	cached := &data.Artist{ID: testArtistID, Name: "Cached"}

	repo := &stubArtistRepo{
		getFunc: func(ctx context.Context, id string) (*data.Artist, error) {
			if id != testArtistID {
				t.Fatalf("unexpected id %q", id)
			}
			return cached, nil
		},
		saveFunc: func(ctx context.Context, artist *data.Artist) error {
			t.Fatalf("save should not be called on cache hit")
			return nil
		},
	}

	mb := &stubMusicBrainz{
		lookupArtistFunc: func(ctx context.Context, id string) (*musicbrainz.Artist, error) {
			t.Fatalf("musicbrainz should not be called on cache hit")
			return nil, nil
		},
	}

	wiki := &stubWikipedia{} // Default behavior is fine for cached response

	req := httptest.NewRequest(http.MethodGet, artistPath, nil)
	res := httptest.NewRecorder()

	artistLookupHandler(repo, mb, wiki).ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf(status200Fmt, res.Code)
	}

	var payload data.Artist
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf(decodeErrFmt, err)
	}
	if payload.Name != "Cached" {
		t.Fatalf("expected cached artist name, got %q", payload.Name)
	}
}

func TestArtistLookupHandlerFetchesAndCaches(t *testing.T) {
	saved := false
	repo := &stubArtistRepo{
		getFunc: func(ctx context.Context, id string) (*data.Artist, error) {
			return nil, nil
		},
		saveFunc: func(ctx context.Context, artist *data.Artist) error {
			saved = true
			if artist.ID != testArtistID {
				t.Fatalf("unexpected artist ID %q", artist.ID)
			}
			return nil
		},
	}

	mb := &stubMusicBrainz{
		lookupArtistFunc: func(ctx context.Context, id string) (*musicbrainz.Artist, error) {
			if id != testArtistID {
				t.Fatalf("unexpected lookup id %q", id)
			}
			return &musicbrainz.Artist{ID: id, Name: "Remote"}, nil
		},
	}

	wiki := &stubWikipedia{} // Default behavior is fine

	req := httptest.NewRequest(http.MethodGet, artistPath, nil)
	res := httptest.NewRecorder()

	artistLookupHandler(repo, mb, wiki).ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf(status200Fmt, res.Code)
	}
	if !saved {
		t.Fatalf("expected artist to be cached")
	}
}

func TestArtistLookupHandlerHandlesNotFound(t *testing.T) {
	repo := &stubArtistRepo{
		getFunc: func(ctx context.Context, id string) (*data.Artist, error) {
			return nil, nil
		},
	}

	mb := &stubMusicBrainz{
		lookupArtistFunc: func(ctx context.Context, id string) (*musicbrainz.Artist, error) {
			return nil, musicbrainz.ErrNotFound
		},
	}

	wiki := &stubWikipedia{} // Default behavior is fine

	req := httptest.NewRequest(http.MethodGet, missingPath, nil)
	res := httptest.NewRecorder()

	artistLookupHandler(repo, mb, wiki).ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", res.Code)
	}
}

func TestArtistLookupHandlerMethodNotAllowed(t *testing.T) {
	repo := &stubArtistRepo{}
	mb := &stubMusicBrainz{}
	wiki := &stubWikipedia{}

	req := httptest.NewRequest(http.MethodPost, artistPath, strings.NewReader(""))
	res := httptest.NewRecorder()

	artistLookupHandler(repo, mb, wiki).ServeHTTP(res, req)

	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", res.Code)
	}
}

func TestArtistLookupHandlerBadRequest(t *testing.T) {
	repo := &stubArtistRepo{}
	mb := &stubMusicBrainz{}
	wiki := &stubWikipedia{}

	req := httptest.NewRequest(http.MethodGet, baseArtistPath, nil)
	res := httptest.NewRecorder()

	artistLookupHandler(repo, mb, wiki).ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf(status400Fmt, res.Code)
	}
}

func TestArtistLookupHandlerRepositoryError(t *testing.T) {
	repo := &stubArtistRepo{
		getFunc: func(ctx context.Context, id string) (*data.Artist, error) {
			return nil, errors.New("boom")
		},
	}
	mb := &stubMusicBrainz{}
	wiki := &stubWikipedia{}

	req := httptest.NewRequest(http.MethodGet, artistPath, nil)
	res := httptest.NewRecorder()

	artistLookupHandler(repo, mb, wiki).ServeHTTP(res, req)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", res.Code)
	}
}

func TestArtistLookupHandlerMusicBrainzError(t *testing.T) {
	repo := &stubArtistRepo{}
	mb := &stubMusicBrainz{
		lookupArtistFunc: func(ctx context.Context, id string) (*musicbrainz.Artist, error) {
			return nil, errors.New("upstream failure")
		},
	}

	wiki := &stubWikipedia{}

	req := httptest.NewRequest(http.MethodGet, artistPath, nil)
	res := httptest.NewRecorder()

	artistLookupHandler(repo, mb, wiki).ServeHTTP(res, req)

	if res.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", res.Code)
	}
}

func TestAlbumLookupHandlerReturnsCachedAlbum(t *testing.T) {
	repo := &stubAlbumRepo{
		getFunc: func(ctx context.Context, id string) (*data.Album, error) {
			if id != testAlbumID {
				t.Fatalf("unexpected id %q", id)
			}
			return &data.Album{ID: id, Title: "Cached"}, nil
		},
		saveFunc: func(ctx context.Context, album *data.Album) error {
			t.Fatalf("save should not be called on cache hit")
			return nil
		},
	}

	mb := &stubMusicBrainz{}

	req := httptest.NewRequest(http.MethodGet, albumPath, nil)
	res := httptest.NewRecorder()

	albumLookupHandler(repo, mb).ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf(status200Fmt, res.Code)
	}

	var payload data.Album
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf(decodeErrFmt, err)
	}
	if payload.Title != "Cached" {
		t.Fatalf("expected cached album title, got %q", payload.Title)
	}
}

func TestAlbumLookupHandlerFetchesAndCaches(t *testing.T) {
	saved := false
	repo := &stubAlbumRepo{
		getFunc: func(ctx context.Context, id string) (*data.Album, error) {
			return nil, nil
		},
		saveFunc: func(ctx context.Context, album *data.Album) error {
			saved = true
			if album.ID != testAlbumID {
				t.Fatalf("unexpected album ID %q", album.ID)
			}
			if album.Year != 1999 {
				t.Fatalf("expected album year 1999, got %d", album.Year)
			}
			return nil
		},
	}

	mb := &stubMusicBrainz{
		lookupReleaseGroupFunc: func(ctx context.Context, id string) (*musicbrainz.ReleaseGroup, error) {
			if id != testAlbumID {
				t.Fatalf("unexpected lookup id %q", id)
			}
			return &musicbrainz.ReleaseGroup{
				ID:               id,
				Title:            "Remote Album",
				PrimaryType:      "Album",
				SecondaryTypes:   []string{"Live"},
				FirstReleaseDate: "1999-06-01",
				ArtistCredit: []musicbrainz.ArtistCredit{
					{
						Name:   remoteArtist,
						Artist: musicbrainz.ReleaseGroupArtist{ID: "artist-1", Name: remoteArtist},
					},
				},
			}, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, albumPath, nil)
	res := httptest.NewRecorder()

	albumLookupHandler(repo, mb).ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf(status200Fmt, res.Code)
	}
	if !saved {
		t.Fatalf("expected album to be cached")
	}

	var payload data.Album
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf(decodeErrFmt, err)
	}
	if payload.ArtistName != remoteArtist {
		t.Fatalf("expected artist name propagated, got %q", payload.ArtistName)
	}
}

func TestAlbumLookupHandlerNotFound(t *testing.T) {
	repo := &stubAlbumRepo{}
	mb := &stubMusicBrainz{
		lookupReleaseGroupFunc: func(ctx context.Context, id string) (*musicbrainz.ReleaseGroup, error) {
			return nil, musicbrainz.ErrNotFound
		},
	}

	req := httptest.NewRequest(http.MethodGet, missingAlbum, nil)
	res := httptest.NewRecorder()

	albumLookupHandler(repo, mb).ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", res.Code)
	}
}

func TestAlbumLookupHandlerBadRequest(t *testing.T) {
	repo := &stubAlbumRepo{}
	mb := &stubMusicBrainz{}

	req := httptest.NewRequest(http.MethodGet, baseAlbumPath, nil)
	res := httptest.NewRecorder()

	albumLookupHandler(repo, mb).ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf(status400Fmt, res.Code)
	}
}

func TestSearchHandlerReturnsResults(t *testing.T) {
	searchResult := &musicbrainz.SearchResult{
		Artists: []musicbrainz.Artist{
			{ID: "artist1", Name: "Test Artist 1"},
			{ID: "artist2", Name: "Test Artist 2"},
		},
		Offset: 0,
		Count:  2,
	}

	mb := &stubMusicBrainz{
		searchArtistsFunc: func(ctx context.Context, query string, limit int, offset int) (*musicbrainz.SearchResult, error) {
			if query != "test query" {
				t.Fatalf("unexpected query %q", query)
			}
			if limit != 25 {
				t.Fatalf("unexpected limit %d", limit)
			}
			if offset != 0 {
				t.Fatalf("unexpected offset %d", offset)
			}
			return searchResult, nil
		},
	}

	handler := searchHandler(mb)
	req := httptest.NewRequest(http.MethodGet, "/search?q=test+query", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf(status200Fmt, resp.Code)
	}

	var result musicbrainz.SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf(decodeErrFmt, err)
	}

	if len(result.Artists) != 2 {
		t.Fatalf("expected 2 artists, got %d", len(result.Artists))
	}
	if result.Artists[0].Name != "Test Artist 1" {
		t.Fatalf("unexpected artist name %q", result.Artists[0].Name)
	}
}

func TestSearchHandlerRequiresQuery(t *testing.T) {
	mb := &stubMusicBrainz{}
	handler := searchHandler(mb)
	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf(status400Fmt, resp.Code)
	}
}
