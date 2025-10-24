import { Component, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Subject, takeUntil, debounceTime, distinctUntilChanged } from 'rxjs';
import { SearchService } from '../../services/search.service';
import { SearchResult, Artist } from '../../models/search.models';

@Component({
  selector: 'app-search',
  standalone: true,
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

  constructor(private searchService: SearchService) {
    // Subscribe to search results
    this.searchService.searchResults$
      .pipe(takeUntil(this.destroy$))
      .subscribe((results: SearchResult | null) => {
        this.searchResults = results;
      });

    // Debounce search input
    this.searchSubject
      .pipe(
        debounceTime(300),
        distinctUntilChanged(),
        takeUntil(this.destroy$)
      )
      .subscribe(query => {
        if (query.trim()) {
          this.performSearch(query.trim());
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
      query,
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

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }
}