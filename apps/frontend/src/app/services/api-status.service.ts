import { Injectable, inject, PLATFORM_ID } from '@angular/core';
import { isPlatformBrowser } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { BehaviorSubject } from 'rxjs';
import { timeout, catchError } from 'rxjs/operators';
import { of } from 'rxjs';

export type ApiStatus = 'checking' | 'waking' | 'ready' | 'error';

@Injectable({ providedIn: 'root' })
export class ApiStatusService {
  private readonly http = inject(HttpClient);
  private readonly platformId = inject(PLATFORM_ID);
  private readonly statusSubject = new BehaviorSubject<ApiStatus>('checking');

  readonly status$ = this.statusSubject.asObservable();

  get status(): ApiStatus {
    return this.statusSubject.value;
  }

  /**
   * Fire a health check against /api/healthz.
   * If the response takes longer than SLOW_THRESHOLD_MS, emit 'waking'
   * so the UI can show a warm-up notice. Resolves to 'ready' on success
   * or 'error' if the request fails outright.
   */
  init(): void {
    if (!isPlatformBrowser(this.platformId)) {
      return;
    }

    const SLOW_THRESHOLD_MS = 800;
    const MAX_WAIT_MS = 30_000;

    const slowTimer = setTimeout(() => {
      if (this.statusSubject.value === 'checking') {
        this.statusSubject.next('waking');
      }
    }, SLOW_THRESHOLD_MS);

    this.http.get('/api/healthz').pipe(
      timeout(MAX_WAIT_MS),
      catchError(() => {
        clearTimeout(slowTimer);
        this.statusSubject.next('error');
        return of(null);
      })
    ).subscribe((result) => {
      clearTimeout(slowTimer);
      if (result !== null) {
        this.statusSubject.next('ready');
      }
    });
  }
}
