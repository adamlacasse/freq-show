# Freq Show!
![Freq Show logo](docs/img/freq-show_logo.png)

Reference material for the somewhat lazy music nerd.

## What This Repo Contains
- Monorepo layout with application code under `apps/` and room for shared libraries in `packages/`.
- Go 1.22 backend (`apps/server`) that proxies to the [MusicBrainz](https://musicbrainz.org/doc/Development/XML_Web_Service/Version_2) API and caches artist/album metadata.
- Pluggable persistence layer with in-memory and SQLite implementations.
- Minimal HTTP API: `/healthz`, `/artists/{mbid}`, `/albums/{mbid}`.
- Development log in `agent-context/development-log.md` capturing ongoing decisions.

Frontend work is not started yet; the focus so far is on the service layer.

## Architecture at a Glance
- **`apps/server/cmd/server`** – Entry point; wires config, datastore, MusicBrainz client, HTTP router, and graceful shutdown.
- **`apps/server/pkg/api`** – HTTP handlers using dependency-injected repositories and MusicBrainz client; handles caching logic.
- **`apps/server/pkg/config`** – Environment-driven configuration (port, shutdown timeout, database driver/URL, MusicBrainz headers/timeouts).
- **`apps/server/pkg/data`** – Domain structs shared across layers (artists, albums, tracks, reviews).
- **`apps/server/pkg/db`** – Repository interfaces plus memory/SQLite store implementations.
- **`apps/server/pkg/sources/musicbrainz`** – Thin client wrapping MusicBrainz REST endpoints with proper headers and response transforms.

See `agent-context/development-log.md` for a chronological narrative of how these pieces evolved.

## Getting Started
1. **Prerequisites**
	- Go 1.22+
	- (Optional) SQLite if you want to inspect the generated database file.

2. **Clone and Install Dependencies**
	```bash
	git clone https://github.com/adamlacasse/freq-show.git
	cd freq-show/apps/server
	go mod download
	```

3. **Configure Environment (optional)**
	Create a `.env` file or export variables. Defaults are sensible for local development:
	- `APP_ENV` (default `development`)
	- `PORT` or `HTTP_PORT` (default `8080`)
	- `SHUTDOWN_TIMEOUT_SECONDS` (default `10`)
	- `DATABASE_DRIVER` (`memory` or `sqlite`, default `sqlite`)
	- `DATABASE_URL` (default `file:freqshow.db?_fk=1` when using SQLite)
	- `MUSICBRAINZ_BASE_URL`, `MUSICBRAINZ_APP_NAME`, `MUSICBRAINZ_APP_VERSION`, `MUSICBRAINZ_CONTACT`, `MUSICBRAINZ_TIMEOUT_SECONDS`

	MusicBrainz requires a contact email and descriptive user agent—update the defaults if you deploy publicly.

4. **Run the Server** (from `apps/server`)
	```bash
	go run ./cmd/server
	```

	The service listens on `http://localhost:8080` by default.

5. **Hit the Endpoints**
	```bash
	curl http://localhost:8080/healthz
	curl http://localhost:8080/artists/5b11f4ce-a62d-471e-81fc-a69a8278c7da   # Nirvana
	curl http://localhost:8080/albums/1b022e01-4da6-387b-8658-8678046e4cef   # Nevermind
	```

6. **Run Tests** (from `apps/server`)
	```bash
	go test ./...
	```

## Development Notes
- The first request for an artist/album fetches from MusicBrainz; subsequent requests return the cached payload.
- The SQLite store persists JSON blobs—use `jq` or SQLite queries to inspect contents.
- `agent-context/development-log.md` doubles as an AI assistant context file; it records milestones and outstanding ideas.

## Roadmap Snapshot
- Flesh out album details (tracks, labels) and artist enrichment beyond MusicBrainz defaults.
- Add search endpoints and begin the Angular front end.
- Expand documentation as new subsystems appear.

Questions or suggestions? Open an issue or drop a note.🎶
