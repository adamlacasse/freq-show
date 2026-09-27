import { Component, DestroyRef, OnInit, computed, inject } from '@angular/core';
import { Router, NavigationEnd, RouterLink, RouterOutlet } from '@angular/router';
import { AsyncPipe } from '@angular/common';
import { filter } from 'rxjs';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { SearchService } from './services/search.service';
import { ApiStatusService } from './services/api-status.service';
import { AuthService } from './services/auth.service';
import { AuthModalComponent } from './components/auth-modal/auth-modal.component';

@Component({
    selector: 'app-root',
    imports: [RouterOutlet, RouterLink, AsyncPipe, AuthModalComponent],
    templateUrl: './app.component.html',
    styleUrl: './app.component.css'
})
export class AppComponent implements OnInit {
  readonly title = 'FreqShow';

  private readonly router = inject(Router);
  private readonly searchService = inject(SearchService);
  private readonly destroyRef = inject(DestroyRef);
  readonly apiStatus = inject(ApiStatusService);
  readonly authService = inject(AuthService);

  readonly collectionRoute = computed(() => {
    const user = this.authService.currentUser();
    return user ? `/collections/${user.id}` : '/collections/adam';
  });

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

  ngOnInit(): void {
    this.authService.checkSession().subscribe();
  }

  onHomeClick(): void {
    this.searchService.requestSearchReset();
  }

  onSignIn(): void {
    this.authService.openModal();
  }

  onSignOut(): void {
    this.authService.logout().subscribe();
  }
}
