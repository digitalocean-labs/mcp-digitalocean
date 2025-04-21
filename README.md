# MCP DigitalOcean Integration

MCP DigitalOcean Integration is an open-source project that provides a comprehensive interface for managing DigitalOcean resources and performing actions using the [DigitalOcean API](https://docs.digitalocean.com/reference/api/). Built on top of the [godo](https://github.com/digitalocean/godo) library and the [MCP framework](https://github.com/mark3labs/mcp-go), this project exposes a wide range of tools and resources to simplify cloud infrastructure management.

> DISCLAIMER: “Use of MCP technology to interact with your DigitalOcean account [can come with risks](https://www.wiz.io/blog/mcp-security-research-briefing)”

## Features

### Resources

Resources provide read-only access to DigitalOcean entities, allowing users to retrieve detailed information about their infrastructure. The following resources are supported:

| **Resource**            | **Description**                                                                 |
|--------------------------|---------------------------------------------------------------------------------|
| **Droplets**             | Retrieve details about droplets, including actions and neighbors.              |
| **Sizes**                | List all available droplet sizes.                                              |
| **Account**              | Fetch account information.                                                     |
| **Balance**              | View current account balance.                                                  |
| **Billing**              | Access billing history.                                                        |
| **Invoices**             | Retrieve a list of all invoices.                                               |
| **Actions**              | Get details about specific actions.                                            |
| **Images**               | Retrieve information about distribution images or specific images.             |
| **CDNs**                 | Fetch details about CDN configurations.                                        |
| **Certificates**         | Retrieve certificate details.                                                  |
| **Domains**              | Access domain and domain record information.                                   |
| **Firewalls**            | Get details about firewalls.                                                   |
| **SSH Keys**             | Retrieve information about SSH keys.                                           |
| **Regions**              | List all available regions.                                                    |
| **Reserved IPs**         | Fetch details about reserved IPv4 and IPv6 addresses.                          |
| **Partner Attachments**  | Retrieve partner attachment details.                                           |
| **VPCs**                 | Get information about Virtual Private Clouds (VPCs).                          |

### Tools

Tools provide actionable capabilities for managing DigitalOcean resources. These tools are grouped by resource type and allow users to perform various operations. Below is an overview of the supported tools:

#### Droplet Tools
- Create, delete, resize, and rename droplets.
- Power on/off, reboot, and snapshot droplets.
- Manage backups, private networking, and IPv6.

#### CDN Tools
- Create and delete CDNs.
- Flush CDN caches.

#### Certificate Tools
- Create and delete certificates.
- Retrieve certificate details.

#### Domain Tools
- Create and delete domains.
- Manage domain records (create, edit, delete).

#### Firewall Tools
- Create and delete firewalls.
- Configure inbound and outbound rules.

#### SSH Key Tools
- Create and delete SSH keys.

#### Reserved IP Tools
- Reserve and release IPv4/IPv6 addresses.
- Assign and unassign reserved IPs to/from droplets.

#### Partner Attachment Tools
- Create, update, and delete partner attachments.
- Retrieve service keys and BGP configurations.

#### VPC Tools
- Create and delete VPCs.
- List VPC members.

---

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/your-repo/mcp-digitalocean.git
   cd mcp-digitalocean
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Build the project:
   ```bash
   make build-bin
   ```

---

## Usage

1. Set up your DigitalOcean API token:
   ```bash
   export DO_TOKEN=your_token
   ```

2. Run the MCP server:
   ```bash
   ./bin/mcp-digitalocean
   ```

3. Use the tools and resources to interact with your DigitalOcean infrastructure.

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
