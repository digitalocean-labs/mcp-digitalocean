## Spaces MCP Tools


This directory contains tools for managing DigitalOcean Spaces object storage and Spaces access keys via the MCP Server. All operations are exposed as tools with argument-based input—no resource URIs are used. Pagination and filtering are supported where applicable.

---

## Supported Tools

### Spaces Access Keys

- **spaces-key-create**  
  Create a new Spaces access key.  
  **Arguments:**
    - `Name` (string, required): Name for the Spaces key

- **spaces-key-delete**  
  Delete a Spaces access key.  
  **Arguments:**
    - `AccessKey` (string, required): Access Key of the Spaces key to delete

- **spaces-key-get**  
  Get information about a specific Spaces access key.  
  **Arguments:**
    - `AccessKey` (string, required): Access Key of the Spaces key to retrieve

- **spaces-key-list**  
  List all Spaces access keys with pagination.  
  **Arguments:**
    - `Page` (number, default: 1): Page number for pagination
    - `PerPage` (number, default: 10, max: 100): Number of items per page

- **spaces-key-update**  
  Update an existing Spaces access key.  
  **Arguments:**
    - `AccessKey` (string, required): Access Key of the Spaces key to update
    - `Name` (string, required): New name for the Spaces key

---

## Example Usage

- **Create a new Spaces access key:**  
  Tool: `spaces-key-create`  
  Arguments:
    - `Name`: `"production-key"`

- **Get a Spaces key by access key:**  
  Tool: `spaces-key-get`  
  Arguments:
    - `AccessKey`: `"AKIA1234567890EXAMPLE"`

- **List all Spaces keys:**  
  Tool: `spaces-key-list`  
  Arguments: `{}`

- **List Spaces keys with pagination:**  
  Tool: `spaces-key-list`  
  Arguments:
    - `Page`: `2`
    - `PerPage`: `20`

- **Update a Spaces key name:**  
  Tool: `spaces-key-update`  
  Arguments:
    - `AccessKey`: `"AKIA1234567890EXAMPLE"`
    - `Name`: `"new-key-name"`

- **Delete a Spaces key:**  
  Tool: `spaces-key-delete`  
  Arguments:
    - `AccessKey`: `"AKIA1234567890EXAMPLE"`

---

## Notes

- All tools use argument-based input; do not use resource URIs.
- Pagination is supported for list endpoints via `Page` and `PerPage` arguments.
- All responses are returned in JSON format for easy parsing and integration.
- Spaces keys provide S3-compatible access to DigitalOcean Spaces object storage.
- When creating a new key, it will have full access permissions by default.
- Access keys cannot be retrieved once created - only their metadata can be viewed.
- Store secret keys securely immediately after creation as they cannot be recovered.
