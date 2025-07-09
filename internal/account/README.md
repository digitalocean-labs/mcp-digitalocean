# DigitalOcean Account Tools

This directory provides tool-based handlers for interacting with DigitalOcean account-related features via the MCP Server. All account operations are exposed as tools that accept structured arguments—no resource URIs are used. Pagination and filtering are supported where applicable.

## Supported Tools

### Actions

- **digitalocean-action-get**
  - Get a specific action by its ID.
  - Arguments:
    - `ID` (number, required): The action ID.

- **digitalocean-action-list**
  - List actions with pagination.
  - Arguments:
    - `Page` (number, default: 1): Page number.
    - `PerPage` (number, default: 30): Items per page.

### Balance

- **digitalocean-balance-get**
  - Get balance information for the user account.
  - Arguments: _none_

### Billing

- **digitalocean-billing-history-list**
  - List billing history with pagination.
  - Arguments:
    - `Page` (number, default: 1): Page number.
    - `PerPage` (number, default: 30): Items per page.

### Invoices

- **digitalocean-invoice-list**
  - List invoices with pagination.
  - Arguments:
    - `Page` (number, default: 1): Page number.
    - `PerPage` (number, default: 30): Items per page.

### SSH Keys

- **digitalocean-key-create**
  - Create a new SSH key.
  - Arguments:
    - `Name` (string, required): Name of the SSH key.
    - `PublicKey` (string, required): Public key content.

- **digitalocean-key-delete**
  - Delete an SSH key by its ID.
  - Arguments:
    - `ID` (number, required): ID of the SSH key to delete.

- **digitalocean-key-get**
  - Get a specific SSH key by its ID.
  - Arguments:
    - `ID` (number, required): ID of the SSH key.

- **digitalocean-key-list**
  - List SSH keys with pagination.
  - Arguments:
    - `Page` (number, default: 1): Page number.
    - `PerPage` (number, default: 30): Items per page.

### Account Information

- **digitalocean-account-get-information**
  - Retrieves account information for the current user.
  - Arguments: _none_

---

## Example Usage

- Get details for action ID 123456:
  - Tool: `digitalocean-action-get`
  - Arguments: `{ "ID": 123456 }`

- List actions (page 2, 50 per page):
  - Tool: `digitalocean-action-list`
  - Arguments: `{ "Page": 2, "PerPage": 50 }`

- Get current account balance:
  - Tool: `digitalocean-balance-get`
  - Arguments: `{}`

- List billing history (first page, 10 items per page):
  - Tool: `digitalocean-billing-history-list`
  - Arguments: `{ "Page": 1, "PerPage": 10 }`

- List invoices (default pagination):
  - Tool: `digitalocean-invoice-list`
  - Arguments: `{}`

- Create a new SSH key:
  - Tool: `digitalocean-key-create`
  - Arguments: `{ "Name": "my-key", "PublicKey": "ssh-rsa AAAA..." }`

- Delete SSH key with ID 98765:
  - Tool: `digitalocean-key-delete`
  - Arguments: `{ "ID": 98765 }`

- Get SSH key by ID:
  - Tool: `digitalocean-key-get`
  - Arguments: `{ "ID": 12345 }`

- List SSH keys (page 3, 20 per page):
  - Tool: `digitalocean-key-list`
  - Arguments: `{ "Page": 3, "PerPage": 20 }`

- Get current account information:
  - Tool: `digitalocean-account-get-information`
  - Arguments: `{}`

---

## Notes

- All tools use argument-based input; do not use resource URIs.
- Pagination is supported for list endpoints via `Page` and `PerPage` arguments.
- All responses are returned as JSON-formatted text.
- Error handling is consistent: errors are returned in the tool result with an error flag and message.