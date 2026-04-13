import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { provideHttpClient } from '@angular/common/http';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, ParamMap, convertToParamMap, provideRouter } from '@angular/router';
import { BehaviorSubject, of, throwError } from 'rxjs';

import { ArtistService } from '../../services/artist.service';
import { ArtistDetailComponent } from './artist-detail.component';

describe('ArtistDetailComponent', () => {
  const artistId = 'artist-1';
  const mockArtist = {
    id: artistId,
    name: 'Test Artist',
    biography: 'Biography',
    genres: [],
    albums: [],
    related: [],
    imageUrl: '',
    aliases: [],
    lifeSpan: { begin: '1990', end: '', ended: false }
  };

  describe('with cached service responses', () => {
    let fixture: ComponentFixture<ArtistDetailComponent>;
    let routeParams$: BehaviorSubject<ParamMap>;
    let httpMock: HttpTestingController;

    beforeEach(async () => {
      routeParams$ = new BehaviorSubject(convertToParamMap({ id: artistId }));

      await TestBed.configureTestingModule({
        imports: [ArtistDetailComponent],
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

      fixture = TestBed.createComponent(ArtistDetailComponent);
      httpMock = TestBed.inject(HttpTestingController);
      fixture.detectChanges();
    });

    afterEach(() => {
      httpMock.verify();
    });

    it('reuses the cached artist on repeat navigation', () => {
      const firstRequest = httpMock.expectOne(`/api/artists/${artistId}`);
      firstRequest.flush(mockArtist);

      routeParams$.next(convertToParamMap({ id: artistId }));
      fixture.detectChanges();

      httpMock.expectNone(`/api/artists/${artistId}`);
      expect(fixture.nativeElement.textContent).toContain('Test Artist');
    });
  });

  describe('retry behavior', () => {
    let fixture: ComponentFixture<ArtistDetailComponent>;
    let artistService: jasmine.SpyObj<ArtistService>;
    let routeParams$: BehaviorSubject<ParamMap>;

    beforeEach(async () => {
      routeParams$ = new BehaviorSubject(convertToParamMap({ id: artistId }));
      artistService = jasmine.createSpyObj<ArtistService>('ArtistService', ['getArtist']);
      spyOn(console, 'error');

      artistService.getArtist.and.returnValues(
        throwError(() => new Error('boom')),
        of(mockArtist)
      );

      await TestBed.configureTestingModule({
        imports: [ArtistDetailComponent],
        providers: [
          provideRouter([]),
          {
            provide: ActivatedRoute,
            useValue: { paramMap: routeParams$.asObservable() }
          },
          {
            provide: ArtistService,
            useValue: artistService
          }
        ]
      }).compileComponents();

      fixture = TestBed.createComponent(ArtistDetailComponent);
      fixture.detectChanges();
    });

    it('shows a retry action when loading fails and reloads on click', () => {
      expect(fixture.nativeElement.textContent).toContain('Failed to load artist information.');

      const buttons = Array.from(fixture.nativeElement.querySelectorAll('button')) as HTMLButtonElement[];
      const retryButton = buttons.find(button => button.textContent?.includes('Try again'));
      expect(retryButton).toBeTruthy();

      retryButton?.click();
      fixture.detectChanges();

      expect(artistService.getArtist).toHaveBeenCalledTimes(2);
      expect(fixture.nativeElement.textContent).toContain('Test Artist');
    });
  });
});
