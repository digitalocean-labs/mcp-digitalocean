// Package openapi implements MCP tools backed by the embedded DigitalOcean
// public OpenAPI 3 specification. Agents can search operations, inspect full
// metadata for an operationId, and execute requests after validation with
// kin-openapi against the live API using godo.Client.Do. DELETE operations use
// a dedicated tool (openapi-execute-delete) with destructiveHint annotations.
package openapi
