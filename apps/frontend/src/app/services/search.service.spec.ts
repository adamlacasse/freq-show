import { TestBed } from '@angular/core/testing';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { SearchService } from './search.service';
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http';

describe('SearchService', () => {
  let service: SearchService;
  let httpMock: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
    imports: [],
    providers: [provideHttpClient(withInterceptorsFromDi()), provideHttpClientTesting()]
});
    service = TestBed.inject(SearchService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
  });

  it('should issue only one HTTP request for multiple subscribers', () => {
    const searchResult = {
      artists: [],
      count: 0,
      offset: 0,
      limit: 10
    };

    const observable = service.searchArtists({ q: 'bowie', limit: 10 });

    observable.subscribe();
    observable.subscribe();

    const req = httpMock.expectOne('/api/search?q=bowie&limit=10');
    expect(req.request.method).toBe('GET');

    req.flush(searchResult);
  });

  it('should clear stale state before a new search and expose errors on failure', () => {
    const observable = service.searchArtists({ q: 'bowie', limit: 10 });

    expect(service.getCurrentResults()).toBeNull();
    expect(service.getCurrentError()).toBeNull();
    expect(service.isSearching).toBeTrue();

    observable.subscribe({
      error: () => {
        // expected below
      }
    });

    const req = httpMock.expectOne('/api/search?q=bowie&limit=10');
    req.flush(
      { error: 'search failed' },
      { status: 500, statusText: 'Server Error' }
    );

    expect(service.getCurrentResults()).toBeNull();
    expect(service.getCurrentError()).toBe('Search failed. Try again.');
    expect(service.isSearching).toBeFalse();
  });

  it('should clear search state explicitly', () => {
    service.searchArtists({ q: 'bowie', limit: 10 }).subscribe({
      error: () => {
        // not reached
      }
    });

    const req = httpMock.expectOne('/api/search?q=bowie&limit=10');

    req.flush({
      artists: [],
      count: 0,
      offset: 0,
      limit: 10
    });

    service.clearSearchState();

    expect(service.getCurrentResults()).toBeNull();
    expect(service.getCurrentError()).toBeNull();
    expect(service.isSearching).toBeFalse();
  });
});
