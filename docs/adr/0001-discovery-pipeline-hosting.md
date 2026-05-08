# ADR-0001: Hosting strategy for the AI music discovery pipeline

**Status:** Proposed (Action Item #1 — embedding provider benchmark — resolved 2026-05-08)
**Date:** 2026-05-05
**Deciders:** Adam LaCasse

## Context

FreqShow's `README.md` and `BACKLOG.md` both name an unbuilt "Discovery Mode" — natural-language listening requests resolved to ranked album recommendations with editorial reasoning ("Saturday morning coffee, jazzy but modern, nothing harsh"). No architecture exists for it yet.

A working reference implementation of the pipeline shape exists outside this repo: the Project 7 capstone in `Merrimack_CSC6314/Module 7/Project 7/`. Its design is portable; its implementation is not. The pipeline has four stages:

1. **Interpret** — instruction-tuned LLM call turns freeform text into structured JSON (mood, sonic qualities, reference artists, avoid list, discovery appetite).
2. **Embed + retrieve** — embedding model produces a vector for the interpreted query; cosine similarity against a pre-embedded corpus returns top-K.
3. **Diversify** — Maximal Marginal Relevance reranks top-K to a diverse top-N, preventing five-near-duplicates results.
4. **Curate** — second LLM call ranks final 5 picks and writes "why it fits" + "what to listen for" prose against a strict JSON schema.

Forces on the hosting decision:

- **Stack mismatch.** FreqShow's backend is Go 1.22 (`apps/server`). Project 7's pipeline is Python (`huggingface_hub`, `sentence-transformers`, `numpy`). Decided: Go wins. The discovery pipeline lives in Go, calling whatever inference services it needs over HTTP.
- **Soft cost ceiling: ~$10–20/month.** Above the existing Render Hobby Legacy plan, until the project monetizes or attracts meaningful user volume. Recurring costs that compound dramatically with corpus growth need a credible bound; small fixed line items are acceptable.
- **Deploy target is Render Hobby (Legacy)** for the Go backend — single web service, ~512 MB RAM, always-on (no idle spin-down), persistent disk attachable as a separate optional feature. The Angular SSR frontend runs as a separate Render service. The 512 MB RAM ceiling is the load-bearing operational fact for this decision.
- **Corpus is no longer 250 hand-curated rows.** In FreqShow the embeddable corpus is whatever has been hydrated into the SQLite cache from MusicBrainz + Wikipedia + Discogs. It grows incrementally and is unbounded in principle. The Project 7 LLM-generated blurb workaround goes away — we have real source text now, which is a strict upgrade in data provenance.
- **Solo developer, no SLO commitments.** Operational simplicity is the scarcest resource. A second runtime is a permanent ops tax, not a one-time setup cost.
- **The same embedding infrastructure unlocks two other backlog items** — "Related Artists" via artist-level embeddings; "themed browsing" via genre-prototype embeddings — so the choice has multiplier effects.

### Where the budget actually gets spent (and where it doesn't)

LLM-call volume at "you and a few friends" is trivial. Two chat-completion calls per query × low query rate fits comfortably inside HF Inference API's free-tier quota, or costs under a dollar per month on `gpt-4o-mini` or Claude Haiku.

Embedding inference is also small at this scale. A complete backfill of a 5K-album corpus at ~500 embedding tokens per album is 2.5M tokens, which on `text-embedding-3-small` ($0.02/1M tokens) is five cents one-time. Live indexing as new albums get hydrated, plus query-time embeddings (~100 tokens per query × low query volume), lands in the sub-dollar/month range. Embedding cost is not where the budget gets spent.

Where the budget *does* get spent is **memory headroom on the 512 MB Hobby Legacy instance**, which is fixed and not paid-for-able without moving to the $25/month Standard plan (above the ceiling). Running a local ~80 MB sentence-transformer model in-process leaves 250–350 MB of resident memory for the Go server, SQLite working set, ONNX runtime, and request-handling overhead. That's "works in dev, gets OOM-killed under bursty traffic" territory, with no swap to soften the spikes. The natural mitigation — swapping to the smaller `paraphrase-MiniLM-L3-v2` — buys headroom by giving back retrieval quality, which is the wrong direction when hosted alternatives cost pennies.

This is the inversion that drives the decision: **memory is the fixed resource; embedding-inference cost is the variable and trivially small one.** Spend the variable resource to protect the fixed one.

## Decision

**Adopt Option C: a pure-Go discovery pipeline that calls hosted embedding and LLM endpoints over HTTP.** No new language, no new service, no in-process ML model. The pipeline lives at `apps/server/pkg/discovery/`, with a handler wired into the existing router. The vector index is persisted in SQLite alongside the existing JSON-blob caches and built incrementally as albums get hydrated.

**Embedding provider for v1:** Voyage `voyage-3-lite` (512-dim). Selected after running the Action Item #1 benchmark (Voyage, HF MiniLM-L6, OpenAI `text-embedding-3-small`) against a 30+ album corpus and 14 representative queries. All three tied on retrieval quality on the rubric used; the choice fell to operational properties. Voyage offers the most generous free tier (covers expected usage indefinitely without billing setup), no cold-load latency on first call, and no minimum-balance gate. HF was comparable on quality but its ~1000-call/day free-tier rate limit and ~20-second cold-load on the first call after idle make it operationally unsuitable for the lazy-embedding-on-hydration path. OpenAI was viable but requires a positive account balance for any call — a gate we hit during benchmark testing, which is exactly the kind of friction the cost-ceiling discussion was meant to avoid in this phase.

**Benchmark methodology caveat.** The benchmark tested embedders against raw user queries, not the *interpreted* queries the production pipeline will actually embed (which are produced by the interpretation LLM call and contain structured fields rather than freeform text). The "no material difference" finding may not hold once the interpretation step is in place — interpreted queries are richer and more semantically loaded. Re-validation against interpreted queries is a follow-up after Phase 3 of the implementation plan lands. If a different provider wins clearly on interpreted queries, the `Embedder` interface makes the swap a one-line change.

**LLM provider for v1:** HF Inference API free tier, using the same instruction-tuned chat models the Project 7 client targets. If free-tier rate limits start hurting, swap to OpenAI `gpt-4o-mini` or Anthropic Claude Haiku — both well under a dollar/month at expected volume, and the chat-completion shape is similar enough that the swap is a client-class change, not a redesign.

Both providers sit behind small Go interfaces (`Embedder.Encode(ctx, text) ([]float32, error)` and a chat-completion equivalent) so swapping providers is a wiring change, not a refactor. A `LocalONNXEmbedder` is *not* implemented in v1 but is named in the design as a future contingency if hosted dependencies become unsuitable or the project moves to a plan with more memory headroom.

The Project 7 *code* is treated as a throwaway reference implementation. The Project 7 *design* — prompts, JSON contracts, MMR formula, fail-loudly policy, candidate-id validation, two-call temperature split — transfers verbatim.

## Options Considered

### Option A: Python sidecar service

A new app `apps/discovery/` in the monorepo running a small FastAPI/Flask service. Loads `sentence-transformers/all-MiniLM-L6-v2` locally, calls HF Inference API for the LLM, owns its own vector index. Go calls it over HTTP. Project 7's Python code transplants almost directly.

| Dimension | Assessment |
|---|---|
| Complexity | High — adds a runtime, build pipeline, and IPC contract to a previously single-language monorepo |
| Cost | Low marginal on inference, but a separate Render service is +$7/month minimum |
| Scalability | Good for moderate corpora; scales by giving the sidecar more RAM |
| Team familiarity | High — Adam just shipped a Python sentence-transformers pipeline in Project 7 |
| Operational burden | Two services to deploy, supervise, observe, version, and keep in sync |
| Memory footprint | Isolated from the Go server's 512 MB ceiling — but only because the second Render service has its own RAM budget |

**Pros:** Direct reuse of Project 7 code. Python is the well-trodden path for `sentence-transformers`. Embedding compute is a sunk hardware cost rather than a metered API charge.
**Cons:** Permanently doubles operational surface area and adds at minimum +$7/month for the second Render service. The same memory isolation it provides is achievable more cheaply by simply not running the model anywhere we own (Option C).

### Option B: Pure Go with local ONNX embeddings

Vendor a quantized ONNX export of `all-MiniLM-L6-v2` (~25 MB int8, ~80 MB fp32) into the Go binary. Run inference via `onnxruntime-go` (CGO bindings to ONNX Runtime). Use `github.com/sugarme/tokenizer` or `github.com/daulet/tokenizers` for the BERT tokenizer to avoid hand-rolling parity with HF's. LLM calls go to HF Inference API directly over HTTP.

| Dimension | Assessment |
|---|---|
| Complexity | Medium — Go ML tooling has matured; vendored tokenizer libraries handle the parity problem |
| Cost | Zero marginal on embeddings; LLM calls inside HF free tier or trivial paid spend |
| Scalability | Excellent on indexing throughput (no rate limits); query-time embedding is in-process and ~10–30 ms |
| Team familiarity | Low — Adam has not done Go-native ML work before, but the surface area is small |
| Operational burden | Low — single binary, no extra services |
| Memory footprint | ~150–250 MB resident for loaded model + runtime — too tight for the 512 MB Hobby Legacy plan under load |
| Speed to first version | Medium — wiring + tokenizer integration + parity validation |

**Pros:** No external dependency for embeddings. Faster query-time embedding than a network round-trip. Indexing throughput bounded by CPU, not API rate limits. Survives provider outages.
**Cons:** **Memory pressure on the 512 MB Hobby Legacy plan is the disqualifier.** Loaded model + ONNX runtime + Go server + SQLite working set + request buffering pushes resident memory into the 250–350 MB range, which is fragile under bursty load with no swap. Mitigations (smaller L3 model, paid Standard plan) either degrade retrieval quality or exceed the cost ceiling. Tokenizer-parity validation and CGO build complexity are real one-time costs that hosted embeddings sidestep.

### Option C: Pure Go with hosted embeddings (recommended)

Go server calls a hosted embedding API for both query-time and indexing embeddings. The discovery pipeline is a Go package under `apps/server/pkg/discovery/`. The vector index is persisted in SQLite (a new `album_embeddings` table holding `(mbid, vec BLOB, model, dim, updated_at)`) and built incrementally as albums get hydrated. Brute-force in-memory cosine over loaded vectors is the v1 retrieval; defer a vector-index extension until measured need.

| Dimension | Assessment |
|---|---|
| Complexity | Low — no new runtime, no model files, no tokenizers; just HTTP clients and float32-slice math |
| Cost | Free tier covers expected usage indefinitely on Voyage (the chosen provider). 5K-album backfill is ~2.5M tokens, well within Voyage's free-tier headroom. Paid tier across providers is comparable (~$0.02/1M tokens) if free tier is ever exhausted. |
| Scalability | Bounded by hosted-API rate limits; paid tiers have generous limits |
| Team familiarity | High — Go HTTP clients and `[]float32` math are well within Adam's Go comfort zone |
| Operational burden | Low — same one-binary deployment as today |
| Memory footprint | Negligible — no in-process model |
| Embedding quality | Higher than locally-runnable L6: `text-embedding-3-small` and Voyage-3-lite both outperform MiniLM-L6 on retrieval benchmarks |
| Speed to first version | Fast — most of the work is wiring, not modeling |

**Pros:** Single-language monorepo stays single-language. No new deployment artifact. Cleanest mental model. No memory pressure on the 512 MB Hobby Legacy ceiling. Better embedding quality than a locally-runnable model. Cost is small at this scale and predictable.
**Cons:** Outbound dependency on the hosted embedding provider (rate limits, downtime, lock-in risk — mitigated by the `Embedder` interface). Cold-start for a freshly hydrated album means a network call before that album is queryable (~100–300 ms typical). Per-call cost grows with corpus growth, but the growth rate stays trivial within any realistic 12-month projection.

## Trade-off Analysis

The decision turns on which resource is fixed and which is variable.

Memory on the 512 MB Hobby Legacy instance is **fixed** — there is no incremental knob. The next plan up (Standard at $25/month) exceeds the cost ceiling, and even on that plan, Option B's local model still consumes the headroom rather than the application benefiting from it. Embedding-inference cost, by contrast, is **variable and trivially small** at FreqShow's expected scale: pennies-per-month, growing linearly with corpus size, with no rate-limit cliff on paid tiers. Spending the variable resource to protect the fixed one is the rational play.

This inverts the conventional default of "self-host where possible to avoid recurring costs." That default assumes the recurring cost is meaningful relative to the fixed cost of running the application, and that self-hosting doesn't constrain the application's primary resource. Neither holds here.

Option A (Python sidecar) solves the memory problem the same way Option C does — by isolating the model from the main service's RAM budget — but pays for it with a permanent second runtime and at minimum +$7/month for the second Render service. Option C achieves the same isolation by not running the model anywhere we own. The operational simplicity advantage is decisive at solo-developer scale.

Option B's tokenizer-parity concern is solvable with mature vendored libraries — that's not the disqualifier. The disqualifier is that ~250 MB resident in a 512 MB ceiling is fragile under load, and the mitigations all cost something the project either can't afford (Standard plan) or shouldn't pay (retrieval quality).

The Project 7 code does not transfer line-for-line under any option, but the pipeline *design* does, and that's the load-bearing intellectual content. Having a working Python reference to compare against during the Go port is worth more than transplanting source.

## Consequences

What becomes easier:

- Deployment, dev environment, and CI stay single-language. One Go binary. One Render service.
- Memory headroom on the Hobby Legacy plan stays comfortable as the SQLite cache grows.
- Embedding model upgrades (e.g., from `text-embedding-3-small` to `-3-large` if quality matters more later) are a one-line change.
- "Related Artists" and "themed browsing" reuse the same `album_embeddings` infrastructure with no new services.
- The `Embedder` interface keeps the door open to local inference if hosted ever becomes unsuitable.

What becomes harder:

- Outbound HTTP dependency on the embedding provider becomes a request-path requirement. Provider downtime = no recommendations during the outage. (Mitigated somewhat by query-side caching of interpreted query → vector for repeat queries.)
- Indexing on cache loss costs a re-backfill against the hosted API. At 5K albums × $0.05 per backfill, this is a non-event; at 50K × $0.50, still acceptable; at 500K × $5, worth re-evaluating.
- Latency floor includes a network hop on every query for the interpreted-query embedding (~100–300 ms typical). Negligible in absolute terms but worth keeping in mind for UX.
- Lock-in risk with the chosen provider. Mitigated by the `Embedder` interface and the SQLite `model` and `dim` columns, which let multiple embedding versions coexist during a swap.

What we'll need to revisit:

- If sustained query volume crosses ~10 qps or the corpus crosses ~50K rows, re-run the cost projection. Even at hosted prices, the math eventually flips.
- If the v1 provider underperforms on the validation benchmark (Action Item #1), swap before locking in.
- If a Render plan upgrade for memory becomes justifiable on other grounds (query latency, larger SQLite working set), Option B becomes attractive again as a way to remove the hosted-API dependency.

## Action Items

1. [x] **Embedding provider benchmark.** *Resolved 2026-05-08:* Voyage `voyage-3-lite` selected. All three providers tied on quality across the curated corpus (30+ albums) and queries (14); operational properties decided it. Benchmark script and inputs live at `scripts/embedding-benchmark/`; raw `results/` are local-only (gitignored). See Decision section for full rationale and the methodology caveat about raw-vs-interpreted queries.
2. [ ] **Confirm SQLite persistence on Render.** Per the dashboard checks (Disks tab, Environment tab, Shell `df -h` and `ls -la freqshow.db`). The answer informs whether re-backfill on deploy is normal-case or rare. Either way, Option C handles it; this is just to know what to expect.
3. [ ] **Design `apps/server/pkg/discovery/` package layout.** Proposed files: `types.go`, `prompts.go`, `embedder.go` (the `Embedder` interface plus a default provider implementation and an `HFEmbedder` contingency), `llm.go` (chat-completion client interface), `interpret.go`, `retrieve.go` (cosine + MMR), `curate.go`, `index.go` (persistence + incremental rebuild), `handler.go`.
4. [ ] **Vector-index storage.** New `album_embeddings` table with `(mbid TEXT PRIMARY KEY, vec BLOB, model TEXT NOT NULL, dim INTEGER NOT NULL, updated_at TIMESTAMP)`. The `model` and `dim` columns let multiple embedding versions coexist during a swap. Brute-force cosine over loaded `[]float32` is fine through ~10K rows.
5. [ ] **Embedding-text builder.** Define which fields from MusicBrainz tags + Wikipedia bio excerpt + Discogs review excerpts concatenate into the per-album embeddable string. This is the spiritual replacement for Project 7's LLM-generated blurbs and deserves its own short design note before coding.
6. [ ] **Build a resumable reindex job as `cmd/reindex`.** Iterates the corpus, embeds any record whose `model` or `dim` differs from the current default, writes results to `album_embeddings`, saves progress every N records. Survives Ctrl-C without losing progress. Useful for first backfill, model swaps, and recovery after cache loss.
7. [ ] **API key management.** Embedding provider key (and any future LLM provider keys) live in Render's environment variables, surfaced in `config.go` analogously to the existing Discogs OAuth fields. Never committed.
8. [ ] **Frontend wiring.** Add a `/discover` route with a single textarea input and a results view that reuses the existing album card component plus a per-pick expandable "why this fits" / "what to listen for" section.
9. [ ] **Update `BACKLOG.md`.** Mark "Discovery Mode" as in-progress under this ADR. Note that "Related Artists" and "themed browsing" can ride on the same embedding table once it exists.
10. [ ] **Write a follow-up implementation plan** (`docs/plans/discovery-pipeline-plan.md`) before any code lands, covering the file-by-file breakdown, prompt constants, validation contracts, and test cases — analogous to the Project 7 implementation plan.
