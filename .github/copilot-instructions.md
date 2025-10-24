# FreqShow Development Guide

## Project Architecture

This is a **Go + Angular monorepo** for a music encyclopedia app that proxies MusicBrainz API with caching.

### Key Directories
- `apps/server/` - Go 1.22 backend with layered architecture 
- `apps/frontend/` - Angular 17 + Tailwind frontend with SSR support
- `agent-context/development-log.md` - Chronicles architectural decisions and evolution

## Backend Patterns (Go)

### Dependency Injection Architecture
The app uses explicit DI throughout. Main wiring happens in `apps/server/cmd/server/main.go`:
```go
// Router gets injected dependencies via RouterConfig
api.NewRouter(api.RouterConfig{
    MusicBrainz: mbClient,
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
3. Transform external response to internal `data.Artist`/`data.Album` structs
4. Save to repository (`store.SaveArtist()`) 
5. Return cached result

### Environment-Driven Configuration
Configuration in `apps/server/pkg/config/config.go` uses environment variables with sensible defaults:
- `DATABASE_DRIVER=sqlite` (or `memory` for testing)
- `DATABASE_URL=file:freqshow.db?_fk=1`
- `MUSICBRAINZ_TIMEOUT_SECONDS=6`

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
cd apps/server && go run ./cmd/server

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

## MusicBrainz Integration

### Client Pattern
`apps/server/pkg/sources/musicbrainz/client.go` implements proper API etiquette:
- Required User-Agent headers with app identification
- Configurable timeouts and rate limiting
- Transforms external JSON to internal domain structs

### Error Handling
APIs return 404 when MusicBrainz has no data (vs 500 for actual errors). This distinction matters for caching behavior.

## Key Conventions

### Import Paths
Always use full module paths: `github.com/adamlacasse/freq-show/apps/server/pkg/...`

### Error Patterns
- Controllers return HTTP status codes, don't log errors (let middleware handle)
- Repository layer returns domain errors that controllers translate to HTTP
- Test helpers use `t.Fatalf()` for setup failures, `t.Errorf()` for assertion failures

### File Organization
- Each package has clear single responsibility (`api`, `db`, `config`, `data`, `sources/musicbrainz`)
- Tests live alongside code, not in separate directories
- Shared domain models in `pkg/data/` avoid circular dependencies