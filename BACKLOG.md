# Backlog

## Shipped

- **AI music discovery pipeline** — All 7 phases complete as of 2026-07-25. Natural-language listening requests resolved to ranked album recommendations with editorial reasoning, via Voyage embeddings + HF Inference LLM. Architecture in [`docs/adr/0001-discovery-pipeline-hosting.md`](docs/adr/0001-discovery-pipeline-hosting.md); implementation detail in [`docs/plans/discovery-pipeline-plan.md`](docs/plans/discovery-pipeline-plan.md). To run the real-key smoke test: `DISCOVERY_E2E=1 DISCOVERY_EMBEDDINGS_API_KEY=<key> DISCOVERY_LLM_API_KEY=<key> go test ./pkg/discovery/ -run TestDiscoveryE2E -v -timeout 120s`.

- **Contextual back navigation from album pages** — Shipped 2026-04-16 (PR #4), with navigation provenance refactored to a typed variant on 2026-09-27 (PR #9). Album pages opened from an artist page show a `Back to Artist` action. Navigation provenance is modeled as a typed variant on `NavigationContextService` rather than a separate ad-hoc flag, preparing the service for direct search return flows.

- **Magic link authentication** — Passwordless email login via Resend, backed by `users`/`login_tokens`/`sessions` SQLite tables, with client-side session management in Angular. Backend provides `POST /auth/request`, `GET`/`POST /auth/verify` (split to avoid email-scanner prefetch burning tokens), `GET /auth/me`, and `POST /auth/logout`. Frontend includes a passwordless magic-link sign-in modal, session state management via `AuthService`, dynamic navigation controls with email display and sign-out, collection routing, rate-limit prompts, and regenerated OpenAPI TypeScript types.

## Downstream (ride on `album_embeddings` table)

- **Related Artists** — Artist-level embeddings for "you might also like" suggestions. Foundation is in place; pick up when discovery is confirmed working in production.
- **Themed browsing** — Genre-prototype embeddings for mood/era-based browsing. Same table, same interface.

## In Progress

## UI

- **Direct Search → Album navigation & return flow** — Allow navigating directly from search results to album detail pages. This will produce `{ source: 'search' }` provenance on `NavigationContextService` and activate the typed return flows ("Back to Search Results" vs. "Back to Search").

- **Expand frontend test coverage** — Coverage is better than the initial MVP, but service-level tests are still missing for `ArtistService` and `AlbumService`, and the UI specs can go deeper on loading states, template rendering, and service interactions.

## Auth / Personalization

- **User-specific collection under auth** — Place collection access and mutation under authentication so collections are strictly user-specific. Enforce session verification on collection mutations (`POST /collections/:userId/albums/:albumId`, update, delete) to ensure users can only modify their own collection, and update frontend routing so "My Collection" resolves to the authenticated user's collection.

- **User profiles & account management** — Enable users to manage their own profile details (display name, bio, avatar, music tastes/preferences) in addition to their record collection.

- **Personalization follow-ons** — Once users can authenticate through the UI, natural next steps: query history, saved picks, and preference memory tied to the logged-in user.

## Data / Integrations

- **Rethink Discogs usage** — The Reviews section currently pulls data from Discogs, which surfaces pressing-specific detail (format, label, catalog number, etc.) rather than general album information. Evaluate whether Discogs is the right source for this use case, or whether a different data source or a narrower Discogs query would better serve the app's goals.

- **Add Spotify deep links for tracks** — Resolve album tracks to Spotify track URLs where possible and show a "Play on Spotify" link from the track listing. Start with outbound deep links only; do not add embedded or in-app playback yet.
