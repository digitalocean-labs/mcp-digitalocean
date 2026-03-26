## Model Catalog MCP Tools

This directory contains tools for managing DigitalOcean Model Catalog via the MCP Server. The Model Catalog provides access to AI models available on DigitalOcean's Gradient AI platform. All operations are exposed as tools with argument-based input—no resource URIs are used.

---

## Supported Tools

### Model Catalog Tools

- **genai-model-catalog-search**  
  Search for models in the catalog using a search query. Returns a list of model UUIDs that match the search criteria.  
  **Arguments:**  
  - `SearchQuery` (string, optional): Search query string to find models. An empty or missing query returns all available models.

- **genai-model-catalog-get-card**  
  Get the model metadata for a specific model UUID.  
  **Arguments:**  
  - `ModelUUID` (string, required): The unique UUID identifier of the model

---

## Example Usage

- **Search for models by name:**  
  Tool: `genai-model-catalog-search`  
  Arguments:  
  - `SearchQuery`: `"llama"`

- **List all available models:**  
  Tool: `genai-model-catalog-search`  
  Arguments: `{}` or `{"SearchQuery": ""}`

- **Get model details:**  
  Tool: `genai-model-catalog-get-card`  
  Arguments:  
  - `ModelUUID`: `"12345678-1234-1234-1234-123456789012"`

---

## Notes

- The search is case-insensitive and matches against model names
- An empty or missing `SearchQuery` returns all available models
- A valid DigitalOcean API token is required for all operations
- Model UUIDs are stable identifiers that can be used for deployment and API access
