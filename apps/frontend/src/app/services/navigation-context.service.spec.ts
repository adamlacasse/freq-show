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

  it('round-trips album provenance', () => {
    const provenance: AlbumProvenance = {
      source: 'artist',
      artistId: 'artist-1',
      artistName: 'Test Artist'
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

  it('round-trips had-search-results values for true and false', () => {
    service.recordSearchResults(true);
    expect(service.getHadSearchResults()).toBeTrue();

    service.recordSearchResults(false);
    expect(service.getHadSearchResults()).toBeFalse();
  });

  it('clears had-search-results values', () => {
    service.recordSearchResults(true);
    service.clearHadSearchResults();

    expect(service.getHadSearchResults()).toBeFalse();
  });

  it('keeps service state independent across search query, provenance, and search results', () => {
    const provenance: AlbumProvenance = {
      source: 'artist',
      artistId: 'artist-1',
      artistName: 'Test Artist'
    };

    service.saveSearchQuery('Nirvana');
    service.setAlbumProvenance(provenance);
    service.recordSearchResults(true);

    service.clearSavedSearchQuery();
    expect(service.getSavedSearchQuery()).toBeNull();
    expect(service.getAlbumProvenance()).toEqual(provenance);
    expect(service.getHadSearchResults()).toBeTrue();

    service.clearAlbumProvenance();
    expect(service.getAlbumProvenance()).toBeNull();
    expect(service.getHadSearchResults()).toBeTrue();

    service.clearHadSearchResults();
    expect(service.getHadSearchResults()).toBeFalse();
  });
});
