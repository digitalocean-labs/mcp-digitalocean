# Onboarding your tools

To onboard a new service you'll need to do the following:

1. **Create a new service directory**: Create a new directory under `pkg/` with the name of your service.
2. **Implement the tools** Within the service directory. 
3. **Update `registry.go`** Add your service to `supportedServices` and update the register function to include your service's tools.
4. **Update the README**: Document your service and its tools in the `README.md` file within your service directory.
5. **Create a PR**: Submit a pull request with your changes.

# Debugging and Troubleshooting

## Local Development

When developing tools for the MCP DigitalOcean integration, you may want to run the MCP server locally to test your changes. To do this, you can use the following command:

> We use goreleaser to build the binaries. To install it, follow the instructions at [goreleaser installation](https://goreleaser.com/install/).

```bash
# build the MCP server
make build-dist
```

Update your IDE mcp server configuration to use the local version:
```json
{
  "mcpServers": {
    "digitalocean": {
      "command": "npx",
      "args": ["/PATH/TO/PROJECT/mcp-digitalocean/scripts/npm", "--services apps"],
      "env": {
        "DIGITALOCEAN_API_TOKEN": ""
      }
    }
  }
}
```

# Using the MCP Inspector

If you need to look into the mcp server itself, you might want to use [mcp inspector](https://modelcontextprotocol.io/docs/tools/inspector).

To run the server with mcp inspector, you can use the following command:

```bash
npx -y @modelcontextprotocol/inspector npx PATH/TO/PROJECT/mcp-digitalocean/scripts/npm --services apps --digitalocean-api-token YOUR_DO_TOKEN
```

