# GenAI Custom Models Tools

This package provides MCP tools for managing custom (bring-your-own) models on DigitalOcean's GenAI platform.

## Overview

The custom models tools enable users to:
- List custom models with status filtering and pagination
- Import models from HuggingFace or other sources
- Update model metadata (name, description, tags)
- Delete custom models

## Tools

### `genai-custom-models-list`
List custom models with optional status filter and pagination.

**Arguments:**
- `status` (string, optional): Filter by status (`STATUS_IMPORTING`, `STATUS_READY`, `STATUS_FAILED`, `STATUS_DELETED`)
- `page` (number, optional): Page number (default: 1)
- `per_page` (number, optional): Results per page (default: 20)

**Returns:** JSON object with array of models and count

```json
{
  "models": [
    {
      "uuid": "...",
      "name": "my-mistral-7b",
      "status": "STATUS_READY",
      "architecture": "MistralForCausalLM",
      "source_type": "SOURCE_TYPE_HUGGINGFACE"
    }
  ],
  "count": 1,
  "max_threshold": 10
}
```

### `genai-custom-models-import`
Import a custom model from an external source (e.g. HuggingFace). Starts an async import job.

**Consent (required every import):** Present import terms and obtain explicit consent (yes) before calling. Set `accept_terms_and_conditions` to `true` only after the user agrees. Consent is validated before Hugging Face `commit_sha` resolution or the DigitalOcean API. This applies to every import, including re-imports of the same model. The tool rejects omitted or `false` values.

**Arguments:**
- `name` (string, optional): Display name (leading/trailing spaces are trimmed when provided)
- `source_type` (string, required): `SOURCE_TYPE_HUGGINGFACE`, `SOURCE_TYPE_SPACES_BUCKET`, `SOURCE_TYPE_SDK_UPLOAD`, `SOURCE_TYPE_FINE_TUNING`
- `source_ref` (object, required): Source reference with `repo_id`, `commit_sha` (optional for HuggingFace; if omitted, fetched from Hugging Face Hub before the import API is called), `access_type`, `hf_token` (for private/gated)
- `accept_terms_and_conditions` (boolean, required): Must be `true` after explicit user consent in chat
- `description` (string, optional): Model description
- `preferred_gpu_region` (string, optional): Preferred GPU region (e.g. `nyc3`)
- `tags` (object, optional): Tags object with a `tags` array of strings

**Returns:** JSON object with model, import job status, and validation steps

```json
{
  "model": { "uuid": "...", "name": "my-model", "status": "STATUS_IMPORTING" },
  "import_job": { "uuid": "...", "status": "...", "files_total": 12, "files_done": 0 },
  "validation_steps": [{ "name": "config_json", "passed": true }],
  "error": ""
}
```

### `genai-custom-models-update-metadata`
Update the name, description, or tags of an existing custom model.

**Arguments:**
- `uuid` (string, required): UUID of the custom model
- `name` (string, optional): New name
- `description` (string, optional): New description
- `tags` (object, optional): New tags object with a `tags` array

At least one of `name`, `description`, or `tags` must be provided.

**Returns:** JSON object with the updated model

### `genai-custom-models-delete`
Delete a custom model by exact UUID or exact name.

**Exact identifier:** Provide `uuid` OR `name` (at least one). Only a full UUID or an exact model name deletes. Partial uuid or partial name returns candidates only — never deletes, even with a single match. Do not substitute values from a partial-match list.

**Consent (required for every delete):** Before calling with `confirm_deletion: true`, tell the user which model will be deleted (name and uuid), that deletion is permanent, and obtain explicit consent (yes).

**Arguments:**
- `name` (string, optional): Exact custom model name (case-sensitive after trimming). Partial names return name matches only.
- `uuid` (string, optional): Exact full model UUID. Partial uuids return uuid matches only.
- `confirm_deletion` (boolean): Must be `true` after explicit user consent when performing a delete

**Returns:** JSON object with status on success

```json
{
  "status": "DELETE_CUSTOM_MODEL_STATUS_SUCCESS",
  "error": ""
}
```

**Returns (name not exact):** JSON object listing candidate models

```json
{
  "message": "no custom model with exact name \"llama\"; ask the user to provide the exact name ...",
  "query": "llama",
  "matches": [
    { "uuid": "...", "name": "my-llama-model", "status": "STATUS_READY" }
  ],
  "query_field": "name",
  "requires_exact_match": true,
  "do_not_substitute_name_or_uuid_from_matches": true
}
```

## Custom Model Status Values

- `STATUS_IMPORTING`: Model files are being imported
- `STATUS_READY`: Model is ready for deployment
- `STATUS_FAILED`: Import failed
- `STATUS_DELETED`: Model has been deleted

## Source Types

- `SOURCE_TYPE_HUGGINGFACE`: Import from HuggingFace Hub
- `SOURCE_TYPE_SPACES_BUCKET`: Import from DigitalOcean Spaces
- `SOURCE_TYPE_SDK_UPLOAD`: Upload via SDK
- `SOURCE_TYPE_FINE_TUNING`: Result of fine-tuning

## Access Types (for source_ref)

- `ACCESS_TYPE_PUBLIC`: Publicly accessible model
- `ACCESS_TYPE_PRIVATE`: Private model (requires token)
- `ACCESS_TYPE_GATED`: Gated model (requires acceptance + token)

## Workflow Example

```
1. Ask the user to accept import terms (storage cost, license, source). Wait for explicit yes.

2. Import a model from HuggingFace:
   genai-custom-models-import
     name: "my-mistral-7b"
     source_type: "SOURCE_TYPE_HUGGINGFACE"
     source_ref: { "repo_id": "mistralai/Mistral-7B-v0.1", "access_type": "ACCESS_TYPE_PUBLIC" }
     accept_terms_and_conditions: true

3. Check import status by listing models:
   genai-custom-models-list
     status: "STATUS_IMPORTING"

4. Once ready, update metadata:
   genai-custom-models-update-metadata
     uuid: "<uuid from step 2>"
     description: "Production model for customer support"
     tags: { "tags": ["production", "v1"] }

5. Confirm deletion with the user (permanent removal). Wait for explicit yes.

6. Delete using the exact name or uuid the user confirmed:
   genai-custom-models-delete
     name: "my-mistral-7b"
     confirm_deletion: true
   # or
   genai-custom-models-delete
     uuid: "<full-uuid>"
     confirm_deletion: true
```
