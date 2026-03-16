# Service Category Filtering

The MCP DigitalOcean server supports granular tool filtering to reduce context window usage. By default, only "basic" tools are loaded for each service.

## Usage

```bash
# Load basic tools for all services (default)
--services droplets,networking,databases

# Load specific categories
--services droplets:basic,networking:lb,databases:postgresql

# Load all tools for a service
--services droplets:all

# Mix and match
--services droplets:basic,droplets:actions,networking:all
```

## Available Categories

### droplets
| Category | Description |
|----------|-------------|
| `basic` | Droplet CRUD operations (create, delete, get, list, power ops) |
| `actions` | Droplet actions (reboot, snapshot, resize, etc.) |
| `images` | Image management and actions |
| `sizes` | List available droplet sizes |
| `all` | All droplet tools |

### networking
| Category | Description |
|----------|-------------|
| `basic` / `lb` | Load balancer management |
| `firewall` | Firewall rules |
| `dns` | Domains and certificates |
| `vpc` | VPC and VPC peering |
| `ip` | Reserved IPs and BYOIP |
| `all` | All networking tools |

### databases
| Category | Description |
|----------|-------------|
| `basic` / `cluster` | Generic cluster operations |
| `postgresql` | PostgreSQL-specific tools |
| `mysql` | MySQL-specific tools |
| `mongodb` | MongoDB-specific tools |
| `redis` | Redis-specific tools |
| `kafka` | Kafka-specific tools |
| `opensearch` | OpenSearch-specific tools |
| `users` | Database user management |
| `firewall` | Database firewall rules |
| `all` | All database tools |

### accounts
| Category | Description |
|----------|-------------|
| `basic` / `info` | Account information |
| `billing` | Balance, billing history, invoices |
| `keys` | SSH key management |
| `actions` | Account actions |
| `all` | All account tools |

### spaces
| Category | Description |
|----------|-------------|
| `basic` / `keys` | Spaces key management |
| `cdn` | CDN endpoints |
| `all` | All spaces tools |

### insights
| Category | Description |
|----------|-------------|
| `basic` / `uptime` | Uptime checks and alerts |
| `alerts` | Alert policies |
| `all` | All insights tools |

### apps, doks, marketplace
These services currently load all tools regardless of category (small tool sets).

## Examples

### Minimal setup for droplet management
```bash
--services droplets:basic,accounts:info
```

### Web application deployment
```bash
--services droplets:basic,networking:lb,networking:dns,databases:postgresql
```

### Full infrastructure management
```bash
--services droplets:all,networking:all,databases:all
```

## Context Window Impact

Using category filtering can significantly reduce context window usage:

| Configuration | Approximate Tools |
|---------------|-------------------|
| All services (no filtering) | ~200 tools |
| All services (basic only) | ~50 tools |
| droplets:basic,networking:lb | ~20 tools |
