import { Injectable } from '@angular/core';

export interface AlbumProvenance {
  source: 'artist';
  artistId: string;
  artistName: string;
}

// TODO(#5): Add explicit search provenance when direct Search -> Album navigation is supported.

/**
 * Tracks contextual navigation state (album provenance, saved search query,
 * whether the last search produced results) so downstream pages can render
 * contextual "back" affordances.
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
 */
@Injectable({
  providedIn: 'root'
})
export class NavigationContextService {
  private savedSearchQuery: string | null = null;
  private albumProvenance: AlbumProvenance | null = null;
  private hadSearchResults = false;

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

  recordSearchResults(hasResults: boolean): void {
    this.hadSearchResults = hasResults;
  }

  getHadSearchResults(): boolean {
    return this.hadSearchResults;
  }

  clearHadSearchResults(): void {
    this.hadSearchResults = false;
  }
}
