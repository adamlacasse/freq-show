import { Component, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { Subject, takeUntil, switchMap, EMPTY } from 'rxjs';
import { ArtistService } from '../../services/artist.service';
import { Artist } from '../../models/artist.models';

@Component({
  selector: 'app-artist-detail',
  standalone: true,
  imports: [CommonModule, RouterLink],
  templateUrl: './artist-detail.component.html',
  styleUrl: './artist-detail.component.css'
})
export class ArtistDetailComponent implements OnInit, OnDestroy {
  artist: Artist | null = null;
  isLoading = false;
  error: string | null = null;
  private destroy$ = new Subject<void>();

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private artistService: ArtistService
  ) {}

  ngOnInit(): void {
    this.route.paramMap
      .pipe(
        takeUntil(this.destroy$),
        switchMap(params => {
          const artistId = params.get('id');
          if (artistId) {
            this.isLoading = true;
            this.error = null;
            return this.artistService.getArtist(artistId);
          }
          return EMPTY;
        })
      )
      .subscribe({
        next: (artist: Artist) => {
          this.artist = artist;
          this.isLoading = false;
        },
        error: (error: any) => {
          console.error('Error loading artist:', error);
          this.error = 'Failed to load artist information.';
          this.isLoading = false;
        }
      });
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  goBack(): void {
    this.router.navigate(['/']);
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

  trackByRelatedId(index: number, relatedId: string): string {
    return relatedId;
  }
}
