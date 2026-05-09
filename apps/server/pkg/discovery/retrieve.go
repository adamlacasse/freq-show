package discovery

import (
	"errors"
	"math"
	"sort"
	"strings"

	"github.com/adamlacasse/freq-show/apps/server/pkg/data"
	"github.com/adamlacasse/freq-show/apps/server/pkg/db"
)

const (
	TopKRetrieval = 30
	TopNAfterMMR  = 10
	MMRLambda     = 0.7
)

type albumLookupFunc func(mbid string) (*data.Album, error)

func retrieveCandidates(
	queryVec []float32,
	records []db.EmbeddingRecord,
	interpreted data.InterpretedQuery,
	alreadyKnown []string,
	albumLookup albumLookupFunc,
) ([]*data.Album, error) {
	if len(queryVec) == 0 {
		return nil, errors.New("discovery: query vector cannot be empty")
	}
	if len(records) == 0 {
		return nil, ErrNoEmbeddedAlbums
	}

	scored := cosineAll(queryVec, records)
	topK := argTopK(scored, TopKRetrieval)
	selected := mmrRerank(records, scored, topK, MMRLambda, TopNAfterMMR)

	albums := make([]*data.Album, 0, len(selected))
	for _, idx := range selected {
		album, err := albumLookup(records[idx].MBID)
		if err != nil {
			return nil, err
		}
		if album != nil {
			albums = append(albums, album)
		}
	}
	return filterAvoid(albums, append(interpreted.Avoid, alreadyKnown...)), nil
}

type scoredIndex struct {
	Index int
	Score float64
}

func cosineAll(queryVec []float32, records []db.EmbeddingRecord) []scoredIndex {
	scored := make([]scoredIndex, 0, len(records))
	for i, record := range records {
		scored = append(scored, scoredIndex{Index: i, Score: cosine(queryVec, record.Vec)})
	}
	return scored
}

func argTopK(scored []scoredIndex, k int) []int {
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})
	if k > len(scored) {
		k = len(scored)
	}
	out := make([]int, k)
	for i := 0; i < k; i++ {
		out[i] = scored[i].Index
	}
	return out
}

func mmrRerank(records []db.EmbeddingRecord, scored []scoredIndex, candidates []int, lambda float64, limit int) []int {
	if limit > len(candidates) {
		limit = len(candidates)
	}
	queryScores := make(map[int]float64, len(scored))
	for _, item := range scored {
		queryScores[item.Index] = item.Score
	}

	selected := make([]int, 0, limit)
	remaining := append([]int(nil), candidates...)
	for len(selected) < limit && len(remaining) > 0 {
		bestPos := 0
		bestScore := math.Inf(-1)
		for pos, idx := range remaining {
			noveltyPenalty := 0.0
			for _, selectedIdx := range selected {
				sim := cosine(records[idx].Vec, records[selectedIdx].Vec)
				if sim > noveltyPenalty {
					noveltyPenalty = sim
				}
			}
			score := lambda*queryScores[idx] - (1-lambda)*noveltyPenalty
			if score > bestScore {
				bestScore = score
				bestPos = pos
			}
		}
		selected = append(selected, remaining[bestPos])
		remaining = append(remaining[:bestPos], remaining[bestPos+1:]...)
	}
	return selected
}

func cosine(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, aNorm, bNorm float64
	for i := range a {
		av := float64(a[i])
		bv := float64(b[i])
		dot += av * bv
		aNorm += av * av
		bNorm += bv * bv
	}
	if aNorm == 0 || bNorm == 0 {
		return 0
	}
	return dot / (math.Sqrt(aNorm) * math.Sqrt(bNorm))
}

func filterAvoid(albums []*data.Album, avoid []string) []*data.Album {
	needles := cleanStrings(avoid)
	if len(needles) == 0 {
		return albums
	}
	out := make([]*data.Album, 0, len(albums))
	for _, album := range albums {
		haystack := strings.ToLower(strings.Join([]string{
			album.Title,
			album.ArtistName,
			album.Genre,
			album.Review.Summary,
		}, " "))
		blocked := false
		for _, needle := range needles {
			if strings.Contains(haystack, strings.ToLower(needle)) {
				blocked = true
				break
			}
		}
		if !blocked {
			out = append(out, album)
		}
	}
	return out
}
