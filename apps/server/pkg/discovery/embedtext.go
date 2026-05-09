package discovery

import (
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/adamlacasse/freq-show/apps/server/pkg/data"
)

const (
	MinEmbeddingTextChars = 120
	maxReviewExcerptRunes = 400
	maxTrackTitles        = 8
)

// BuildAlbumEmbeddingText turns FreqShow album metadata into natural language
// for document embeddings. Thin records return an empty string.
func BuildAlbumEmbeddingText(album *data.Album, artist *data.Artist) string {
	if album == nil {
		return ""
	}

	artistName := firstNonEmpty(album.ArtistName, artistName(artist))
	var parts []string
	if album.Title != "" && artistName != "" {
		identity := fmt.Sprintf("%q by %s", album.Title, artistName)
		if album.Year > 0 {
			identity += fmt.Sprintf(", released %d", album.Year)
		}
		parts = append(parts, identity+".")
	}

	if genre := albumGenre(album, artist); genre != "" {
		parts = append(parts, "Genre: "+genre+".")
	}
	if album.Review.Summary != "" {
		prefix := "Review"
		if album.Review.Source != "" {
			prefix = album.Review.Source
		}
		if album.Review.Rating > 0 {
			prefix += fmt.Sprintf(" (%.1f/10)", album.Review.Rating)
		}
		parts = append(parts, prefix+": "+strings.TrimSpace(album.Review.Summary))
	}
	if album.Review.Text != "" {
		parts = append(parts, truncateText(album.Review.Text, maxReviewExcerptRunes))
	}
	if tracks := trackTitles(album.Tracks); tracks != "" {
		parts = append(parts, "Tracks include: "+tracks+".")
	}

	text := strings.Join(parts, " ")
	if utf8.RuneCountInString(text) < MinEmbeddingTextChars {
		return ""
	}
	return text
}

func buildQueryEmbeddingText(interpreted data.InterpretedQuery) string {
	var parts []string
	if interpreted.Mood != "" {
		parts = append(parts, "The listener wants music with a "+interpreted.Mood+" mood.")
	}
	if len(interpreted.SonicQualities) > 0 {
		parts = append(parts, "Sonic qualities: "+strings.Join(cleanStrings(interpreted.SonicQualities), ", ")+".")
	}
	if len(interpreted.ReferenceArtists) > 0 {
		parts = append(parts, "Reference artists: "+strings.Join(cleanStrings(interpreted.ReferenceArtists), ", ")+".")
	}
	if len(interpreted.EraHints) > 0 {
		parts = append(parts, "Era hints: "+strings.Join(cleanStrings(interpreted.EraHints), ", ")+".")
	}
	if len(interpreted.Avoid) > 0 {
		parts = append(parts, "Avoid: "+strings.Join(cleanStrings(interpreted.Avoid), ", ")+".")
	}
	if interpreted.DiscoveryAppetite != "" {
		parts = append(parts, "Discovery appetite is "+interpreted.DiscoveryAppetite+".")
	}
	return strings.Join(parts, " ")
}

func spotifySearchURL(album *data.Album) string {
	if album == nil {
		return ""
	}
	query := strings.TrimSpace(album.ArtistName + " " + album.Title)
	if query == "" {
		return ""
	}
	return "https://open.spotify.com/search/" + url.PathEscape(query)
}

func albumGenre(album *data.Album, artist *data.Artist) string {
	if strings.TrimSpace(album.Genre) != "" {
		return strings.TrimSpace(album.Genre)
	}
	if artist == nil || len(artist.Genres) == 0 {
		return ""
	}
	genres := cleanStrings(artist.Genres)
	if len(genres) > 5 {
		genres = genres[:5]
	}
	return strings.Join(genres, ", ")
}

func artistName(artist *data.Artist) string {
	if artist == nil {
		return ""
	}
	return artist.Name
}

func trackTitles(tracks []data.Track) string {
	limit := len(tracks)
	if limit > maxTrackTitles {
		limit = maxTrackTitles
	}
	titles := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		title := strings.TrimSpace(tracks[i].Title)
		if title != "" {
			titles = append(titles, title)
		}
	}
	return strings.Join(titles, ", ")
}

func truncateText(text string, maxRunes int) string {
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) <= maxRunes {
		return text
	}
	runes := []rune(text)
	return strings.TrimSpace(string(runes[:maxRunes])) + "..."
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
