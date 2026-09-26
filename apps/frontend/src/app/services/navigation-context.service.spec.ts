import { NavigationContextService, AlbumProvenance } from './navigation-context.service';

describe('NavigationContextService', () => {
  let service: NavigationContextService;

  beforeEach(() => {
    service = new NavigationContextService();
  });

  it('round-trips saved search query values', () => {
    service.saveSearchQuery('Radiohead');

    expect(service.getSavedSearchQuery()).toBe('Radiohead');
  });

  it('trims saved search query values and stores null for blank input', () => {
    service.saveSearchQuery('  Bjork  ');
    expect(service.getSavedSearchQuery()).toBe('Bjork');

    service.saveSearchQuery('   ');
    expect(service.getSavedSearchQuery()).toBeNull();
  });

  it('clears saved search query values', () => {
    service.saveSearchQuery('Bowie');
    service.clearSavedSearchQuery();

    expect(service.getSavedSearchQuery()).toBeNull();
  });

  it('round-trips artist album provenance', () => {
    const provenance: AlbumProvenance = {
      source: 'artist',
      artistId: 'artist-1',
      artistName: 'Test Artist'
    };

    service.setAlbumProvenance(provenance);

    expect(service.getAlbumProvenance()).toEqual(provenance);
  });

  it('round-trips search album provenance', () => {
    const provenance: AlbumProvenance = {
      source: 'search',
      query: 'Radiohead',
      hadResults: true
    };

    service.setAlbumProvenance(provenance);

    expect(service.getAlbumProvenance()).toEqual(provenance);
  });

  it('clears album provenance', () => {
    service.setAlbumProvenance({
      source: 'artist',
      artistId: 'artist-1',
      artistName: 'Test Artist'
    });
    service.clearAlbumProvenance();

    expect(service.getAlbumProvenance()).toBeNull();
  });

  it('keeps saved search query independent of album provenance', () => {
    const provenance: AlbumProvenance = {
      source: 'artist',
      artistId: 'artist-1',
      artistName: 'Test Artist'
    };

    service.saveSearchQuery('Nirvana');
    service.setAlbumProvenance(provenance);

    service.clearSavedSearchQuery();
    expect(service.getSavedSearchQuery()).toBeNull();
    expect(service.getAlbumProvenance()).toEqual(provenance);

    service.clearAlbumProvenance();
    expect(service.getAlbumProvenance()).toBeNull();
  });
});
