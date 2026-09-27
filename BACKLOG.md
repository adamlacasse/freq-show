# Backlog

## Shipped

- **AI music discovery pipeline** — All 7 phases complete as of 2026-07-25. Natural-language listening requests resolved to ranked album recommendations with editorial reasoning, via Voyage embeddings + HF Inference LLM. Architecture in [`docs/adr/0001-discovery-pipeline-hosting.md`](docs/adr/0001-discovery-pipeline-hosting.md); implementation detail in [`docs/plans/discovery-pipeline-plan.md`](docs/plans/discovery-pipeline-plan.md). To run the real-key smoke test: `DISCOVERY_E2E=1 DISCOVERY_EMBEDDINGS_API_KEY=<key> DISCOVERY_LLM_API_KEY=<key> go test ./pkg/discovery/ -run TestDiscoveryE2E -v -timeout 120s`.

- **Contextual back navigation from album pages** — Shipped 2026-09-27 (PR #4, follow-up fix in PR #9). Album pages opened from an artist page show a `Back to Artist` action; search-originated visits distinguish returning to prior results vs. an empty search screen. Navigation provenance is modeled as a typed variant on `NavigationContextService` rather than a separate ad-hoc flag.

- **Magic link authentication** — Shipped 2026-09-26 (PR #10). Passwordless email login via Resend, backed by new `users`/`login_tokens`/`sessions` SQLite tables. `POST /auth/request` issues a one-time token; `GET`/`POST /auth/verify` confirms and consumes it (split to avoid email-scanner prefetch burning the token) and sets a session cookie. `/discover` stays optionally authenticated: anonymous requests keep the per-IP limit, logged-in requests get a more generous per-user limit. Hardened post-review: CORS credential handling, atomic token consumption, per-email request cooldown, rate-limiter eviction sweep.

## Downstream (ride on `album_embeddings` table)

- **Related Artists** — Artist-level embeddings for "you might also like" suggestions. Foundation is in place; pick up when discovery is confirmed working in production.
- **Themed browsing** — Genre-prototype embeddings for mood/era-based browsing. Same table, same interface.

## In Progress

## UI

- **Expand frontend test coverage** — Coverage is better than the initial MVP, but service-level tests are still missing for `ArtistService` and `AlbumService`, and the UI specs can go deeper on loading states, template rendering, and service interactions.

## Auth / Personalization

- **Personalization follow-ons** — Now that sessions exist (magic link auth), natural next steps: query history, saved picks, and preference memory tied to the logged-in user.

## Data / Integrations

- **Rethink Discogs usage** — The Reviews section currently pulls data from Discogs, which surfaces pressing-specific detail (format, label, catalog number, etc.) rather than general album information. Evaluate whether Discogs is the right source for this use case, or whether a different data source or a narrower Discogs query would better serve the app's goals.

- **Add Spotify deep links for tracks** — Resolve album tracks to Spotify track URLs where possible and show a "Play on Spotify" link from the track listing. Start with outbound deep links only; do not add embedded or in-app playback yet.
