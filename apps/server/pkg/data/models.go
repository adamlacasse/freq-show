package data

type Artist struct {
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

type CollectionItem struct {
	ID               int    `json:"id"`
	UserID           string `json:"userId"`
	AlbumID          string `json:"albumId"`
	Format           string `json:"format"`
	CustomArtistName string `json:"customArtistName,omitempty"`
	AddedAt          string `json:"addedAt"`
	Album            *Album `json:"album,omitempty"`
}
