import { Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, BehaviorSubject } from 'rxjs';
import type { components, operations } from '../models/openapi-types.generated';

type SearchResult = components['schemas']['SearchResult'];
type SearchArtistsQuery = operations['searchArtists']['parameters']['query'];

@Injectable({
  providedIn: 'root'
})
export class SearchService {
  private apiUrl = '/api';
  private searchResultsSubject = new BehaviorSubject<SearchResult | null>(null);
  
  public searchResults$ = this.searchResultsSubject.asObservable();
  public isSearching = false;

  constructor(private http: HttpClient) {}

  searchArtists(params: SearchArtistsQuery): Observable<SearchResult> {
    this.isSearching = true;
    
    let httpParams = new HttpParams()
      .set('q', params.q);
    
    if (params.limit) {
      httpParams = httpParams.set('limit', params.limit.toString());
    }
    
    if (params.offset) {
      httpParams = httpParams.set('offset', params.offset.toString());
    }

    const searchObservable = this.http.get<SearchResult>(`${this.apiUrl}/search`, { params: httpParams });
    
    searchObservable.subscribe({
      next: (result) => {
        this.searchResultsSubject.next(result);
        this.isSearching = false;
      },
      error: () => {
        this.isSearching = false;
      }
    });

    return searchObservable;
  }

  clearSearchResults(): void {
    this.searchResultsSubject.next(null);
  }

  getCurrentResults(): SearchResult | null {
    return this.searchResultsSubject.value;
  }
}