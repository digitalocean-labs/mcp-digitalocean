# E2E Tests

This directory contains end-to-end (E2E) tests for the project. These tests are designed to validate the complete functionality of the system by simulating real-world scenarios without having 
requiring setting up cursor or claude. 

The tests will perform the following actions:

- Set up MCP Server with HTTP transport in a container. 
- Run the integration tests using the provided DigitalOcean API token and URL.

Usage:

```bash
DIGITALOCEAN_API_TOKEN=your_token_here  DIGITALOCEAN_API_URL=https://api.digitalocean.com go test -v -tags=integration ./testing/...
```

Optional environment variables:

- `MCP_SERVER_URL` — use an existing MCP server instead of starting a testcontainer (e.g. `https://genai-custom-models.mcp.digitalocean.com/mcp` for custom-models-only E2E).
- `GENAI_EVALUATION_TEST_AGENT_WORKSPACE_NAME` — when set, enables `TestGenAIListEvaluationTestCases` in [e2e_genai_evaluation_test.go](e2e_genai_evaluation_test.go) against that agent workspace.

GenAI custom models Hugging Face commit resolution and import behavior are covered in [e2e_custom_models_test.go](e2e_custom_models_test.go) and [e2e_custom_models_huggingface_test.go](e2e_custom_models_huggingface_test.go).

MicroDroplet / MicroVM tools (`--services microdroplets`, REST `/v2/microvms`) are covered in [e2e_microdroplet_test.go](e2e_microdroplet_test.go). List/get/checkpoint-list run by default (skip when the token lacks access). Optional create→delete: set `MICRODROPLET_E2E_OCI_REF` (and optionally `MICRODROPLET_E2E_REGION`, default `nyc1`).

