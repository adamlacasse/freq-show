import { Component, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { Subject, takeUntil, debounceTime, distinctUntilChanged } from 'rxjs';
import { SearchService } from '../../services/search.service';
import type { components } from '../../models/openapi-types.generated';

type SearchResult = components['schemas']['SearchResult'];
type Artist = components['schemas']['SearchArtist'];

@Component({
    selector: 'app-search',
    imports: [CommonModule, FormsModule],
    templateUrl: './search.component.html',
    styleUrls: ['./search.component.css']
})
export class SearchComponent implements OnDestroy {
  searchQuery = '';
  searchResults: SearchResult | null = null;
  isSearching = false;
  private destroy$ = new Subject<void>();
  private searchSubject = new Subject<string>();

  constructor(
    private searchService: SearchService,
    private router: Router
  ) {
    // Subscribe to search results
    this.searchService.searchResults$
      .pipe(takeUntil(this.destroy$))
      .subscribe((results: SearchResult | null) => {
        this.searchResults = results;
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
    this.searchService.clearSearchResults();
    this.searchResults = null;
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
    this.router.navigate(['/artists', artist.id]);
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }
}