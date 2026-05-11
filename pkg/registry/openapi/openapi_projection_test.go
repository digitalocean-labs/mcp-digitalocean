package openapi

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatOperation_golden(t *testing.T) {
	t.Parallel()
	const yaml = `openapi: 3.0.0
info:
  title: T
  version: "1"
servers:
  - url: https://api.example.com
paths:
  /v1/items/{id}:
    get:
      operationId: items_get
      summary: Get item
      tags:
        - Items
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
          description: Item id
      responses:
        "200":
          description: OK
`
	c := newTestOpenAPIClientYAML(t, yaml)
	op, err := c.GetOperation("items_get")
	require.NoError(t, err)

	want := "**GET** `/v1/items/{id}`\n\n" +
		"Get item\n\n" +
		"**Tags:** Items\n\n" +
		"### Parameters\n\n" +
		"- id (path, string, required) — Item id\n\n" +
		"### Responses\n\n" +
		"- **200:** OK"

	require.Equal(t, want, formatOperation(op))
}

func TestDescribeRequestBody_inProjection(t *testing.T) {
	t.Parallel()
	const yaml = `openapi: 3.0.0
info:
  title: T
  version: "1"
servers:
  - url: https://api.example.com
paths:
  /v1/items:
    post:
      operationId: items_create
      summary: Create
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                name:
                  type: string
                  description: Item name
      responses:
        "201":
          description: Created
`
	c := newTestOpenAPIClientYAML(t, yaml)
	op, err := c.GetOperation("items_create")
	require.NoError(t, err)
	require.Contains(t, op.RequestBody, "Content-Type: application/json")
	require.Contains(t, op.RequestBody, "name")
	require.Contains(t, formatOperation(op), "### Request body")
}
