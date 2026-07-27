import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import type { components } from '../models/openapi-types.generated';

export interface CollectionItem {
  id: number;
  userId: string;
  albumId: string;
  format: string;
  addedAt: string;
  album?: components['schemas']['Album'];
}

@Injectable({
  providedIn: 'root'
})
export class CollectionService {
  private apiUrl = '/api';

  constructor(private http: HttpClient) {}

  getCollection(userId: string): Observable<CollectionItem[]> {
    return this.http.get<CollectionItem[]>(`${this.apiUrl}/collections/${userId}`);
  }

  addAlbumToCollection(userId: string, albumId: string, format: string = 'Vinyl'): Observable<any> {
    return this.http.post(`${this.apiUrl}/collections/${userId}/albums/${albumId}`, { format });
  }

  removeAlbumFromCollection(userId: string, albumId: string): Observable<any> {
    return this.http.delete(`${this.apiUrl}/collections/${userId}/albums/${albumId}`);
  }
}
