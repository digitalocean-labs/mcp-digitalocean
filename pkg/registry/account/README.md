# DigitalOcean Account Tools

This directory provides tool-based handlers for interacting with DigitalOcean account-related features via the MCP Server. All account operations are exposed as tools that accept structured arguments—no resource URIs are used. Pagination and filtering are supported where applicable.

## Supported Tools

### Actions

- **action-get**
  - Get a specific action by its ID.
  - Arguments:
    - `ID` (number, required): The action ID.

- **action-list**
  - List actions with pagination.
  - Arguments:
    - `Page` (number, default: 1): Page number.
    - `PerPage` (number, default: 30): Items per page.

### Balance

- **balance-get**
  - Get balance information for the user account.
  - Arguments: _none_

### Billing

- **billing-history-list**
  - List billing history with pagination.
  - Arguments:
    - `Page` (number, default: 1): Page number.
    - `PerPage` (number, default: 30): Items per page.

### Invoices

- **invoice-list**
  - List invoices with pagination.
  - Arguments:
    - `Page` (number, default: 1): Page number.
    - `PerPage` (number, default: 30): Items per page.

### SSH Keys

- **key-create**
  - Create a new SSH key.
  - Arguments:
    - `Name` (string, required): Name of the SSH key.
    - `PublicKey` (string, required): Public key content.

- **key-delete**
  - Delete an SSH key.
  - Arguments:
    - `ID` (number, required): The SSH key ID.

- **key-get**
  - Get a specific SSH key by ID.
  - Arguments:
    - `ID` (number, required): The SSH key ID.

- **key-list**
  - List SSH keys with pagination.
  - Arguments:
    - `Page` (number, default: 1): Page number.
    - `PerPage` (number, default: 30): Items per page.

### Account Info

- **account-get-information**
  - Get information about the current account.
  - Arguments: _none_

---

## Example Usage

- Get details for action ID 123456:
  - Tool: `action-get`
  - Arguments: `{ "ID": 123456 }`

- List actions (page 2, 50 per page):
  - Tool: `action-list`
  - Arguments: `{ "Page": 2, "PerPage": 50 }`

- Get current account balance:
  - Tool: `balance-get`
  - Arguments: `{}`

- List billing history (first page, 10 items per page):
  - Tool: `billing-history-list`
  - Arguments: `{ "Page": 1, "PerPage": 10 }`

- List invoices (default pagination):
  - Tool: `invoice-list`
  - Arguments: `{}`

- Create a new SSH key:
  - Tool: `key-create`
  - Arguments: `{ "Name": "my-key", "PublicKey": "ssh-rsa AAAA..." }`

- Delete SSH key with ID 98765:
  - Tool: `key-delete`
  - Arguments: `{ "ID": 98765 }`

- Get SSH key by ID:
  - Tool: `key-get`
  - Arguments: `{ "ID": 12345 }`

- List SSH keys (page 3, 20 per page):
  - Tool: `key-list`
  - Arguments: `{ "Page": 3, "PerPage": 20 }`

- Get current account information:
  - Tool: `account-get-information`
  - Arguments: `{}`

---

## Notes

- All tools use argument-based input; do not use resource URIs.
- Pagination is supported for list endpoints via `Page` and `PerPage` arguments.
- All responses are returned as JSON-formatted text.
- Error handling is consistent: errors are returned in the tool result with an error flag and message.