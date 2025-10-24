# FreqShow

![Freq Show logo](docs/img/freq-show_logo.png)

**Deep cuts, no ads.** A music encyclopedia for listeners who still read liner notes.

> **Current Status**: Full-featured music browser with artist search, detailed artist pages, complete album information with track listings, and comprehensive navigation. [Try it live](#quick-start) by searching for artists like "Nirvana" or "Beatles" and exploring their complete discographies!

## What This Repo Contains
- Monorepo layout with application code under `apps/` and room for shared libraries in `packages/`.
- Go 1.22 backend (`apps/server`) that proxies to the [MusicBrainz](https://musicbrainz.org/doc/Development/XML_Web_Service/Version_2) API and caches artist/album metadata.
- Pluggable persistence layer with in-memory and SQLite implementations.
- HTTP API: `/healthz`, `/artists/{mbid}`, `/albums/{mbid}`, and `/search?q={query}` with complete artist/album data and track listings.
- Angular 17 + Tailwind frontend (`apps/frontend`) with search, artist detail pages, album detail pages, and full navigation flow.
- Development log in `agent-context/development-log.md` capturing ongoing decisions.

## Architecture at a Glance

### Backend (Go)
- **`apps/server/cmd/server`** – Entry point; wires config, datastore, MusicBrainz client, HTTP router, and graceful shutdown.
- **`apps/server/pkg/api`** – HTTP handlers using dependency-injected repositories and MusicBrainz client; handles caching logic and CORS.
- **`apps/server/pkg/config`** – Environment-driven configuration (port, shutdown timeout, database driver/URL, MusicBrainz headers/timeouts).
- **`apps/server/pkg/data`** – Domain structs shared across layers (artists, albums, tracks, reviews).
- **`apps/server/pkg/db`** – Repository interfaces plus memory/SQLite store implementations.
- **`apps/server/pkg/sources/musicbrainz`** – Client wrapping MusicBrainz REST endpoints (lookup + search) with proper headers and response transforms.

### Frontend (Angular + Tailwind)
- **`apps/frontend/src/app/models`** – TypeScript interfaces matching backend API responses.
- **`apps/frontend/src/app/services`** – Angular services for HTTP communication with the Go backend.
- **`apps/frontend/src/app/components`** – Reusable UI components including the search component.
- **`apps/frontend/src/app/pages`** – Route-level components like the homepage with integrated search.

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
	- Search for artists like "Beatles" or "Nirvana"
	- Click on any artist to view their detailed biography and complete discography
	- Click on any album to see full track listings with durations
	- Navigate seamlessly: search → artist → album → tracks

## Backend Configuration

For backend-only development, you can configure environment variables (optional):
	Create a `.env` file or export variables. Defaults are sensible for local development:
	- `APP_ENV` (default `development`)
	- `PORT` or `HTTP_PORT` (default `8080`)
	- `SHUTDOWN_TIMEOUT_SECONDS` (default `10`)
	- `DATABASE_DRIVER` (`memory` or `sqlite`, default `sqlite`)
	- `DATABASE_URL` (default `file:freqshow.db?_fk=1` when using SQLite)
	- `MUSICBRAINZ_BASE_URL`, `MUSICBRAINZ_APP_NAME`, `MUSICBRAINZ_APP_VERSION`, `MUSICBRAINZ_CONTACT`, `MUSICBRAINZ_TIMEOUT_SECONDS`

	MusicBrainz requires a contact email and descriptive user agent—update the defaults if you deploy publicly.

## API Testing

You can test the backend endpoints directly:
	```bash
	curl http://localhost:8080/healthz
	curl http://localhost:8080/artists/5b11f4ce-a62d-471e-81fc-a69a8278c7da   # Nirvana with full discography
	curl http://localhost:8080/albums/1b022e01-4da6-387b-8658-8678046e4cef   # Nevermind with all 12 tracks
	curl "http://localhost:8080/search?q=beatles&limit=5"                     # Search artists with rich metadata
	```
	
	**Sample Response** (album with tracks):
	```json
	{
		"id": "1b022e01-4da6-387b-8658-8678046e4cef",
		"title": "Nevermind",
		"tracks": [
			{"number": 1, "title": "Smells Like Teen Spirit", "length": "5:01"},
			{"number": 2, "title": "In Bloom", "length": "4:15"},
			...
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

- **🔍 Artist Search** - Real-time search with MusicBrainz integration and rich result cards
- **👤 Artist Detail Pages** - Complete artist information with biography, discography, and metadata
- **💽 Album Detail Pages** - Full album information with track listings, durations, and release details
- **🎵 Track Listings** - Complete tracklists with track numbers, titles, and precise durations
- **🧭 Seamless Navigation** - Intuitive flow from search to artist to album to tracks
- **🎨 Branded UI** - Dark theme with FreqShow design language and professional typography  
- **⚡ Fast Backend** - Go API with intelligent MusicBrainz caching and SQLite persistence
- **🔄 Reactive Frontend** - Angular 17 with RxJS state management and Tailwind CSS
- **📱 Responsive Design** - Optimized experience on desktop and mobile devices

## What's Next

**Enhanced Search & Discovery:**
- **Album Search** - Extend search to include release groups/albums alongside artists
- **Search Result Caching** - Cache popular search queries for improved performance
- **Result Pagination** - Handle large search result sets with proper pagination

**Content Enrichment:**
- **Review Integration** - Add curated review excerpts from open sources
- **Artist Relationships** - Show "similar artists" and musical connections  
- **Genre Exploration** - Add genre filtering and discovery features
- **Album Artwork** - Integrate cover art from MusicBrainz Cover Art Archive

**Advanced Features:**
- **Advanced Search** - Filter by genre, year, country, album type
- **Personal Collections** - Save favorite artists and albums
- **Discovery Mode** - Algorithmic recommendations and themed browsing

See `agent-context/development-log.md` for detailed technical roadmap.

Questions or suggestions? Open an issue or drop a note.🎶
