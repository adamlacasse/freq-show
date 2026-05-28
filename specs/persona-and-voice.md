# Persona and Voice

This document is authoritative for who FreqShow's user-facing surface serves and the voice it speaks in.

## Summary

FreqShow's in-app UX serves exactly one persona: **the music nerd**. The app's product surface — every screen, every string, every navigational affordance — must address this persona and only this persona. Engineer-facing and hiring-manager-facing framing (architecture overviews, "what's working," "current sources," API-doc references, technical capability lists) does not belong in the app and is routed out to **adamlacasse.dev**, which links into the live app and explains how it works for those audiences.

The repo `README.md` continues to address developers landing on the GitHub project; this spec governs the **runtime app surface**, not developer docs.

## Personas

### In scope: the music nerd

A wandering listener who reads liner notes and falls into rabbit holes. Their typical entry is a specific curiosity ("which Yes albums does Chris Squire NOT play on?"), and their typical session moves laterally across people, releases, scenes, and labels via dense interlinks — closer in spirit to Wikipedia or AllMusic than to a streaming app's recommendation feed.

### Out of scope (in this codebase): engineer, hiring manager, portfolio reviewer

These audiences are served on **adamlacasse.dev**, not in the FreqShow app. Content that exists primarily to explain the app's construction, demonstrate its completeness, advertise its data sources, or showcase its developer must not appear in the app's user-facing surface.

## Required Behavior

### Voice

- All user-facing copy must speak to a listener exploring music, not to an evaluator inspecting a product.
- Schema, database, and implementation vocabulary must not surface in user-visible strings. Forbidden examples include but are not limited to: `in our database`, `in our sources`, `release group`, `primary type`, `secondary types`, `classifications` (as a category label), `tags` (when used as a data-model term distinct from genres), `the hydrated album index`, `ranked picks`, `avoid filters`.
- Hosting and infrastructure conditions (cold starts, deployment state, service health) must not be narrated to the user in server terms. If a slow-start experience must be communicated, frame it from the listener's point of view.
- Status framing that addresses the maturity or completeness of the app (`what is live now`, `core experiences`, `already working end to end`, `current scope`) must not appear in user-facing copy.

### Attribution

- Attribution to upstream data sources (MusicBrainz, Wikipedia, Discogs, etc.) is permitted and is required where the source's license demands it.
- Attribution must be framed as tasteful credit (`Bio via Wikipedia`, `Genres via MusicBrainz`) at the point of the contributed content, not as a build manifest in the chrome of the app.
- Attribution links, when present, must target the human-facing page being credited (the Wikipedia article, the MusicBrainz entity, the Discogs page). Links to API documentation are prohibited as user-facing affordances.

### Linking

- Every named entity surfaced from the data layer (artists, albums, labels, tracks, related people, etc.) is a candidate for a link to the entity's own page. The persona expects to wander via these links.
- This spec does not require any specific link to be live, but it forbids the opposite practice: surfacing API-doc links, source-shopping links, or other developer-oriented external links as primary interactive affordances on the user-facing surface.

### Out-of-persona content

If product copy is being added or changed, the implementer must check it against this spec. If a string addresses an engineer or hiring manager, it does not ship. If it must exist somewhere, it belongs at adamlacasse.dev.

## Worked Examples

| Out of persona | In persona |
| --- | --- |
| `Source data: MusicBrainz` (CTA linking to MusicBrainz API docs) | removed; attribution lives as tasteful credit at the point of contributed content |
| `What is live now` + bulleted status panel on the home page | removed; the home page is an entrance to wandering, not a status board |
| `FreqShow stitches together open data sources that match the app's current scope.` | removed |
| `FreqShow will search the hydrated album index and return five ranked picks.` | `Describe what you want to hear. FreqShow will pick five records that fit.` |
| `No picks matched after the avoid filters.` | `No records matched what you asked for. Try loosening one of the constraints.` |
| `We couldn't find biographical information for this artist in our sources.` | `No biography yet for this artist.` |
| `Genre tags haven't been assigned to this artist yet in our database.` | `No genres listed for this artist yet.` |
| `Genre data sourced from MusicBrainz community tags` | `Genres via MusicBrainz` |
| `Information sourced from Wikipedia` | `Bio via Wikipedia` |
| `Service is waking up — first request may be slow.` | `Just a moment — pulling the first record off the shelf.` |
| `Additional classifications:` | `Also classified as:` |
| `Release Type` (field label) | `Type` |
| `First Release Date` (field label) | `First released` |
| `Genres & Tags` (section header) | `Genres` |

## Where the engineer / hiring-manager narrative belongs

Architecture overviews, technical capability lists, "how it's built" prose, monorepo structure, language/framework choices, and developer-onboarding instructions belong on **adamlacasse.dev**, which links into the live FreqShow app. The README.md in this repo serves developers landing on the GitHub project and may continue to address them; the runtime app may not.

## How to apply

When adding or changing user-facing strings, components, or screens, an implementer (human or agent) must verify against this spec:

- Read every new or changed user-visible string and ask: would a listener wandering for music care about this? If no, cut or reframe.
- If a string exposes data-model vocabulary, rewrite it in everyday English.
- If a string narrates the app's own construction or status, remove it.
- If attribution is being added, frame it as credit at the point of use; never link to API docs.

Conflicts with other specs are resolved by `specs/index.md`'s precedence rules.
