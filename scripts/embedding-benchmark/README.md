# Embedding Provider Benchmark

This is a one-shot eval to pick the v1 embedding provider for FreqShow's discovery pipeline (per ADR-0001 Action Item #1). It is intentionally *not* production code — it predates `pkg/sources/embeddings/` and uses Python SDKs directly. Re-run it whenever you're considering a provider swap or evaluating a new candidate.

## What it does

1. Loads a hand-curated corpus of albums (`corpus.json`) and natural-language queries (`queries.json`).
2. For each provider with an API key in the environment, embeds the full corpus and every query.
3. For each query, computes cosine similarity against the corpus and ranks the top-K matches.
4. Writes a side-by-side markdown report (`results/results.md`) showing top-5 retrieval per provider per query, plus a per-provider cost summary.

The output is designed for **eyeballing**, not automated scoring. ADR-0001 explicitly calls for "gut-check ground truth" — the `expected_vibe` field on each query is the human reviewer's note-to-self about what good results should look like.

## Setup

```bash
cd scripts/embedding-benchmark
python -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
```

API keys for the providers can come from either source:

- **Repo `.env` (recommended).** The script auto-loads `.env` from the script dir or repo root at startup. Anything in `freq-show/.env` is picked up automatically — the same file `dev.sh` already loads for the Go server. Existing shell env vars are not overwritten, so explicit exports still win.
- **Shell exports.** If you'd rather not put benchmark keys in the repo `.env`, just `export OPENAI_API_KEY=...` etc. before running.

Missing keys cause that provider to be skipped with a clear log line — partial runs are fine.

## Run

Recommended first-run sequence (see "Local testing strategy" below for why):

```bash
# 1. Validate corpus + queries JSON, eyeball the embedding texts. No API calls.
python benchmark.py --show-text | less

# 2. Run one provider on a tiny subset to confirm wiring.
python benchmark.py --providers openai --query-ids saturday-jazz --limit-corpus 8

# 3. Full run, all providers, all data.
python benchmark.py
```

Optional flags:

```
--providers openai,huggingface,voyage   # default: all
--query-ids id1,id2                     # subset queries by ID; default: all
--limit-corpus N                        # only embed first N albums; default: no limit
--show-text                             # dry run: print embedding texts and exit, no API calls
--top-k 5                                # default: 5
--out results/                           # default
--corpus corpus.json                     # default
--queries queries.json                   # default
```

Output (only when not `--show-text`):

- `results/results.md` — human-readable side-by-side comparison.
- `results/rankings.json` — raw top-K rankings for every (provider, query) pair, in case you want to do follow-up analysis.

## Local testing strategy

Run things in this order so you spend the smallest amount of effort and API quota to catch each class of bug.

1. **`--show-text` first.** Zero API cost. Catches malformed JSON, missing fields, the `MinEmbeddingTextChars=120` cutoff being tripped, and any ugly prose from the text builder. Read 5 or 6 of them and ask: would I want this fed into an embedding model?
2. **One provider, two queries, eight albums.** `--providers openai --query-ids <id1>,<id2> --limit-corpus 8`. Confirms env vars, network, JSON parsing of the response, and report rendering — for a few cents and a few seconds. Pick OpenAI first if you have a key; it's the most reliable of the three. HF free tier sometimes returns a 503 cold-load on first call (the script retries once after 20s, but it's a slower first impression).
3. **All providers, two queries, eight albums.** Same subset, all keys set. Confirms each provider's wiring independently before you commit to a full run.
4. **Full run.** All providers, full corpus, full queries. Read `results/results.md` top-to-bottom. The `expected_vibe` field on each query is your gut-check anchor.

If a provider fails partway through, the script logs the error and skips it — you'll still get useful results from the others. Re-run with `--providers <failed_one>` to retry just the broken one.

## Curating the corpus

The committed `corpus.json` is a 12-album starter spanning genres (jazz, soul, hip-hop, ambient, post-rock, folk, metal, electronic) and eras (1959–2015). Before relying on benchmark results, **expand it to ~30 albums** drawn from music you actually want FreqShow to recommend well. Coverage matters more than count — a wider net surfaces more provider-quality differences.

Likewise, `queries.json` has 6 starter queries covering reference-anchored, mood-anchored, era-anchored, and discovery-mode shapes. Add 4 more in your own voice.

## How to interpret the output

The report renders each query as a 3-column markdown table — one column per provider — with the top-5 albums listed in each. Read across the row at rank 1: do all three providers pick something defensible? At rank 5: which provider's tail still looks coherent vs. which is reaching?

Cost matters less than quality at this scale (a full benchmark pass is fractions of a cent for any of the three), so weight the eyeballing heavily. If two providers tie on quality, prefer the cheaper one. If one is clearly better, use it.

After running, update ADR-0001 with the choice and a one-paragraph note about why.

## Why Python and not Go

This script will run a handful of times in its life — once now, again on swaps, occasionally for a new candidate. The production discovery code is Go because production needs Go. The eval is Python because Python has the shortest path between intent and result for an eval. Don't import the eval pattern into production.
