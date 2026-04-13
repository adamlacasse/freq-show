import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, of, tap, finalize, shareReplay } from 'rxjs';
import type { components } from '../models/openapi-types.generated';

type Album = components['schemas']['Album'];

@Injectable({
  providedIn: 'root'
})
export class AlbumService {
  private apiUrl = '/api';
  private readonly cacheLimit = 20;
  private cache = new Map<string, Album>();
  private inFlight = new Map<string, Observable<Album>>();

  constructor(private http: HttpClient) {}

  getAlbum(id: string): Observable<Album> {
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

    const request$ = this.http.get<Album>(`${this.apiUrl}/albums/${id}`).pipe(
      tap(album => this.storeInCache(id, album)),
      finalize(() => {
        this.inFlight.delete(id);
      }),
      shareReplay({ bufferSize: 1, refCount: true })
    );

    this.inFlight.set(id, request$);
    return request$;
  }

  private storeInCache(id: string, album: Album): void {
    if (this.cache.has(id)) {
      this.cache.delete(id);
    }

    this.cache.set(id, album);

    while (this.cache.size > this.cacheLimit) {
      const oldestKey = this.cache.keys().next().value as string | undefined;
      if (!oldestKey) {
        break;
      }
      this.cache.delete(oldestKey);
    }
  }
}
