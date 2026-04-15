# Backlog

## UI

- **Contextual back navigation from album pages** — The album detail page should support richer return flows based on how the user arrived there. If an album was opened from an artist page, show a `Back to Artist` action that returns to that artist. Search return behavior should also distinguish between going back to prior search results vs. going back to an empty search screen, rather than sending every user to the same generic destination. This likely requires preserving navigation provenance and search state in the frontend.

- **Expand frontend test coverage** — Coverage is better than the initial MVP, but service-level tests are still missing for `ArtistService` and `AlbumService`, and the UI specs can go deeper on loading states, template rendering, and service interactions.

## Data / Integrations

- **Rethink Discogs usage** — The Reviews section currently pulls data from Discogs, which surfaces pressing-specific detail (format, label, catalog number, etc.) rather than general album information. Evaluate whether Discogs is the right source for this use case, or whether a different data source or a narrower Discogs query would better serve the app's goals.

- **Add Spotify deep links for tracks** — Resolve album tracks to Spotify track URLs where possible and show a "Play on Spotify" link from the track listing. Start with outbound deep links only; do not add embedded or in-app playback yet.
