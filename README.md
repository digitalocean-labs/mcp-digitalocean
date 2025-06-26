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
      "args": ["@digitalocean/mcp-digitalocean", "--services apps"]
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
                ]
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
| **Droplets**   | Create, manage, and monitor droplets (virtual machines) on DigitalOcean.                                            |
| **Account**    | Get information about your DigitalOcean account.                                                                    |
| **Networking** | Secure and control the traffic to your applications with private networking, traffic filtering, and load balancing. |

### Service Tools

Each service provides a toolset to interact with DigitalOcean.

| **Service**    | **Tools**                                                                                            |
|----------------|------------------------------------------------------------------------------------------------------|
| **Account**    | `get-account-info`, `get-account-balance`, `get-account-usage`                                       |
| **Apps**       | `create-app`, `get-app-info`, `update-app`, `delete-app`, `get-app-usage`, `list-apps`, `deploy-app` |
| **Droplets**   |                                                                                                      |
| **Networking** |                                                                                                      |

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
