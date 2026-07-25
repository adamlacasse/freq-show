# Backlog

## Shipped

- **AI music discovery pipeline** — All 7 phases complete as of 2026-07-25. Natural-language listening requests resolved to ranked album recommendations with editorial reasoning, via Voyage embeddings + HF Inference LLM. Architecture in [`docs/adr/0001-discovery-pipeline-hosting.md`](docs/adr/0001-discovery-pipeline-hosting.md); implementation detail in [`docs/plans/discovery-pipeline-plan.md`](docs/plans/discovery-pipeline-plan.md). To run the real-key smoke test: `DISCOVERY_E2E=1 DISCOVERY_EMBEDDINGS_API_KEY=<key> DISCOVERY_LLM_API_KEY=<key> go test ./pkg/discovery/ -run TestDiscoveryE2E -v -timeout 120s`.

## Downstream (ride on `album_embeddings` table)

- **Related Artists** — Artist-level embeddings for "you might also like" suggestions. Foundation is in place; pick up when discovery is confirmed working in production.
- **Themed browsing** — Genre-prototype embeddings for mood/era-based browsing. Same table, same interface.

## In Progress

## UI

- **Contextual back navigation from album pages** — The album detail page should support richer return flows based on how the user arrived there. If an album was opened from an artist page, show a `Back to Artist` action that returns to that artist. Search return behavior should also distinguish between going back to prior search results vs. going back to an empty search screen, rather than sending every user to the same generic destination. This likely requires preserving navigation provenance and search state in the frontend.

- **Expand frontend test coverage** — Coverage is better than the initial MVP, but service-level tests are still missing for `ArtistService` and `AlbumService`, and the UI specs can go deeper on loading states, template rendering, and service interactions.

## Auth / Personalization

- **Magic link authentication** — Passwordless email login to unlock personalization and tighten discovery cost protection beyond IP rate limiting. Stack fits existing infrastructure: `users` and `sessions` tables in SQLite, `POST /auth/request` sends a time-limited token link via Resend (free tier), `GET /auth/verify?token=...` creates a session cookie. Start with protecting `/discover` behind optional login; anonymous users still get the IP rate limit. Once sessions exist, query history, saved picks, and preference memory are natural follow-ons.

## Data / Integrations

- **Rethink Discogs usage** — The Reviews section currently pulls data from Discogs, which surfaces pressing-specific detail (format, label, catalog number, etc.) rather than general album information. Evaluate whether Discogs is the right source for this use case, or whether a different data source or a narrower Discogs query would better serve the app's goals.

- **Add Spotify deep links for tracks** — Resolve album tracks to Spotify track URLs where possible and show a "Play on Spotify" link from the track listing. Start with outbound deep links only; do not add embedded or in-app playback yet.
