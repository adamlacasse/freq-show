import { Component, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { Observable, Subject, Subscription, takeUntil } from 'rxjs';
import { ArtistService } from '../../services/artist.service';
import { NavigationContextService } from '../../services/navigation-context.service';
import type { components } from '../../models/openapi-types.generated';

type BaseArtist = components['schemas']['Artist'];

type RelatedArtist = {
  id: string;
  name: string;
  relationshipType?: string;
};

type Artist = Omit<BaseArtist, 'related'> & {
  biographyUrl?: string;
  related: RelatedArtist[];
};

@Component({
    selector: 'app-artist-detail',
    imports: [CommonModule, RouterLink],
    templateUrl: './artist-detail.component.html',
    styleUrl: './artist-detail.component.css'
})
export class ArtistDetailComponent implements OnInit, OnDestroy {
  artist: Artist | null = null;
  isLoading = false;
  error: string | null = null;
  private artistId: string | null = null;
  private activeLoad?: Subscription;
  private destroy$ = new Subject<void>();

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private artistService: ArtistService,
    private navigationContextService: NavigationContextService
  ) {}

  ngOnInit(): void {
    this.route.paramMap
      .pipe(takeUntil(this.destroy$))
      .subscribe(params => {
        const artistId = params.get('id');
        if (artistId) {
          this.loadArtist(artistId);
          return;
        }

        this.artistId = null;
        this.artist = null;
        this.error = 'Artist id is required.';
      });
  }

  ngOnDestroy(): void {
    this.activeLoad?.unsubscribe();
    this.destroy$.next();
    this.destroy$.complete();
  }

  goBack(): void {
    this.router.navigate(['/']);
  }

  retry(): void {
    if (this.artistId) {
      this.loadArtist(this.artistId);
    }
  }

  private loadArtist(artistId: string): void {
    this.artistId = artistId;
    this.isLoading = true;
    this.error = null;
    this.activeLoad?.unsubscribe();

    this.activeLoad = (this.artistService.getArtist(artistId) as unknown as Observable<Artist>)
      .subscribe({
        next: (artist: Artist) => {
          this.artist = artist;
          this.isLoading = false;
        },
        error: (error: unknown) => {
          console.error('Error loading artist:', error);
          this.artist = null;
          this.error = 'Failed to load artist information.';
          this.isLoading = false;
        }
      });
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

  getYearsActive(artist: Artist): string {
    const span = this.getLifeSpanDisplay(artist);
    return span ? `Active: ${span}` : '';
  }

  trackByAlbumId(index: number, album: any): string {
    return album.id;
  }

  trackByRelatedArtistId(index: number, relatedArtist: RelatedArtist): string {
    return relatedArtist.id;
  }

  onAlbumClick(album: any): void {
    // Guard against a click arriving before artistId is populated; writing an
    // undefined artistId into provenance would cause the album page's back
    // button to later navigate to /artists/undefined.
    if (!this.artistId) {
      return;
    }
    this.navigationContextService.setAlbumProvenance({
      source: 'artist',
      artistId: this.artistId,
      artistName: this.artist?.name ?? ''
    });
    this.router.navigate(['/albums', album.id]);
  }

  get sortedAlbums() {
    if (!this.artist?.albums) {
      return [];
    }
    
    return [...this.artist.albums].sort((a, b) => {
      // Sort by year (most recent first)
      // Handle cases where year might be 0 or undefined
      const yearA = a.year || 0;
      const yearB = b.year || 0;
      
      if (yearA === yearB) {
        // If years are the same, sort alphabetically by title
        return a.title.localeCompare(b.title);
      }
      
      // Sort by year descending (newest first)
      return yearB - yearA;
    });
  }
}
