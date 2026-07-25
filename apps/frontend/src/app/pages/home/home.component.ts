import { Component } from '@angular/core';
import { SearchComponent } from '../../components/search/search.component';

@Component({
    selector: 'app-home',
    imports: [SearchComponent],
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
}
