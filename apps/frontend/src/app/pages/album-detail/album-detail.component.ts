import { Component, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { Subject, Subscription, takeUntil } from 'rxjs';
import { AlbumService } from '../../services/album.service';
import { NavigationContextService, AlbumProvenance } from '../../services/navigation-context.service';
import type { components } from '../../models/openapi-types.generated';

type Album = components['schemas']['Album'];

@Component({
    selector: 'app-album-detail',
    imports: [CommonModule],
    templateUrl: './album-detail.component.html',
    styleUrl: './album-detail.component.css'
})
export class AlbumDetailComponent implements OnInit, OnDestroy {
  album: Album | null = null;
  isLoading = false;
  error: string | null = null;
  provenance: AlbumProvenance | null = null;
  backLabel = 'Back to Search';
  private albumId: string | null = null;
  private activeLoad?: Subscription;
  private destroy$ = new Subject<void>();

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private albumService: AlbumService,
    private navigationContextService: NavigationContextService
  ) {}

  ngOnInit(): void {
    this.route.paramMap
      .pipe(takeUntil(this.destroy$))
      .subscribe(params => {
        const freshProvenance = this.navigationContextService.getAlbumProvenance();
        if (freshProvenance) {
          this.provenance = freshProvenance;
          this.navigationContextService.clearAlbumProvenance();
        }

        const hadSearchResults = this.navigationContextService.getHadSearchResults();
        if (hadSearchResults) {
          this.navigationContextService.clearHadSearchResults();
        }

        if (freshProvenance || hadSearchResults) {
          this.backLabel = this.computeBackLabel(hadSearchResults);
        }

        const albumId = params.get('id');
        if (albumId) {
          this.loadAlbum(albumId);
          return;
        }

        this.albumId = null;
        this.album = null;
        this.error = 'Album id is required.';
      });
  }

  ngOnDestroy(): void {
    this.activeLoad?.unsubscribe();
    this.destroy$.next();
    this.destroy$.complete();
  }

  goBack(): void {
    if (this.provenance?.source === 'artist') {
      this.router.navigate(['/artists', this.provenance.artistId]);
      return;
    }
    this.router.navigate(['/']);
  }

  private computeBackLabel(hadSearchResults: boolean): string {
    if (this.provenance?.source === 'artist') {
      return this.provenance.artistName
        ? 'Back to ' + this.provenance.artistName
        : 'Back to Artist';
    }

    // TODO: Model search as explicit album provenance source when direct Search -> Album navigation exists.
    if (this.provenance === null && hadSearchResults) {
      return 'Back to Search Results';
    }
    return 'Back to Search';
  }

  retry(): void {
    if (this.albumId) {
      this.loadAlbum(this.albumId);
    }
  }

  private loadAlbum(albumId: string): void {
    this.albumId = albumId;
    this.isLoading = true;
    this.error = null;
    this.activeLoad?.unsubscribe();

    this.activeLoad = this.albumService.getAlbum(albumId)
      .subscribe({
        next: (album: Album) => {
          this.album = album;
          this.isLoading = false;
        },
        error: (error: unknown) => {
          console.error('Error loading album:', error);
          this.album = null;
          this.error = 'Failed to load album information.';
          this.isLoading = false;
        }
      });
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
