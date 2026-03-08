import { TestBed } from '@angular/core/testing';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { SearchService } from './search.service';

describe('SearchService', () => {
  let service: SearchService;
  let httpMock: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule]
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
});
