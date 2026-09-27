import { Injectable, computed, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, catchError, of, tap } from 'rxjs';
import type { components } from '../models/openapi-types.generated';

export type User = components['schemas']['User'];
export type AuthRequestBody = components['schemas']['AuthRequestBody'];
export type AuthRequestResponse = components['schemas']['AuthRequestResponse'];
export type AuthLogoutResponse = components['schemas']['AuthLogoutResponse'];

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private readonly http = inject(HttpClient);
  private readonly apiUrl = '/api';

  readonly currentUser = signal<User | null>(null);
  readonly isAuthenticated = computed(() => this.currentUser() !== null);
  readonly isCheckingSession = signal<boolean>(false);
  readonly isModalOpen = signal<boolean>(false);

  checkSession(): Observable<User | null> {
    this.isCheckingSession.set(true);
    return this.http.get<User>(`${this.apiUrl}/auth/me`, { withCredentials: true }).pipe(
      tap({
        next: (user) => {
          this.currentUser.set(user);
          this.isCheckingSession.set(false);
        },
        error: () => {
          this.currentUser.set(null);
          this.isCheckingSession.set(false);
        }
      }),
      catchError(() => of(null))
    );
  }

  requestMagicLink(email: string): Observable<AuthRequestResponse> {
    return this.http.post<AuthRequestResponse>(
      `${this.apiUrl}/auth/request`,
      { email },
      { withCredentials: true }
    );
  }

  logout(): Observable<AuthLogoutResponse> {
    return this.http.post<AuthLogoutResponse>(
      `${this.apiUrl}/auth/logout`,
      {},
      { withCredentials: true }
    ).pipe(
      tap(() => {
        this.currentUser.set(null);
      }),
      catchError(() => {
        this.currentUser.set(null);
        return of({ status: 'ok', message: 'signed out' });
      })
    );
  }

  openModal(): void {
    this.isModalOpen.set(true);
  }

  closeModal(): void {
    this.isModalOpen.set(false);
  }
}
