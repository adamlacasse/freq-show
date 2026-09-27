import { Injectable } from '@angular/core';

/**
 * Describes how the user arrived at an album page, as a discriminated union
 * so every entry point describes its origin through the same typed channel
 * (see issue #5). The `search` variant is not yet produced by any navigation
 * flow -- today's only path to an album is Search -> Artist -> Album, which
 * always yields `artist` provenance -- but it lets `AlbumDetailComponent`
 * render the correct "Back to Search" / "Back to Search Results" label as
 * soon as a direct Search -> Album flow is added, without another shape
 * change.
 */
export type AlbumProvenance =
  | { source: 'artist'; artistId: string; artistName: string }
  | { source: 'search'; query: string; hadResults: boolean };

/**
 * Tracks contextual navigation state (album provenance, saved search query)
 * so downstream pages can render contextual "back" affordances.
 *
 * State is held in memory on a root-provided singleton and is therefore
 * session-scoped: it does NOT survive a page reload or a direct deep-link
 * into an album URL. Consumers should treat a missing provenance as "unknown
 * origin" and fall back to a safe default label rather than assuming the user
 * arrived via any particular route.
 *
 * Because fields are module-global until a consumer reads-and-clears them,
 * setters on this service are effectively one-shot handoffs between a
 * source page (e.g. artist detail) and its immediate navigation target
 * (e.g. album detail). Writing provenance and then not navigating to the
 * target will leave stale state that the next consumer will pick up.
 *
 * Album provenance is intentionally the single source of truth for the
 * album back button. An earlier version of this service also tracked a
 * standalone `hadSearchResults` boolean, set on a search -> artist
 * navigation and read back on the *next* album page visit regardless of
 * how that visit was reached. Because the flag wasn't tied to the
 * provenance it was meant to describe, it could linger across an
 * unrelated navigation (e.g. search -> artist -> home -> collection ->
 * album) and mislabel the back button as "Back to Search Results" for a
 * visit that had nothing to do with that earlier search. Model any new
 * "how did the user get here" signal as an `AlbumProvenance` variant
 * rather than a side-channel flag.
 */
@Injectable({
  providedIn: 'root'
})
export class NavigationContextService {
  private savedSearchQuery: string | null = null;
  private albumProvenance: AlbumProvenance | null = null;

  saveSearchQuery(query: string): void {
    this.savedSearchQuery = query.trim() || null;
  }

  getSavedSearchQuery(): string | null {
    return this.savedSearchQuery;
  }

  clearSavedSearchQuery(): void {
    this.savedSearchQuery = null;
  }

  setAlbumProvenance(provenance: AlbumProvenance): void {
    this.albumProvenance = provenance;
  }

  getAlbumProvenance(): AlbumProvenance | null {
    return this.albumProvenance;
  }

  clearAlbumProvenance(): void {
    this.albumProvenance = null;
  }
}
