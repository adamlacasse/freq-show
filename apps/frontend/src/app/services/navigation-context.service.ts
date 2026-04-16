import { Injectable } from '@angular/core';

export interface AlbumProvenance {
  source: 'artist';
  artistId: string;
  artistName: string;
}

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
