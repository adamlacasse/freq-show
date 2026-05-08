"""
Embedding provider benchmark for FreqShow's discovery pipeline.

Compares OpenAI text-embedding-3-small, HuggingFace sentence-transformers/all-MiniLM-L6-v2,
and Voyage voyage-3-lite on a hand-picked corpus and a set of natural-language queries.
Output is a side-by-side markdown report for human eyeballing.

Run from this directory:
    python benchmark.py [--providers openai,huggingface,voyage] [--top-k 5]

API keys are read from the environment:
    OPENAI_API_KEY, HF_TOKEN, VOYAGE_API_KEY

Missing keys cause that provider to be skipped (logged, not fatal).
"""

from __future__ import annotations

import argparse
import json
import os
import sys
import time
from dataclasses import dataclass, field
from pathlib import Path
from typing import Callable, TypedDict

import numpy as np


# ── .env auto-loader (no python-dotenv dep) ───────────────────────────────────


def _autoload_dotenv() -> None:
    """Load .env into os.environ if found. Does NOT overwrite existing vars.

    Looks in the script dir first, then the repo root two levels up. Logs which
    file (if any) was loaded so the behavior is visible in run output.
    """
    here = Path(__file__).resolve().parent
    candidates = [here / ".env", here.parent.parent / ".env"]
    for path in candidates:
        if not path.is_file():
            continue
        loaded = 0
        for line in path.read_text().splitlines():
            line = line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            key, _, value = line.partition("=")
            key = key.strip()
            value = value.strip().strip('"').strip("'")
            if key and key not in os.environ:
                os.environ[key] = value
                loaded += 1
        print(f"[env] loaded {loaded} vars from {path}", file=sys.stderr)
        return
    print("[env] no .env found in script dir or repo root; using shell env only", file=sys.stderr)


_autoload_dotenv()


# ── Constants ─────────────────────────────────────────────────────────────────

DEFAULT_TOP_K = 5
DEFAULT_OUT_DIR = "results"
DEFAULT_CORPUS = "corpus.json"
DEFAULT_QUERIES = "queries.json"


class _ProviderConfig(TypedDict):
    name: str
    model: str
    dim: int
    price_per_1m_tokens_usd: float
    env_var: str


# Provider definitions: (id, display_name, model, default_dim, $/1M tokens, env_var)
PROVIDER_REGISTRY: dict[str, _ProviderConfig] = {
    "openai": {
        "name": "OpenAI text-embedding-3-small",
        "model": "text-embedding-3-small",
        "dim": 1536,
        "price_per_1m_tokens_usd": 0.02,
        "env_var": "OPENAI_API_KEY",
    },
    "huggingface": {
        "name": "HF all-MiniLM-L6-v2",
        "model": "sentence-transformers/all-MiniLM-L6-v2",
        "dim": 384,
        "price_per_1m_tokens_usd": 0.0,  # free tier; rate-limited
        "env_var": "HF_TOKEN",
    },
    "voyage": {
        "name": "Voyage voyage-3-lite",
        "model": "voyage-3-lite",
        "dim": 512,
        "price_per_1m_tokens_usd": 0.02,
        "env_var": "VOYAGE_API_KEY",
    },
}


# ── Embedding text builder (mirrors the Go production builder spec) ──────────


def build_embedding_text(album: dict) -> str:
    """Build the prose blurb that gets fed to the embedder.

    Mirrors the spec in docs/plans/discovery-pipeline-plan.md §"The Embedding-Text Builder".
    Stays in sync with what the Go BuildAlbumEmbeddingText will produce so benchmark
    results carry over to production.
    """
    parts = []
    parts.append(f'"{album["title"]}" by {album["artist"]}, released {album["year"]}.')
    if genre := album.get("genre"):
        parts.append(f"Genre: {genre}.")
    if summary := album.get("review_summary"):
        parts.append(summary)
    if tracks := album.get("tracks"):
        track_list = ", ".join(tracks[:8])
        parts.append(f"Tracks include: {track_list}.")
    return " ".join(parts)


# ── Provider clients ──────────────────────────────────────────────────────────


@dataclass
class EmbeddingResult:
    vectors: np.ndarray  # (N, dim) float32
    approx_tokens: int
    duration_seconds: float
    error: str | None = None


def embed_openai(texts: list[str], model: str) -> EmbeddingResult:
    from openai import OpenAI

    client = OpenAI()  # reads OPENAI_API_KEY from env
    start = time.perf_counter()
    try:
        resp = client.embeddings.create(input=texts, model=model)
    except Exception as e:
        return EmbeddingResult(
            vectors=np.zeros((0, 0), dtype=np.float32),
            approx_tokens=0,
            duration_seconds=time.perf_counter() - start,
            error=f"openai api error: {e}",
        )
    vectors = np.array([d.embedding for d in resp.data], dtype=np.float32)
    tokens = resp.usage.total_tokens if resp.usage else _approx_tokens(texts)
    return EmbeddingResult(
        vectors=vectors,
        approx_tokens=tokens,
        duration_seconds=time.perf_counter() - start,
    )


def embed_huggingface(texts: list[str], model: str) -> EmbeddingResult:
    from huggingface_hub import InferenceClient

    client = InferenceClient(token=os.environ.get("HF_TOKEN"))
    start = time.perf_counter()

    def _call(payload: list[str]):
        # Batch call. Returns a list-of-vectors when given a list; the SDK normalizes shape.
        return client.feature_extraction(payload, model=model)

    try:
        try:
            v = _call(texts)
        except Exception as e:
            # Cold-start: HF often returns 503 with `estimated_time` on the first call after idle.
            # Wait once and retry.
            msg = str(e).lower()
            if "loading" in msg or "503" in msg or "estimated_time" in msg:
                print(f"[hf]   model {model} appears to be cold-loading; waiting 20s and retrying...", file=sys.stderr)
                time.sleep(20)
                v = _call(texts)
            else:
                raise

        arr = np.array(v, dtype=np.float32)
        # Shape can be (N, dim) for batched sentence-transformers,
        # or (N, seq_len, dim) for token-level models — mean-pool the latter.
        if arr.ndim == 3:
            arr = arr.mean(axis=1)
        elif arr.ndim == 1:
            # Single-vector edge case if the SDK collapsed a one-element batch.
            arr = arr.reshape(1, -1)
    except Exception as e:
        return EmbeddingResult(
            vectors=np.zeros((0, 0), dtype=np.float32),
            approx_tokens=0,
            duration_seconds=time.perf_counter() - start,
            error=f"hf api error: {e}",
        )

    return EmbeddingResult(
        vectors=arr,
        approx_tokens=_approx_tokens(texts),
        duration_seconds=time.perf_counter() - start,
    )


def embed_voyage(texts: list[str], model: str) -> EmbeddingResult:
    import voyageai

    client = voyageai.Client()  # type: ignore[attr-defined]  # reads VOYAGE_API_KEY from env
    start = time.perf_counter()
    try:
        resp = client.embed(texts, model=model, input_type="document")
    except Exception as e:
        return EmbeddingResult(
            vectors=np.zeros((0, 0), dtype=np.float32),
            approx_tokens=0,
            duration_seconds=time.perf_counter() - start,
            error=f"voyage api error: {e}",
        )
    vectors = np.array(resp.embeddings, dtype=np.float32)
    tokens = getattr(resp, "total_tokens", _approx_tokens(texts))
    return EmbeddingResult(
        vectors=vectors,
        approx_tokens=tokens,
        duration_seconds=time.perf_counter() - start,
    )


PROVIDER_FNS: dict[str, Callable[[list[str], str], EmbeddingResult]] = {
    "openai": embed_openai,
    "huggingface": embed_huggingface,
    "voyage": embed_voyage,
}


def _approx_tokens(texts: list[str]) -> int:
    """Rough token count: chars / 4. Used when the provider doesn't report exact usage."""
    return sum(len(t) for t in texts) // 4


# ── Retrieval ─────────────────────────────────────────────────────────────────


def cosine_top_k(query_vec: np.ndarray, doc_vecs: np.ndarray, k: int) -> list[tuple[int, float]]:
    """Return [(idx, similarity), ...] sorted descending."""
    q = query_vec / (np.linalg.norm(query_vec) + 1e-12)
    d = doc_vecs / (np.linalg.norm(doc_vecs, axis=1, keepdims=True) + 1e-12)
    sims = d @ q
    idx_sorted = np.argsort(-sims)[:k]
    return [(int(i), float(sims[i])) for i in idx_sorted]


# ── Orchestration ─────────────────────────────────────────────────────────────


@dataclass
class ProviderRun:
    id: str
    name: str
    model: str
    corpus_result: EmbeddingResult | None = None
    query_result: EmbeddingResult | None = None
    rankings: dict[str, list[tuple[int, float]]] = field(default_factory=dict)


def run_benchmark(
    providers: list[str],
    corpus: list[dict],
    queries: list[dict],
    top_k: int,
) -> dict[str, ProviderRun]:
    runs: dict[str, ProviderRun] = {}
    corpus_texts = [build_embedding_text(a) for a in corpus]
    query_texts = [q["query"] for q in queries]

    for pid in providers:
        meta = PROVIDER_REGISTRY[pid]
        run = ProviderRun(id=pid, name=meta["name"], model=meta["model"])
        runs[pid] = run

        if not os.environ.get(meta["env_var"]):
            print(f"[skip] {meta['name']}: env var {meta['env_var']} not set", file=sys.stderr)
            run.corpus_result = EmbeddingResult(
                vectors=np.zeros((0, 0), dtype=np.float32),
                approx_tokens=0,
                duration_seconds=0.0,
                error=f"{meta['env_var']} not set",
            )
            continue

        print(f"[run]  {meta['name']}: embedding {len(corpus_texts)} corpus items...", file=sys.stderr)
        run.corpus_result = PROVIDER_FNS[pid](corpus_texts, meta["model"])
        if run.corpus_result.error:
            print(f"[err]  {meta['name']}: {run.corpus_result.error}", file=sys.stderr)
            continue

        print(f"[run]  {meta['name']}: embedding {len(query_texts)} queries...", file=sys.stderr)
        run.query_result = PROVIDER_FNS[pid](query_texts, meta["model"])
        if run.query_result.error:
            print(f"[err]  {meta['name']}: {run.query_result.error}", file=sys.stderr)
            continue

        for qi, q in enumerate(queries):
            run.rankings[q["id"]] = cosine_top_k(
                run.query_result.vectors[qi], run.corpus_result.vectors, top_k
            )

    return runs


# ── Reporting ─────────────────────────────────────────────────────────────────


def format_markdown_report(
    runs: dict[str, ProviderRun],
    corpus: list[dict],
    queries: list[dict],
    top_k: int,
) -> str:
    out = []
    out.append("# Embedding Provider Benchmark — Results\n")
    out.append(f"_Corpus: {len(corpus)} albums. Queries: {len(queries)}. Top-K: {top_k}._\n")

    active = [r for r in runs.values() if not (r.corpus_result and r.corpus_result.error)]
    skipped = [r for r in runs.values() if r.corpus_result and r.corpus_result.error]

    if skipped:
        out.append("## Skipped providers\n")
        for r in skipped:
            err = r.corpus_result.error if r.corpus_result else "unknown error"
            out.append(f"- **{r.name}** — {err}\n")
        out.append("")

    if not active:
        out.append("\n## All providers failed\n")
        out.append("Errors are listed above. Common causes:\n")
        out.append("- Missing or invalid API key (check `.env`)")
        out.append("- OpenAI 429 `insufficient_quota` — top up account balance, or run with `--providers huggingface,voyage`")
        out.append("- HF 503 cold-load — re-run; the script retries once but the model may need longer to wake up")
        out.append("- Network reachability — confirm outbound HTTPS works")
        out.append("\nIf you scoped the run with `--providers <one>` and that one failed, drop the flag to let the others try.\n")
        return "\n".join(out)

    # Per-query side-by-side
    out.append("## Per-query rankings\n")
    for q in queries:
        out.append(f"### {q['id']}\n")
        out.append(f"> {q['query']}\n")
        out.append(f"\n**Expected vibe:** {q.get('expected_vibe', '_(none provided)_')}\n")

        header = ["Rank"] + [r.name for r in active]
        out.append("\n| " + " | ".join(header) + " |")
        out.append("|" + "|".join(["---"] * len(header)) + "|")

        for rank in range(1, top_k + 1):
            cells = [str(rank)]
            for r in active:
                if q["id"] not in r.rankings or rank > len(r.rankings[q["id"]]):
                    cells.append("—")
                    continue
                idx, sim = r.rankings[q["id"]][rank - 1]
                album = corpus[idx]
                cells.append(f"{album['artist']} — _{album['title']}_ ({album['year']}) · {sim:.3f}")
            out.append("| " + " | ".join(cells) + " |")
        out.append("")

    # Cost & timing summary
    out.append("## Cost and timing summary\n")
    out.append("| Provider | Model | Corpus tokens | Query tokens | Corpus time | Query time | Est. cost (this run) |")
    out.append("|---|---|---|---|---|---|---|")
    for r in active:
        meta = PROVIDER_REGISTRY[r.id]
        ct = r.corpus_result.approx_tokens if r.corpus_result else 0
        qt = r.query_result.approx_tokens if r.query_result else 0
        cd = r.corpus_result.duration_seconds if r.corpus_result else 0.0
        qd = r.query_result.duration_seconds if r.query_result else 0.0
        cost = (ct + qt) * meta["price_per_1m_tokens_usd"] / 1_000_000
        cost_str = "free tier" if meta["price_per_1m_tokens_usd"] == 0 else f"${cost:.6f}"
        out.append(
            f"| {r.name} | `{r.model}` | {ct} | {qt} | {cd:.2f}s | {qd:.2f}s | {cost_str} |"
        )
    out.append("")
    out.append(
        "_Token counts for HF are approximated as `chars / 4`. OpenAI and Voyage report exact usage._\n"
    )

    out.append("## How to read this\n")
    out.append(
        "Read each query's table left-to-right at rank 1: do all providers pick something defensible? "
        "Then scan to rank 5: which provider's tail still looks coherent? "
        "The `expected_vibe` row above each table is your gut-check ground truth.\n"
    )
    out.append(
        "Cost is essentially noise at this scale. Pick on quality. If two providers tie, pick the cheaper one.\n"
    )
    return "\n".join(out)


def write_rankings_json(runs: dict[str, ProviderRun], corpus: list[dict], path: Path) -> None:
    """Dump raw rankings for follow-up analysis."""
    payload = {}
    for pid, r in runs.items():
        payload[pid] = {
            "name": r.name,
            "model": r.model,
            "rankings": {
                qid: [
                    {"album_id": corpus[idx]["id"], "similarity": sim}
                    for idx, sim in top_list
                ]
                for qid, top_list in r.rankings.items()
            },
            "error": (r.corpus_result.error if r.corpus_result else None),
        }
    path.write_text(json.dumps(payload, indent=2))


# ── Main ──────────────────────────────────────────────────────────────────────


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument(
        "--providers",
        default=",".join(PROVIDER_REGISTRY.keys()),
        help="Comma-separated provider IDs (default: all)",
    )
    ap.add_argument("--top-k", type=int, default=DEFAULT_TOP_K)
    ap.add_argument("--corpus", default=DEFAULT_CORPUS)
    ap.add_argument("--queries", default=DEFAULT_QUERIES)
    ap.add_argument("--out", default=DEFAULT_OUT_DIR)
    ap.add_argument(
        "--show-text",
        action="store_true",
        help="Print built embedding texts and exit. Validates corpus JSON and the text builder. No API calls.",
    )
    ap.add_argument(
        "--query-ids",
        default="",
        help="Comma-separated query IDs to include (default: all from queries.json)",
    )
    ap.add_argument(
        "--limit-corpus",
        type=int,
        default=0,
        help="If >0, only use the first N corpus albums. Useful for fast iteration.",
    )
    args = ap.parse_args()

    providers = [p.strip() for p in args.providers.split(",") if p.strip()]
    for p in providers:
        if p not in PROVIDER_REGISTRY:
            print(f"unknown provider: {p}", file=sys.stderr)
            return 2

    corpus = json.loads(Path(args.corpus).read_text())
    queries = json.loads(Path(args.queries).read_text())

    if not corpus:
        print("corpus is empty", file=sys.stderr)
        return 2
    if not queries:
        print("queries is empty", file=sys.stderr)
        return 2

    # Apply subset filters.
    if args.limit_corpus > 0:
        corpus = corpus[: args.limit_corpus]
        print(f"[subset] corpus limited to first {len(corpus)} albums", file=sys.stderr)

    if args.query_ids.strip():
        wanted = {qid.strip() for qid in args.query_ids.split(",") if qid.strip()}
        before = len(queries)
        queries = [q for q in queries if q["id"] in wanted]
        if not queries:
            print(f"--query-ids matched nothing. Available IDs: {[q['id'] for q in json.loads(Path(args.queries).read_text())]}", file=sys.stderr)
            return 2
        print(f"[subset] {len(queries)} of {before} queries selected: {[q['id'] for q in queries]}", file=sys.stderr)

    # Show-text mode: print everything that would be embedded and exit. No API calls.
    if args.show_text:
        print("# Embedding texts (dry run — no API calls)\n")
        print(f"## Corpus ({len(corpus)} albums)\n")
        for i, album in enumerate(corpus, 1):
            text = build_embedding_text(album)
            char_count = len(text)
            approx_tok = char_count // 4
            print(f"### {i}. {album['id']}")
            print(f"_({char_count} chars, ~{approx_tok} tokens)_\n")
            print(f"> {text}\n")
        print(f"\n## Queries ({len(queries)} queries)\n")
        for i, q in enumerate(queries, 1):
            print(f"### {i}. {q['id']}")
            print(f"> {q['query']}\n")
            if vibe := q.get("expected_vibe"):
                print(f"**Expected:** {vibe}\n")
        # Quick sanity stats
        thin = [a for a in corpus if len(build_embedding_text(a)) < 120]
        if thin:
            print(f"\n## Warnings\n")
            print(f"- {len(thin)} corpus item(s) have <120 chars of embedding text and would be skipped by the production builder: {[a['id'] for a in thin]}\n")
        return 0

    out_dir = Path(args.out)
    out_dir.mkdir(parents=True, exist_ok=True)

    runs = run_benchmark(providers, corpus, queries, args.top_k)

    report = format_markdown_report(runs, corpus, queries, args.top_k)
    (out_dir / "results.md").write_text(report)
    write_rankings_json(runs, corpus, out_dir / "rankings.json")

    print(report)
    print(f"\n[wrote] {out_dir / 'results.md'}", file=sys.stderr)
    print(f"[wrote] {out_dir / 'rankings.json'}", file=sys.stderr)

    # Non-zero exit if every provider failed.
    if all(r.corpus_result and r.corpus_result.error for r in runs.values()):
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
