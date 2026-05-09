# Discovery Pipeline — Implementation Plan

> **Goal:** Ship a `/discover` endpoint on the Go backend that takes a freeform natural-language listening request, interprets it with a hosted LLM, embeds the interpreted intent with a hosted embedding model, retrieves and diversity-reranks candidates from the existing `albums` SQLite cache, and returns a ranked set of five recommendations with editorial "why it fits" / "what to listen for" reasoning. Wire a minimal `/discover` page in the Angular SSR frontend that calls the endpoint and renders the picks. No new runtime, no new service. Pipeline shape is ported from the Project 7 reference; corpus, index, and host are FreqShow-native.

This plan is the build-time companion to **[ADR-0001: Hosting strategy for the AI music discovery pipeline](../adr/0001-discovery-pipeline-hosting.md)**. The ADR settled *what* and *why*; this plan settles *how* and *in what order*. If they conflict, the ADR wins and this plan gets corrected.

---

## Status

_Last updated 2026-05-08._

| Phase | Status | Notes |
|---|---|---|
| 1. Foundations | ✅ Complete (2026-05-08) | Discovery types, `EmbeddingRepository` interface, SQLite migration for `album_embeddings`, vector encode/decode helpers, `DiscoveryConfig` in `config.go`, `RouterConfig.Embeddings` plumbed, MemoryStore + SQLite tests added |
| 2. Provider clients | ✅ Complete (2026-05-08) | `pkg/sources/embeddings/` and `pkg/sources/llm/` added with Voyage/OpenAI/Hugging Face embedding clients and Hugging Face/OpenAI/Anthropic chat clients; request/response parsing covered by local HTTP tests |
| 3. Discovery package | ✅ Complete (2026-05-08) | `pkg/discovery/` added with prompts, interpretation retry/validation, embedding-text builder, cosine+MMR retrieval, curation retry/validation, and pipeline tests |
| 4. HTTP wiring | ✅ Complete (2026-05-08) | `POST /discover` handler, server discovery-service wiring, OpenAPI contract, generated frontend API types, and best-effort lazy embedding hook in `getOrFetchAlbum` |
| 5. Reindex CLI | ✅ Complete (2026-05-08) | `cmd/reindex` added with `--limit`, `--dry-run`, and `--prune-old`; DB repositories can list albums missing embeddings for the current model |
| 6. Frontend | ✅ Complete (2026-05-08) | Angular `/discover` route, `DiscoverService`, form/result UI, route/nav wiring, and component smoke tests added |
| 7. Documentation | ✅ Complete (2026-05-08) | BACKLOG, README, and `agent-context/development-log.md` updated for the discovery pipeline implementation |

**To resume work:** read this Status section, jump to the next non-✅ phase in [Step-by-Step Implementation](#step-by-step-implementation), and proceed. The package layout, constants, prompts, JSON schemas, and validation rules below stay current across phases — those are the spec, not the progress tracker.

**Open questions / decisions parked:**

- Whether the Render SQLite cache currently sits on a persistent disk or rebuilds on each deploy (ADR-0001 Action Item #2). Doesn't block Phase 2 because the lazy-embedding-on-hydration path and the reindex CLI both work either way; only the *frequency* of full backfills changes. Worth answering before Phase 5 ships.
- Whether to validate the "no material difference between providers" benchmark finding against *interpreted* queries once Phase 3 lands. Documented as a follow-up in ADR-0001's methodology caveat. Not gating any phase, but a useful sanity check before declaring the discovery feature done.

---

## What This Plan Is Testing

This is the first AI-shaped feature in FreqShow and the first piece that crosses both backend and frontend in a single user-facing flow. It exercises:

- A two-call LLM pipeline (interpretation + curation) with strict JSON-schema contracts and one-shot retry on shape failure.
- A hosted embedding integration with a swappable provider interface so the v1 default (OpenAI `text-embedding-3-small`) is not load-bearing.
- A new SQLite table that coexists with the existing JSON-blob caches without disturbing migrations or existing read paths.
- An incremental indexing strategy: embeddings are produced lazily on album hydration and backfilled by a separate `cmd/reindex` CLI — the first FreqShow code that runs as a long-lived non-server process.
- A new HTTP endpoint shape — POST with a JSON body — that's a departure from the existing GET-by-ID endpoints and worth getting right as a template for future endpoints.
- A minimal frontend route that reuses existing album-card components but introduces the editorial "why it fits" prose, a new visual primitive.

It is **not** testing fine-tuning, vector-DB engineering (brute-force cosine over the loaded `[]float32` slice is correct at this scale), real-time streaming responses, multi-turn dialogue, or "Related Artists" / "themed browsing" — those are downstream backlog items that can later ride on the same `album_embeddings` table.

---

## Scope and Deferrals

### In scope for v1

- New Go package `apps/server/pkg/discovery/` with the full pipeline.
- New Go packages `apps/server/pkg/sources/embeddings/` and `apps/server/pkg/sources/llm/` for the swappable provider clients.
- New SQLite table `album_embeddings` and an `EmbeddingRepository` interface in `pkg/db/`.
- Lazy embedding on album hydration (extend `getOrFetchAlbum` to enqueue/perform embedding when the album is newly cached).
- New CLI tool `apps/server/cmd/reindex/` for backfilling and model-swap reindexing.
- New `POST /discover` HTTP endpoint with a JSON request/response contract.
- New Angular route `/discover` with a single textarea, a results list reusing the album card, and per-pick "why it fits" prose.
- New env vars in `config.go` for embedding-provider and LLM-provider keys, surfaced analogously to the existing Discogs OAuth fields.
- An integration-light test pass: handler unit tests with the embedder and LLM mocked; one end-to-end smoke test with a real (paid) provider key, gated behind a `DISCOVERY_E2E=1` env var.

### Deferred (named in the ADR but not built here)

- Artist-level embeddings (the foundation for "Related Artists").
- Genre-prototype embeddings (the foundation for "themed browsing").
- A `LocalONNXEmbedder` implementation. The interface is designed to accept one; no implementation is shipped.
- Provider benchmarking infrastructure beyond a single hand-run script. The ADR's Action Item #1 (benchmark `text-embedding-3-small` vs HF MiniLM-L6 vs Voyage `voyage-3-lite`) happens before code lands and informs the v1 default; it does not need to live in the repo as a re-runnable test.
- A vector index extension (`sqlite-vss` etc). Brute-force in-memory cosine over the loaded `[]float32` matrix is the v1 retrieval path and is fine through ~10K rows.
- Per-query embedding cache. Worth adding once query volume justifies it; not necessary at "you and a few friends" scale.
- Streaming the curation response back to the client. The whole result lands in one JSON payload.

---

## Required Setup

### Environment

- Go 1.22+ (matches existing `go.mod`).
- A working `dev.sh` flow as documented in the README.
- Two new environment variables, set on Render and in `.env` (added to `.env.example`):
  - `DISCOVERY_EMBEDDINGS_PROVIDER` — one of `openai` (default), `huggingface`, `voyage`. Lowercase string.
  - `DISCOVERY_EMBEDDINGS_API_KEY` — the API key for the chosen provider.
  - `DISCOVERY_LLM_PROVIDER` — one of `huggingface` (default), `openai`, `anthropic`.
  - `DISCOVERY_LLM_API_KEY` — the API key for the chosen provider.
- Optional override: `DISCOVERY_EMBEDDING_MODEL` (e.g., `text-embedding-3-small`, `sentence-transformers/all-MiniLM-L6-v2`, `voyage-3-lite`). The default per-provider is hard-coded.
- Optional override: `DISCOVERY_LLM_MODEL` for the chat model.

The pattern matches the existing `REVIEWS_DISCOGS_*` envs in `config.go`. Never committed.

### Dependencies

No new third-party libraries needed. The HTTP clients use `net/http`. JSON handling uses `encoding/json`. SQL uses `database/sql` with the existing `modernc.org/sqlite` driver. Vector math uses raw `[]float32` slices and `math.Sqrt` — no `gonum`, no `numpy` analogue. Keep the dependency surface small.

---

## Package Layout

```
apps/server/
├── cmd/
│   ├── server/main.go               # existing
│   └── reindex/main.go              # NEW — backfill/reindex CLI
├── pkg/
│   ├── api/
│   │   ├── router.go                # existing — extend NewRouter to mount /discover
│   │   └── discover_handler.go      # NEW — POST /discover handler closure
│   ├── config/
│   │   └── config.go                # existing — add DiscoveryConfig fields
│   ├── data/
│   │   ├── models.go                # existing — extend with discovery types
│   │   └── discovery.go             # NEW — DiscoveryQuery, InterpretedQuery, Pick, DiscoveryResult
│   ├── db/
│   │   ├── db.go                    # existing — add EmbeddingRepository interface
│   │   ├── sqlite.go                # existing — extend migrate(), add embedding methods
│   │   └── memory.go                # implicit — extend MemoryStore with embedding methods
│   ├── discovery/
│   │   ├── prompts.go               # NEW — Prompt A (interpretation) + Prompt B (curation) constants
│   │   ├── interpret.go             # NEW — interpretQuery(client, raw) → InterpretedQuery
│   │   ├── retrieve.go              # NEW — cosine top-K + MMR rerank + avoid filter
│   │   ├── curate.go                # NEW — curate(client, interpreted, candidates) → []Pick
│   │   ├── embedtext.go             # NEW — buildAlbumEmbeddingText(album, artist) → string
│   │   ├── pipeline.go              # NEW — top-level Run() that orchestrates all four stages
│   │   └── pipeline_test.go         # NEW — unit tests with mocked embedder and LLM
│   └── sources/
│       ├── embeddings/
│       │   ├── client.go            # NEW — Embedder interface + provider router
│       │   ├── openai.go            # NEW — OpenAIEmbedder
│       │   ├── huggingface.go       # NEW — HFEmbedder (free-tier-aware)
│       │   └── voyage.go            # NEW — VoyageEmbedder
│       └── llm/
│           ├── client.go            # NEW — ChatCompleter interface + provider router
│           ├── huggingface.go       # NEW — HFChatCompleter (default, free-tier)
│           ├── openai.go            # NEW — OpenAIChatCompleter
│           └── anthropic.go         # NEW — AnthropicChatCompleter (Claude Haiku)
└── go.mod / go.sum                  # unchanged

apps/frontend/src/app/
├── app.routes.ts                    # existing — add /discover route
├── pages/discover/                  # NEW
│   ├── discover.component.ts
│   ├── discover.component.html
│   ├── discover.component.css
│   └── discover.component.spec.ts
└── services/
    └── discover.service.ts          # NEW
```

This layout matches the existing convention: external integrations under `pkg/sources/<name>/`, persistence under `pkg/db/`, business logic under its own package (`pkg/discovery/`), HTTP wiring in `pkg/api/`. `cmd/reindex/` is a sibling of `cmd/server/` exactly the way Go standard layout wants it.

---

## Constants Reference

Every magic number lives at the top of its file as a named constant. Centralizing them up front means tuning is one-place-edit.

| Constant | Value | Used by | Why this value |
| --- | --- | --- | --- |
| `DefaultEmbeddingProvider` | `"voyage"` | discovery config | per ADR-0001 (selected after benchmark, 2026-05-08) |
| `DefaultEmbeddingModel` | `"voyage-3-lite"` | Voyage embedder | 512-dim, free tier covers expected usage |
| `DefaultEmbeddingDim` | `512` | sanity checks, sqlite schema | matches `voyage-3-lite` |
| `DefaultLLMProvider` | `"huggingface"` | discovery config | free tier covers expected volume |
| `DefaultLLMModelCandidates` | `("meta-llama/Llama-3.1-8B-Instruct", "Qwen/Qwen2.5-7B-Instruct", "HuggingFaceH4/zephyr-7b-beta", "microsoft/Phi-3.5-mini-instruct")` | HF chat client | Project 6/7's working fallback list |
| `TopKRetrieval` | `30` | retrieve.go | enough headroom for MMR |
| `TopNAfterMMR` | `10` | retrieve.go | enough variety for curation |
| `FinalPicks` | `5` | curate.go | the project's "show off" number |
| `MMRLambda` | `0.7` | retrieve.go | mild diversity penalty, relevance dominant |
| `InterpretationTemperature` | `0.3` | Prompt A call | stable structured output |
| `InterpretationMaxTokens` | `400` | Prompt A call | enough for the schema |
| `CurationTemperature` | `0.7` | Prompt B call | prose with voice but grounded |
| `CurationMaxTokens` | `900` | Prompt B call | five picks × ~150 tokens reasoning |
| `EmbeddingHTTPTimeout` | `10 * time.Second` | embeddings clients | slow networks tolerated |
| `LLMHTTPTimeout` | `30 * time.Second` | llm clients | LLM responses can be slow |
| `ReindexBatchSave` | `25` | cmd/reindex | save progress every N records |
| `MinEmbeddingTextChars` | `120` | embedtext.go | guards against records too thin to embed usefully |

Where these constants live: provider-specific defaults live in the provider files (`embeddings/openai.go`, `llm/huggingface.go`); pipeline knobs (`TopK`, `MMR`, temperatures) live in `discovery/pipeline.go`.

---

## Architectural Decisions Within This Plan

ADR-0001 settled the macro-architecture; this section captures the smaller decisions that come up when you actually sit down to write the code, and that are easier to make once than to re-derive each time.

### 1. `Embedder` and `ChatCompleter` are interfaces, not concrete types

The existing `MusicBrainzClient` / `WikipediaClient` / `ReviewsClient` types in `router.go` are interfaces defined at the consumer (the router) and implemented by the concrete client structs. Follow the same pattern.

```go
// pkg/sources/embeddings/client.go
type Embedder interface {
    Encode(ctx context.Context, text string) ([]float32, error)
    EncodeBatch(ctx context.Context, texts []string) ([][]float32, error)
    Model() string
    Dim() int
}

// pkg/sources/llm/client.go
type ChatCompleter interface {
    ChatComplete(ctx context.Context, req ChatRequest) (ChatResponse, error)
    Model() string
}

type ChatRequest struct {
    SystemPrompt string
    UserPrompt   string
    Temperature  float64
    MaxTokens    int
}

type ChatResponse struct {
    Content string
    Model   string
}
```

Why both `Encode` and `EncodeBatch`: the reindex job hits `EncodeBatch` for throughput; the live query path hits `Encode` for the single interpreted-query string. Providers that don't natively batch can implement `EncodeBatch` as a loop over `Encode`.

`Model()` and `Dim()` exist so the index path can write the `model` and `dim` columns without the caller knowing which provider it has.

### 2. Provider selection is a tiny router function, not a registry

```go
// pkg/sources/embeddings/client.go
func NewFromConfig(cfg Config) (Embedder, error) {
    switch strings.ToLower(cfg.Provider) {
    case "openai":
        return newOpenAIEmbedder(cfg)
    case "huggingface":
        return newHFEmbedder(cfg)
    case "voyage":
        return newVoyageEmbedder(cfg)
    default:
        return nil, fmt.Errorf("embeddings: unsupported provider %q", cfg.Provider)
    }
}
```

No registry pattern, no init-time side effects, no global state. The provider is selected exactly once at server start. Same shape on the LLM side.

### 3. `album_embeddings` is a sibling table, not a column on `albums`

Two reasons. First, embeddings are a different lifecycle than the cached album payload: an album can be hydrated and useful to the existing `/albums/{id}` endpoint long before its embedding lands. Second, model swaps want to coexist with old embeddings during a rolling reindex — having `model` and `dim` as primary-key-adjacent columns lets two model versions live in the table briefly without colliding.

Schema:

```sql
CREATE TABLE IF NOT EXISTS album_embeddings (
    mbid       TEXT NOT NULL,
    model      TEXT NOT NULL,
    dim        INTEGER NOT NULL,
    vec        BLOB NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    PRIMARY KEY (mbid, model)
);

CREATE INDEX IF NOT EXISTS album_embeddings_model_idx
    ON album_embeddings (model);
```

The `(mbid, model)` composite primary key lets a single album hold an embedding for `text-embedding-3-small` and another for `voyage-3-lite` simultaneously during a swap. The retrieval path filters by the *current* model; once the swap is complete, `cmd/reindex --prune-old` deletes rows for the old model.

`vec` is stored as raw little-endian float32 bytes — 4 × dim bytes per row, ~6 KB at dim=1536. Encode/decode helpers live in `pkg/db/sqlite.go`:

```go
func encodeVector(v []float32) []byte { ... }   // little-endian f32 bytes
func decodeVector(b []byte) []float32 { ... }
```

No JSON for the vector — the encode/decode overhead is real at corpus scale and the bytes-on-disk are smaller.

### 4. Lazy embedding on hydration, not eager

When `getOrFetchAlbum` lands a new album in the cache, kick off the embedding inline (synchronous, before returning to the caller). At expected volume (one user request → one album lookup → one embed call), this adds ~150 ms to a cold-fetch response and zero to a warm-fetch response. It's a lower-effort path than a worker queue and the latency is acceptable.

If embedding fails (network error, rate limit), log it, do not fail the album lookup. The next time `cmd/reindex` runs, missing embeddings get backfilled.

### 5. Retrieval candidate set comes from `album_embeddings`, not `albums`

The retrieval path queries `album_embeddings` for the current model, decodes vectors, computes cosine similarity, returns the top-K mbids. It then joins back to `albums` for the metadata used by curation and rendering. Albums without embeddings are simply not candidates yet — they need to flow through the lazy embedding path or get picked up by reindex.

This is correct behavior. The alternative (embed at query time for any uncached candidate) defeats the whole architecture.

### 6. The interpreted query is not stored

Project 7 stored the interpreted-query JSON for transparency in the CLI output. FreqShow's HTTP response includes it for the same reason — it's useful for debugging in the browser dev tools and for the eventual "edit my interpreted query" feature — but it's not persisted anywhere on the server. Each query interprets fresh.

If we later add per-user query history, that's a separate feature with its own storage.

### 7. Avoid-filter runs *after* retrieval, *before* curation

Same rule as Project 7. Filtering before retrieval distorts the embedding nearest-neighbor result. After retrieval, the filter is just a list of substring checks on `artist_name` and `title`. Case-insensitive substring match — `"Bon Iver"` filters anything by Bon Iver, `"Kid A"` filters that specific album.

### 8. JSON parsing is belt-and-suspenders, identical to Project 7

```go
// pkg/discovery/jsonparse.go (or inline in interpret.go / curate.go)
func parseJSONObject(raw string) (map[string]any, error) {
    start := strings.Index(raw, "{")
    end := strings.LastIndex(raw, "}")
    if start == -1 || end == -1 || end < start {
        return nil, fmt.Errorf("no JSON object found in response: %.200q", raw)
    }
    var out map[string]any
    if err := json.Unmarshal([]byte(raw[start:end+1]), &out); err != nil {
        return nil, fmt.Errorf("invalid JSON in response: %w", err)
    }
    return out, nil
}
```

One retry on shape failure (one re-call to the LLM with a "your last response was not valid JSON in the required shape, here is the schema again" reminder), then fail loudly. Matches Project 7.

### 9. Logging via `log` (Go stdlib), structured prefixes

The existing codebase uses `log.Printf` directly. Match it. Each pipeline stage gets a prefix:

```
discovery: interpret: calling LLM (model=meta-llama/Llama-3.1-8B-Instruct, temp=0.3)
discovery: interpret: parsed interpretation in 1.4s
discovery: retrieve: 30 candidates by cosine, 10 after MMR (lambda=0.7), 9 after avoid filter
discovery: curate: calling LLM (model=meta-llama/Llama-3.1-8B-Instruct, temp=0.7)
discovery: curate: 5 picks ready in 2.1s
```

No structured logging framework. The existing code doesn't use one and we're not adding one for this feature alone.

### 10. Error handling: the existing `apiError` pattern

Reuse `newAPIError(status, msg)` from `pkg/api/router.go` for client-facing failures. Unwrapped errors bubble up to `handleAPIError` which returns a generic 500. Log the unwrapped error before returning the apiError.

Specific status codes:

| Failure | Status | Message |
| --- | --- | --- |
| Empty / missing query in request body | 400 | `"query field required"` |
| Query > 1000 chars | 400 | `"query too long (max 1000 characters)"` |
| LLM provider auth failure | 502 | `"LLM provider unreachable"` |
| Embedding provider auth failure | 502 | `"embedding provider unreachable"` |
| LLM returns invalid JSON twice | 502 | `"LLM produced invalid response"` |
| No embedded albums in corpus yet | 503 | `"no albums embedded yet — try again after the catalog warms up"` |
| MMR returned 0 candidates after avoid filter | 200 with `picks: []` | not an error |

---

## Step-by-Step Implementation

Build bottom-up so each layer is testable before the layer above lands. Top-to-bottom file order in the package is different from build order; matches the layout in [Package Layout](#package-layout).

### Phase 1 — Foundations (no LLM, no embeddings yet) — ✅ Complete (2026-05-08)

**Step 1.1 — Discovery types in `pkg/data/discovery.go`.**

```go
package data

type DiscoveryQuery struct {
    Query        string   `json:"query"`
    AlreadyKnown []string `json:"alreadyKnown,omitempty"`
}

type InterpretedQuery struct {
    Mood              string   `json:"mood"`
    EraHints          []string `json:"eraHints"`
    SonicQualities    []string `json:"sonicQualities"`
    ReferenceArtists  []string `json:"referenceArtists"`
    Avoid             []string `json:"avoid"`
    DiscoveryAppetite string   `json:"discoveryAppetite"` // "low" | "medium" | "high"
}

type Pick struct {
    Rank             int    `json:"rank"`
    AlbumID          string `json:"albumId"`
    Title            string `json:"title"`
    ArtistName       string `json:"artistName"`
    Year             int    `json:"year"`
    WhyItFits        string `json:"whyItFits"`
    WhatToListenFor  string `json:"whatToListenFor"`
    SpotifySearchURL string `json:"spotifySearchUrl"`
}

type DiscoveryResult struct {
    Interpreted InterpretedQuery `json:"interpreted"`
    Picks       []Pick           `json:"picks"`
}
```

JSON tags use camelCase to match the existing `data.Artist` / `data.Album` style. No new dependencies.

**Step 1.2 — `EmbeddingRepository` interface in `pkg/db/db.go`.**

```go
type EmbeddingRepository interface {
    GetEmbedding(ctx context.Context, mbid, model string) ([]float32, error)
    SaveEmbedding(ctx context.Context, mbid, model string, dim int, vec []float32) error
    LoadAllForModel(ctx context.Context, model string) ([]EmbeddingRecord, error)
    DeleteOtherModels(ctx context.Context, keepModel string) (int, error)
}

type EmbeddingRecord struct {
    MBID  string
    Vec   []float32
}
```

`LoadAllForModel` returns the entire model's vectors at once; the discovery handler keeps them in process memory between requests. At dim=1536, 10K albums = ~60 MB resident. Acceptable on the Hobby Legacy plan; revisit if the corpus grows past ~50K.

**Step 1.3 — `MemoryStore` and `SQLiteStore` implementations.**

`MemoryStore` gets a `map[string]map[string][]float32` keyed by `(mbid, model)` plus the same lock as artists/albums.

`SQLiteStore` extends `migrate` with the `album_embeddings` table from Decision 3 and adds the four `EmbeddingRepository` methods. Use the `encodeVector` / `decodeVector` helpers; do not store the vector as JSON.

**Step 1.4 — Wire `EmbeddingRepository` into `RouterConfig`.**

Same shape as `Artists db.ArtistRepository`. The single `Store` instance implements all three repository interfaces.

**Step 1.5 — Add `DiscoveryConfig` to `pkg/config/config.go`.**

```go
type DiscoveryConfig struct {
    EmbeddingsProvider string
    EmbeddingsAPIKey   string
    EmbeddingsModel    string  // "" → provider default
    LLMProvider        string
    LLMAPIKey          string
    LLMModel           string  // "" → provider default
}
```

Add `Discovery DiscoveryConfig` to `Config`. Add a `resolveDiscovery()` function matching the style of `resolveReviews()`. Update `.env.example` with the four new env vars and inline comments describing each.

**Step 1.6 — Smoke test the foundations.**

Write `pkg/db/sqlite_test.go` cases for `SaveEmbedding` → `GetEmbedding` → `LoadAllForModel` → `DeleteOtherModels`. Run `go test ./...`. No discovery code exists yet but the persistence layer is verified.

### Phase 2 — Provider clients — 🟡 Next

**Step 2.1 — `pkg/sources/embeddings/client.go`.**

The `Embedder` interface (Decision 1), `Config` struct, and `NewFromConfig` router (Decision 2). No model-specific code — that's in the provider files.

**Step 2.2 — `pkg/sources/embeddings/voyage.go` (the v1 default per ADR-0001).**

```go
type VoyageEmbedder struct {
    apiKey string
    model  string
    dim    int
    http   *http.Client
}

func newVoyageEmbedder(cfg Config) (*VoyageEmbedder, error) {
    if strings.TrimSpace(cfg.APIKey) == "" {
        return nil, errors.New("embeddings: voyage api key required")
    }
    model := cfg.Model
    if model == "" { model = "voyage-3-lite" }
    return &VoyageEmbedder{
        apiKey: cfg.APIKey,
        model:  model,
        dim:    512, // voyage-3-lite native dim
        http:   &http.Client{Timeout: 10 * time.Second},
    }, nil
}

func (e *VoyageEmbedder) Encode(ctx context.Context, text string) ([]float32, error) { ... }
func (e *VoyageEmbedder) EncodeBatch(ctx context.Context, texts []string) ([][]float32, error) { ... }
func (e *VoyageEmbedder) Model() string { return e.model }
func (e *VoyageEmbedder) Dim() int { return e.dim }
```

POST to `https://api.voyageai.com/v1/embeddings` with body `{"model": ..., "input": [...], "input_type": "document"}` for corpus embeds and `"input_type": "query"` for query embeds — Voyage's API distinguishes them and quality improves slightly when used correctly. Parse the response's `data[i].embedding` into `[]float32`. Return a wrapped error on non-2xx.

**Step 2.3 — `pkg/sources/embeddings/openai.go` (alternative provider).**

POST to `https://api.openai.com/v1/embeddings` with body `{"model": ..., "input": [...]}`, parse the response's `data[i].embedding` into `[]float32`. Default model `text-embedding-3-small` (dim=1536). Standard OpenAI HTTP shape. Return a wrapped error on non-2xx. Implementation pattern is the same as `VoyageEmbedder`.

**Step 2.4 — `pkg/sources/embeddings/huggingface.go` (alternative provider, also useful for the embedding path's fallback contingency).**

POST to `https://api-inference.huggingface.co/pipeline/feature-extraction/{model}` with `{"inputs": [...]}`. Free-tier-aware: on 503 with `estimated_time` (model loading), wait the indicated duration once and retry. On 429, surface a typed `ErrRateLimit` error. Default model is `sentence-transformers/all-MiniLM-L6-v2` (dim=384).

**Step 2.5 — `pkg/sources/llm/client.go`.**

The `ChatCompleter` interface, `ChatRequest` / `ChatResponse` structs, `Config`, `NewFromConfig` router. Mirror the embeddings package structure.

**Step 2.6 — `pkg/sources/llm/huggingface.go`.**

Port the model-routing fallback from Project 6's `LLMClient`. Iterate `DefaultLLMModelCandidates` until one routes successfully; pin that model for the rest of the session. Use the HF Inference Providers `chat_completion` route. Return `ChatResponse{Content, Model}`.

**Step 2.7 — `pkg/sources/llm/openai.go` and `anthropic.go`.**

Standard chat-completion shapes. OpenAI `gpt-4o-mini` default. Anthropic Claude Haiku via the Messages API. Both straight HTTP; no SDK dependency.

**Step 2.8 — Smoke test each provider.**

A small `cmd/discovery-smoke/main.go` (committed under `cmd/` and used during dev, can be deleted before merge) that takes a provider name and calls `Encode("hello world")` or `ChatComplete(...)` against a simple prompt. Run once per provider with real keys to confirm wiring, then delete the smoke-test file or move it under a build tag.

### Phase 3 — Discovery package

**Step 3.1 — `pkg/discovery/prompts.go`.**

The two prompt constants. See [Prompts Specification](#prompts-specification) below for verbatim text. Exported as `interpretSystemPrompt`, `interpretUserPromptTemplate`, `curateSystemPrompt`, `curateUserPromptTemplate`. Templates use `text/template`-style placeholders or `fmt.Sprintf` — `text/template` is cleaner for the multi-line user prompts.

**Step 3.2 — `pkg/discovery/embedtext.go`.**

```go
func BuildAlbumEmbeddingText(album *data.Album, artist *data.Artist) string { ... }
```

See [The Embedding-Text Builder](#the-embedding-text-builder) below for the field-by-field spec. Returns the prose string that gets fed to the embedder. If the result is < `MinEmbeddingTextChars`, returns empty string and the caller skips embedding (or logs a "thin record" warning). This is the spiritual replacement for Project 7's LLM-generated blurbs.

**Step 3.3 — `pkg/discovery/interpret.go`.**

```go
func interpretQuery(
    ctx context.Context,
    llm llm.ChatCompleter,
    raw string,
    alreadyKnown []string,
) (data.InterpretedQuery, error) {
    userPrompt := renderInterpretUserPrompt(raw, alreadyKnown)
    resp, err := llm.ChatComplete(ctx, llm.ChatRequest{
        SystemPrompt: interpretSystemPrompt,
        UserPrompt:   userPrompt,
        Temperature:  InterpretationTemperature,
        MaxTokens:    InterpretationMaxTokens,
    })
    if err != nil { ... }
    parsed, err := parseInterpretation(resp.Content)
    if err != nil {
        // one retry with a shape-reminder appended to the user prompt
        ...
    }
    if err := validateInterpretation(parsed); err != nil { ... }
    return parsed, nil
}
```

`parseInterpretation` uses the JSON helper from Decision 8. `validateInterpretation` checks every required key, list-typed keys are lists, `discoveryAppetite ∈ {low, medium, high}`. One retry on parse or validation failure, then fail loudly.

**Step 3.4 — `pkg/discovery/retrieve.go`.**

```go
func retrieveCandidates(
    queryVec []float32,
    records []db.EmbeddingRecord,
    interpreted data.InterpretedQuery,
    alreadyKnown []string,
    albumLookup func(mbid string) *data.Album, // injected to fetch album metadata
) ([]*data.Album, error) {
    // 1. Top-K by cosine
    sims := cosineAll(queryVec, records)
    topK := argTopK(sims, TopKRetrieval)
    // 2. MMR rerank
    selected := mmrRerank(records, topK, queryVec, MMRLambda, TopNAfterMMR)
    // 3. Hydrate album metadata
    albums := make([]*data.Album, 0, len(selected))
    for _, idx := range selected {
        if a := albumLookup(records[idx].MBID); a != nil {
            albums = append(albums, a)
        }
    }
    // 4. Avoid filter
    avoid := append(slices.Clone(interpreted.Avoid), alreadyKnown...)
    return filterAvoid(albums, avoid), nil
}
```

`cosineAll`, `argTopK`, `mmrRerank`, `filterAvoid` are small helpers in the same file. `mmrRerank` is the only nontrivial one — port the formula from the Project 7 plan verbatim.

```go
// MMR(d_i) = λ · sim(d_i, query) − (1 − λ) · max_{d_j ∈ S} sim(d_i, d_j)
// Greedy: pick the top-K candidate maximizing MMR given currently-selected set S.
```

**Step 3.5 — `pkg/discovery/curate.go`.**

```go
func curate(
    ctx context.Context,
    llm llm.ChatCompleter,
    interpreted data.InterpretedQuery,
    candidates []*data.Album,
) ([]data.Pick, error) {
    userPrompt := renderCurateUserPrompt(interpreted, candidates)
    resp, err := llm.ChatComplete(ctx, llm.ChatRequest{
        SystemPrompt: curateSystemPrompt,
        UserPrompt:   userPrompt,
        Temperature:  CurationTemperature,
        MaxTokens:    CurationMaxTokens,
    })
    if err != nil { ... }
    parsed, err := parseCuration(resp.Content)
    if err != nil { /* one retry */ }
    if err := validateCuration(parsed, candidates); err != nil { ... }
    return enrichPicks(parsed, candidates), nil
}
```

`validateCuration` checks: exactly `FinalPicks` items; each has `rank`, `albumId`, `whyItFits`, `whatToListenFor`; `albumId` matches one of the candidate IDs exactly (no hallucinations); `rank` values are `{1..FinalPicks}` with no duplicates. One retry on shape failure.

`enrichPicks` joins each pick to the matching candidate album to populate `Title`, `ArtistName`, `Year`, and `SpotifySearchURL`.

**Step 3.6 — `pkg/discovery/pipeline.go`.**

The orchestrator. Pulls everything together:

```go
type Service struct {
    Embedder      embeddings.Embedder
    LLM           llm.ChatCompleter
    Embeddings    db.EmbeddingRepository
    Albums        db.AlbumRepository
    // ... possibly a sync.Mutex around an in-memory cache of records
}

func (s *Service) Run(ctx context.Context, q data.DiscoveryQuery) (data.DiscoveryResult, error) {
    // 1. interpret
    interpreted, err := interpretQuery(ctx, s.LLM, q.Query, q.AlreadyKnown)
    // 2. embed the query
    queryText := buildQueryEmbeddingText(interpreted)
    queryVec, err := s.Embedder.Encode(ctx, queryText)
    // 3. load corpus embeddings
    records, err := s.Embeddings.LoadAllForModel(ctx, s.Embedder.Model())
    if len(records) == 0 { return _, errNoEmbeddedAlbums }
    // 4. retrieve + MMR + avoid
    candidates, err := retrieveCandidates(queryVec, records, interpreted, q.AlreadyKnown, albumLookup)
    // 5. curate
    picks, err := curate(ctx, s.LLM, interpreted, candidates)
    return data.DiscoveryResult{Interpreted: interpreted, Picks: picks}, nil
}
```

`buildQueryEmbeddingText` formats the interpreted query as a natural-language sentence (not JSON) — Project 7's `build_query_text` pattern, ported.

The corpus records can be cached in `s` between requests. Add a small TTL or invalidate-on-write — easy to bolt on later, not necessary for v1 if `LoadAllForModel` is fast enough on a 5K-album corpus (~50 ms).

**Step 3.7 — `pkg/discovery/pipeline_test.go`.**

Mock the `Embedder` and `ChatCompleter` interfaces. Test cases in [Validation Test Cases](#validation-test-cases). Cover happy-path, validation failures, retry-on-shape-failure, fail-loudly on second failure, empty-corpus edge case.

### Phase 4 — HTTP wiring

**Step 4.1 — `pkg/api/discover_handler.go`.**

```go
func discoverHandler(svc *discovery.Service) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if !assertMethod(w, r, http.MethodPost) { return }
        var q data.DiscoveryQuery
        if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
            writeJSON(w, http.StatusBadRequest, errorResponse{"invalid request body"})
            return
        }
        if strings.TrimSpace(q.Query) == "" {
            writeJSON(w, http.StatusBadRequest, errorResponse{"query field required"})
            return
        }
        if utf8.RuneCountInString(q.Query) > 1000 {
            writeJSON(w, http.StatusBadRequest, errorResponse{"query too long (max 1000 characters)"})
            return
        }
        result, err := svc.Run(r.Context(), q)
        if err != nil { handleAPIError(w, err); return }
        writeJSON(w, http.StatusOK, result)
    }
}
```

**Step 4.2 — Mount in `NewRouter`.**

Add to `RouterConfig`: `Discovery *discovery.Service`. Add to `NewRouter`: `mux.Handle("/discover", discoverHandler(cfg.Discovery))`. Wire in `cmd/server/main.go`: build the embedder and LLM via `NewFromConfig`, build the `*discovery.Service`, pass to `NewRouter`.

**Step 4.3 — Lazy embedding hook in `getOrFetchAlbum`.**

Add an `Embedder` and `Embeddings` (the repo) to the album handler's dependencies. After a successful fresh fetch + cache, before returning, if `Embedder != nil`:

```go
text := discovery.BuildAlbumEmbeddingText(domainAlbum, artistForAlbum) // need the artist
if text != "" {
    vec, err := embedder.Encode(ctx, text)
    if err == nil {
        _ = embRepo.SaveEmbedding(ctx, domainAlbum.ID, embedder.Model(), embedder.Dim(), vec)
    } else {
        log.Printf("discovery: lazy embed failed for %s: %v", domainAlbum.ID, err)
    }
}
```

Note: `BuildAlbumEmbeddingText` needs the artist's name and biography excerpt. The `Album` struct has `ArtistName` but not the bio. Either fetch the artist (which the album path doesn't currently do) or build the text from album-only fields. Recommended: build from album-only fields in v1; folding in artist context is a follow-up.

### Phase 5 — Reindex CLI

**Step 5.1 — `cmd/reindex/main.go`.**

Flags:

- `--prune-old` — after backfill, delete embeddings for any model other than the current default.
- `--limit N` — only process the first N missing-embedding albums (useful for testing on a subset).
- `--dry-run` — log what would be embedded without making API calls.

Behavior:

1. Load config like `cmd/server/main.go` does.
2. Open the SQLite store.
3. Build the embedder via `embeddings.NewFromConfig(cfg.Discovery)`.
4. Query `albums` LEFT JOIN `album_embeddings ON mbid = albums.id AND model = <current>` — albums where the join is null are the work list.
5. For each album, build the embedding text, call `Encode`, write the result, log progress.
6. Save progress every `ReindexBatchSave` records (transactional commit).
7. On Ctrl-C, save current progress and exit cleanly. Re-running picks up where it left off because the work list is "albums without an embedding for the current model."

This is a one-binary, one-call tool. No worker pool — sequential is fine at FreqShow's volume.

### Phase 6 — Frontend

**Step 6.1 — `services/discover.service.ts`.**

```ts
@Injectable({ providedIn: 'root' })
export class DiscoverService {
  constructor(private http: HttpClient) {}
  discover(query: string, alreadyKnown: string[] = []): Observable<DiscoveryResult> {
    return this.http.post<DiscoveryResult>('/api/discover', { query, alreadyKnown });
  }
}
```

Models go in `models/discovery-types.ts` (or extend `openapi-types.generated.ts` if there's an OpenAPI flow).

**Step 6.2 — `pages/discover/discover.component.ts`.**

A single textarea, an "Already know about" comma-separated input (optional), a Discover button, and a results section. While loading: a spinner. On error: a message. On success: the interpreted query in a collapsible accordion (debug aid), then 5 picks.

Each pick is the existing album card (reuse from search results) plus an expandable section showing `whyItFits` and `whatToListenFor`. Spotify search URL renders as a "Find on Spotify" button.

**Step 6.3 — Add `/discover` to `app.routes.ts`.**

```ts
{ path: 'discover', loadComponent: () => import('./pages/discover/discover.component').then(m => m.DiscoverComponent) }
```

**Step 6.4 — Frontend test (`discover.component.spec.ts`).**

Cover: empty input doesn't submit; submitting calls the service; loading state renders; error state renders; picks render correctly. Match the test depth of the existing `home.component.spec.ts`.

### Phase 7 — Documentation and backlog

**Step 7.1 — Update `BACKLOG.md`.**

Move "Discovery Mode" from "What's Next" to "In Progress" with a pointer to ADR-0001 and this plan. Add a note that "Related Artists" and "themed browsing" can ride on the same `album_embeddings` table once it exists.

**Step 7.2 — Update `README.md`.**

Add a brief "Discovery Mode" section under "Current Features" once the feature lands. Update the env-vars list to include the four new `DISCOVERY_*` vars. Update the API examples with a `POST /discover` curl invocation.

**Step 7.3 — Update `agent-context/development-log.md`.**

A new entry summarizing the architectural choices (link to ADR-0001), the corpus strategy (lazy embed + reindex CLI), and any surprises encountered during the build.

---

## Prompts Specification

These are the verbatim system prompts. The user prompts are templates rendered at call time.

### Prompt A — Query Interpretation

**System prompt** (file `pkg/discovery/prompts.go` constant `interpretSystemPrompt`):

```
You are a music librarian who turns natural-language listening requests into structured search criteria for a music discovery system.

Given a user's freeform request, output ONLY a single JSON object with this exact schema:

{
  "mood": "string, short phrase capturing the emotional/atmospheric register",
  "eraHints": ["array of decade or year-range strings, may be empty"],
  "sonicQualities": ["array of descriptors like 'dense', 'sparse', 'electronic', 'analog warmth', 'instrumental'"],
  "referenceArtists": ["array of artists the user named, may be empty"],
  "avoid": ["array of qualities, artists, or albums the user explicitly wants to avoid"],
  "discoveryAppetite": "low | medium | high — how far from the references the user wants to stretch"
}

`discoveryAppetite` is your best guess at the user's openness:
- "low" if they want something close to references they named
- "medium" if they want adjacent territory
- "high" if they want surprise

Example user request:
"I love Radiohead's In Rainbows but want something less melancholy, more textural — instrumental preferred. Maybe something with electronic elements but organic-feeling."

Example output:
{
  "mood": "textural and organic, less melancholy than In Rainbows",
  "eraHints": [],
  "sonicQualities": ["textural", "organic", "electronic but warm", "instrumental or near-instrumental"],
  "referenceArtists": ["Radiohead"],
  "avoid": ["heavy melancholy", "vocal-driven"],
  "discoveryAppetite": "medium"
}

Output only the JSON. No prose. No markdown fences.
```

**User prompt template:**

```
Listening request: {{.Query}}

Artists the user already knows well and does not want repeated: {{.AlreadyKnownOrNone}}
```

`AlreadyKnownOrNone` is `"(none)"` if the slice is empty, else comma-separated.

### Prompt B — Curation and Reasoning

**System prompt** (constant `curateSystemPrompt`):

```
You are a music critic and discovery guide. You receive a structured listening request and a list of candidate albums. Pick the best 5 picks and explain why each fits.

Output ONLY a single JSON object with this exact schema:

{
  "picks": [
    {
      "rank": 1,
      "albumId": "must match one of the input candidate IDs exactly",
      "whyItFits": "2-3 sentences referencing both the user's request and the album's qualities",
      "whatToListenFor": "1-2 sentences naming a specific musical detail to notice"
    }
    // ... 5 entries total, ranks 1 through 5
  ]
}

Rules:
- Use ONLY albumIds present in the candidate list. Do not invent.
- If `discoveryAppetite` is `high`, prefer picks the user is unlikely to have already heard.
- If `discoveryAppetite` is `low`, prefer picks closely tied to the named reference artists.
- `whyItFits` should reference the user's mood, sonic qualities, or reference artists by name.
- `whatToListenFor` should be a concrete musical observation (an instrument moment, a structural choice, a textural feature) — not generic praise.
- Output ONLY the JSON. No prose. No markdown fences.
```

**User prompt template:**

```
Interpreted listening request:
{{.InterpretedJSON}}

Candidate albums:
{{range $i, $a := .Candidates}}
{{add $i 1}}. albumId: {{$a.ID}}
   {{$a.Title}} — {{$a.ArtistName}} ({{$a.Year}})
   Genre: {{$a.Genre}}
   {{if $a.Review.Summary}}Review summary: {{$a.Review.Summary}}{{end}}
{{end}}

Pick 5. Output the JSON.
```

The `add` template func is registered when parsing.

---

## JSON Schemas

These are the wire formats and their validation rules — the source of truth for what Go validators must enforce.

### `POST /discover` request

```json
{
  "query": "saturday morning, jazzy but modern, nothing harsh",
  "alreadyKnown": ["Kamasi Washington"]
}
```

- `query` is required, non-empty, ≤ 1000 chars after `strings.TrimSpace`.
- `alreadyKnown` is optional, may be empty or omitted.

### `POST /discover` response (200 OK)

```json
{
  "interpreted": {
    "mood": "...",
    "eraHints": [...],
    "sonicQualities": [...],
    "referenceArtists": [...],
    "avoid": [...],
    "discoveryAppetite": "medium"
  },
  "picks": [
    {
      "rank": 1,
      "albumId": "<mbid>",
      "title": "...",
      "artistName": "...",
      "year": 2019,
      "whyItFits": "...",
      "whatToListenFor": "...",
      "spotifySearchUrl": "https://open.spotify.com/search/..."
    }
    // 5 entries
  ]
}
```

If the avoid filter empties the candidate set entirely, return `picks: []` with HTTP 200. Not an error.

### Validation rules for LLM output

Interpretation:
- All seven keys present, no extras (extras are tolerated and ignored).
- List-typed keys are JSON arrays (may be empty).
- `discoveryAppetite ∈ {"low", "medium", "high"}`.

Curation:
- `picks` is exactly 5 items.
- Each pick has `rank`, `albumId`, `whyItFits`, `whatToListenFor`.
- `rank` values are exactly `{1, 2, 3, 4, 5}` with no duplicates.
- Each `albumId` matches one of the input candidates' MBIDs exactly.

Failure mode for either schema: one retry with a "your last response was not valid JSON in the required shape; produce only the JSON object" reminder appended to the user prompt. If the second attempt also fails, return the 502 from the [error table](#10-error-handling-the-existing-apierror-pattern).

---

## The Embedding-Text Builder

This function produces the per-album prose string fed to the embedder. It is the spiritual replacement for Project 7's LLM-generated blurbs and the single most important piece of "data engineering" in this feature.

### Field selection

For an album, in order of preference:

1. **Identity anchors** — `Title`, `ArtistName`, `Year`. These ground the embedding so retrieval can match on naming.
2. **Genre / type** — `Album.Genre` if non-empty, else first 3-5 of `Artist.Genres` (the album record doesn't currently carry genres of its own; the artist's genres are a reasonable proxy and often more specific than the album-level genre tag).
3. **Review summary** — `Album.Review.Summary` if non-empty.
4. **Review excerpt** — first ~400 chars of `Album.Review.Text` if non-empty, with `...` if truncated.
5. **Track titles** — concatenate `Album.Tracks[].Title` separated by `, `, capped at the first 8 titles. Track titles often carry mood signal ("Black Star", "How To Disappear Completely").

If after concatenation the result is < `MinEmbeddingTextChars` (120), return empty string. The album is too thin to embed usefully right now. The reindex CLI will retry it after more data has been hydrated.

### Output format

A short paragraph, not a structured field list. Embedding models are trained on natural language; sentences retrieve better than `key: value` dumps. Suggested structure:

```
"In Rainbows" by Radiohead, released 2007. Genre: art rock. Pitchfork (8.5/10): A pivot from the band's electronic experiments back toward warmth, with emotionally direct songwriting and inventive arrangements. Tracks include: 15 Step, Bodysnatchers, Nude, Weird Fishes/Arpeggi, All I Need, Faust Arp, Reckoner, House of Cards.
```

The exact joining text matters less than: (a) album title and artist appear in natural-language form, (b) genre / qualities appear, (c) review prose is present if available, (d) it reads like text a human wrote.

### Skipping vs. embedding

- Empty result (< MinEmbeddingTextChars): skip embedding, log `discovery: embedtext: thin record %s, skipping` at debug level.
- Embedding HTTP failure: log warn, do not fail the album cache. Reindex CLI will pick it up.
- Successful embedding: persist via `SaveEmbedding`.

---

## HTTP API

### `POST /discover`

Request: see [Schemas](#post-discover-request).

Response codes:

- 200 OK — full result including possibly empty `picks` array.
- 400 Bad Request — invalid JSON body, empty query, or query > 1000 chars.
- 502 Bad Gateway — LLM or embedding provider unreachable, or LLM produced invalid output twice.
- 503 Service Unavailable — corpus has zero embeddings (cold start before any albums have been hydrated). Message tells the user to come back after some catalog warming.

### `GET /healthz` extension (optional)

Add a `discovery` field to the health response indicating whether the discovery service is wired up. Helpful for confirming env var configuration on Render without making a real LLM call.

```json
{ "status": "ok", "discovery": "ready" }
```

`"ready"` if both providers initialized; `"unconfigured"` if either env var is missing; `"degraded: <reason>"` if init failed.

---

## The `cmd/reindex` Tool

Usage:

```
go run ./cmd/reindex                          # backfill missing embeddings for the current model
go run ./cmd/reindex --limit 50               # backfill at most 50 albums (smoke test)
go run ./cmd/reindex --prune-old              # after backfill, delete embeddings for non-current models
go run ./cmd/reindex --dry-run                # log work plan without making API calls
```

Behavior:

1. Resolve config exactly like `cmd/server/main.go`.
2. Open the SQLite store (read-write).
3. Build the embedder via `embeddings.NewFromConfig(cfg.Discovery)`.
4. SELECT albums with no `album_embeddings` row for the current model. (`LEFT JOIN ... WHERE album_embeddings.mbid IS NULL`.)
5. For each album: build text, encode, save, advance.
6. Commit a transaction every `ReindexBatchSave` (25) records.
7. Catch SIGINT/SIGTERM, commit current batch, exit 0.
8. If `--prune-old`, after backfill: `DELETE FROM album_embeddings WHERE model != ?` with the current model.

Exit codes:

- 0 — backfill complete (or `--dry-run` finished).
- 1 — config or store init failed.
- 2 — provider auth failed.
- 130 — interrupted; partial progress saved.

---

## Common Mistakes to Avoid

- **Embedding the JSON of the interpreted query directly.** Build natural-language prose from the structured fields first. Embedding `{"mood": "textural", ...}` puts braces and field names in the input.
- **Letting the curator see the raw user query.** It receives the *interpreted* JSON. Mixing them invites the LLM to overweight the freeform phrasing and ignore the structured signal.
- **Using `text/template` for the system prompts.** They're constants, not templates. Only the user prompts are templates.
- **Hard-coding the API key.** Always via env vars surfaced in `config.go`. Never in source. Never logged.
- **Trusting the LLM's `albumId` without validation.** The model can invent IDs. The validator must check `albumId ∈ {ids of input candidates}` and retry on miss.
- **Filtering avoid before retrieval.** Let the embedder do its job, then apply hard filters. Filtering before retrieval distorts the nearest-neighbor result.
- **Forgetting `(mbid, model)` as the composite primary key.** Without `model` in the PK, you can't run two embedding versions in parallel during a swap.
- **Storing vectors as JSON.** 4 × dim raw bytes is smaller and faster. Encode/decode helpers in `pkg/db/sqlite.go`.
- **Loading embeddings on every request.** Cache the records in the `discovery.Service` between requests. Invalidate when the reindex CLI runs (or just on a 5-minute TTL — simpler).
- **Failing the album lookup when lazy embedding fails.** Embedding is best-effort. Album lookup is the user's request. They're independent.
- **Mounting `/discover` outside `/api/`.** The Angular SSR proxy in `apps/frontend/server.ts` strips `/api` before forwarding. Frontend calls `/api/discover`; backend mounts `/discover`.
- **Forgetting the existing CORS middleware.** Already wraps the router. Just mount and go.
- **Reusing `searchCache`'s 5-minute TTL for discovery results.** Search cache is fine for raw MusicBrainz queries; discovery results are user-specific and shouldn't be cached across users. If caching is added later, it should be per-user-per-query.
- **Building the candidate list from `albums` instead of `album_embeddings`.** Albums without embeddings can't be retrieval candidates. The retrieval path must filter to embedded ones.
- **Forgetting to update `BACKLOG.md`.** The "Related Artists" and "themed browsing" notes need to point at the embedding table that now exists.

---

## Validation Test Cases

Run these before considering the feature shippable.

| Scenario | Expected Behavior |
| --- | --- |
| Bad LLM API key | First LLM call returns auth error; handler returns 502 with `"LLM provider unreachable"`. Logs the underlying error. |
| Bad embedding API key | Query embedding fails with 502; handler returns 502 with `"embedding provider unreachable"`. |
| Empty corpus (no `album_embeddings` rows) | Handler returns 503 with the cold-start message. |
| Empty `query` | Handler returns 400 with `"query field required"`. No LLM calls made. |
| `query` > 1000 chars | Handler returns 400 with `"query too long (max 1000 characters)"`. |
| Happy path: "I love Radiohead's In Rainbows but want something less melancholy, more textural" | Returns 200 with an interpretation matching the few-shot example structurally; 5 picks; no Radiohead picks; at least 2 instrumental or near-instrumental. |
| Happy path: "Saturday morning coffee, jazzy but modern, nothing harsh" | Returns 200 with picks skewing toward contemporary jazz / nu-jazz; no death metal slips through. |
| `alreadyKnown: ["Bon Iver"]` with a query that would naturally surface Bon Iver | Bon Iver does not appear in the picks. Substring match on artist name. |
| `discoveryAppetite: "high"` (after interpretation) | Curation prose explicitly mentions stretching beyond references; picks skew toward less canonical names. |
| LLM returns invalid JSON on first call | Service retries once with a shape reminder. |
| LLM returns invalid JSON twice | Returns 502 with `"LLM produced invalid response"`. Logs both raw responses. |
| LLM hallucinates an `albumId` not in the candidates | Retries once. If the retry also hallucinates, returns 502. |
| Avoid filter empties candidate set | Returns 200 with `picks: []`. Not an error. |
| Run two queries in the same process | Both succeed; corpus embeddings loaded once and cached in `discovery.Service`. |
| `cmd/reindex` on a fresh empty SQLite | No-op (no albums). Exits 0. |
| `cmd/reindex` after hydrating 5 albums | Embeds all 5, writes them, exits 0. |
| `cmd/reindex` interrupted with SIGINT during a batch | Commits the current batch's progress, exits 130. Re-running picks up where it left off. |
| `cmd/reindex --prune-old` after a model swap | Removes embeddings for the previous model, leaves current ones. |
| Concurrent lazy embedding from two album lookups | Both succeed; SQLite UPSERT handles the race. |
| Frontend `/discover` route with valid query | Renders 5 picks with collapsible `whyItFits` / `whatToListenFor` and Spotify links. |
| Frontend with empty query | Submit button disabled or input validation prevents submit. |
| Frontend during loading | Spinner visible, results section hidden. |
| Frontend on 503 (cold corpus) | Shows the cold-start message, suggests visiting some artist pages first. |

---

## Final Checklist

- [ ] `pkg/discovery/` exists with the file layout in [Package Layout](#package-layout).
- [ ] `pkg/sources/embeddings/` and `pkg/sources/llm/` exist with provider router + at least one implementation each.
- [ ] `pkg/data/discovery.go` defines `DiscoveryQuery`, `InterpretedQuery`, `Pick`, `DiscoveryResult` with camelCase JSON tags.
- [ ] `pkg/db/db.go` adds `EmbeddingRepository` interface; `MemoryStore` and `SQLiteStore` implement it.
- [ ] SQLite migration creates `album_embeddings` with the composite PK from Decision 3.
- [ ] Vector blob encoding is raw little-endian f32, not JSON.
- [ ] `pkg/config/config.go` surfaces all four `DISCOVERY_*` env vars; `.env.example` documents them.
- [ ] No API keys committed anywhere.
- [ ] Two LLM calls per query, different temperatures, different system prompts.
- [ ] Prompt A (interpretation) uses a system role, structured JSON schema, and the few-shot example.
- [ ] Prompt B (curation) uses a system role, structured input format with the interpreted query plus numbered candidates, and `discoveryAppetite` instruction.
- [ ] Embedder model and dim are written to every embedding row.
- [ ] Retrieval is two-stage: top-30 cosine, then MMR rerank to top-10 with `λ=0.7`.
- [ ] Avoid filter runs after retrieval, before curation, with case-insensitive substring matching on artist or title.
- [ ] Final 5 picks are chosen by the LLM curation step from the post-filter candidates.
- [ ] All API/parse failures fail loudly with a clear message and the right HTTP status.
- [ ] One retry on JSON shape failure for each LLM call; second failure is fatal.
- [ ] No `_ = err` swallows. No silent fallbacks.
- [ ] `cmd/reindex` works, is resumable, and skips already-embedded records for the current model.
- [ ] `cmd/reindex --prune-old` removes embeddings for non-current models.
- [ ] Lazy embedding in `getOrFetchAlbum` is best-effort; embedding failures don't fail album lookups.
- [ ] HTTP `POST /discover` request and response shapes match [JSON Schemas](#json-schemas).
- [ ] Angular `/discover` route renders interpreted query (collapsed by default), 5 picks with expandable reasoning, and Spotify search links.
- [ ] All validation test cases above pass.
- [ ] `BACKLOG.md`, `README.md`, and `agent-context/development-log.md` are updated.
- [ ] `go test ./...` passes; `go vet ./...` is clean.

---

## Expected Outcome

When `dev.sh` is running with valid `DISCOVERY_*` env vars and at least a few albums in the cache:

1. The user visits `/discover`, types "Saturday morning coffee, jazzy but modern, nothing harsh" into the textarea, and clicks Discover.
2. The Angular `DiscoverService` POSTs to `/api/discover` (proxied to the Go backend's `/discover`).
3. The handler parses the request, validates it, and calls `discovery.Service.Run`.
4. `Run` interprets the query via the LLM (one ChatComplete call, `temp=0.3`), parses and validates the JSON, embeds the natural-language form of the interpreted query (one Encode call), loads the corpus embeddings for the current model from SQLite, computes top-30 by cosine, MMR-reranks to top-10, applies the avoid filter, calls the LLM for curation (one ChatComplete call, `temp=0.7`), parses and validates the JSON, joins picks back to album metadata, and returns the result.
5. The frontend renders 5 picks with `whyItFits` / `whatToListenFor` reasoning and Spotify search links.
6. Console logs across the run include stage banners with timing for interpret / retrieve / curate. Logs are clean — no shape-failure retries on the happy path.
7. Total request time is ~3-5 seconds, dominated by the two LLM calls.
8. `cmd/reindex` can be run on demand to backfill any albums hydrated since the last reindex; in normal operation, lazy embedding on hydration handles new albums without manual intervention.
9. `BACKLOG.md` shows "Discovery Mode" as in-progress with pointers to ADR-0001 and this plan; "Related Artists" and "themed browsing" are noted as ready to ride on the same `album_embeddings` table.
