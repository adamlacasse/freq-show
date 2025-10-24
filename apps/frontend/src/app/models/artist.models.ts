export interface Artist {
  id: string;
  name: string;
  biography: string;
  genres: string[] | null;
  albums: Album[] | null;
  related: string[] | null;
  imageUrl: string;
  country?: string;
  type?: string;
  disambiguation?: string;
  aliases?: string[];
  lifeSpan: LifeSpan;
}

export interface Album {
  id: string;
  title: string;
  artistId: string;
  artistName?: string;
  primaryType?: string;
  secondaryTypes?: string[];
  firstReleaseDate?: string;
  year: number;
  genre: string;
  label: string;
  tracks: Track[];
  review: Review;
  coverUrl: string;
}

export interface Track {
  number: number;
  title: string;
  length: string;
}

export interface Review {
  source: string;
  author: string;
  rating: number;
  summary: string;
  text: string;
  url: string;
}

export interface LifeSpan {
  begin?: string;
  end?: string;
  ended?: boolean;
}