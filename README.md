# MCP DigitalOcean Integration

MCP DigitalOcean Integration is an open-source project that provides a comprehensive interface for managing DigitalOcean resources and performing actions using the [DigitalOcean API](https://docs.digitalocean.com/reference/api/). Built on top of the [godo](https://github.com/digitalocean/godo) library and the [MCP framework](https://github.com/mark3labs/mcp-go), this project exposes a wide range of tools to simplify cloud infrastructure management.

> **DISCLAIMER:** "Use of MCP technology to interact with your DigitalOcean account [can come with risks](https://www.wiz.io/blog/mcp-security-research-briefing)"

## 🚀 What's New in v1.0.12

### Performance & Reliability
- **In-memory Caching**: Configurable caching layer with TTL support for improved response times
- **Rate Limiting**: Token bucket rate limiter to prevent API abuse and respect rate limits
- **Enhanced Configuration**: Comprehensive configuration management with environment variables
- **Health Checks**: Built-in health monitoring for API connectivity and component status

### Observability
- **Metrics Collection**: Request/response metrics, success rates, and performance monitoring
- **Structured Logging**: Enhanced JSON logging with contextual information
- **Error Handling**: Improved error categorization and retry logic

### Code Quality
- **Better Architecture**: Modular design with dependency injection
- **Comprehensive Testing**: Enhanced test coverage for new components
- **Documentation**: Improved inline documentation and examples

## Prerequisites

- Node.js (v18 or later)
- NPM (v8 or later)

You can find installation guides at [https://nodejs.org/en/download](https://nodejs.org/en/download)

Verify your installation:
```bash
node --version
npm --version
```

## Quick Test

To verify the MCP server works correctly, you can test it directly from the command line:
```bash
npx @digitalocean/mcp --services apps
```

## Configuration

### Environment Variables

The MCP server supports extensive configuration through environment variables:

#### Required
- `DIGITALOCEAN_API_TOKEN`: Your DigitalOcean API token

#### Optional
- `SERVICES`: Comma-separated list of services to enable (default: all)
- `LOG_LEVEL`: Logging level (debug, info, warn, error) (default: info)
- `DIGITALOCEAN_API_ENDPOINT`: API endpoint URL (default: https://api.digitalocean.com)

#### Performance & Reliability
- `REQUEST_TIMEOUT`: API request timeout (default: 30s)
- `MAX_RETRIES`: Maximum retry attempts (default: 4)
- `RETRY_WAIT_MIN`: Minimum retry wait time (default: 1s)
- `RETRY_WAIT_MAX`: Maximum retry wait time (default: 30s)

#### Caching
- `CACHE_ENABLED`: Enable/disable caching (default: true)
- `CACHE_TTL`: Cache time-to-live (default: 5m)

#### Rate Limiting
- `RATE_LIMIT_ENABLED`: Enable/disable rate limiting (default: true)
- `RATE_LIMIT_RPS`: Requests per second limit (default: 100)

### Example Configuration

```bash
export DIGITALOCEAN_API_TOKEN="your_token_here"
export SERVICES="apps,droplets,networking"
export LOG_LEVEL="debug"
export CACHE_ENABLED="true"
export CACHE_TTL="10m"
export RATE_LIMIT_RPS="50"
```

## Installation

### Claude Code

To add the DigitalOcean MCP server to [Claude Code](https://www.anthropic.com/claude-code), run the following command in your terminal:

```bash
claude mcp add digitalocean-mcp \
  -e DIGITALOCEAN_API_TOKEN=YOUR_DO_API_TOKEN \
  -- npx @digitalocean/mcp --services apps,databases
```

This will:
- Add the MCP server under the default (local) scope — meaning it's only available inside the current folder.
- Register it with the name `digitalocean-mcp`.
- Enable the `apps` and `databases` services.
- Pass your DigitalOcean API token securely to the server.
- Store the configuration in your global Claude config at `~/.claude.json`, scoped to the current folder.

#### Verify Installation
To confirm it's been added:
```bash
claude mcp list
```

#### Inspect Details
To inspect details:
```bash
claude mcp get digitalocean-mcp
```

#### Remove Server
To remove it:
```bash
claude mcp remove digitalocean-mcp
```

### Claude Desktop

Alternatively, add the following to your claude_desktop_config.json file.

```json
{
  "mcpServers": {
    "digitalocean": {
      "command": "npx",
      "args": ["@digitalocean/mcp", "--services apps"],
      "env": {
        "DIGITALOCEAN_API_TOKEN": "YOUR_API_TOKEN"
      }
    }
  }
}
```

### Claude Code (User Scope)

Local scope is great when you're testing or only using the server in one project. User scope is better if you want it available everywhere.

If you'd like to make the server available globally (so you don't have to re-add it in each project), you can use the `user` scope:

```bash
claude mcp add -s user digitalocean-mcp-user-scope \
  -e DIGITALOCEAN_API_TOKEN=YOUR_DO_API_TOKEN \
  -- npx @digitalocean/mcp --services apps,databases
```

This will:
- Make the server available in all folders, not just the one you're in
- Scope it to your user account
- Store it in your global Claude config at `~/.claude.json`

To remove it:
```bash
claude mcp remove -s user digitalocean-mcp-user-scope
```

### Cursor

[![Install MCP Server](https://cursor.com/deeplink/mcp-install-dark.svg)](https://cursor.com/en/install-mcp?name=digitalocean&config=eyJjb21tYW5kIjoibnB4IEBkaWdpdGFsb2NlYW4vbWNwIC0tc2VydmljZXMgYXBwcyIsImVudiI6eyJESUdJVEFMT0NFQU5fQVBJX1RPS0VOIjoiWU9VUl9BUElfVE9LRU4ifX0%3D)

Add the following to your Cursor settings file located at `~/.cursor/config.json`:

```json
{
  "mcpServers": {
    "digitalocean": {
      "command": "npx",
      "args": ["@digitalocean/mcp", "--services", "apps"],
      "env": {
        "DIGITALOCEAN_API_TOKEN": "YOUR_API_TOKEN"
      }
    }
  }
}
```

#### Verify Installation in Cursor

1. Open Cursor and open Command Pallet ( `Shift + ⌘ + P` on Mac or `Ctrl+ Shift + P` on Windows/Linux )
2. Search for "MCP" in the command pallet search bar
3. Select "View: Open MCP Settings"
4. Select "Tools & Integrations" from the left sidebar
5. You should see "digitalocean" listed under Available MCP Servers
6. Click on "N tools enabled" (N is the number of tools currently enabled). 

#### Debugging in Cursor

To check MCP server logs and debug issues:
1. Open the Command Palette (⌘+Shift+P on Mac or Ctrl+Shift+P on Windows/Linux)
2. Type "Developer: Toggle Developer Tools" and press Enter
3. Navigate to the Console tab to view MCP server logs
4. You'll find MCP related logs as you interact with the MCP server

#### Testing the Connection

In Cursor's chat, try asking: "List all my DigitalOcean apps" - this should trigger the MCP server to fetch your apps if properly configured. If you are getting an 401 error or authentication related errors, it is likely due to misconfiguring your access token.

### VS Code

Add the following to your VS Code MCP configuration file:

```json
{
  "mcp": {
    "inputs": [],
    "servers": {
      "mcpDigitalOcean": {
        "command": "npx",
        "args": [
          "@digitalocean/mcp",
          "--services",
          "apps"
        ],
        "env": {
          "DIGITALOCEAN_API_TOKEN": "YOUR_API_TOKEN"
        }
      }
    }
  }
}
```

#### Verify Installation in VS Code

1. Open Cursor and open Command Pallet ( `Shift + ⌘ + P` on Mac or `Ctrl+ Shift + P` on Windows/Linux )
2. Search for "MCP" in the command pallet search bar
3. Select "MCP: List Servers"
4. Verify that "mcpDigitalOcean" appears in the list of configured servers

#### Viewing Available Tools

To see what tools are available from the MCP server:
1. Open the Command Palette (⌘+Shift+P on Mac or Ctrl+Shift+P on Windows/Linux)
2. Select "Agent" mode in the chatbox, 
3. Click "Configure tools" on the right, and check for digitalocean related tools under `MCP Server: mcpDigitalocean`. You should be able to list available tools like `app-create`, `app-list`, `app-delete`, etc.

#### Debugging in VS Code

To troubleshoot MCP connections:
1. Open the Command Palette (⌘+Shift+P on Mac or Ctrl+Shift+P on Windows/Linux)
2. Type "Developer: Toggle Developer Tools" and press Enter
3. Navigate to the Console tab to view MCP server logs
3. Check for connection status and error messages

If you are getting an 401 error or authentication related errors, it is likely due to misconfiguring your access token.

## Configuration

To configure tools, you use the `--services` flag to specify which service you want to enable. It is highly recommended to only
enable the services you need to reduce context size and improve accuracy. See list of supported services below.

```bash
npx @digitalocean/mcp --services apps,droplets
```

## Supported Services

The MCP DigitalOcean Integration supports the following services, allowing users to manage their DigitalOcean infrastructure effectively

| Service      | Description                                                                             |
|--------------|-----------------------------------------------------------------------------------------|
| apps         | Manage DigitalOcean App Platform applications, including deployments and configurations. |
| droplets     | Create, manage, resize, snapshot, and monitor droplets (virtual machines) on DigitalOcean. |
| accounts     | Get information about your DigitalOcean account, billing, balance, invoices, and SSH keys. |
| networking   | Manage domains, DNS records, certificates, firewalls, reserved IPs, VPCs, and CDNs. |
| insights     | Monitors your resources, endpoints and alert you when they're slow, unavailable, or SSL certificates are expiring. |
| spaces       | DigitalOcean Spaces object storage and Spaces access keys for S3-compatible storage. |
| databases    | Provision, manage, and monitor managed database clusters (Postgres, MySQL, Redis, etc.). |
| marketplace  | Discover and manage DigitalOcean Marketplace applications. |
| doks         | Manage DigitalOcean Kubernetes clusters and node pools. |

## Documentation

Each service provides a detailed README describing all available tools, resources, arguments, and example queries. See the following files for full documentation:

- [Apps Service](internal/apps/README.md)
- [Droplet Service](internal/droplet/README.md)
- [Account Service](internal/account/README.md)
- [Networking Service](internal/networking/README.md)
- [Databases Service](internal/dbaas/README.md)
- [Insights Service](internal/insights/README.md)
- [Spaces Service](internal/spaces/README.md)
- [Marketplace Service](internal/marketplace/README.md)
- [DOKS Service](internal/doks/README.md)

## Example Tools

- Deploy an app from a GitHub repo: `create-app-from-spec`
- Resize a droplet: `droplet-resize`
- Add a new SSH key: `key-create`
- Create a new domain: `domain-create`
- Enable backups on a droplet: `droplet-enable-backups`
- Flush a CDN cache: `cdn-flush-cache`
- Create a VPC peering connection: `vpc-peering-create`
- Delete a VPC peering connection: `vpc-peering-delete`

## Contributing

Contributions are welcome! If you encounter any issues or have ideas for improvements, feel free to open an issue or submit a pull request.

1. Fork the repository.
2. Create a new branch for your feature or bug fix.
3. Submit a pull request with a clear description of your changes.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
