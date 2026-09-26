package data

type Artist struct {
	// CacheVersion records which generation of the fetch logic produced this
	// record. Cached artists are otherwise never refreshed, so a payload
	// written by an older, buggier fetch path would persist indefinitely.
	// Bump the corresponding constant in the api package to force a refetch.
	CacheVersion int `json:"cacheVersion,omitempty"`

	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Biography      string          `json:"biography"`
	BiographyURL   string          `json:"biographyUrl,omitempty"`
	Genres         []string        `json:"genres"`
	Albums         []Album         `json:"albums"`
	Related        []RelatedArtist `json:"related"`
	ImageURL       string          `json:"imageUrl"`
	Country        string          `json:"country,omitempty"`
	Type           string          `json:"type,omitempty"`
	Disambiguation string          `json:"disambiguation,omitempty"`
	Aliases        []string        `json:"aliases"`
	LifeSpan       LifeSpan        `json:"lifeSpan"`
}

type RelatedArtist struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	RelationshipType string `json:"relationshipType,omitempty"`
}

type LifeSpan struct {
	Begin string `json:"begin,omitempty"`
	End   string `json:"end,omitempty"`
	Ended bool   `json:"ended,omitempty"`
}

type Album struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	ArtistID         string   `json:"artistId"`
	ArtistName       string   `json:"artistName,omitempty"`
	PrimaryType      string   `json:"primaryType,omitempty"`
	SecondaryTypes   []string `json:"secondaryTypes"`
	FirstReleaseDate string   `json:"firstReleaseDate,omitempty"`
	Year             int      `json:"year"`
	Genre            string   `json:"genre"`
	Label            string   `json:"label"`
	Tracks           []Track  `json:"tracks"`
	Review           Review   `json:"review"`
	CoverURL         string   `json:"coverUrl"`
}

type Track struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Length string `json:"length"`
}

type Review struct {
	Source  string  `json:"source"`
	Author  string  `json:"author"`
	Rating  float64 `json:"rating"`
	Summary string  `json:"summary"`
	Text    string  `json:"text"`
	URL     string  `json:"url"`
}

// User represents an account created lazily on first successful magic-link
// verification. There is no password — identity is the verified email
// address itself.
type User struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	CreatedAt string `json:"createdAt"`
}

// Session represents an active login session backing a session cookie.
// The raw session token is never stored — callers look sessions up by a
// hash of the cookie value (see pkg/auth).
type Session struct {
	UserID    string `json:"userId"`
	ExpiresAt string `json:"expiresAt"`
}

type CollectionItem struct {
	ID               int    `json:"id"`
	UserID           string `json:"userId"`
	AlbumID          string `json:"albumId"`
	Format           string `json:"format"`
	CustomArtistName string `json:"customArtistName,omitempty"`
	CustomTitle      string `json:"customTitle,omitempty"`
	CustomYear       int    `json:"customYear,omitempty"`
	AddedAt          string `json:"addedAt"`
	Album            *Album `json:"album,omitempty"`
}
