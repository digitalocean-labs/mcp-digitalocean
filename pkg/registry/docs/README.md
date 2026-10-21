## Documentation Tools

Read-only tools for querying DigitalOcean's public documentation. No authentication required.

---

## Supported Tools

- **docs-search**
  Keyword search over the titles, descriptions, and URLs of every public docs page (~4,600 pages, including individual support articles and API/CLI/SDK reference pages). Returns ranked results with canonical URL, product, page type, last-updated date, and an excerpt: the passage of the page body that mentions the query terms when available, otherwise the page description. Ranking does not search page bodies.
  **Arguments:**
    - `Query` (string, required): Search keywords
    - `Limit` (number, default: 10): Maximum number of results to return
    - `Product` (string, optional): Restrict results to one product or area (e.g., `"droplets"`, `"app platform"`, `"vpc"`, `"support"`, `"api"`)

- **docs-get-page**
  Fetch the markdown content of a specific docs page, whole or in windows.
  **Arguments:**
    - `URL` (string, required): Full URL or path of the docs page (e.g., `https://docs.digitalocean.com/products/droplets/getting-started/quickstart/` or `/products/droplets/getting-started/quickstart/`)
    - `MaxLength` (number, optional): Maximum UTF-8 bytes to return (integer from 0 to 10,485,760); omit or use 0 for the whole page. Windows end at character boundaries. A budget too small for the next character returns an argument error. The truncation note gives the `StartIndex` to continue from.
    - `StartIndex` (number, optional): UTF-8 byte offset to start from (integer from 0 to 10,485,760, default 0). An offset inside a character advances to the next character

- **docs-find-for-service**
  List documentation pages for a given DigitalOcean service.
  **Arguments:**
    - `Service` (string, required): DigitalOcean service name (e.g., `"droplets"`, `"kubernetes"`, `"app platform"`, `"databases"`)

- **docs-get-quickstart**
  Get the quickstart or getting-started guide for a service.
  **Arguments:**
    - `Service` (string, required): DigitalOcean service name (e.g., `"droplets"`, `"kubernetes"`, `"app platform"`)

- **docs-troubleshoot**
  Find support and troubleshooting articles matching an error message or symptom by keyword, ranking support articles and troubleshooting-style titles first. Returns the content of the best matching page (up to 16 KB, with a pointer to the rest), plus related pages with excerpts.
  **Arguments:**
    - `Symptom` (string, required): Error message, symptom description, or keywords (e.g., `"520 status code"`, `"deployment failed health check"`)
    - `Limit` (number, default: 5): Maximum number of results to return

- **docs-get-related**
  Extract and categorize all outbound documentation links from a specific docs page. Returns links grouped by type (how-to, reference, support, getting-started, concept, details).
  **Arguments:**
    - `URL` (string, required): Full URL or path of the docs page to extract links from

---

## How It Works

- `docs-search` and `docs-troubleshoot` rank pages from `docs.digitalocean.com/llms-index.json`, the published per-page metadata index (title, description, product, section, canonical URL, last-modified). It covers every public page, including support articles that the curated `llms.txt` overview omits. If a JSON refresh fails, the client keeps a previously loaded JSON index and retries after 10 minutes. With no previous JSON index it falls back to `llms.txt`; a failed refresh of that fallback also keeps the last good index.
- Ranking is keyword-based: query terms are lowercased, stripped of punctuation and stop words, and matched against the tokens of the title, description, product, and URL path: a term matches a token exactly, by light stem (`configuration`/`configure`), or as a prefix of at least four characters (`postgres`/`postgresql`), never as an arbitrary substring (`spaces` does not match `namespace`). Fields are tokenized once when the index loads, so a query costs about 10 ms over 4,600 pages. Title matches receive more weight than description and URL matches, pages matching every term receive an additive boost, and terms that match very few pages (such as an error code) weigh double so they are not drowned out by common product words. Page bodies are not used for ranking; see `corpus-search-followup.md` for what that would take.
- Excerpts come from `docs-corpus.jsonl`, the published page-body feed. After ranking, the client schedules only the shards covering the returned pages, in result order (at most 8 per query, at most 4 corpus loads at a time across the client, 5 s enrichment deadline, 30 min shard cache). Concurrent queries share manifest and shard loads. A caller can stop waiting without cancelling another query's load; shared loads have their own 5 s deadline, and cancellations are not negative-cached. Each excerpt preserves source case and UTF-8 boundaries, considers the first 8 occurrences of each query term within the first 64 KB of the body, and chooses a window of up to 300 bytes plus ellipses. Whitespace snapping preserves a complete query hit. Missing, oversized, slow, or unmatched content falls back to the description; the corpus is never loaded in full.
- `docs-find-for-service` uses per-service `llms.txt` files
- Uses `markdown_url` from the cached page index when available, including the last good index during refresh failures. Falls back to the `index.html.md` endpoint and then the original page URL without loading the index for a page read; only `docs.digitalocean.com` URLs are fetched
All requests and redirects must stay on the HTTPS `docs.digitalocean.com` origin. Each HTTP request has a 15 s timeout; oversized bodies are rejected even without a reliable `Content-Length` (10 MB accepted-body cap). Page windows do not bypass this response cap.
- In-memory caching (30 min for pages, 1 hour for indexes); concurrent cold loads of the index share one fetch
- Supports common service name aliases (e.g., "k8s" → "kubernetes", "gpu" → "bare-metal-gpus")

### Ranking regression set

`testdata/regression_queries.json` is a judged query set (28 queries: support, how-to, quickstart, API/CLI/SDK reference, and prefix/abbreviation queries such as `mongo backups`, with expected canonical URLs). `TestRegressionSet_Fixture` runs it against the 218-page sample in `testdata/llms-index.json` (each target plus its strongest competitors) and requires every expected page in the top 5; MRR is gated loosely at 0.80 so tie-break changes do not fail the build (measured 0.88). Additional tests reject unrelated substring matches and exercise rarity weighting at 4,600-page scale. The sample does not reproduce the full live universe.

`TestLiveRegressionSet` (build tag `integration`, not run in standard CI) runs the set against the live index and requires Recall@5 >= 0.9, MRR >= 0.75, and under 50 ms per query of ranking time; measured 28/28, 0.875, 11 ms. `TestLiveContentExcerpts` reports excerpt coverage and shard-fetch latency with a fresh corpus cache per query; measured 136/140 results and 28/28 top-1 results with a body excerpt, cold avg 267 ms / max 535 ms, warm 3 ms. Add a judged row before changing ranking.

---

## Example Usage

- **Search documentation:**
  Tool: `docs-search`
  Arguments:
    - `Query`: `"kubernetes networking"`
    - `Limit`: `5`

- **Fetch a specific page:**
  Tool: `docs-get-page`
  Arguments:
    - `URL`: `"/products/droplets/getting-started/quickstart/"`

- **Browse a service:**
  Tool: `docs-find-for-service`
  Arguments:
    - `Service`: `"app platform"`

- **Get a quickstart guide:**
  Tool: `docs-get-quickstart`
  Arguments:
    - `Service`: `"databases"`

- **Diagnose an error:**
  Tool: `docs-troubleshoot`
  Arguments:
    - `Symptom`: `"520 status code from my app"`
    - `Limit`: `3`

- **Find related pages:**
  Tool: `docs-get-related`
  Arguments:
    - `URL`: `"/products/app-platform/how-to/manage-domains/"`

---

## Notes

- All tools are read-only and do not require a DigitalOcean API token.
- All tools use argument-based input; do not use resource URIs.
- All responses are returned as markdown text.
- Service name aliases are supported (e.g., "k8s" for "kubernetes", "postgres" for "postgresql").
