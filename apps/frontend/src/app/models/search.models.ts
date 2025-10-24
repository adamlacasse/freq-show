export interface Artist {
  id: string;
  name: string;
  country?: string;
  type?: string;
  disambiguation?: string;
  aliases?: string[];
  lifeSpan: LifeSpan;
}

export interface LifeSpan {
  begin?: string;
  end?: string;
  ended?: boolean;
}

export interface SearchResult {
  artists: Artist[];
  offset: number;
  count: number;
}

export interface SearchParams {
  query: string;
  limit?: number;
  offset?: number;
}