# MCP DigitalOcean Integration

MCP DigitalOcean Integration is an open-source project that provides a comprehensive interface for managing DigitalOcean resources and performing actions using the [DigitalOcean API](https://docs.digitalocean.com/reference/api/). Built on top of the [godo](https://github.com/digitalocean/godo) library and the [MCP framework](https://github.com/mark3labs/mcp-go), this project exposes a wide range of tools and resources to simplify cloud infrastructure management.

> DISCLAIMER: “Use of MCP technology to interact with your DigitalOcean account [can come with risks](https://www.wiz.io/blog/mcp-security-research-briefing)”

## Installation

Prerequisites:

- Node.js (v18 or later)
- NPM (v8 or later)

### Local Installation

```bash
npx @digitalocean/mcp-digitalocean --services apps,droplets --log-level debug
```

### Using Cursor IDE

```json
{
  "mcpServers": {
    "digitalocean": {
      "command": "npx",
      "args": ["@digitalocean/mcp-digitalocean", "--services apps"],
      "env": {
        "DIGITALOCEAN_API_TOKEN": "YOUR_API_TOKEN"
      }
    }
  }
}
```

### Using VSCode
```json
{
    "mcp": {
        "inputs": [],
        "servers": {
            "mcpDigitalOcean": {
                "command": "npx",
                "args": [
                    "@digitalocean/mcp-digitalocean",
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

### Supported Services

The MCP DigitalOcean Integration supports a variety of services, allowing users to manage their DigitalOcean infrastructure effectively. The following services are currently supported:

| **Service**    | **Description**                                                                                                     |
|----------------|---------------------------------------------------------------------------------------------------------------------|
| **Apps**       | Manage DigitalOcean App Platform applications, including deployments and configurations.                            |
| **Droplets**   | Create, manage, resize, snapshot, and monitor droplets (virtual machines) on DigitalOcean.                          |
| **Account**    | Get information about your DigitalOcean account, billing, balance, invoices, and SSH keys.                          |
| **Networking** | Manage domains, DNS records, certificates, firewalls, reserved IPs, VPCs, CDNs, and partner attachments.            |

### Service Tools

Each service provides a toolset to interact with DigitalOcean.

| **Service**    | **Tools** (examples, see per-service README for full list)                                           |
|----------------|------------------------------------------------------------------------------------------------------|
| **Account**    | `digitalocean-key-create`, `digitalocean-key-delete`, `account://current`, `balance://current`, `billing://{last}`, `invoice://{last}`, `actions://{id}`, `keys://{id}` |
| **Apps**       | `digitalocean-create-app-from-spec`, `digitalocean-apps-update`, `digitalocean-apps-delete`, `digitalocean-apps-get-info`, `digitalocean-apps-usage`, `digitalocean-apps-get-deployment-status`, `digitalocean-apps-list` |
| **Droplets**   | `digitalocean-droplet-create`, `digitalocean-droplet-delete`, `digitalocean-droplet-power-cycle`, `digitalocean-droplet-resize`, `digitalocean-droplet-snapshot`, `digitalocean-droplet-enable-backups`, `digitalocean-droplet-get-neighbors`, `digitalocean-droplet-rename`, `digitalocean-droplet-rebuild`, `digitalocean-droplet-get-kernels`, ... (see [Droplet README](./internal/droplet/README.md)) |
| **Networking** | `digitalocean-domain-create`, `digitalocean-domain-delete`, `digitalocean-domain-record-create`, `digitalocean-domain-record-delete`, `digitalocean-certificate-create`, `digitalocean-certificate-delete`, `digitalocean-firewall-create`, `digitalocean-firewall-delete`, `digitalocean-reserved-ip-reserve`, `digitalocean-reserved-ip-release`, `digitalocean-vpc-create`, `digitalocean-vpc-delete`, `digitalocean-vpc-peering-create`, `digitalocean-vpc-peering-delete`, `digitalocean-cdn-create`, `digitalocean-cdn-delete`, `digitalocean-partner-attachment-create`, ... (see [Networking README](./internal/networking/README.md)) |

---
## Service Documentation

Each service provides a detailed README describing all available tools, resources, arguments, and example queries.
See the following files for full documentation:

- [Apps Service README](./internal/apps/README.md)
- [Droplet Service README](./internal/droplet/README.md)
- [Account Service README](./internal/account/README.md)
- [Networking Service README](./internal/networking/README.md)

---

### Example Resource URIs

Each service exposes resources that can be queried directly. Examples:

- **Account:** `account://current`, `balance://current`, `billing://3`, `invoice://6`, `actions://123456`, `keys://987654`
- **Apps:** `apps://{id}`, `apps://{id}/deployments/{deployment_id}`
- **Droplets:** `droplets://{id}`, `droplets://{id}/actions/{action_id}`, `images://distribution`, `images://{id}`, `sizes://all`
- **Networking:** `domains://{name}`, `domains://{name}/records/{record_id}`, `certificates://{id}`, `firewalls://{id}`, `reserved_ipv4://{ip}`, `vpcs://{id}`, `cdn://{id}`, `partner_attachment://{id}`, `vpc_peering://{id}`

---

### Example Tool Usage

- Deploy an app from a GitHub repo: `digitalocean-create-app-from-spec`
- Resize a droplet: `digitalocean-droplet-resize`
- Add a new SSH key: `digitalocean-key-create`
- Create a new domain: `digitalocean-domain-create`
- Enable backups on a droplet: `digitalocean-droplet-enable-backups`
- Flush a CDN cache: `digitalocean-cdn-flush-cache`
- List all available droplet sizes: `sizes://all`
- Get account balance: `balance://current`
- Create a VPC peering connection: `digitalocean-vpc-peering-create`
- Delete a VPC peering connection: `digitalocean-vpc-peering-delete`

---


## Configuring Tools

To configure tools, you use the `--services` flag to specify which service you want to enable. It is highly recommended to only
enable the services you need to reduce context size and improve accuracy.

```bash
npx @digitalocean/mcp-digitalocean --services apps,droplets
```

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
