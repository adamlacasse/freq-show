# Navigation Return Flows

This document is authoritative for the frontend return-navigation behavior on search, artist, and album pages.

## Summary

FreqShow should use deterministic, app-state-driven return navigation instead of a single hardcoded `router.navigate(['/'])` flow. The frontend should preserve search context in memory during the current SPA session, track when an album page was reached from an artist page, and expose explicit return actions that match how the user arrived.

This feature does not change backend behavior or the OpenAPI contract.

## Required Behavior

### Search state preservation

- Preserve the current search query in frontend state alongside search results and search errors.
- Preserve the current search query and results across navigation from home -> artist -> album during the current SPA session.
- Do not clear remembered search state automatically when leaving `/`.
- Keep explicit reset behavior only for user actions that intentionally start over:
  - header brand link
  - header `Home` link
  - `Back to Search`
  - explicit clear/reset actions inside the search UI
- A remembered zero-result search still counts as restorable search state.
- Remembered search state is in-memory only. Reloads, fresh tabs, and direct deep links start with no remembered search context.

### Artist page return behavior

- Artist pages must no longer use a generic `goBack()` that always navigates to `/`.
- If remembered search results exist, the artist page should show `Back to Search Results`.
- If remembered search results do not exist, the artist page should show `Back to Search`.
- `Back to Search Results` must navigate to `/` without clearing the remembered search query or results.
- `Back to Search` must clear remembered search state before navigating to `/`.
- The same return CTA behavior must appear in both:
  - the normal loaded page
  - the error state

### Album page return behavior

- Album pages must no longer use a single generic `Back to Search` action for every case.
- Album pages must always show exactly one search-return CTA:
  - `Back to Search Results` when remembered search results exist
  - `Back to Search` otherwise
- If the album was reached from an artist page during the current SPA session, the album page must also show `Back to Artist`.
- When both conditions apply, show both buttons at once.
- `Back to Artist` must navigate to the originating artist route captured at the time the album was opened.
- The same CTA set and ordering must appear in both:
  - the top-of-page navigation row
  - the error state CTA row
- CTA ordering must be:
  - `Back to Artist` first when present
  - search-return CTA second

### Provenance tracking

- When a user opens an album from an artist page, the frontend must capture album-entry provenance in memory.
- Provenance must include:
  - album id
  - artist id
  - optional artist name for display if useful
- Provenance is keyed to the specific album route being opened.
- Provenance is in-memory only and must not be encoded in route params or query params.
- Direct album entry, reloads, and fresh sessions must not fabricate `Back to Artist`.

## Frontend Interfaces

### Search state service

The search service should expose enough state to support deterministic return behavior.

Required capabilities:

- store and retrieve the current query string
- store and retrieve the current search result payload
- store and retrieve the current search error
- explicitly clear all remembered search state
- report whether search results are currently restorable

Minimum expected API shape:

- `getCurrentQuery(): string`
- `setCurrentQuery(query: string): void`
- `getCurrentResults(): SearchResult | null`
- `getCurrentError(): string | null`
- `hasRestorableResults(): boolean`
- existing reset/clear methods must clear query, results, and errors together

### Return provenance service

Introduce a small in-memory frontend service dedicated to return provenance for album pages.

Minimum expected API shape:

- `setAlbumOrigin(albumId: string, artistId: string, artistName?: string): void`
- `getAlbumOrigin(albumId: string): AlbumOrigin | null`
- `clearAlbumOrigin(albumId?: string): void`

Expected stored type:

```ts
type AlbumOrigin = {
  albumId: string;
  artistId: string;
  artistName?: string;
};
```

## Component-Level Expectations

### Search component

- Hydrate the search input from remembered query state when the component is created.
- Push input changes into the search state service as the user types.
- Do not clear remembered state on component destroy.
- Explicit user-driven clear/reset actions should still clear both local UI state and shared search state.

### App shell

- The app shell must stop clearing search state solely because the router left `/`.
- Header home navigation remains the explicit blank-search action and should continue clearing remembered state before navigating home.

### Artist detail page

- When an album card is clicked, capture album provenance before routing to the album page.
- Use explicit destination methods instead of a generic `goBack()`.

### Album detail page

- Resolve visible return CTAs from two independent pieces of state:
  - remembered search results
  - artist-origin provenance for the current album id
- Do not rely on `Location.back()` or browser history stack heuristics for the primary feature behavior.

## Test Requirements

### Service tests

- Search service:
  - query persistence
  - reset clearing query/results/error
  - `hasRestorableResults()` returning true for both populated and zero-result searches
- Return provenance service:
  - set/get for a specific album id
  - no provenance for unrelated album ids
  - clear behavior

### Component tests

- Search component restores remembered query/results when recreated.
- Search component destroy does not clear remembered search state.
- App component no longer clears search state on non-home route transitions.
- App component still clears search state for explicit home-link usage.
- Artist detail page shows `Back to Search Results` when restorable search state exists.
- Artist detail page shows `Back to Search` when restorable search state does not exist.
- Artist detail page uses the same CTA behavior in the error state.
- Album detail page shows both `Back to Artist` and `Back to Search Results` after search -> artist -> album navigation.
- Album detail page shows `Back to Artist` and `Back to Search` when artist provenance exists but search state does not.
- Album detail page hides `Back to Artist` on direct album entry.
- Album detail page mirrors the same CTA behavior in the error state.

### Flow test

- Cover the end-to-end in-app flow:
  - run a search
  - open an artist
  - open an album from that artist
  - return to the artist
  - return to preserved search results

## Defaults and Assumptions

- Remembered search state survives in-app route changes only within the current SPA session.
- Browser reloads and direct deep links reset remembered search state and album provenance.
- On direct artist or album entry with no remembered state:
  - artist page shows `Back to Search`
  - album page shows `Back to Search`
  - album page does not show `Back to Artist`
- The feature should prefer explicit, deterministic destinations over browser-history-dependent behavior.
