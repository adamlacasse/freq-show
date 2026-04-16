import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { provideHttpClient } from '@angular/common/http';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, ParamMap, Router, convertToParamMap, provideRouter } from '@angular/router';
import { BehaviorSubject, of, throwError } from 'rxjs';

import { AlbumService } from '../../services/album.service';
import { NavigationContextService, AlbumProvenance } from '../../services/navigation-context.service';
import { AlbumDetailComponent } from './album-detail.component';

describe('AlbumDetailComponent', () => {
  const albumId = 'album-1';
  const mockAlbum = {
    id: albumId,
    title: 'Test Album',
    artistId: 'artist-1',
    artistName: 'Test Artist',
    primaryType: 'Album',
    secondaryTypes: [],
    firstReleaseDate: '1999-01-01',
    year: 1999,
    genre: '',
    label: '',
    tracks: [],
    review: {
      source: '',
      author: '',
      rating: 0,
      summary: '',
      text: '',
      url: ''
    },
    coverUrl: ''
  };

  describe('with cached service responses', () => {
    let fixture: ComponentFixture<AlbumDetailComponent>;
    let routeParams$: BehaviorSubject<ParamMap>;
    let httpMock: HttpTestingController;

    beforeEach(async () => {
      routeParams$ = new BehaviorSubject(convertToParamMap({ id: albumId }));

      await TestBed.configureTestingModule({
        imports: [AlbumDetailComponent],
        providers: [
          provideRouter([]),
          provideHttpClient(),
          provideHttpClientTesting(),
          {
            provide: ActivatedRoute,
            useValue: { paramMap: routeParams$.asObservable() }
          }
        ]
      }).compileComponents();

      fixture = TestBed.createComponent(AlbumDetailComponent);
      httpMock = TestBed.inject(HttpTestingController);
      fixture.detectChanges();
    });

    afterEach(() => {
      httpMock.verify();
    });

    it('reuses the cached album on repeat navigation', () => {
      const firstRequest = httpMock.expectOne(`/api/albums/${albumId}`);
      firstRequest.flush(mockAlbum);

      routeParams$.next(convertToParamMap({ id: albumId }));
      fixture.detectChanges();

      httpMock.expectNone(`/api/albums/${albumId}`);
      expect(fixture.nativeElement.textContent).toContain('Test Album');
    });
  });

  describe('retry behavior', () => {
    let fixture: ComponentFixture<AlbumDetailComponent>;
    let albumService: jasmine.SpyObj<AlbumService>;
    let routeParams$: BehaviorSubject<ParamMap>;

    beforeEach(async () => {
      routeParams$ = new BehaviorSubject(convertToParamMap({ id: albumId }));
      albumService = jasmine.createSpyObj<AlbumService>('AlbumService', ['getAlbum']);
      const navContextService = jasmine.createSpyObj<NavigationContextService>(
        'NavigationContextService',
        ['getAlbumProvenance', 'clearAlbumProvenance', 'getHadSearchResults', 'clearHadSearchResults']
      );
      navContextService.getAlbumProvenance.and.returnValue(null);
      navContextService.getHadSearchResults.and.returnValue(false);
      spyOn(console, 'error');

      albumService.getAlbum.and.returnValues(
        throwError(() => new Error('boom')),
        of(mockAlbum)
      );

      await TestBed.configureTestingModule({
        imports: [AlbumDetailComponent],
        providers: [
          provideRouter([]),
          {
            provide: ActivatedRoute,
            useValue: { paramMap: routeParams$.asObservable() }
          },
          {
            provide: AlbumService,
            useValue: albumService
          },
          { provide: NavigationContextService, useValue: navContextService }
        ]
      }).compileComponents();

      fixture = TestBed.createComponent(AlbumDetailComponent);
      fixture.detectChanges();
    });

    it('shows a retry action when loading fails and reloads on click', () => {
      expect(fixture.nativeElement.textContent).toContain('Failed to load album information.');

      const buttons = Array.from(fixture.nativeElement.querySelectorAll('button')) as HTMLButtonElement[];
      const retryButton = buttons.find(button => button.textContent?.includes('Try again'));
      expect(retryButton).toBeTruthy();

      retryButton?.click();
      fixture.detectChanges();

      expect(albumService.getAlbum).toHaveBeenCalledTimes(2);
      expect(fixture.nativeElement.textContent).toContain('Test Album');
    });
  });

  describe('contextual back navigation', () => {
    let fixture: ComponentFixture<AlbumDetailComponent>;
    let routeParams$: BehaviorSubject<ParamMap>;
    let albumService: jasmine.SpyObj<AlbumService>;
    let navContextService: jasmine.SpyObj<NavigationContextService>;

    async function setup(
      provenance: AlbumProvenance | null,
      hadSearchResults: boolean
    ): Promise<void> {
      routeParams$ = new BehaviorSubject(convertToParamMap({ id: albumId }));
      albumService = jasmine.createSpyObj<AlbumService>('AlbumService', ['getAlbum']);
      navContextService = jasmine.createSpyObj<NavigationContextService>(
        'NavigationContextService',
        ['getAlbumProvenance', 'clearAlbumProvenance', 'getHadSearchResults', 'clearHadSearchResults']
      );

      albumService.getAlbum.and.returnValue(of(mockAlbum));
      navContextService.getAlbumProvenance.and.returnValue(provenance);
      navContextService.getHadSearchResults.and.returnValue(hadSearchResults);

      await TestBed.configureTestingModule({
        imports: [AlbumDetailComponent],
        providers: [
          provideRouter([]),
          {
            provide: ActivatedRoute,
            useValue: { paramMap: routeParams$.asObservable() }
          },
          { provide: AlbumService, useValue: albumService },
          { provide: NavigationContextService, useValue: navContextService }
        ]
      }).compileComponents();

      fixture = TestBed.createComponent(AlbumDetailComponent);
      fixture.detectChanges();
    }

    it('shows "Back to [Artist Name]" and navigates to artist page when provenance is artist', async () => {
      const provenance: AlbumProvenance = { source: 'artist', artistId: 'artist-1', artistName: 'Test Artist' };
      await setup(provenance, false);

      const backButton = (Array.from(fixture.nativeElement.querySelectorAll('button')) as HTMLButtonElement[])
        .find(b => b.textContent?.includes('Back to Test Artist'));
      expect(backButton).toBeTruthy();

      const router = TestBed.inject(Router);
      spyOn(router, 'navigate');
      fixture.componentInstance.goBack();
      expect(router.navigate).toHaveBeenCalledWith(['/artists', 'artist-1']);
    });

    it('shows "Back to Search Results" when no provenance but had prior search results', async () => {
      await setup(null, true);

      const backButton = (Array.from(fixture.nativeElement.querySelectorAll('button')) as HTMLButtonElement[])
        .find(b => b.textContent?.includes('Back to Search Results'));
      expect(backButton).toBeTruthy();
    });

    it('shows "Back to Search" when no provenance and no prior search results', async () => {
      await setup(null, false);

      const backButton = (Array.from(fixture.nativeElement.querySelectorAll('button')) as HTMLButtonElement[])
        .find(b => b.textContent?.includes('Back to Search'));
      expect(backButton).toBeTruthy();
      expect(fixture.nativeElement.textContent).not.toContain('Back to Search Results');
    });
  });
});
