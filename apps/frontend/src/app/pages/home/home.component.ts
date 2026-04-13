import { CommonModule } from '@angular/common';
import { Component } from '@angular/core';
import { SearchComponent } from '../../components/search/search.component';

@Component({
    selector: 'app-home',
    imports: [CommonModule, SearchComponent],
    templateUrl: './home.component.html',
    styleUrl: './home.component.css'
})
export class HomeComponent {
  readonly hero = {
    eyebrow: 'FreqShow',
    headline: 'Deep cuts, no ads.',
    description:
      'Search artists, read biographies from Wikipedia, browse sorted discographies from MusicBrainz, and open album pages with track listings and reviews.',
  };

  readonly highlights = [
    {
      title: 'Artist search',
      description: 'Find artists with country, type, life span, and alias metadata surfaced directly from the API.',
    },
    {
      title: 'Artist detail pages',
      description: 'Read biographies, genre tags, and chronological album lists without extra navigation noise.',
    },
    {
      title: 'Album detail pages',
      description: 'Inspect release details, track listings, and review data in a single focused view.',
    },
  ];

  readonly sources = [
    {
      title: 'MusicBrainz',
      description: 'Structured artist and release metadata, search, release groups, tracks, and tags.',
    },
    {
      title: 'Wikipedia',
      description: 'Artist biographies with fallback search and content cleanup.',
    },
    {
      title: 'Discogs',
      description: 'Album review and release context when a matching record is available.',
    },
  ];
}
