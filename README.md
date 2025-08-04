# MCP DigitalOcean Integration

MCP DigitalOcean Integration is an open-source project that provides a comprehensive interface for managing DigitalOcean resources and performing actions using the [DigitalOcean API](https://docs.digitalocean.com/reference/api/). Built on top of the [godo](https://github.com/digitalocean/godo) library and the [MCP framework](https://github.com/mark3labs/mcp-go), this project exposes a wide range of tools to simplify cloud infrastructure management.

> DISCLAIMER: “Use of MCP technology to interact with your DigitalOcean account [can come with risks](https://www.wiz.io/blog/mcp-security-research-briefing)”

## Installation

Prerequisites:

- Node.js (v18 or later)
- NPM (v8 or later)

#### Local Installation

```bash
npx @digitalocean/mcp --services apps 
```
---

#### Using with Claude Code

To add the DigitalOcean MCP server to [Claude Code](https://www.anthropic.com/claude-code), run the following command in your terminal:

```bash
claude mcp add digitalocean-mcp \
  -e DIGITALOCEAN_API_TOKEN=YOUR_DO_API_TOKEN \
  -- npx @digitalocean/mcp --services apps,databases
```

This will:

* Add the MCP server under the **default (local)** scope — meaning it's only available inside the current folder.
* Register it with the name `digitalocean-mcp`.
* Enable the `apps` and `databases` services.
* Pass your DigitalOcean API token securely to the server.
* Store the configuration in your global Claude config at `~/.claude.json`, scoped to the current folder.

To confirm it's been added:

```bash
claude mcp list
```

To inspect details:

```bash
claude mcp get digitalocean-mcp
```

To remove it:

```bash
claude mcp remove digitalocean-mcp
```

---

### Want to use it everywhere?
**Local scope** is great when you're testing or only using the server in one project.
**User scope** is better if you want it available everywhere.

If you’d like to make the server available **globally** (so you don’t have to re-add it in each project), you can use the `user` scope:

```bash
claude mcp add -s user digitalocean-mcp-user-scope \
  -e DIGITALOCEAN_API_TOKEN=YOUR_DO_API_TOKEN \
  -- npx @digitalocean/mcp --services apps,databases
```

This will:

* Make the server available in **all folders**, not just the one you’re in
* Scope it to **your user account**
* Store it in your global Claude config at `~/.claude.json`

To remove it:

```bash
claude mcp remove -s user digitalocean-mcp-user-scope
```

---

#### Using Cursor IDE

[![Install MCP Server](https://cursor.com/deeplink/mcp-install-dark.svg)](https://cursor.com/install-mcp?name=digitalocean&config=JTdCJTIyY29tbWFuZCUyMiUzQSUyMm5weCUyMCU0MGRpZ2l0YWxvY2VhbiUyRm1jcCUyMC0tc2VydmljZXMlMjBhcHBzJTIyJTJDJTIyZW52JTIyJTNBJTdCJTIyRElHSVRBTE9DRUFOX0FQSV9UT0tFTiUyMiUzQSUyMllPVVJfQVBJX1RPS0VOJTIyJTdEJTdE)

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

#### Using VSCode
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

## Configuring Tools

To configure tools, you use the `--services` flag to specify which service you want to enable. It is highly recommended to only
enable the services you need to reduce context size and improve accuracy. See list of supported services below.

```bash
npx @digitalocean/mcp --services apps,droplets
```

## Supported Services

The MCP DigitalOcean Integration supports the following services, allowing users to manage their DigitalOcean infrastructure effectively

| **Service**     | **Description**                                                                                                    |
|-----------------|--------------------------------------------------------------------------------------------------------------------|
| **apps**        | Manage DigitalOcean App Platform applications, including deployments and configurations.                           |
| **droplets**    | Create, manage, resize, snapshot, and monitor droplets (virtual machines) on DigitalOcean.                         |
| **accounts**    | Get information about your DigitalOcean account, billing, balance, invoices, and SSH keys.                         |
| **networking**  | Manage domains, DNS records, certificates, firewalls, reserved IPs, VPCs, and CDNs.                                |
| **insights**    | Monitors your resources, endpoints and alert you when they're slow, unavailable, or SSL certificates are expiring. |
| **spaces**      | DigitalOcean Spaces object storage and Spaces access keys for S3-compatible storage.                               |
| **databases**   | Provision, manage, and monitor managed database clusters (Postgres, MySQL, Redis, etc.).                           |
| **marketplace** | Discover and manage DigitalOcean Marketplace applications.                                                         |
| **doks**        | Manage DigitalOcean Kubernetes clusters and node pools.                                                            |                                                   |
---
### Service Documentation

Each service provides a detailed README describing all available tools, resources, arguments, and example queries.
See the following files for full documentation:

- [Apps Service](./internal/apps/README.md)
- [Droplet Service](./internal/droplet/README.md)
- [Account Service](./internal/account/README.md)
- [Networking Service](./internal/networking/README.md)
- [Databases Service](./internal/dbaas/README.md)
- [Insights Service](./internal/insights/README.md)
- [Spaces Service](./internal/spaces/README.md)
- [Marketplace Service](./internal/marketplace/README.md)
- [DOKS Service](./internal/doks/README.md)

---

### Example Tool Usage

- Deploy an app from a GitHub repo: `create-app-from-spec`
- Resize a droplet: `droplet-resize`
- Add a new SSH key: `key-create`
- Create a new domain: `domain-create`
- Enable backups on a droplet: `droplet-enable-backups`
- Flush a CDN cache: `cdn-flush-cache`
- Create a VPC peering connection: `vpc-peering-create`
- Delete a VPC peering connection: `vpc-peering-delete`

---

## Contributing

Contributions are welcome! If you encounter any issues or have ideas for improvements, feel free to open an issue or submit a pull request.

### How to Contribute
1. Fork the repository.
2. Create a new branch for your feature or bug fix.
3. Submit a pull request with a clear description of your changes.

---

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
