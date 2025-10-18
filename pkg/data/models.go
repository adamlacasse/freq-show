package data

type Artist struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Biography      string   `json:"biography"`
	Genres         []string `json:"genres"`
	Albums         []Album  `json:"albums"`
	Related        []string `json:"related"`
	ImageURL       string   `json:"imageUrl"`
	Country        string   `json:"country,omitempty"`
	Type           string   `json:"type,omitempty"`
	Disambiguation string   `json:"disambiguation,omitempty"`
	Aliases        []string `json:"aliases,omitempty"`
	LifeSpan       LifeSpan `json:"lifeSpan"`
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
	SecondaryTypes   []string `json:"secondaryTypes,omitempty"`
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
