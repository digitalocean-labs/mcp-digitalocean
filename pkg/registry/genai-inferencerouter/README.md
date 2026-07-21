# GenAI Inference Router MCP Tools

## What is a model router?

A **model router** (sometimes called an inference router) is a **named GenAI configuration** in your DigitalOcean account. It does not run inference by itself; it defines **how your app or agent should choose models** for different kinds of work. Concretely, a router has:

- **Policies** — Each policy ties a **task** to an **ordered list of model ids** and a **selection policy** (for example, prefer the fastest or cheapest model among the candidates for that task). A task is either a **built-in** `task_slug` (e.g. code generation, summarization) or a **custom task** you describe with a name and short `description`.
- **Fallback models** — A **required** ordered list the API can use when primary model choices in a policy are not available, so traffic still has a path to complete.

### Use cases and how routing fits together (for agents)

From DigitalOcean’s product documentation, an **Inference Router** is meant for **production routing** over [serverless inference](https://docs.digitalocean.com/products/inference/how-to/use-serverless-inference/) and [dedicated inference](https://docs.digitalocean.com/products/inference/how-to/use-dedicated-inference/). Instead of hard-coding one model per call, you define **tasks** (preset routes tuned by DigitalOcean or **custom** routes you name and describe), attach **model pools** and a **selection policy** (cost vs latency tradeoffs; the control panel also describes “optimal” presets and manual ordering for custom tasks), and **fallback models** so unmatched or degraded traffic still completes. The router evaluates each request against those tasks and policies. This MCP’s create/update/list/get/delete and preset-list tools map to the **account-level router configuration** (`/v2/gen-ai/models/routers` and related endpoints) that backs that behavior. After a router exists, client applications typically invoke it as a **drop-in model target** (for example `model` set to `router:<your-router-name>` in Chat Completions or Responses against the inference runtime—see the official how-to for exact URLs, headers such as model affinity, and response details like which route was selected).

**Further reading**

- [How to Use Inference Router](https://docs.digitalocean.com/products/inference/how-to/use-inference-router/) — end-to-end: concepts, control panel vs API (`POST /v2/gen-ai/models/routers`), preset vs custom tasks, fallbacks, calling the router from inference APIs, playground and metrics.
- [DigitalOcean Inference Engine](https://www.digitalocean.com/products/inference-engine) — where Inference Router sits alongside serverless, batch, and dedicated inference, evaluations, and the broader “single control plane” story.

When you use these MCP tools, you are managing that **routing configuration** through the typed **`godo.GradientAI`** client (same auth, base URL, and transport as the rest of this MCP server). Responses are formatted JSON aligned with the API (`tasks` for presets, `model_routers` / `model_router`, and `config` on get/create/update).

## godo surface

Calls map to:

- `GradientAI.CreateInferenceRouter` — create (`POST /v2/gen-ai/models/routers`)
- `GradientAI.ListInferenceRouters` — list (`GET …` with `page`, `per_page`)
- `GradientAI.GetInferenceRouter` — get by UUID
- `GradientAI.UpdateInferenceRouter` — update (`PUT …/{uuid}`)
- `GradientAI.DeleteInferenceRouter` — delete by UUID
- `GradientAI.ListInferenceRouterTaskPresets` — list preset tasks (`GET /v2/gen-ai/models/routers/tasks/presets` with `page`, `per_page`)

## Built-in `task_slug` values (how to choose)

**Prefer the live catalog** — Call **`genai-inference-router-task-presets`** (wraps `ListInferenceRouterTaskPresets`) for current `task_slug` values, display names, suggested models, and pagination metadata. The API remains authoritative if a slug is unavailable in your account or region.

**Practical ways to pick a slug:**

1. **`genai-inference-router-task-presets`** — Returns each preset’s `task_slug` and related fields.
2. **Copy from an existing router** — Call `genai-inference-router-list` or `genai-inference-router-get` and read `task_slug` under `model_router.config.policies`.
3. **Avoid slugs for one-off work** — Use a **`custom_task`** policy with `name` and `description` instead of `task_slug` (the [e2e test](../../../testing/e2e_genai_inferencerouter_test.go) does this so tests do not depend on a specific catalog in every environment).

**Static reference** (snapshot for quick scanning; may drift—use **`genai-inference-router-task-presets`** or the docs above for current values):

- **General:** `brainstorming-ideation`, `classification-labeling`, `opinion-advice-recommendation`, `planning-task-decomposition`, `summarization`, `text-extraction-structured-output`, `translation`
- **Writing:** `creative-writing`, `email-professional-communication-drafting`, `long-form-article-blog-writing`, `rewriting-editing`, `social-media-short-form-content`
- **Software engineering:** `bug-fixing`, `code-completion-inline`, `code-generation`, `code-performance-optimization`, `test-writing-code-verification`
- **Knowledge base & document intelligence:** `knowledge-base-customer-support`, `long-context-retrieval-aggregation`, `long-document-qa`, `rag-system-quality-evaluation`, `retrieval-quality-cross-domain-ir`, `text-and-table-grounded-reasoning`

## Tools

### `genai-inference-router-create`

**Arguments**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `Name` | string | yes | Router name |
| `PoliciesJson` | string | no | JSON array of policies (omit or `[]` if the API allows); see below |
| `FallbackModels` | string[] | **yes** | At least one model id, sent as `fallback_models` (required by the API) |

**Create request body (via godo):** `name`, optional **`policies`**, and required non-empty **`fallback_models`**. Each policy must define a **task**:

- **Built-in task:** set **`task_slug`** (e.g. `code-generation`, `summarization`, `bug-fixing`) and **`models`** (ordered model id strings). Include **`selection_policy`** with **`prefer`** set to `fastest` or `cheapest`.
- **Custom task:** use **`custom_task`** with **`name`** and **`description`** instead of `task_slug`, and still set **`models`** and **`selection_policy`** as needed.

Policies that only set `model` plus `usecase_class` (with no task) are rejected by the API with an error like `policy 0 task is required`.

**Example** (equivalent JSON body sent by godo; `PoliciesJson` is only the `policies` array):

```json
{
  "name": "my-router",
  "policies": [
    {
      "task_slug": "code-generation",
      "models": ["openai-gpt-5", "anthropic-claude-4.6-sonnet"],
      "selection_policy": { "prefer": "fastest" }
    }
  ],
  "fallback_models": ["openai-gpt-oss-120b"]
}
```

**Custom task** policy example:

```json
{
  "custom_task": {
    "name": "Code reviewer",
    "description": "Review patches for correctness and style."
  },
  "models": ["openai-gpt-5.2"],
  "selection_policy": { "prefer": "cheapest" }
}
```

Pass the contents of `policies` (a JSON array) as the `PoliciesJson` string. **List** and **get** return `model_router.config.policies` in the same general shape (`task_slug` or `custom_task`, `models`, `selection_policy`). List summaries and full router payloads may include **`regions`** when the API returns them; that field is not set by this MCP on create (see current `godo.InferenceRouterCreateRequest`).

### `genai-inference-router-list`

**Arguments**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `Page` | number | no | Page (default 1) |
| `PerPage` | number | no | Page size (default 1000, max 1000) |

### `genai-inference-router-get`

**Arguments**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `UUID` | string | yes | Model router UUID |

### `genai-inference-router-delete`

**Arguments**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `UUID` | string | yes | Model router UUID to delete |

Returns formatted JSON (typically `{"uuid":"..."}`). If the API responds with an empty body on success, the tool still returns a JSON object containing the requested UUID.

### `genai-inference-router-task-presets`

**Arguments**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `Page` | number | no | Page (default 1) |
| `PerPage` | number | no | Page size (default 1000, max 1000) |

Returns JSON with a `tasks` array (preset `task_slug`, names, categories, suggested models, etc.) plus optional `meta` and `links`. Use this when choosing built-in slugs for `PoliciesJson` on create/update.

### `genai-inference-router-update`

**Arguments**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `UUID` | string | yes | Model router UUID |
| `Name` | string | no | New name (omit if unchanged) |
| `Description` | string | no | New description (omit if unchanged) |
| `PoliciesJson` | string | no | JSON array of policies (omit if unchanged); must decode as a JSON array when non-empty |
| `FallbackModels` | string[] | no | New ordered fallbacks; pass this argument only when updating fallbacks |

At least one of `Name`, `Description`, non-empty `PoliciesJson`, or `FallbackModels` must be provided (matches `godo` validation).

## Enabling the service

Register with `--services genai-inferencerouter` (or include it in `SERVICES`). A valid `DIGITALOCEAN_API_TOKEN` is required.

## Notes

- The GenAI model router API **requires** at least one `fallback_models` entry on create, so the MCP enforces a non-empty `FallbackModels` list for create (mirroring `godo` client validation).
- Preview / unreleased APIs may only work on specific API hosts or accounts.
- Response bodies are returned as formatted JSON text.
- **Integration tests / tokens:** If `DELETE …/v2/gen-ai/models/routers/{uuid}` returns **403** (“not authorized”), your Personal Access Token can list or create routers but **cannot delete** them—use a token with **write** (not read-only) access to the account’s Inference / GenAI resources, or delete the router in the [control panel](https://cloud.digitalocean.com/) under **INFERENCE → Inference Router**.
