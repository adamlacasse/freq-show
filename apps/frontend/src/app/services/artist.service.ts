import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import type { components } from '../models/openapi-types.generated';

type Artist = components['schemas']['Artist'];

@Injectable({
  providedIn: 'root'
})
export class ArtistService {
  private apiUrl = 'http://localhost:8080';

  constructor(private http: HttpClient) {}

  getArtist(id: string): Observable<Artist> {
    return this.http.get<Artist>(`${this.apiUrl}/artists/${id}`);
  }
}