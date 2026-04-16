import { ComponentFixture, TestBed } from '@angular/core/testing';
import { Observable, BehaviorSubject, Subject, of } from 'rxjs';
import { Router } from '@angular/router';
import type { components } from '../../models/openapi-types.generated';

import { SearchComponent } from './search.component';
import { SearchService } from '../../services/search.service';
import { NavigationContextService } from '../../services/navigation-context.service';

type SearchResult = components['schemas']['SearchResult'];
type Artist = components['schemas']['SearchArtist'];

describe('SearchComponent', () => {
  let fixture: ComponentFixture<SearchComponent>;
  let component: SearchComponent;

  let searchResultsSubject: BehaviorSubject<SearchResult | null>;
  let searchErrorSubject: BehaviorSubject<string | null>;
  let searchResetSubject: Subject<void>;

  let searchService: {
    searchResults$: Observable<SearchResult | null>;
    searchError$: Observable<string | null>;
    searchReset$: Observable<void>;
    searchArtists: jasmine.Spy;
    clearSearchState: jasmine.Spy;
  };
  let navigationContextService: jasmine.SpyObj<NavigationContextService>;
  let router: jasmine.SpyObj<Router>;

  function createComponent(): void {
    fixture = TestBed.createComponent(SearchComponent);
    component = fixture.componentInstance;
  }

  beforeEach(async () => {
    searchResultsSubject = new BehaviorSubject<SearchResult | null>(null);
    searchErrorSubject = new BehaviorSubject<string | null>(null);
    searchResetSubject = new Subject<void>();

    searchService = {
      searchResults$: searchResultsSubject.asObservable(),
      searchError$: searchErrorSubject.asObservable(),
      searchReset$: searchResetSubject.asObservable(),
      searchArtists: jasmine.createSpy('searchArtists').and.returnValue(
        of({ artists: [], offset: 0, count: 0 } as SearchResult)
      ),
      clearSearchState: jasmine.createSpy('clearSearchState')
    };

    navigationContextService = jasmine.createSpyObj<NavigationContextService>(
      'NavigationContextService',
      ['getSavedSearchQuery', 'clearSavedSearchQuery', 'saveSearchQuery', 'recordSearchResults']
    );
    navigationContextService.getSavedSearchQuery.and.returnValue(null);

    router = jasmine.createSpyObj<Router>('Router', ['navigate']);

    await TestBed.configureTestingModule({
      imports: [SearchComponent],
      providers: [
        { provide: SearchService, useValue: searchService },
        { provide: NavigationContextService, useValue: navigationContextService },
        { provide: Router, useValue: router }
      ]
    }).compileComponents();
  });

  it('restores a saved query on init, performs a search, and clears the saved query', () => {
    navigationContextService.getSavedSearchQuery.and.returnValue('Radio');
    createComponent();
    const performSearchSpy = spyOn(component, 'performSearch').and.callThrough();

    fixture.detectChanges();

    expect(component.searchQuery).toBe('Radio');
    expect(performSearchSpy).toHaveBeenCalledOnceWith('Radio');
    expect(navigationContextService.clearSavedSearchQuery).toHaveBeenCalledTimes(1);
  });

  it('does nothing on init when no saved query is available', () => {
    navigationContextService.getSavedSearchQuery.and.returnValue(null);
    createComponent();
    const performSearchSpy = spyOn(component, 'performSearch').and.callThrough();

    fixture.detectChanges();

    expect(component.searchQuery).toBe('');
    expect(performSearchSpy).not.toHaveBeenCalled();
    expect(navigationContextService.clearSavedSearchQuery).not.toHaveBeenCalled();
  });

  it('saves the last executed query when an artist is clicked', () => {
    createComponent();
    fixture.detectChanges();

    const artist: Artist = {
      id: 'artist-1',
      name: 'Radiohead',
      lifeSpan: {},
      aliases: []
    };
    component.searchResults = { artists: [artist], offset: 0, count: 1 };

    component.performSearch('Radio');
    component.searchQuery = 'Radiohead';
    component.onArtistClick(artist);

    expect(navigationContextService.saveSearchQuery).toHaveBeenCalledWith('Radio');
    expect(navigationContextService.recordSearchResults).toHaveBeenCalledWith(true);
  });
});
