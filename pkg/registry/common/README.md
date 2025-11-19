# Common MCP Tools

This directory contains shared tool-based handlers and utility functions that are always included as part of the DigitalOcean MCP integration. The contents of this directory provide utilities and abstractions leveraged across all DigitalOcean MCP resources and tools.

## Purpose

The `common` package is designed to:

- Provide utility functions for extracting and parsing arguments.
- Offer core tools that are universally useful for DigitalOcean interactions (such as region listings).
- Serve as a central place for logic and helpers that are not specific to a single DigitalOcean product or service, but are required by many.

## Included Tools

### Regions Tool

- **region-list**
  - Lists all available DigitalOcean regions, including their features and droplet size availability.
  - Supports pagination.
  - **Arguments:**
    - `Page` (number, default: 1): Page number.
    - `PerPage` (number, default: 50): Items per page.

#### Example Usage

- List all regions (default pagination):
  - Tool: `region-list`
  - Arguments: `{}`

- List regions, page 2, 20 per page:
  - Tool: `region-list`
  - Arguments: `{ "Page": 2, "PerPage": 20 }`

## Notes

- All tools use argument-based input; do not use resource URIs.
- Pagination is supported for list endpoints via `Page` and `PerPage` arguments.
- All responses are returned as JSON-formatted text.
- Error handling is consistent: errors are returned in the tool result with an error flag and message.