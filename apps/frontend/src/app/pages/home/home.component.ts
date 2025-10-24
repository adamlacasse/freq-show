import { CommonModule } from '@angular/common';
import { Component } from '@angular/core';
import { SearchComponent } from '../../components/search/search.component';

type RoadmapStatus = 'done' | 'in-progress' | 'up-next';

@Component({
  selector: 'app-home',
  standalone: true,
  imports: [CommonModule, SearchComponent],
  templateUrl: './home.component.html',
  styleUrl: './home.component.css'
})
export class HomeComponent {
  readonly hero = {
    eyebrow: 'freqshow prototype',
    headline: 'Deep cuts, no ads.',
    description:
      'FreqShow is a music encyclopedia for listeners who still read liner notes. The Go API is already caching MusicBrainz data; this Angular front end is where the stories, credits, and critic voices come to life.',
  };

  readonly highlights = [
    {
      title: 'Artist dossiers',
      description: 'Quick-loading biographies, discographies, and session players built from clean MusicBrainz and Discogs data.',
    },
    {
      title: 'Album narratives',
      description: 'Treat release groups like long-form stories with context, studio notes, and standout track callouts.',
    },
    {
      title: 'Critical lens',
      description: 'Summaries from trusted critics plus room for future editorial blends—respecting copyright while adding insight.',
    },
  ];

  readonly roadmap: Array<{
    phase: string;
    status: RoadmapStatus;
    badge: { label: string; class: string };
    description: string;
  }> = [
    {
      phase: 'Phase 1 · Data Core',
      status: 'done',
      badge: { label: 'shipping', class: 'bg-freq-teal/20 text-freq-teal' },
      description: 'Go server, MusicBrainz ingestion, in-memory + SQLite caching, and REST endpoints for artists and albums.',
    },
    {
      phase: 'Phase 2 · Angular Explorer',
      status: 'in-progress',
      badge: { label: 'in progress', class: 'bg-freq-rose/15 text-freq-rose' },
      description: 'Standalone Angular app with Tailwind styling, SSR, and first artist/album browse experiences.',
    },
    {
      phase: 'Phase 3 · Critical Context',
      status: 'up-next',
      badge: { label: 'up next', class: 'bg-freq-amber/20 text-freq-amber' },
      description: 'Curated review excerpts, genre essays, and playful discovery modes for power listeners.',
    },
  ];

  readonly principles = [
    {
      title: 'Research first',
      description: 'Designed like a hybrid of a record-store conversation and a reference desk, minus the ad clutter.',
    },
    {
      title: 'Respect the sources',
      description: 'Lean on open data APIs today and extend with licensed or self-authored criticism when it is responsible.',
    },
    {
      title: 'Built to tinker',
      description: 'Modular Go backend + Angular front end make it easy to swap data sources or fork for your own collection.',
    },
  ];
}
