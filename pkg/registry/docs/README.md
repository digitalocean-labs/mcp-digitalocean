## Documentation Tools

Read-only tools for querying DigitalOcean's public documentation. No authentication required.

---

## Supported Tools

- **docs-search**
  Full-text search across all DigitalOcean documentation.
  **Arguments:**
    - `Query` (string, required): Search query string
    - `Limit` (number, default: 10): Maximum number of results to return

- **docs-get-page**
  Fetch the full markdown content of a specific docs page.
  **Arguments:**
    - `URL` (string, required): Full URL or path of the docs page (e.g., `https://docs.digitalocean.com/products/droplets/getting-started/quickstart/` or `/products/droplets/getting-started/quickstart/`)
    - `IncludeStructured` (boolean, default: false): If true, returns JSON with `markdown` and an `actions` array (doctl and `curl` examples extracted from the page).

- **docs-search-semantic**
  Two-stage search: reuse the llms.txt keyword ranker for a first pass, then BM25 over fetched markdown excerpts from `llms-index.json`. Better when the query does not match titles in `llms.txt`.
  **Arguments:**
    - `Query` (string, required)
    - `Limit` (number, default: 10)

- **docs-get-api-spec**
  Returns one operation object from the embedded bundled public DigitalOcean v2 OpenAPI JSON (gzip in this package).
  **Arguments:**
    - `Method` (string, required): `GET`, `POST`, `PUT`, `PATCH`, or `DELETE`
    - `Path` (string, required): Path template exactly as in the spec (example: `/v2/apps/{id}`)

- **docs-list-regions** (docs-only MCP profile only)
  Returns a static JSON array of common region slugs. No API token. When other services are enabled alongside docs, use the shared `region-list` tool instead.

- **docs-find-for-service**
  List documentation pages for a given DigitalOcean service.
  **Arguments:**
    - `Service` (string, required): DigitalOcean service name (e.g., `"droplets"`, `"kubernetes"`, `"app platform"`, `"databases"`)

- **docs-get-quickstart**
  Get the quickstart or getting-started guide for a service.
  **Arguments:**
    - `Service` (string, required): DigitalOcean service name (e.g., `"droplets"`, `"kubernetes"`, `"app platform"`)

- **docs-troubleshoot**
  Search for troubleshooting pages matching an error message or symptom. Returns the full content of the best matching support page, plus links to related pages.
  **Arguments:**
    - `Symptom` (string, required): Error message, symptom description, or keywords (e.g., `"520 status code"`, `"deployment failed health check"`)
    - `Limit` (number, default: 5): Maximum number of results to return

- **docs-get-related**
  Extract and categorize all outbound documentation links from a specific docs page. Returns links grouped by type (how-to, reference, support, getting-started, concept, details).
  **Arguments:**
    - `URL` (string, required): Full URL or path of the docs page to extract links from

---

## How It Works

- Indexes `docs.digitalocean.com/llms.txt`, `llms-index.json`, and per-service `llms.txt` files
- Fetches raw markdown via the `index.html.md` endpoint (and `markdown_url` from `llms-index.json` for semantic search)
- In-memory caching (30 min for pages, 1 hour for indexes)
- Supports common service name aliases (e.g., "k8s" → "kubernetes", "gpu" → "bare-metal-gpus")

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
- Most responses are markdown text; `docs-get-page` with `IncludeStructured: true` and `docs-list-regions` return JSON text.
- Service name aliases are supported (e.g., "k8s" for "kubernetes", "postgres" for "postgresql").
