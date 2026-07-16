## DigitalOcean Vector Database (Managed Weaviate) Tools

This directory provides tools for managing DigitalOcean Managed Weaviate vector database clusters via the MCP server. All operations are exposed as tools with argument-based input, and the list endpoint supports pagination.

Vector databases are a public preview offering. Authentication reuses the caller's DigitalOcean API token (Bearer token), the same as every other service in this server.

---

## Supported Tools

- **vector-db-create**
Create a new managed Weaviate vector database cluster. Returns the full cluster object, including its `id`, `status`, and connection `endpoints` (`endpoints.http`, `endpoints.grpc`).
**Arguments:**
  - `Name` (string, required): Name of the cluster
  - `Region` (string, required): Region slug where the cluster will be created. Currently only `tor1` is available.
  - `Size` (string, required): Resource tier. One of `small`, `medium`, `large`.
  - `ProjectId` (string, required): ID of the project to assign the cluster to. **Write-only** — it is accepted on create but is **not** returned in any response.
  - `Tags` (array, optional): Tags to apply to the cluster
- **vector-db-list**
List vector database clusters. Supports pagination. Returns a summary of each cluster including its connection endpoints.
**Arguments:**
  - `Page` (number, default: 1): Page number
  - `PerPage` (number, default: 20): Clusters per page (capped at 50)
- **vector-db-get**
Get a cluster by ID. Returns the full cluster object including `status`, `config`, and `endpoints`.
**Arguments:**
  - `ID` (string, required): ID of the cluster
- **vector-db-resize**
Resize a cluster to a new resource tier.
**Arguments:**
  - `ID` (string, required): ID of the cluster to resize
  - `Size` (string, required): New resource tier. One of `small`, `medium`, `large`.
- **vector-db-delete**
Delete a cluster by ID. This is irreversible.
**Arguments:**
  - `ID` (string, required): ID of the cluster to delete
- **vector-db-get-credentials**
Get the admin credentials (`user_id` and `api_token`) for a cluster. Combine these with the cluster's `endpoints` to connect a Weaviate client.
**Arguments:**
  - `ID` (string, required): ID of the cluster

---

## Example Queries

- "Create a small Weaviate vector database named `search-index` in `tor1` under project `<project-id>`."
- "List all my vector databases."
- "Get the details of vector database `<id>`, including its connection endpoints."
- "Resize vector database `<id>` to medium."
- "Get the admin credentials for vector database `<id>` so I can connect a Weaviate client."
- "Delete vector database `<id>`."

---

## Notes

- All tools use argument-based input; do not use resource URIs.
- Numeric MCP arguments are handled as numbers, and in tests they are commonly represented as `float64`.
- `vector-db-list` returns a filtered summary of each cluster (`id`, `name`, `region`, `size`, `status`, `created_at`, `tags`, `endpoint_http`, `endpoint_grpc`); `vector-db-get` and `vector-db-create` return the full cluster object.
- `ProjectId` is required on create but write-only — it will not appear in the response. Retrieve the cluster afterwards to see its assigned state.
- `Size` values are `small`, `medium`, and `large`. The tools do not hard-validate the value; the API rejects anything else.
- `tor1` is currently the only region where vector databases can be provisioned. The tools do not hard-validate the region.
- `vector-db-get-credentials` returns admin secrets — handle the response accordingly.
