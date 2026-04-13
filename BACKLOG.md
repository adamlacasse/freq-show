# Backlog

## UI

- **Remove placeholder text** — Audit all placeholder/dummy content across the UI and remove anything that doesn't reflect current functionality. The app should only describe what it actually does. This includes the "coming soon" nav links (Artists, Albums, Genres, Reviews) in the header — if these pages aren't being built imminently, the links should be removed.

- **Reset search state on navigation** — When the user navigates away from search (e.g. hits the Home button), the search query and results should be cleared so they don't persist unexpectedly on return.

- **Resolve related artists** — The artist detail page displays related artist IDs as raw strings instead of resolving them to names. These should either be fetched and displayed as clickable links, or hidden until they can be.

- **Add retry UI on errors** — When an API request fails, the user sees a generic message with no way to recover other than refreshing the browser. Add a "Try again" button to error states on search, artist detail, and album detail pages.

- **Client-side cache for detail pages** — Artist and album detail pages re-fetch from the server on every navigation. A lightweight in-memory cache (e.g. last 20 viewed items) would make back-navigation feel instant and reduce unnecessary API calls.

- **Expand frontend test coverage** — Nearly all component tests are shallow "creates successfully" stubs with no assertions on error handling, loading states, template rendering, or service interactions. The search service has one meaningful test; the artist and album services have none.

## Data / Integrations

- **Rethink Discogs usage** — The Reviews section currently pulls data from Discogs, which surfaces pressing-specific detail (format, label, catalog number, etc.) rather than general album information. Evaluate whether Discogs is the right source for this use case, or whether a different data source or a narrower Discogs query would better serve the app's goals.
