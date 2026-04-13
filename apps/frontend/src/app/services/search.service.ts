import { Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, BehaviorSubject, Subject, Subscription, shareReplay } from 'rxjs';
import type { components, operations } from '../models/openapi-types.generated';

type SearchResult = components['schemas']['SearchResult'];
type SearchArtistsQuery = operations['searchArtists']['parameters']['query'];

@Injectable({
  providedIn: 'root'
})
export class SearchService {
  private apiUrl = '/api';
  private searchResultsSubject = new BehaviorSubject<SearchResult | null>(null);
  private searchErrorSubject = new BehaviorSubject<string | null>(null);
  private searchResetSubject = new Subject<void>();
  private activeSearchSubscription?: Subscription;
  
  public searchResults$ = this.searchResultsSubject.asObservable();
  public searchError$ = this.searchErrorSubject.asObservable();
  public searchReset$ = this.searchResetSubject.asObservable();
  public isSearching = false;

  constructor(private http: HttpClient) {}

  searchArtists(params: SearchArtistsQuery): Observable<SearchResult> {
    this.isSearching = true;
    this.searchErrorSubject.next(null);
    this.searchResultsSubject.next(null);
    this.activeSearchSubscription?.unsubscribe();
    
    let httpParams = new HttpParams()
      .set('q', params.q);
    
    if (params.limit) {
      httpParams = httpParams.set('limit', params.limit.toString());
    }
    
    if (params.offset) {
      httpParams = httpParams.set('offset', params.offset.toString());
    }

    const searchObservable = this.http.get<SearchResult>(`${this.apiUrl}/search`, { params: httpParams }).pipe(
      shareReplay({ bufferSize: 1, refCount: true })
    );
    
    this.activeSearchSubscription = searchObservable.subscribe({
      next: (result) => {
        this.searchResultsSubject.next(result);
        this.isSearching = false;
        this.activeSearchSubscription = undefined;
      },
      error: () => {
        this.searchErrorSubject.next('Search failed. Try again.');
        this.isSearching = false;
        this.activeSearchSubscription = undefined;
      }
    });

    return searchObservable;
  }

  clearSearchResults(): void {
    this.activeSearchSubscription?.unsubscribe();
    this.activeSearchSubscription = undefined;
    this.searchResultsSubject.next(null);
    this.searchErrorSubject.next(null);
    this.isSearching = false;
  }

  clearSearchState(): void {
    this.clearSearchResults();
  }

  requestSearchReset(): void {
    this.clearSearchState();
    this.searchResetSubject.next();
  }

  getCurrentResults(): SearchResult | null {
    return this.searchResultsSubject.value;
  }

  getCurrentError(): string | null {
    return this.searchErrorSubject.value;
  }
}
