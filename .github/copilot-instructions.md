# FreqShow Development Guide

## Project Architecture

This is a **Go + Angular monorepo** for a music encyclopedia app that integrates MusicBrainz, Wikipedia, and Discogs APIs with caching.

### Key Directories
- `apps/server/` - Go 1.22 backend with layered architecture 
- `apps/frontend/` - Angular 17 + Tailwind frontend with SSR support
- `agent-context/development-log.md` - Chronicles architectural decisions and evolution
- `.env` - Environment configuration with API credentials (OAuth for Discogs reviews)

## Backend Patterns (Go)

### Dependency Injection Architecture
The app uses explicit DI throughout. Main wiring happens in `apps/server/cmd/server/main.go`:
```go
// Router gets injected dependencies via RouterConfig
api.NewRouter(api.RouterConfig{
    MusicBrainz: mbClient,
    Wikipedia:   wikiClient,
    Reviews:     reviewsClient,
    Artists:     store, 
    Albums:      store,
})
```

### Repository Pattern with Pluggable Storage
- **Interfaces**: `apps/server/pkg/db/db.go` defines `ArtistRepository`, `AlbumRepository`, `Store`
- **Implementations**: Memory store (dev) and SQLite store (production) both satisfy same interfaces
- **Usage**: Controllers always use interfaces, never concrete types

### Cache-First API Strategy
All `/artists/{mbid}` and `/albums/{mbid}` endpoints follow this pattern:
1. Check local repository first (`store.GetArtist()`)
2. If cache miss, fetch from MusicBrainz API 
3. Fetch supplementary data (Wikipedia biographies, Discogs reviews)
4. Transform external response to internal `data.Artist`/`data.Album` structs
5. Save to repository (`store.SaveArtist()`) 
6. Return cached result

### Environment-Driven Configuration
Configuration in `apps/server/pkg/config/config.go` uses environment variables with sensible defaults:
- `DATABASE_DRIVER=sqlite` (or `memory` for testing)
- `DATABASE_URL=file:freqshow.db?_fk=1`
- `MUSICBRAINZ_TIMEOUT_SECONDS=6`
- `REVIEWS_DISCOGS_CONSUMER_KEY` and `REVIEWS_DISCOGS_CONSUMER_SECRET` for OAuth

## Frontend Patterns (Angular)

### Service-Component Architecture
- **Services**: `src/app/services/search.service.ts` handles HTTP + state management via BehaviorSubject
- **Models**: `src/app/models/search.models.ts` mirrors backend response structures exactly
- **Components**: Smart components in `pages/`, reusable components in `components/`

### Shared State Pattern
```typescript
// Services expose reactive streams for components to consume
public searchResults$ = this.searchResultsSubject.asObservable();
```

## Development Workflows

### Running the Stack
```bash
# Terminal 1: Backend (from repo root)
cd apps/server && go build ./cmd/server && ./run.sh
# This loads .env with OAuth credentials automatically

# Terminal 2: Frontend (from repo root)  
cd apps/frontend && npm start
```

### Testing
- **Go**: `go test ./...` (uses standard testing with mock interfaces)
- **Angular**: `npm test` (Karma + Jasmine, headless Chrome)
- **Pattern**: Test files live alongside source (`*_test.go`, `*.spec.ts`)

### Database Switching
Change `DATABASE_DRIVER` env var:
- `memory` - In-memory storage (tests, rapid iteration)
- `sqlite` - Persistent SQLite (development, production)

## External API Integration

### MusicBrainz Integration
`apps/server/pkg/sources/musicbrainz/client.go` implements proper API etiquette:
- Required User-Agent headers with app identification
- Configurable timeouts and rate limiting
- Transforms external JSON to internal domain structs

### Wikipedia Integration
`apps/server/pkg/sources/wikipedia/client.go` fetches artist biographies:
- Intelligent fallback search strategies (artist name, "band", "musician" variations)
- Content cleaning and length limiting
- Graceful error handling

### Discogs Reviews Integration
`apps/server/pkg/sources/reviews/client.go` fetches album reviews and ratings:
- OAuth consumer key/secret authentication via query parameters
- Multi-source aggregation pattern (currently Discogs, extensible for future sources)
- Community ratings, release notes, and metadata
- Graceful degradation (albums work even if reviews fail)

### Error Handling
APIs return 404 when no data exists (vs 500 for actual errors). This distinction matters for caching behavior.

## Key Conventions

### Import Paths
Always use full module paths: `github.com/adamlacasse/freq-show/apps/server/pkg/...`

### Error Patterns
- Controllers return HTTP status codes, don't log errors (let middleware handle)
- Repository layer returns domain errors that controllers translate to HTTP
- Test helpers use `t.Fatalf()` for setup failures, `t.Errorf()` for assertion failures

### File Organization
- Each package has clear single responsibility (`api`, `db`, `config`, `data`, `sources/musicbrainz`, `sources/wikipedia`, `sources/reviews`)
- Tests live alongside code, not in separate directories
- Shared domain models in `pkg/data/` avoid circular dependencies
- Environment variables loaded from `.env` via `run.sh` script