## Documentation Tools

Read-only tools for querying DigitalOcean's public documentation. No authentication required.

---

## Supported Tools

- **docs-search**
  Keyword search over the titles, descriptions, and URLs of every public docs page (~4,600 pages, including individual support articles and API/CLI/SDK reference pages). Returns ranked results with canonical URL, product, page type, last-updated date, and a one-line excerpt. It does not search page bodies.
  **Arguments:**
    - `Query` (string, required): Search keywords
    - `Limit` (number, default: 10): Maximum number of results to return (1 to 50)

- **docs-get-page**
  Fetch the full markdown content of a specific docs page.
  **Arguments:**
    - `URL` (string, required): Full URL or path of the docs page (e.g., `https://docs.digitalocean.com/products/droplets/getting-started/quickstart/` or `/products/droplets/getting-started/quickstart/`)

- **docs-find-for-service**
  List documentation pages for a given DigitalOcean service.
  **Arguments:**
    - `Service` (string, required): DigitalOcean service name (e.g., `"droplets"`, `"kubernetes"`, `"app platform"`, `"databases"`)

- **docs-get-quickstart**
  Get the quickstart or getting-started guide for a service.
  **Arguments:**
    - `Service` (string, required): DigitalOcean service name (e.g., `"droplets"`, `"kubernetes"`, `"app platform"`)

- **docs-troubleshoot**
  Find support and troubleshooting articles matching an error message or symptom by keyword, ranking support articles and troubleshooting-style titles first. Returns the full content of the best matching page, plus links to related pages.
  **Arguments:**
    - `Symptom` (string, required): Error message, symptom description, or keywords (e.g., `"520 status code"`, `"deployment failed health check"`)
    - `Limit` (number, default: 5): Maximum number of results to return (1 to 50)

- **docs-get-related**
  Extract and categorize all outbound documentation links from a specific docs page. Returns links grouped by type (how-to, reference, support, getting-started, concept, details).
  **Arguments:**
    - `URL` (string, required): Full URL or path of the docs page to extract links from

---

## How It Works

- `docs-search` and `docs-troubleshoot` rank pages from `docs.digitalocean.com/llms-index.json`, the published per-page metadata index (title, description, product, section, canonical URL, last-modified). It covers every public page, including support articles that the curated `llms.txt` overview omits. If `llms-index.json` is unavailable the client falls back to `llms.txt`, and if a refresh fails it keeps serving the last good index.
- Ranking is keyword-based: query terms are lowercased, stripped of punctuation and stop words, and matched (with light stemming) against title, description, product, and URL path. Title matches outrank description and URL matches, pages matching every term are boosted, and terms that match very few pages (such as an error code) weigh double so they are not drowned out by common product words. Page bodies are not searched; see `corpus-search-followup.md` for the plan to add content-backed retrieval from `docs-corpus.jsonl`.
- `docs-find-for-service` uses per-service `llms.txt` files
- Fetches raw markdown via the `index.html.md` endpoint; only `docs.digitalocean.com` URLs are fetched
- In-memory caching (30 min for pages, 1 hour for indexes); concurrent cold loads of the index share one fetch
- Supports common service name aliases (e.g., "k8s" → "kubernetes", "gpu" → "bare-metal-gpus")

### Ranking regression set

`testdata/regression_queries.json` is a small judged query set (support, how-to, quickstart, and API/CLI/SDK reference queries with expected canonical URLs). `TestRegressionSet_Fixture` runs it offline against `testdata/llms-index.json`; `TestLiveRegressionSet` (build tag `integration`) runs it against the live index and logs Recall@5, MRR, and cold/warm latency. Add a row there before changing ranking.

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
