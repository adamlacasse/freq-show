# FreqShow

![Freq Show logo](docs/img/freq-show_logo.png)

**Deep cuts, no ads.** A music encyclopedia for listeners who still read liner notes.

> **Current Status**: Fully functional music encyclopedia with artist search, rich artist biographies from Wikipedia, genre classification, chronologically sorted discographies, complete album pages with track listings, and seamless navigation. Features professional dark theme UI and intelligent data caching. [Try it live](#quick-start) by searching for artists like "Nirvana" or "Beatles" to explore their biographies, genres, and complete chronological discographies!

## What This Repo Contains
- **Modern Monorepo**: Clean layout with application code under `apps/` and room for shared libraries in `packages/`.
- **Go 1.22 Backend** (`apps/server`): High-performance API that integrates MusicBrainz metadata with Wikipedia biographies, intelligent genre classification, and comprehensive caching.
- **Multi-Source Data Integration**: MusicBrainz API for structured music data + Wikipedia API for artist biographies with smart fallback strategies.
- **Pluggable Architecture**: In-memory and SQLite persistence implementations with full dependency injection.
- **Rich REST API**: `/healthz`, `/artists/{mbid}`, `/albums/{mbid}`, and `/search?q={query}` endpoints serving complete artist/album data with genres, biographies, and track listings.
- **Angular 17 Frontend** (`apps/frontend`): Professional UI with search, artist detail pages with biographies and genres, album detail pages, chronological discography sorting, and seamless navigation.
- **Comprehensive Documentation**: Development log in `agent-context/development-log.md` capturing architectural decisions and evolution.

## Architecture at a Glance

### Backend (Go)
- **`apps/server/cmd/server`** – Entry point; wires config, datastore, MusicBrainz + Wikipedia clients, HTTP router, and graceful shutdown.
- **`apps/server/pkg/api`** – HTTP handlers using dependency-injected repositories and external API clients; handles caching logic, CORS, and multi-source data aggregation.
- **`apps/server/pkg/config`** – Environment-driven configuration supporting MusicBrainz, Wikipedia, database, and server settings.
- **`apps/server/pkg/data`** – Rich domain structs with comprehensive artist metadata, album details, track information, and biography support.
- **`apps/server/pkg/db`** – Repository interfaces plus memory/SQLite store implementations with JSON blob caching.
- **`apps/server/pkg/sources/musicbrainz`** – Comprehensive client with tag filtering, search, artist/album lookups, and track listing integration.
- **`apps/server/pkg/sources/wikipedia`** – Intelligent biography client with fallback search strategies and content cleaning.

### Frontend (Angular + Tailwind)
- **`apps/frontend/src/app/models`** – Rich TypeScript interfaces for artists, albums, tracks, and search results.
- **`apps/frontend/src/app/services`** – Reactive Angular services with RxJS state management and HTTP caching.
- **`apps/frontend/src/app/components`** – Professional search component with debouncing and rich result cards.
- **`apps/frontend/src/app/pages`** – Complete page components: home with search, artist detail with biographies/genres/sorted discographies, and album detail with full track listings.

See `agent-context/development-log.md` for a chronological narrative of how these pieces evolved.

## Quick Start

To run both the backend API and frontend simultaneously:

1. **Prerequisites**
	- Go 1.22+
	- Node.js 18+ and npm
	- (Optional) SQLite if you want to inspect the generated database file

2. **Clone and Install**
	```bash
	git clone https://github.com/adamlacasse/freq-show.git
	cd freq-show
	```

3. **Start Backend** (Terminal 1)
	```bash
	cd apps/server
	go mod download
	go run ./cmd/server/main.go
	# Backend runs on http://localhost:8080
	```

4. **Start Frontend** (Terminal 2)
	```bash
	cd apps/frontend
	npm install
	npm start
	# Frontend runs on http://localhost:4200
	```

5. **Try It Out**
	- Visit http://localhost:4200
	- Search for artists like "Beatles", "Nirvana", or "Miles Davis"
	- **Explore Rich Artist Pages**: Read Wikipedia biographies, browse genre classifications, and view chronologically sorted discographies
	- **Dive into Albums**: Click any album to see complete track listings with precise durations
	- **Discover Musical History**: Navigate seamlessly from search → artist biography/genres → chronological albums → detailed tracks

## Backend Configuration

For backend-only development, you can configure environment variables (optional):

**Server & Database:**
- `APP_ENV` (default `development`)
- `PORT` or `HTTP_PORT` (default `8080`)  
- `SHUTDOWN_TIMEOUT_SECONDS` (default `10`)
- `DATABASE_DRIVER` (`memory` or `sqlite`, default `sqlite`)
- `DATABASE_URL` (default `file:freqshow.db?_fk=1` when using SQLite)

**MusicBrainz API:**
- `MUSICBRAINZ_BASE_URL` (default `https://musicbrainz.org/ws/2`)
- `MUSICBRAINZ_APP_NAME`, `MUSICBRAINZ_APP_VERSION`, `MUSICBRAINZ_CONTACT`
- `MUSICBRAINZ_TIMEOUT_SECONDS` (default `6`)

**Wikipedia API:**  
- `WIKIPEDIA_BASE_URL` (default `https://en.wikipedia.org/api/rest_v1`)
- `WIKIPEDIA_USER_AGENT` (default `FreqShow/1.0 (https://github.com/adamlacasse/freq-show)`)
- `WIKIPEDIA_TIMEOUT_SECONDS` (default `8`)

Create a `.env` file or export variables as needed. Defaults are sensible for local development.
**Note**: MusicBrainz requires a contact email and descriptive user agent—update the defaults if you deploy publicly.

## API Testing

You can test the backend endpoints directly:
	```bash
	curl http://localhost:8080/healthz
	curl http://localhost:8080/artists/5b11f4ce-a62d-471e-81fc-a69a8278c7da   # Nirvana with biography, genres, full discography
	curl http://localhost:8080/albums/1b022e01-4da6-387b-8658-8678046e4cef   # Nevermind with all 12 tracks
	curl "http://localhost:8080/search?q=beatles&limit=5"                     # Search artists with rich metadata
	```
	
	**Sample Response** (artist with biography and genres):
	```json
	{
		"id": "5b11f4ce-a62d-471e-81fc-a69a8278c7da",
		"name": "Nirvana",
		"biography": "Nirvana was an American rock band formed in Aberdeen, Washington, in 1987. Founded by lead singer and guitarist Kurt Cobain and bassist Krist Novoselic, the band went through a succession of drummers, most notably Dave Grohl, who joined in 1990.",
		"genres": ["grunge", "alternative rock", "punk rock"],
		"albums": [
			{"id": "1b022e01-4da6-387b-8658-8678046e4cef", "title": "Nevermind", "year": 1991},
			{"id": "7c3218b7-75e0-4e8c-971f-f097b6c308c5", "title": "In Utero", "year": 1993}
		]
	}
	```

## Development

**Run Tests** (from `apps/server`)
```bash
go test ./...
```

**Frontend Development Server**
```bash
cd apps/frontend
npm start
# Runs with hot reload on http://localhost:4200
```

## Development Notes
- **Caching Strategy**: First request fetches from MusicBrainz; subsequent requests return cached payload from SQLite.
- **Database**: SQLite stores JSON blobs—use `jq` or SQL queries to inspect: `sqlite3 apps/server/freqshow.db ".tables"`
- **CORS**: Enabled for `http://localhost:4200` in development mode.
- **Search Performance**: MusicBrainz search API is rate-limited; results are not currently cached (future enhancement).
- **Documentation**: `agent-context/development-log.md` contains detailed development history and architectural decisions.

## Current Features

### 🎵 Rich Content Integration
- **� Wikipedia Biographies** - Intelligent artist biography fetching with fallback search strategies and content cleaning
- **🏷️ Genre Classification** - MusicBrainz tags filtered and classified into meaningful genre information
- **📅 Chronological Sorting** - Discographies sorted by release year (newest first) with visual year badges
- **🎶 Complete Track Listings** - Full album tracks with numbers, titles, and precise durations (MM:SS format)

### 🔍 Search & Discovery
- **⚡ Real-time Search** - Debounced artist search with MusicBrainz integration and rich result cards
- **🎯 Rich Metadata** - Artist country, type, life spans, aliases, and disambiguation in search results
- **🧭 Seamless Navigation** - Intuitive flow: search → artist biography/genres → chronological albums → track details

### 🎨 Professional UI/UX
- **� Branded Dark Theme** - FreqShow design language with teal/rose/amber accent colors
- **📱 Responsive Design** - Optimized experience on desktop, tablet, and mobile devices  
- **🎪 Visual Hierarchy** - Color-coded genre tags, prominent year badges, and clear content organization
- **♿ Accessibility** - Semantic HTML, keyboard navigation, and screen reader friendly

### ⚙️ Technical Excellence
- **🚀 High-Performance Backend** - Go API with multi-source data aggregation and intelligent caching
- **💾 Smart Persistence** - SQLite caching for instant subsequent loads of artist/album data
- **🔄 Reactive Frontend** - Angular 17 with RxJS state management and component-based architecture
- **� Monorepo Structure** - Clean separation between backend API and frontend applications

## What's Next

**Enhanced Search & Discovery:**
- **Related Artists** - Use MusicBrainz relationship data to show musical connections and similar artists
- **Album Search** - Extend search to include release groups/albums alongside artists  
- **Genre Navigation** - Browse artists by genre with filtering and discovery features
- **Advanced Search** - Filter by genre, year, country, album type, and other metadata

**Visual & Content Enhancements:**
- **Artist Images** - Integrate artist photos from Last.fm, Discogs, or other sources
- **Album Artwork** - Display cover art from MusicBrainz Cover Art Archive or external APIs
- **Review Integration** - Add curated review excerpts from open sources and music journalism
- **Timeline Views** - Visual artist evolution and music history timelines

**User Experience Features:**
- **Personal Collections** - Save favorite artists, albums, and create custom playlists
- **Discovery Mode** - Algorithmic recommendations and themed browsing experiences
- **Search Result Caching** - Cache popular queries for improved performance and offline capability
- **Export Features** - Generate shareable artist/album reports and music discovery lists

See `agent-context/development-log.md` for detailed technical roadmap.

Questions or suggestions? Open an issue or drop a note.🎶
