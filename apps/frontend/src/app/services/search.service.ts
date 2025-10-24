import { Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, BehaviorSubject } from 'rxjs';
import { SearchResult, SearchParams } from '../models/search.models';

@Injectable({
  providedIn: 'root'
})
export class SearchService {
  private apiUrl = 'http://localhost:8080';
  private searchResultsSubject = new BehaviorSubject<SearchResult | null>(null);
  
  public searchResults$ = this.searchResultsSubject.asObservable();
  public isSearching = false;

  constructor(private http: HttpClient) {}

  searchArtists(params: SearchParams): Observable<SearchResult> {
    this.isSearching = true;
    
    let httpParams = new HttpParams()
      .set('q', params.query);
    
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