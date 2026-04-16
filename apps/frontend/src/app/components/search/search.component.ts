import { Component, OnDestroy, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { Subject, takeUntil, debounceTime, distinctUntilChanged } from 'rxjs';
import { SearchService } from '../../services/search.service';
import { NavigationContextService } from '../../services/navigation-context.service';
import type { components } from '../../models/openapi-types.generated';

type SearchResult = components['schemas']['SearchResult'];
type Artist = components['schemas']['SearchArtist'];

@Component({
    selector: 'app-search',
    imports: [CommonModule, FormsModule],
    templateUrl: './search.component.html',
    styleUrls: ['./search.component.css']
})
export class SearchComponent implements OnDestroy, OnInit {
  searchQuery = '';
  searchResults: SearchResult | null = null;
  searchError: string | null = null;
  isSearching = false;
  private destroy$ = new Subject<void>();
  private searchSubject = new Subject<string>();

  constructor(
    private searchService: SearchService,
    private router: Router,
    private navigationContextService: NavigationContextService
  ) {
    // Subscribe to search results
    this.searchService.searchResults$
      .pipe(takeUntil(this.destroy$))
      .subscribe((results: SearchResult | null) => {
        this.searchResults = results;
      });

    this.searchService.searchError$
      .pipe(takeUntil(this.destroy$))
      .subscribe((error: string | null) => {
        this.searchError = error;
      });

    this.searchService.searchReset$
      .pipe(takeUntil(this.destroy$))
      .subscribe(() => {
        this.resetComponentState(true);
      });

    // Debounce search input. 800ms gives MusicBrainz (~1 req/s limit) enough
    // breathing room even if the user types steadily. Also require at least
    // 2 characters to avoid unnecessary round-trips for single keystrokes.
    this.searchSubject
      .pipe(
        debounceTime(800),
        distinctUntilChanged(),
        takeUntil(this.destroy$)
      )
      .subscribe(query => {
        const trimmed = query.trim();
        if (trimmed.length >= 2) {
          this.performSearch(trimmed);
        } else {
          this.clearSearch();
        }
      });
  }

  onSearchInput(event: any): void {
    const query = event.target.value;
    this.searchQuery = query;
    this.searchSubject.next(query);
  }

  performSearch(query: string): void {
    this.isSearching = true;
    this.searchError = null;
    this.searchService.searchArtists({
      q: query,
      limit: 10
    }).subscribe({
      next: () => {
        this.isSearching = false;
      },
      error: (error: any) => {
        console.error('Search failed:', error);
        this.isSearching = false;
      }
    });
  }

  clearSearch(): void {
    this.searchService.clearSearchState();
    this.resetComponentState(false);
  }

  retrySearch(): void {
    const trimmed = this.searchQuery.trim();
    if (trimmed.length >= 2) {
      this.performSearch(trimmed);
    }
  }

  getArtistDisplayInfo(artist: Artist): string {
    let info = artist.name;
    
    if (artist.disambiguation) {
      info += ` (${artist.disambiguation})`;
    }
    
    if (artist.country) {
      info += ` • ${artist.country}`;
    }
    
    if (artist.type) {
      info += ` • ${artist.type}`;
    }
    
    return info;
  }

  getLifeSpanDisplay(artist: Artist): string {
    const lifeSpan = artist.lifeSpan;
    if (!lifeSpan.begin && !lifeSpan.end) {
      return '';
    }
    
    let span = '';
    if (lifeSpan.begin) {
      span = lifeSpan.begin;
    }
    
    if (lifeSpan.end || lifeSpan.ended) {
      span += ' – ';
      if (lifeSpan.end) {
        span += lifeSpan.end;
      }
    } else if (lifeSpan.begin) {
      span += ' – present';
    }
    
    return span;
  }

  trackByArtistId(index: number, artist: Artist): string {
    return artist.id;
  }

  onArtistClick(artist: Artist): void {
    this.navigationContextService.saveSearchQuery(this.searchQuery);
    this.navigationContextService.recordSearchResults(
      !!(this.searchResults && this.searchResults.artists.length > 0)
    );
    this.router.navigate(['/artists', artist.id]);
  }

  ngOnInit(): void {
    const savedQuery = this.navigationContextService.getSavedSearchQuery();
    if (savedQuery) {
      this.searchQuery = savedQuery;
      this.performSearch(savedQuery);
      this.navigationContextService.clearSavedSearchQuery();
    }
  }

  ngOnDestroy(): void {
    this.resetComponentState(true);
    this.searchService.clearSearchState();
    this.destroy$.next();
    this.destroy$.complete();
  }

  private resetComponentState(clearQuery: boolean): void {
    this.searchResults = null;
    this.searchError = null;
    this.isSearching = false;

    if (clearQuery) {
      this.searchQuery = '';
    }
  }
}
