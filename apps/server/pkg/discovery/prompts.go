package discovery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/adamlacasse/freq-show/apps/server/pkg/data"
)

const interpretSystemPrompt = `You are a music librarian who turns natural-language listening requests into structured search criteria for a music discovery system.

Given a user's freeform request, output ONLY a single JSON object with this exact schema:

{
  "mood": "string, short phrase capturing the emotional/atmospheric register",
  "eraHints": ["array of decade or year-range strings, may be empty"],
  "sonicQualities": ["array of descriptors like 'dense', 'sparse', 'electronic', 'analog warmth', 'instrumental'"],
  "referenceArtists": ["array of artists the user named, may be empty"],
  "avoid": ["array of qualities, artists, or albums the user explicitly wants to avoid"],
  "discoveryAppetite": "low | medium | high — how far from the references the user wants to stretch"
}

` + "`discoveryAppetite`" + ` is your best guess at the user's openness:
- "low" if they want something close to references they named
- "medium" if they want adjacent territory
- "high" if they want surprise

Example user request:
"I love Radiohead's In Rainbows but want something less melancholy, more textural — instrumental preferred. Maybe something with electronic elements but organic-feeling."

Example output:
{
  "mood": "textural and organic, less melancholy than In Rainbows",
  "eraHints": [],
  "sonicQualities": ["textural", "organic", "electronic but warm", "instrumental or near-instrumental"],
  "referenceArtists": ["Radiohead"],
  "avoid": ["heavy melancholy", "vocal-driven"],
  "discoveryAppetite": "medium"
}

Output only the JSON. No prose. No markdown fences.`

const interpretUserPromptTemplate = `Listening request: {{.Query}}

Artists the user already knows well and does not want repeated: {{.AlreadyKnownOrNone}}`

const curateSystemPrompt = `You are a music critic and discovery guide. You receive a structured listening request and a list of candidate albums. Pick the best 5 picks and explain why each fits.

Output ONLY a single JSON object with this exact schema:

{
  "picks": [
    {
      "rank": 1,
      "albumId": "must match one of the input candidate IDs exactly",
      "whyItFits": "2-3 sentences referencing both the user's request and the album's qualities",
      "whatToListenFor": "1-2 sentences naming a specific musical detail to notice"
    }
    // ... requested number of entries total, ranks 1 through the requested count
  ]
}

Rules:
- Use ONLY albumIds present in the candidate list. Do not invent.
- If ` + "`discoveryAppetite`" + ` is ` + "`high`" + `, prefer picks the user is unlikely to have already heard.
- If ` + "`discoveryAppetite`" + ` is ` + "`low`" + `, prefer picks closely tied to the named reference artists.
- ` + "`whyItFits`" + ` should reference the user's mood, sonic qualities, or reference artists by name.
- ` + "`whatToListenFor`" + ` should be a concrete musical observation (an instrument moment, a structural choice, a textural feature) — not generic praise.
- Output ONLY the JSON. No prose. No markdown fences.`

const curateUserPromptTemplate = `Interpreted listening request:
{{.InterpretedJSON}}

Candidate albums:
{{range $i, $a := .Candidates}}
{{add $i 1}}. albumId: {{$a.ID}}
   {{$a.Title}} — {{$a.ArtistName}} ({{$a.Year}})
   Genre: {{$a.Genre}}
   {{if $a.Review.Summary}}Review summary: {{$a.Review.Summary}}{{end}}
{{end}}

Pick {{.PickCount}}. Output the JSON.`

const invalidJSONReminder = "Your last response was not valid JSON in the required shape. Produce only the JSON object, with no prose and no markdown fences."

func renderInterpretUserPrompt(raw string, alreadyKnown []string) (string, error) {
	known := "(none)"
	if cleaned := cleanStrings(alreadyKnown); len(cleaned) > 0 {
		known = strings.Join(cleaned, ", ")
	}
	return renderTemplate(interpretUserPromptTemplate, map[string]string{
		"Query":              strings.TrimSpace(raw),
		"AlreadyKnownOrNone": known,
	})
}

func renderCurateUserPrompt(interpreted data.InterpretedQuery, candidates []*data.Album) (string, error) {
	encoded, err := json.MarshalIndent(interpreted, "", "  ")
	if err != nil {
		return "", fmt.Errorf("discovery: encode interpreted query: %w", err)
	}
	return renderTemplate(curateUserPromptTemplate, struct {
		InterpretedJSON string
		Candidates      []*data.Album
		PickCount       int
	}{
		InterpretedJSON: string(encoded),
		Candidates:      candidates,
		PickCount:       pickCount(candidates),
	})
}

func renderTemplate(src string, data any) (string, error) {
	tpl, err := template.New("prompt").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}).Parse(src)
	if err != nil {
		return "", fmt.Errorf("discovery: parse prompt template: %w", err)
	}
	var out bytes.Buffer
	if err := tpl.Execute(&out, data); err != nil {
		return "", fmt.Errorf("discovery: render prompt template: %w", err)
	}
	return out.String(), nil
}

func cleanStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
