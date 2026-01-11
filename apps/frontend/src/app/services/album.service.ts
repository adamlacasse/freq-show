import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import type { components } from '../models/openapi-types.generated';

type Album = components['schemas']['Album'];

@Injectable({
  providedIn: 'root'
})
export class AlbumService {
  private apiUrl = '/api';

  constructor(private http: HttpClient) {}

  getAlbum(id: string): Observable<Album> {
    return this.http.get<Album>(`${this.apiUrl}/albums/${id}`);
  }
}