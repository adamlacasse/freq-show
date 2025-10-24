# FreqShow

![Freq Show logo](docs/img/freq-show_logo.png)

**Deep cuts, no ads.** A music encyclopedia for listeners who still read liner notes.

> **Current Status**: MVP functional with working artist search, Go backend API, and Angular frontend. [Try it live](#quick-start) by searching for your favorite artists!

## What This Repo Contains
- Monorepo layout with application code under `apps/` and room for shared libraries in `packages/`.
- Go 1.22 backend (`apps/server`) that proxies to the [MusicBrainz](https://musicbrainz.org/doc/Development/XML_Web_Service/Version_2) API and caches artist/album metadata.
- Pluggable persistence layer with in-memory and SQLite implementations.
- HTTP API: `/healthz`, `/artists/{mbid}`, `/albums/{mbid}`, and `/search?q={query}` for artist search.
- Angular 17 + Tailwind frontend (`apps/frontend`) with working search functionality and branded UI.
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
	go run ./cmd/server
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
	- Use the search box to find artists like "Beatles" or "Nirvana"
	- Results are fetched from MusicBrainz in real-time

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
	curl http://localhost:8080/artists/5b11f4ce-a62d-471e-81fc-a69a8278c7da   # Nirvana
	curl http://localhost:8080/albums/1b022e01-4da6-387b-8658-8678046e4cef   # Nevermind
	curl "http://localhost:8080/search?q=beatles&limit=5"                     # Search artists
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

- **🔍 Artist Search** - Real-time search with MusicBrainz integration
- **🎨 Branded UI** - Dark theme with FreqShow design language
- **⚡ Fast Backend** - Go API with SQLite caching and CORS support
- **🔄 Reactive Frontend** - Angular 17 with RxJS and Tailwind CSS
- **📱 Responsive Design** - Works on desktop and mobile devices

## What's Next

**Immediate Priorities:**
- **Artist Detail Pages** - Click search results to view full artist bios and discographies  
- **Album Search** - Extend search to include release groups/albums
- **Result Pagination** - Handle large search result sets efficiently

**Future Enhancements:**
- Search result caching and pagination
- Album detail pages with track listings and credits
- Review integration from open sources
- Artist relationship mapping and "similar artists"
- Genre exploration and filtering

See `agent-context/development-log.md` for detailed technical roadmap.

Questions or suggestions? Open an issue or drop a note.🎶
