package data

// DiscoveryQuery is the user-facing input to POST /discover.
// AlreadyKnown lists artists the user has indicated they're already familiar
// with and don't want repeated in the picks.
type DiscoveryQuery struct {
	Query        string   `json:"query"`
	AlreadyKnown []string `json:"alreadyKnown,omitempty"`
}

// InterpretedQuery is the structured form produced by the interpretation LLM
// call (Prompt A). It is what gets embedded for retrieval, and it is also
// passed to the curation LLM call (Prompt B) so the curator can honor
// signals like DiscoveryAppetite that don't appear in the raw text.
type InterpretedQuery struct {
	Mood              string   `json:"mood"`
	EraHints          []string `json:"eraHints"`
	SonicQualities    []string `json:"sonicQualities"`
	ReferenceArtists  []string `json:"referenceArtists"`
	Avoid             []string `json:"avoid"`
	DiscoveryAppetite string   `json:"discoveryAppetite"` // "low" | "medium" | "high"
}

// Pick is one of the five recommendations returned by the curation LLM call,
// joined back to album metadata for rendering by the frontend.
type Pick struct {
	Rank             int    `json:"rank"`
	AlbumID          string `json:"albumId"`
	Title            string `json:"title"`
	ArtistName       string `json:"artistName"`
	Year             int    `json:"year"`
	WhyItFits        string `json:"whyItFits"`
	WhatToListenFor  string `json:"whatToListenFor"`
	SpotifySearchURL string `json:"spotifySearchUrl"`
}

// DiscoveryResult is the response body of POST /discover. The interpreted
// query is included for transparency — the frontend can render it
// (collapsed by default) so users can see how their request was understood.
type DiscoveryResult struct {
	Interpreted InterpretedQuery `json:"interpreted"`
	Picks       []Pick           `json:"picks"`
}
