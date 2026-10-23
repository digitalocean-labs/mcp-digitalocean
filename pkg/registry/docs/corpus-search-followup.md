# Follow-up: content-backed docs retrieval from `docs-corpus.jsonl`

Status: option A (lazy shard fetch for excerpts) is shipped; option B
(content-backed ranking) remains a proposal. This documents what the
metadata ranker over `llms-index.json` deliberately does not do, what the
excerpt layer adds, and how to add content-backed retrieval without
regressing the credential-free, closed-world design of the docs tools.

## Where the first slice stops

`docs-search` and `docs-troubleshoot` rank the complete page universe
(~4,600 pages) by title, description, product, and URL. They do not search
page bodies. On the judged regression set (`testdata/regression_queries.json`)
that moved Recall@5 from 1/24 to 24/24, so the candidate universe, not body
text, was the bottleneck.

Body text still matters for two things:

1. Queries whose terms appear only in the body (an exact error string, a flag
   name, a config key). Not addressed; stage 1 still gates candidates.
2. Query-focused excerpts for citations. Addressed by option A below: the
   excerpt is now a query-focused body window from a bounded set of hits, falling
   back to the description (median 74 characters; 141 pages have none).

## The content feed

`https://docs.digitalocean.com/docs-corpus.jsonl` (schema 0.2.1) is a
manifest: one `corpus` header line, one inline `doc` record for the homepage,
and 538 `shard` records of the form
`{"kind":"shard","url":".../<section>/docs-corpus.jsonl","doc_count":N}`.
Each shard is JSONL of `doc` chunks with `canonical_url`, `markdown_url`,
`title`, `description`, `product`, `content`, `checksum`, `last_modified`,
`chunk_id`, and `doc_id`.

Measured on 2026-10-19:

| Property | Value |
| --- | --- |
| Chunks | 4,896 (4,897 including the inline homepage record) |
| Shards | 538 |
| Total size | 32.1 MB |
| Largest shard | `reference/api/reference/docs-corpus.jsonl`, 14.0 MB (exceeds the 10 MB per-response cap) |
| Next largest | `products/marketplace/`, 1.9 MB; `support/`, 0.7 MB |
| Full fetch, 8 workers | 16.4 s wall, p50 0.16 s / p95 0.42 s per shard |
| Shard -> page mapping | every chunk's `canonical_url` starts with its shard URL's directory (4,896/4,896) |
| Coverage vs `llms-index.json` | 356 corpus pages not in the index (Kubernetes changelog versions); 60 index pages not in the corpus (API reference redirect stubs, homepage) |
| Cache headers | `cache-control: public, max-age=300`, strong `etag`, `last-modified` |

A naive BM25 over full chunk content scored worse than metadata-only ranking
on the regression set (17/24 vs 24/24): long reference pages that mention a
term many times outrank the short how-to page the query is about. Content
retrieval therefore needs field weighting (title >> body) and evaluation
before it ships, not just a bigger index.

## Why this needs more than a per-process in-memory index

The MCP server runs in two shapes: a long-lived HTTP deployment and a
short-lived stdio process started per client session (the npm package). A
full corpus index per process means 538 requests and 32 MB on every cold
start, which is unacceptable for stdio and wasteful for HTTP. Embedding a
snapshot in the binary was already rejected in #360 because it goes stale.

## Options

### A. Lazy shard fetch for the returned results (shipped, in-repo)

Metadata retrieval stays stage 1. For the results a query returns (at most
`Limit`), `ContentExcerpts` maps each canonical URL to its shard by walking
up the path over the manifest, fetches only those shards (deduplicated, at
most 8 per query in result order, at most 4 concurrent corpus loads across
one client, 5 s enrichment deadline, 30 min shard cache). Concurrent queries
share each manifest/shard load. A cancelled caller stops waiting without
cancelling another caller's load; shared loads have their own 5 s deadline.
Origin failures are negative-cached for 10 min. A single deadline expiry is
not, so one stall is retried by the next query; three in a row for the same
key are, so a persistently slow origin does not add the full budget to every
query.

The client rejects a declared oversized response before downloading its body
and also checks the streamed/decompressed size using a one-byte overflow
probe. Bodies over 10 MB are rejected, never parsed or cached as a successful
prefix. All requests and redirects stay on the HTTPS docs origin.

Excerpts preserve source case and UTF-8 boundaries and contain at least one
complete query-term match (same token rule as ranking: exact, stem, or
4+ character prefix). Selection considers the first 8 hits per term in the
first 64 KB of the page body, scoring the actual whitespace-adjusted windows
of up to 300 bytes plus ellipses. It does not promise the globally best
passage in the full page. Anything unresolved falls back to the description.

Measured on 2026-10-20 against the live site over the 28-query regression
set (top-5 per query), with a fresh corpus cache for every cold measurement
so "cold" includes the manifest fetch:

| Metric | Value |
| --- | --- |
| Results with a body excerpt | 136 / 140 (the misses are API reference pages; that shard is 14 MB and is refused by the size cap) |
| Top-1 results with a body excerpt | 28 / 28 |
| Added latency, cold | avg 267 ms, max 535 ms |
| Added latency, warm | ~3 ms |

Body text is deliberately not used to re-rank. The metadata ranker is at
28/28 and MRR 0.875 on this set. The earlier BM25-over-content
measurement (17/24) showed that body frequency favours long reference pages;
a re-ranker needs body-only queries in the regression set first (see
Evaluation) so there is something to measure it against.

The shard-prefix mapping is empirically exact (4,896/4,896) but is not a
documented contract of the schema; the client tolerates misses by falling
back to the description. A `prefix` field on manifest records would make it
a contract.

Recall for body-only terms does not improve: stage 1 still gates candidates.

### B. Corpus-backed search service (outside this repository)

A small service that polls the manifest, fetches changed shards (by `etag` or
per-chunk `checksum`), and maintains a lexical index (BM25F over title,
description, URL, headings, body) with optional embeddings. The MCP server
calls it over HTTPS with no credentials. The deployed `docs` MCP already
runs behind App Platform; the service can live alongside it.

This is the only option that improves recall for body-only terms, and it is
the only one where the 14 MB API reference shard can be indexed at all.

### Recommendation

A is shipped inside the existing tools. B is the next step if body-only
recall matters; it should also improve `docs-search` and `docs-troubleshoot`
in place rather than add a tool. The excerpt layer from A stays useful under
B as the fallback when the service is unavailable.

## Design for option B

### Indexing

- Poll `docs-corpus.jsonl` every 5 minutes (matches `max-age=300`). Compare
  the manifest `etag`; on change, diff shard list and `doc_count`.
- Fetch changed shards with `If-None-Match`. Replace a shard's chunks
  atomically in the index keyed by `chunk_id`; drop chunks whose `doc_id`
  disappeared. Skip a chunk whose `checksum` is unchanged.
- Index fields: title (weight 3), description (1), URL path tokens (2),
  product + section (1), body (1, truncated to 20 KB per chunk). Tokenize
  as the MCP client does today (lowercase, split on non-alphanumerics, light
  stem) so the two rankers agree on vocabulary.
- Keep `llms-index.json` as the authority for "is this a page", so the 356
  changelog-version chunks that are not pages do not surface as results.

### Query path

- `GET /v1/search?q=...&limit=...&scope=support|all` returns
  `{url, markdown_url, title, product, section, last_modified, score,
  excerpt}`. Excerpt is the highest-density query-term window in the body.
- The MCP server keeps the current metadata ranker as a fallback: if the
  service is unreachable or returns an error within 2 s, it answers from the
  in-memory `llms-index.json` and notes the degraded mode in the result
  footer. Zero-result behaviour stays as today (friendly message, no error).

### Cache invalidation

- Service: index is rebuilt per shard on manifest change; no TTL-based
  staleness beyond the 5 minute poll.
- MCP server: response cache keyed by normalized query, TTL 5 minutes, plus
  the existing 30 minute page cache for `docs-get-page`. A manifest `etag`
  exposed by the service in a response header lets the client drop its query
  cache when the corpus changes.

### Latency budget

- Service p50 < 50 ms, p95 < 150 ms for a 5k-chunk lexical index in memory.
- End-to-end from the MCP server p95 < 400 ms including the network hop;
  cold start of the MCP process unchanged (no corpus fetch on the client).

### Rollback

- The service is additive. Disabling it (env var `DOCS_SEARCH_SERVICE_URL`
  unset or feature flag off) returns the tools to the metadata ranker with
  identical output shape, so rollback is a config change, not a release.
- Keep the regression set green under both modes in CI.

### Telemetry

- Service: query count, zero-result rate, p50/p95 latency, index age
  (seconds since last successful manifest poll), shards failed, chunks
  indexed.
- MCP server: per-tool counts split by `mode=service|metadata-fallback`,
  fallback rate, and the rank position of the result the agent subsequently
  fetched with `docs-get-page` (click-through proxy) when the same session
  can be correlated.

### Evaluation

- Extend `testdata/regression_queries.json` with body-only queries (exact
  error strings, flag names) that the metadata ranker is known to miss, so
  the service has something to win on.
- Gate deploys on Recall@5 >= 0.95 and MRR not lower than the metadata
  ranker on the full set, measured against the live index in CI
  (`TestLiveRegressionSet` gates Recall@5 >= 0.9, MRR >= 0.75 and ranking
  latency < 50 ms/query when run with the `integration` tag; standard CI
  does not run it). Standard CI gates on top-5 recall for every judgment
  and MRR >= 0.80 against the 218-page fixture. That sample cannot
  establish live ranking quality; separate negative-match tests and a
  synthetic 4,600-page rarity test cover specific failure modes.
- Report excerpt quality separately: fraction of top-1 results whose excerpt
  contains at least one query term.
