````markdown
# 🎶 FreqShow Project Notes

## 1. Project Overview

**FreqShow** is a web application designed as an **encyclopedic music information explorer** for serious music enthusiasts—especially Gen X listeners and collectors—who enjoy reading about artists, albums, genres, and reviews.  

The idea was inspired by **AllMusic.com**, but FreqShow aims to be:
- Faster and cleaner (less ad clutter)
- Open and customizable
- Built for discovery and critical appreciation, not social or user-generated noise

In short:
> A music metadata and review explorer for nerds who still read liner notes.

---

## 2. Design Goals and Philosophy

### 🎯 Target Audience
- Music obsessives and collectors with broad tastes (rock, jazz, R&B, punk, electronic, etc.)
- Users who value *critical* reviews and factual data (not crowdsourced chatter)
- Often from Gen X or older millennial cohorts—people who remember CDs, vinyl, and music journalism.

### 🧭 Guiding Principles
1. **“Research-first” experience:** FreqShow should feel like a hybrid between an encyclopedia and a record store conversation.
2. **No ads or algorithmic junk:** Prioritize performance, readability, and depth.
3. **Critically curated:** Prefer professional or aggregated critical reviews to user reviews.
4. **Modular & hackable:** The app should be friendly to developers who might want to extend it (via APIs or plugins).
5. **Fun and whimsical, not pretentious:** Self-aware music nerdery with personality.

### 🧢 Name Origin
**FreqShow** plays on the dual meanings of:
- “Freq” = audio frequency (technical/musical angle)
- “Freak show” = tongue-in-cheek nod to music obsessives

Example taglines:
- *FreqShow — for those who still read liner notes.*
- *FreqShow — deep cuts, no ads.*
- *FreqShow — a database for the musically obsessed.*

---

## 3. Backend Plan (Go)

### 💡 Language Choice
Backend implemented in **Go (Golang)** for its performance, simplicity, and native concurrency.
Go is a strong fit for:
- API servers
- Concurrent web scraping or data aggregation tasks
- Static typing and easy deployment

### ⚙️ Core Responsibilities
- **Data aggregation layer:** Pull metadata and reviews from various APIs and public data sources.
- **Normalization:** Standardize artist, album, and genre data structures.
- **Storage:** Cache results in a local database (likely SQLite or Postgres).
- **REST or GraphQL API:** Serve structured data to the Angular frontend.

### 🧩 Modules / Packages
1. `freqshow/api` — REST handlers (artist, album, review endpoints)
2. `freqshow/data` — models and schema definitions
3. `freqshow/sources` — adapters for external APIs (MusicBrainz, Discogs, etc.)
4. `freqshow/review` — stores critical reviews (parsed or summarized)
5. `freqshow/db` — handles database connections, migrations, and caching

### 🔌 Current API Endpoints
| Endpoint | Method | Description | Example |
|----------|--------|-------------|---------|
| `/healthz` | GET | Health check | `curl localhost:8080/healthz` |
| `/artists/{mbid}` | GET | Get artist by MusicBrainz ID | `curl localhost:8080/artists/5b11f4ce-a62d-471e-81fc-a69a8278c7da` |
| `/albums/{mbid}` | GET | Get album by MusicBrainz ID | `curl localhost:8080/albums/1b022e01-4da6-387b-8658-8678046e4cef` |
| `/search?q={query}&limit={n}&offset={n}` | GET | Search artists | `curl "localhost:8080/search?q=beatles&limit=5"` |

### 🔌 Data Sources & APIs
| Source | Status | Use | Implementation Notes |
|--------|--------|-----|----------------------|
| **MusicBrainz API** | ✅ **Active** | Artist, album, release metadata, genres/tags | Comprehensive client in `pkg/sources/musicbrainz` with tag filtering |
| **Wikipedia API** | ✅ **Active** | Artist biographies | Smart fallback client in `pkg/sources/wikipedia` with content cleaning |
| **SQLite Database** | ✅ **Active** | Local caching and persistence | All API responses cached for performance |
| **Discogs API** | 📋 *Planned* | Album details, credits, catalog numbers | Requires API key, generous limits |
| **Last.fm API** | 📋 *Planned* | Artist images, additional tag data | Good for supplemental metadata |
| **RateYourMusic (RYM)** | ❌ *Avoided* | Ratings, reviews | No public API — scraping risk |
| **AllMusic** | 📋 *Research* | Review source | No API; possible future licensed use |

> **Current approach**: Combine MusicBrainz structured data with Wikipedia biographical content, all cached locally for fast access.

---

## 4. Frontend Plan (Angular)

### 🧠 Motivation
The developer (you) is experienced in React but switching to **Angular** professionally, so this project will double as a hands-on learning experience.

### 🏗️ Architecture
- **Angular 17+**
- **TypeScript**
- **Tailwind CSS** for clean, consistent styling
- **RxJS** for handling asynchronous API calls
- **Routing** for different views (Artists, Albums, Genres, Reviews)
- Optional: **NgRx** for state management once the app grows

### 🧩 Core Components Status
| Component | Status | Description |
|------------|---------|-------------|
| `SearchComponent` | ✅ **Built** | Debounced search with rich artist result cards |
| `HomeComponent` | ✅ **Built** | Landing page with integrated search functionality |
| `AppComponent` | ✅ **Built** | Application shell with branded header/footer |
| `ArtistDetailComponent` | ✅ **Built** | Rich artist pages with Wikipedia biographies, MusicBrainz genres/tags, chronologically sorted discographies, and comprehensive metadata display |
| `AlbumDetailComponent` | ✅ **Built** | Complete album pages with detailed track listings, release information, and metadata |
| `SearchService` | ✅ **Built** | Reactive search state management with debouncing |
| `ArtistService` | ✅ **Built** | Artist data fetching and caching |
| `AlbumService` | ✅ **Built** | Album data fetching and caching |
| `ReviewCard` | 📋 *Planned* | Displays critic review excerpts |
| `GenreExplorer` | 📋 *Planned* | Overview of genres and subgenres |

### 🖋️ Visual Style
- Minimalist but warm; evoke a record-store feel
- Subtle animation and polish using Framer Motion–style equivalents for Angular
- Whimsical “freq show” flair — maybe waveform dividers or vinyl-inspired UI touches

---

## 5. Current Data Model

```go
// Current Go data model implementation in pkg/data/models.go

type Artist struct {
    ID             string   `json:"id"`
    Name           string   `json:"name"`
    Biography      string   `json:"biography"`        // Wikipedia integration
    Genres         []string `json:"genres"`           // MusicBrainz tags (filtered)
    Albums         []Album  `json:"albums"`
    Related        []string `json:"related"`
    ImageURL       string   `json:"imageUrl"`
    Country        string   `json:"country,omitempty"`
    Type           string   `json:"type,omitempty"`
    Disambiguation string   `json:"disambiguation,omitempty"`
    Aliases        []string `json:"aliases,omitempty"`
    LifeSpan       LifeSpan `json:"lifeSpan"`
}

type Album struct {
    ID               string   `json:"id"`
    Title            string   `json:"title"`
    ArtistID         string   `json:"artistId"`
    ArtistName       string   `json:"artistName,omitempty"`
    PrimaryType      string   `json:"primaryType,omitempty"`
    SecondaryTypes   []string `json:"secondaryTypes,omitempty"`
    FirstReleaseDate string   `json:"firstReleaseDate,omitempty"`
    Year             int      `json:"year"`             // Chronological sorting
    Genre            string   `json:"genre"`
    Label            string   `json:"label"`
    Tracks           []Track  `json:"tracks"`           // Full track listings
    Review           Review   `json:"review"`
    CoverURL         string   `json:"coverUrl"`
}

type Track struct {
    Number int    `json:"number"`
    Title  string `json:"title"`
    Length string `json:"length"`                       // MM:SS format
}

type LifeSpan struct {
    Begin string `json:"begin,omitempty"`
    End   string `json:"end,omitempty"`
    Ended bool   `json:"ended,omitempty"`
}
```

---

## 6. MVP Roadmap Status

### Phase 1: Data & API foundation ✅ **COMPLETE**
- [x] Create Go project structure (`cmd/server`, `pkg/api`, `pkg/data`, etc.)
- [x] Integrate MusicBrainz API for basic artist and album data + search
- [x] Store fetched results locally (SQLite/in-memory)
- [x] Expose REST endpoints:
  - [x] `/artists/:id`
  - [x] `/albums/:id`
  - [x] `/search?q=` with pagination support
- [x] Add logging, error handling, rate limiting, and CORS

### Phase 2: Angular client prototype ✅ **COMPLETE**
- [x] Bootstrap Angular app with SSR and Tailwind
- [x] Create functional search component with rich UI
- [x] Connect to backend REST API with reactive services
- [x] Implement search and display artist metadata (country, type, life spans, aliases)
- [x] Use Tailwind for styling with FreqShow brand theme
- [x] Create comprehensive Artist detail components with full navigation
- [x] Create complete Album detail components with tracklists
- [x] Implement artist biography integration (Wikipedia API)
- [x] Add genre/tag display (MusicBrainz tags)
- [x] Implement chronological discography sorting

### Phase 3: Content Enhancement ⚡ **IN PROGRESS**
- [x] Wikipedia biography integration with smart fallback strategies
- [x] MusicBrainz genre/tag classification and filtering
- [x] Comprehensive track listing with duration formatting
- [ ] Artist image integration (Last.fm, Discogs, or other sources)
- [ ] Album artwork and cover art display
- [ ] Related artists functionality using MusicBrainz relationships
- [ ] Review integration from open sources

### Phase 4: UX & Personality
- [ ] Branding: logo, color palette, typography
- [ ] Easter eggs / “deep cuts” mode for power users
- [ ] Optional AI-generated summaries (trained on critic-style prose)

---

## 7. Long-Term Vision

- A self-hostable, ad-free music encyclopedia for collectors
- Possibly integrate with user’s local music library (via file metadata)
- Optional offline mode (cached data)
- Potential future companion mobile app

---

## 8. Notes on AI and Metadata Ethics
- Respect copyright and API terms of service
- Prioritize open data sources (MusicBrainz, Wikidata)
- Avoid scraping or republishing proprietary reviews without permission
- Consider using AI to **summarize** existing reviews, not reproduce them verbatim

---

## 9. Summary

**FreqShow** = *music discovery for those who read reviews and credits, not comments.*  
A playful, respectful, developer-built revival of the encyclopedic music culture once embodied by AllMusic.

---

## 10. Meta

- Author: Adam LaCasse (project founder and developer)
- Backend: Go  
- Frontend: Angular + Tailwind  
- License: TBD (probably MIT or AGPL)  
- Current state: MVP functional with working search, backend API complete, Angular frontend operational
- ChatGPT context preserved: design philosophy, technical direction, persona tone

## 11. Development Log

- **2025-10-12:** Backend skeleton scaffolding landed (`cmd/server`, `pkg/api`, `pkg/data`, `pkg/db`, `pkg/sources`). Minimal router exposes `/healthz`. Runtime configuration now resolved via environment variables (`APP_ENV`, `PORT`, `SHUTDOWN_TIMEOUT_SECONDS`) with defaults suitable for local runs and Fly.io deployment. `.env.example` added to document expected values.
- **2025-10-12:** MusicBrainz client implemented with configurable base URL, timeout, and user agent metadata. API router now exposes `GET /artists/{mbid}` proxying to MusicBrainz, returning JSON responses with basic error handling.
- **2025-10-18:** Introduced in-memory artist repository (`pkg/db`) and expanded domain models for richer metadata. HTTP router now composes dependencies via `RouterConfig`, serves `GET /artists/{mbid}` from cache when available, and persists fresh MusicBrainz responses to the store for reuse.
- **2025-10-18:** Added unit tests covering the in-memory repository cloning behavior and the `/artists/{mbid}` handler flow (cache hit, cache miss, error branches) to safeguard upcoming persistence refactors.
- **2025-10-18:** Introduced SQLite-backed persistence (configurable via `DATABASE_DRIVER`/`DATABASE_URL`) with JSON payload storage and migrations. Server now selects between in-memory and SQLite repositories at startup; new tests cover SQLite CRUD behavior.
- **2025-10-18:** Expanded domain to albums: MusicBrainz client now fetches release groups, HTTP router exposes `GET /albums/{mbid}` with caching, and both in-memory and SQLite stores persist album payloads alongside artists. Added unit tests across API and persistence layers plus gofmt/go test verification.
- **2025-10-18:** Bootstrapped Angular 17 + Tailwind frontend (`apps/frontend`) with SSR scaffold, branded application shell, and roadmap-focused landing page. Tailwind theme tokens added for FreqShow palette; npm build verified.
- **2025-10-24:** Implemented full-stack search functionality: Added `SearchArtists` method to MusicBrainz client, created `GET /search?q={query}` endpoint with pagination support, and built Angular search service with reactive state management. Search component features debounced input, loading states, and rich artist result cards displaying metadata like country, type, disambiguation, and life spans. Added CORS middleware for local development. All endpoints tested and working with real MusicBrainz data. Frontend homepage now has functional artist search replacing placeholder button.
- **2025-10-24:** Built artist detail pages with complete navigation flow: Created `ArtistDetailComponent` with comprehensive artist information display, added artist service for individual lookups, implemented routing for `/artists/:id`, and made search results clickable. Artist detail pages show rich metadata including biography placeholders, genre information, aliases, life spans, and discography sections with proper loading states and error handling. Added keyboard accessibility and proper semantic HTML. Full user journey now works: search → click result → view detailed artist page → navigate back to search.
- **2025-10-24:** Implemented complete album browsing experience: Enhanced MusicBrainz client with `GetArtistReleaseGroups` method to fetch artist discographies, populated artist detail pages with real album data (up to 50 albums per artist), created full-featured `AlbumDetailComponent` with release information, tracklist placeholders, and review sections, added album service and routing for `/albums/:id`. Complete navigation flow now works: search → artist detail → album detail. Albums display year, type, and release metadata with professional loading states and error handling. Backend properly caches albums for both new artists and updates existing cached artists with discography data.
- **2025-10-24:** Added comprehensive track listing functionality: Extended MusicBrainz client with `GetReleaseGroupTracks` method that finds representative releases and fetches detailed track data including titles, durations, and track numbers. Updated backend API to populate album track arrays with real data from MusicBrainz recordings API. Enhanced data transformation layer to convert millisecond durations to MM:SS format and handle track numbering. Album detail pages now display complete tracklists with professional formatting—12 tracks shown for Nirvana's "Nevermind" including "Smells Like Teen Spirit" (5:01), "In Bloom" (4:15), etc. All existing tests updated and passing. Track data properly cached in SQLite alongside album metadata.
- **2025-10-31:** Implemented rich artist metadata enhancement: Added MusicBrainz tags integration with `inc=tags` parameter to fetch genre information, filtering out non-genre tags (nationalities, instruments, etc.) with smart classification. Created comprehensive Wikipedia API client (`pkg/sources/wikipedia`) with intelligent fallback search strategies (artist name, "band", "musician", "singer" variations), content cleaning (removing pronunciation guides, limiting to 3 sentences/500 chars), and proper error handling. Updated backend data transformation to populate genres from tags and biographies from Wikipedia. Enhanced artist detail page UI with professional genre display (color-coded primary/secondary genres, visual hierarchy), improved biography section with source attribution, and enhanced empty states. All data sources properly integrated with dependency injection and environment configuration.
- **2025-10-31:** Added discography chronological sorting: Implemented `sortedAlbums` computed property in `ArtistDetailComponent` to sort albums by year in descending order (newest first) with alphabetical fallback for same-year releases. Enhanced discography UI with sorting indicator, prominent year badges on each album card, improved visual hierarchy emphasizing release years, and music-themed icons. Updated album card layout to better accommodate chronological browsing patterns. Users can now easily explore artist evolution from latest releases back to earliest work, matching standard music database navigation expectations.
- **2025-10-31:** Implemented album reviews with Discogs API integration: Created comprehensive reviews client (`pkg/sources/reviews`) with OAuth consumer key/secret authentication for Discogs database API. Implemented multi-source review aggregation architecture with fallback pattern, Discogs search by artist/album with proper query formatting, release detail fetching with community ratings and release notes conversion. Extended configuration system with `ReviewsConfig` supporting both personal tokens and OAuth credentials via environment variables (`REVIEWS_DISCOGS_CONSUMER_KEY`, `REVIEWS_DISCOGS_CONSUMER_SECRET`). Integrated reviews into album lookup flow with graceful degradation—albums return successfully even if review fetching fails. Reviews include Discogs source attribution, community ratings (e.g., 4.7/5 from 1043 users), detailed release notes/liner information, and direct links to Discogs pages. Fixed OAuth authentication format (query parameters vs headers), corrected JSON unmarshaling for Discogs API responses, and implemented proper error handling. Created `run.sh` script to automatically load `.env` file with OAuth credentials for local development. Updated all documentation with reviews feature and proper startup instructions. All tests passing with reviews fully cached in SQLite alongside album data.

````
`````

````