import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, of, tap, finalize, shareReplay } from 'rxjs';
import type { components } from '../models/openapi-types.generated';

type Artist = components['schemas']['Artist'];

@Injectable({
  providedIn: 'root'
})
export class ArtistService {
  private apiUrl = '/api';
  private readonly cacheLimit = 20;
  private cache = new Map<string, Artist>();
  private inFlight = new Map<string, Observable<Artist>>();

  constructor(private http: HttpClient) {}

  getArtist(id: string): Observable<Artist> {
    const cached = this.cache.get(id);
    if (cached) {
      this.cache.delete(id);
      this.cache.set(id, cached);
      return of(cached);
    }

    const inFlight = this.inFlight.get(id);
    if (inFlight) {
      return inFlight;
    }

    const request$ = this.http.get<Artist>(`${this.apiUrl}/artists/${id}`).pipe(
      tap(artist => this.storeInCache(id, artist)),
      finalize(() => {
        this.inFlight.delete(id);
      }),
      shareReplay({ bufferSize: 1, refCount: true })
    );

    this.inFlight.set(id, request$);
    return request$;
  }

  private storeInCache(id: string, artist: Artist): void {
    if (this.cache.has(id)) {
      this.cache.delete(id);
    }

    this.cache.set(id, artist);

    while (this.cache.size > this.cacheLimit) {
      const oldestKey = this.cache.keys().next().value as string | undefined;
      if (!oldestKey) {
        break;
      }
      this.cache.delete(oldestKey);
    }
  }
}
