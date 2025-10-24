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
| Source | Use | Notes |
|--------|-----|-------|
| **MusicBrainz API** | Artist, album, release metadata | Free, structured, reliable |
| **Discogs API** | Album details, credits, catalog numbers | Requires API key, generous limits |
| **Last.fm API** | Tags, genres, basic descriptions | Good for supplemental genre data |
| **Wikipedia / Wikidata** | Context and bios | Can use for fallback info |
| **RateYourMusic (RYM)** | Ratings, reviews | No public API — scraping risk; avoid for now |
| **AllMusic (informational)** | Review source | No API; possible future scraping or licensed use |

> Goal: combine structured data (MusicBrainz/Discogs) with curated review text from reliable sources (public domain or licensed).

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
| `ArtistDetail` | 📋 *Planned* | Bio, discography, related artists |
| `AlbumDetail` | 📋 *Planned* | Tracklist, credits, reviews |
| `ReviewCard` | 📋 *Planned* | Displays critic review excerpts |
| `GenreExplorer` | 📋 *Planned* | Overview of genres and subgenres |

### 🖋️ Visual Style
- Minimalist but warm; evoke a record-store feel
- Subtle animation and polish using Framer Motion–style equivalents for Angular
- Whimsical “freq show” flair — maybe waveform dividers or vinyl-inspired UI touches

---

## 5. Data Model Sketch

```go
// Simplified Go data model examples

type Artist struct {
    ID          string
    Name        string
    Biography   string
    Genres      []string
    Albums      []Album
    Related     []string // artist IDs
    ImageURL    string
}

type Album struct {
    ID          string
    Title       string
    ArtistID    string
    Year        int
    Genre       string
    Label       string
    Tracks      []Track
    Review      Review
    CoverURL    string
}

type Review struct {
    Source      string
    Author      string
    Rating      float64
    Summary     string
    Text        string
    URL         string
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
- [ ] Create Artist and Album detail components (*next priority*)

### Phase 3: Enrichment & Reviews
- [ ] Experiment with combining review data from open sources
- [ ] Add support for displaying multiple critical voices (where available)
- [ ] Implement caching and pagination

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

````