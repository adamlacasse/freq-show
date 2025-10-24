import { Component, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { Subject, takeUntil, switchMap, EMPTY } from 'rxjs';
import { AlbumService } from '../../services/album.service';
import { Album } from '../../models/artist.models';

@Component({
  selector: 'app-album-detail',
  standalone: true,
  imports: [CommonModule, RouterLink],
  templateUrl: './album-detail.component.html',
  styleUrl: './album-detail.component.css'
})
export class AlbumDetailComponent implements OnInit, OnDestroy {
  album: Album | null = null;
  isLoading = false;
  error: string | null = null;
  private destroy$ = new Subject<void>();

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private albumService: AlbumService
  ) {}

  ngOnInit(): void {
    this.route.paramMap
      .pipe(
        takeUntil(this.destroy$),
        switchMap(params => {
          const albumId = params.get('id');
          if (albumId) {
            this.isLoading = true;
            this.error = null;
            return this.albumService.getAlbum(albumId);
          }
          return EMPTY;
        })
      )
      .subscribe({
        next: (album: Album) => {
          this.album = album;
          this.isLoading = false;
        },
        error: (error: any) => {
          console.error('Error loading album:', error);
          this.error = 'Failed to load album information.';
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

  getReleaseYear(): string {
    if (this.album?.year && this.album.year > 0) {
      return this.album.year.toString();
    }
    if (this.album?.firstReleaseDate) {
      const year = this.album.firstReleaseDate.substring(0, 4);
      if (year && !isNaN(Number(year))) {
        return year;
      }
    }
    return 'Unknown';
  }

  trackByTrackNumber(index: number, track: any): number {
    return track.number;
  }
}
