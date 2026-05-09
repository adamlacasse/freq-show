import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import type { components } from '../models/openapi-types.generated';

export type DiscoveryQuery = components['schemas']['DiscoveryQuery'];
export type DiscoveryResult = components['schemas']['DiscoveryResult'];

@Injectable({
  providedIn: 'root'
})
export class DiscoverService {
  private apiUrl = '/api';

  constructor(private http: HttpClient) {}

  discover(query: DiscoveryQuery): Observable<DiscoveryResult> {
    return this.http.post<DiscoveryResult>(`${this.apiUrl}/discover`, query);
  }
}
