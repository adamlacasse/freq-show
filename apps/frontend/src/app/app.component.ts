import { Component, DestroyRef, inject } from '@angular/core';
import { Router, NavigationEnd, RouterLink, RouterOutlet } from '@angular/router';
import { AsyncPipe } from '@angular/common';
import { filter } from 'rxjs';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { SearchService } from './services/search.service';
import { ApiStatusService } from './services/api-status.service';

@Component({
    selector: 'app-root',
    imports: [RouterOutlet, RouterLink, AsyncPipe],
    templateUrl: './app.component.html',
    styleUrl: './app.component.css'
})
export class AppComponent {
  readonly title = 'FreqShow';

  private readonly router = inject(Router);
  private readonly searchService = inject(SearchService);
  private readonly destroyRef = inject(DestroyRef);
  readonly apiStatus = inject(ApiStatusService);

  constructor() {
    this.apiStatus.init();

    this.router.events
      .pipe(
        filter((event): event is NavigationEnd => event instanceof NavigationEnd),
        takeUntilDestroyed(this.destroyRef)
      )
      .subscribe((event) => {
        if (event.urlAfterRedirects !== '/') {
          this.searchService.requestSearchReset();
        }
      });
  }

  onHomeClick(): void {
    this.searchService.requestSearchReset();
  }
}
