import { CommonModule } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import { Component, inject } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { DiscoverService, DiscoveryResult } from '../../services/discover.service';
import { AuthService } from '../../services/auth.service';

@Component({
  selector: 'app-discover',
  imports: [CommonModule, FormsModule, RouterLink],
  templateUrl: './discover.component.html',
  styleUrl: './discover.component.css'
})
export class DiscoverComponent {
  readonly authService = inject(AuthService);
  private readonly discoverService = inject(DiscoverService);

  query = '';
  alreadyKnown = '';
  isLoading = false;
  error: string | null = null;
  isRateLimited = false;
  result: DiscoveryResult | null = null;

  readonly examples = [
    'Saturday morning coffee, jazzy but modern, nothing harsh',
    'Like In Rainbows, but warmer and more instrumental',
    'Late-night electronic records with organic percussion',
  ];

  submit(): void {
    const query = this.query.trim();
    if (!query || this.isLoading) {
      return;
    }

    this.isLoading = true;
    this.error = null;
    this.isRateLimited = false;
    this.result = null;

    this.discoverService.discover({
      query,
      alreadyKnown: this.parseKnownArtists()
    }).subscribe({
      next: (result: DiscoveryResult) => {
        this.result = result;
        this.isLoading = false;
      },
      error: (err: HttpErrorResponse) => {
        if (err.status === 429) {
          this.isRateLimited = true;
        }
        this.error = this.errorMessage(err);
        this.isLoading = false;
      }
    });
  }

  useExample(example: string): void {
    this.query = example;
    this.error = null;
  }

  trackByAlbumId(_: number, pick: DiscoveryResult['picks'][number]): string {
    return pick.albumId;
  }

  private parseKnownArtists(): string[] {
    return this.alreadyKnown
      .split(/[\n,]/)
      .map(value => value.trim())
      .filter(Boolean);
  }

  private errorMessage(err: HttpErrorResponse): string {
    const backendMessage = typeof err.error?.error === 'string' ? err.error.error : '';
    if (backendMessage) {
      return backendMessage;
    }
    if (err.status === 429) {
      return 'You have reached the discovery limit. Sign in to get higher limits.';
    }
    if (err.status === 503) {
      return 'Discovery is not ready yet. Browse a few albums or run the reindex job, then try again.';
    }
    if (err.status === 400) {
      return 'Enter a shorter listening request and try again.';
    }
    return 'Discovery failed. Try again in a moment.';
  }
}
