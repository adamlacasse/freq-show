import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { CollectionService, CollectionItem } from '../../services/collection.service';

@Component({
  selector: 'app-collection',
  standalone: true,
  imports: [CommonModule, RouterModule, FormsModule],
  templateUrl: './collection.component.html',
  styleUrls: []
})
export class CollectionComponent implements OnInit {
  collection: CollectionItem[] = [];
  filteredCollection: CollectionItem[] = [];
  isLoading = true;
  error: string | null = null;
  searchTerm = '';
  userId = 'adam'; // Hardcoded for MVP

  editingItemId: number | null = null;
  editArtistName = '';
  editTitle = '';
  editYear: number | null = null;

  constructor(private collectionService: CollectionService) {}

  ngOnInit(): void {
    this.loadCollection();
  }

  loadCollection(): void {
    this.isLoading = true;
    this.error = null;
    this.collectionService.getCollection(this.userId).subscribe({
      next: (data) => {
        this.collection = this.sortCollection(data);
        this.filteredCollection = this.collection;
        this.isLoading = false;
      },
      error: (err) => {
        console.error('Error loading collection:', err);
        this.error = 'Failed to load your collection.';
        this.isLoading = false;
      }
    });
  }

  filterCollection(): void {
    const term = this.searchTerm.toLowerCase().trim();
    if (!term) {
      this.filteredCollection = this.collection;
      return;
    }

    this.filteredCollection = this.collection.filter(item => {
      const albumTitle = (item.customTitle || item.album?.title || '').toLowerCase();
      const artistName = (item.customArtistName || item.album?.artistName || '').toLowerCase();
      return albumTitle.includes(term) || artistName.includes(term);
    });
  }

  clearSearch(): void {
    this.searchTerm = '';
    this.filterCollection();
  }

  startEditing(item: CollectionItem, event: Event): void {
    event.stopPropagation();
    event.preventDefault();
    this.editingItemId = item.id;
    this.editArtistName = item.customArtistName || item.album?.artistName || '';
    this.editTitle = item.customTitle || item.album?.title || '';
    this.editYear = item.customYear || item.album?.year || null;
  }

  cancelEditing(event?: Event): void {
    if (event) {
      event.stopPropagation();
      event.preventDefault();
    }
    this.editingItemId = null;
    this.editArtistName = '';
    this.editTitle = '';
    this.editYear = null;
  }

  saveCollectionItem(item: CollectionItem, event: Event): void {
    event.stopPropagation();
    event.preventDefault();
    const newArtist = this.editArtistName.trim();
    const newTitle = this.editTitle.trim();
    const newYear = this.editYear ? Number(this.editYear) : 0;

    const finalArtist = (newArtist && newArtist !== item.album?.artistName) ? newArtist : '';
    const finalTitle = (newTitle && newTitle !== item.album?.title) ? newTitle : '';
    const finalYear = (newYear && newYear !== item.album?.year) ? newYear : 0;

    this.collectionService.updateCollectionItem(
      this.userId, item.albumId, item.format, finalArtist, finalTitle, finalYear
    ).subscribe({
      next: () => {
        item.customArtistName = finalArtist || undefined;
        item.customTitle = finalTitle || undefined;
        item.customYear = finalYear || undefined;
        this.collection = this.sortCollection(this.collection);
        this.filterCollection();
        this.editingItemId = null;
      },
      error: (err) => {
        console.error('Failed to update collection item:', err);
      }
    });
  }

  private sortCollection(items: CollectionItem[]): CollectionItem[] {
    return items.slice().sort((a, b) => {
      // If one item is missing album metadata, push it to the end
      if (!a.album && !b.album) return 0;
      if (!a.album) return 1;
      if (!b.album) return -1;

      const artistA = a.customArtistName || a.album.artistName || '';
      const artistB = b.customArtistName || b.album.artistName || '';

      const keyA = this.getSmartArtistKey(artistA);
      const keyB = this.getSmartArtistKey(artistB);

      if (!keyA && !keyB) return 0;
      if (!keyA) return 1;
      if (!keyB) return -1;

      const artistComp = keyA.localeCompare(keyB);
      if (artistComp !== 0) {
        return artistComp;
      }

      // Secondary sort: Release Year (ascending)
      const yearA = a.customYear || a.album.year || 0;
      const yearB = b.customYear || b.album.year || 0;
      return yearA - yearB;
    });
  }

  private getSmartArtistKey(name: string): string {
    if (!name) return '';
    let cleaned = name.trim();

    // Strip leading "The " or "the "
    if (/^the\s+/i.test(cleaned)) {
      cleaned = cleaned.replace(/^the\s+/i, '');
    }

    let mainPart = cleaned;
    let suffix = '';
    const joinerMatch = cleaned.match(/\s+(&|and|\/)\s+/i);
    if (joinerMatch && joinerMatch.index !== undefined) {
      mainPart = cleaned.substring(0, joinerMatch.index).trim();
      suffix = cleaned.substring(joinerMatch.index);
    }

    const lowerNormalized = mainPart.toLowerCase().replace(/['"’`.\-]/g, '');

    const knownGroups = new Set([
      'steely dan', 'pink floyd', 'led zeppelin', 'fleetwood mac', 
      'alice cooper', 'alabama shakes', 'bachman–turner overdrive', 
      'bachman-turner overdrive', 'dixie dregs', 'sex pistols', 
      't. rex', 't rex', 'van halen', 'talking heads', 'iron maiden',
      'judas priest', 'jethro tull', 'uriah heep', 'grateful dead',
      'king crimson', 'lynyrd skynyrd', 'sly & the family stone',
      'sly and the family stone', 'soft machine', 'thin lizzy', 'jack bruce band',
      'jeff beck group', 'black sabbath', 'the manhattan transfer', 'manhattan transfer',
      'jimi hendrix experience', 'the jimi hendrix experience', 'allman brothers band',
      'the allman brothers band', 'deep purple', 'blue oyster cult', 'blue öyster cult',
      'daryl hall & john oates', 'emerson, lake & palmer', 'boz scaggs'
    ]);

    const isGroup = knownGroups.has(lowerNormalized) || 
                    /\b(band|group|orchestra|ensemble|quartet|trio|duo|project|syndicate|machine|overdrive|brothers|sisters|experience|transfer)\b/i.test(mainPart);

    if (!isGroup) {
      const words = mainPart.split(/\s+/);
      if (words.length >= 2 && words.length <= 3) {
        const lastName = words[words.length - 1];
        const firstNames = words.slice(0, words.length - 1).join(' ');
        mainPart = `${lastName} ${firstNames}`;
      }
    }

    return (mainPart + suffix).toLowerCase().replace(/[^a-z0-9\s]/g, '').replace(/\s+/g, ' ').trim();
  }

  trackById(index: number, item: CollectionItem): number {
    return item.id;
  }
}
