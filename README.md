# MCP DigitalOcean Integration

MCP DigitalOcean Integration is an open-source project that provides a comprehensive interface for managing DigitalOcean resources and performing actions using the [DigitalOcean API](https://docs.digitalocean.com/reference/api/). Built on top of the [godo](https://github.com/digitalocean/godo) library and the [MCP framework](https://github.com/mark3labs/mcp-go), this project exposes a wide range of tools to simplify cloud infrastructure management.

> **DISCLAIMER:** "Use of MCP technology to interact with your DigitalOcean account [can come with risks](https://www.wiz.io/blog/mcp-security-research-briefing)"

---

## Security: Never Hardcode Your API Token

> **WARNING:** Do NOT paste your DigitalOcean API token directly into any config file (e.g., `claude_desktop_config.json`, `~/.cursor/config.json`, VS Code settings). If you commit these files to GitHub, your token will be exposed and GitHub will automatically block or revoke it to protect your account.

The safe approach is to store your token in an **environment variable** on your machine and reference it from your config file. Follow the steps below for your operating system before proceeding with any client installation.

---

## Step 1: Set Up Your API Token as an Environment Variable

### macOS / Linux

#### Option A: Set it in your shell profile (Recommended — persists across reboots)

1. Open a terminal.

2. Determine which shell you are using:
   ```bash
   echo $SHELL
   ```
   - If it outputs `/bin/zsh` → edit `~/.zshrc`
   - If it outputs `/bin/bash` → edit `~/.bashrc` or `~/.bash_profile`

3. Open the file in a text editor:
   ```bash
   # For zsh (default on macOS Monterey and later)
   nano ~/.zshrc

   # For bash
   nano ~/.bashrc
   ```

4. Add the following line at the bottom of the file:
   ```bash
   export DIGITALOCEAN_API_TOKEN="your_actual_token_here"
   ```
   Replace `your_actual_token_here` with your real token from the [DigitalOcean API Tokens page](https://cloud.digitalocean.com/account/api/tokens).

5. Save and exit:
   - In `nano`: press `Ctrl + O`, then `Enter` to save, then `Ctrl + X` to exit.

6. Reload your shell to apply the change:
   ```bash
   source ~/.zshrc    # or source ~/.bashrc
   ```

7. Verify it is set correctly:
   ```bash
   echo $DIGITALOCEAN_API_TOKEN
   ```
   You should see your token printed in the terminal.

#### Option B: Use a `.env` file (for project-level isolation)

1. In your project root folder, create a `.env` file:
   ```bash
   touch .env
   ```

2. Open it and add:
   ```
   DIGITALOCEAN_API_TOKEN=your_actual_token_here
   ```

3. **Immediately** add `.env` to your `.gitignore` so it is never committed:
   ```bash
   echo ".env" >> .gitignore
   ```

4. To load the `.env` file into your current terminal session:
   ```bash
   export $(grep -v '^#' .env | xargs)
   ```

5. Verify:
   ```bash
   echo $DIGITALOCEAN_API_TOKEN
   ```

> **Note:** Option B only sets the variable for the current terminal session. You need to run step 4 each time you open a new terminal. For a permanent solution, use Option A.

---

### Windows

#### Option A: Set it as a System Environment Variable (Recommended — persists across reboots)

1. Open **Start Menu** and search for **"Environment Variables"**.
2. Click **"Edit the system environment variables"**.
3. In the System Properties dialog, click the **"Environment Variables..."** button.
4. Under **"User variables"**, click **"New..."**.
5. Fill in:
   - **Variable name:** `DIGITALOCEAN_API_TOKEN`
   - **Variable value:** `your_actual_token_here`
6. Click **OK** on all dialogs to save.
7. **Restart your terminal** (Command Prompt or PowerShell) for the change to take effect.
8. Verify in PowerShell:
   ```powershell
   echo $env:DIGITALOCEAN_API_TOKEN
   ```
   Or in Command Prompt:
   ```cmd
   echo %DIGITALOCEAN_API_TOKEN%
   ```

#### Option B: Set it temporarily in PowerShell (current session only)

```powershell
$env:DIGITALOCEAN_API_TOKEN = "your_actual_token_here"
```

#### Option C: Use a `.env` file on Windows

1. In your project folder, create a file named `.env` (no extension) with this content:
   ```
   DIGITALOCEAN_API_TOKEN=your_actual_token_here
   ```

2. Add `.env` to your `.gitignore`:
   ```
   .env
   ```

3. To load it in PowerShell:
   ```powershell
   Get-Content .env | ForEach-Object {
     if ($_ -match "^\s*([^#][^=]*)=(.*)$") {
       [System.Environment]::SetEnvironmentVariable($matches[1].Trim(), $matches[2].Trim(), "Process")
     }
   }
   ```

4. Verify:
   ```powershell
   echo $env:DIGITALOCEAN_API_TOKEN
   ```

---

## Step 2: Get Your DigitalOcean API Token

1. Log in to your DigitalOcean account.
2. Navigate to **API** → **Tokens** in the left sidebar, or go directly to: `https://cloud.digitalocean.com/account/api/tokens`
3. Click **"Generate New Token"**.
4. Give it a name (e.g., `mcp-local-dev`), set expiry, and choose the required scopes.
5. Copy the token immediately — it will only be shown once.
6. Store it using one of the methods described in Step 1 above.

---

## Installation

### Remote MCP (Recommended)

The easiest way to get started is to use DigitalOcean's hosted MCP services. Each service is deployed as a standalone MCP server accessible via HTTPS, allowing you to connect without running any local server. You can connect to multiple endpoints simultaneously by adding multiple entries to your configuration.

#### Available Services

| Service      | Remote MCP URL                              | Description                                                                             |
|--------------|---------------------------------------------|-----------------------------------------------------------------------------------------|
| apps         | https://apps.mcp.digitalocean.com/mcp       | Manage DigitalOcean App Platform applications, including deployments and configurations. |
| accounts     | https://accounts.mcp.digitalocean.com/mcp   | Get information about your DigitalOcean account, billing, balance, invoices, and SSH keys. |
| databases    | https://databases.mcp.digitalocean.com/mcp  | Provision, manage, and monitor managed database clusters (Postgres, MySQL, Redis, etc.). |
| doks         | https://doks.mcp.digitalocean.com/mcp       | Manage DigitalOcean Kubernetes clusters and node pools. |
| droplets     | https://droplets.mcp.digitalocean.com/mcp   | Create, manage, resize, snapshot, and monitor droplets (virtual machines) on DigitalOcean. |
| docr         | https://docr.mcp.digitalocean.com/mcp       | Manage DigitalOcean Container Registry repositories, tags, manifests, and garbage collection. |
| insights     | https://insights.mcp.digitalocean.com/mcp   | Monitors your resources, endpoints and alert you when they're slow, unavailable, or SSL certificates are expiring. |
| marketplace  | https://marketplace.mcp.digitalocean.com/mcp| Discover and manage DigitalOcean Marketplace applications. |
| networking   | https://networking.mcp.digitalocean.com/mcp | Manage domains, DNS records, certificates, firewalls, load balancers, reserved IPs, BYOIP Prefixes, VPCs, and CDNs. |
| spaces       | https://spaces.mcp.digitalocean.com/mcp     | DigitalOcean Spaces object storage and Spaces access keys for S3-compatible storage. |

---

### Claude Code

#### Remote MCP (Recommended)

Make sure you have completed Step 1 and your `DIGITALOCEAN_API_TOKEN` environment variable is set. Then run:

```bash
claude mcp add --transport http digitalocean-apps https://apps.mcp.digitalocean.com/mcp \
  --header "Authorization: Bearer $DIGITALOCEAN_API_TOKEN"
```

> **How this works:** The `$DIGITALOCEAN_API_TOKEN` is expanded by your shell at the time you run the command. The actual token value is stored securely in Claude's config — you never paste the token into the command yourself.

You can add multiple services the same way:

```bash
claude mcp add --transport http digitalocean-databases https://databases.mcp.digitalocean.com/mcp \
  --header "Authorization: Bearer $DIGITALOCEAN_API_TOKEN"

claude mcp add --transport http digitalocean-droplets https://droplets.mcp.digitalocean.com/mcp \
  --header "Authorization: Bearer $DIGITALOCEAN_API_TOKEN"
```

See the [Available Services](#available-services) section for the complete list of available endpoints.

#### Local Installation

```bash
claude mcp add digitalocean-mcp \
  -e DIGITALOCEAN_API_TOKEN=$DIGITALOCEAN_API_TOKEN \
  -- npx @digitalocean/mcp --services apps,databases
```

This will:
- Add the MCP server under the default (local) scope — meaning it's only available inside the current folder.
- Register it with the name `digitalocean-mcp`.
- Enable the `apps` and `databases` services.
- Pass your DigitalOcean API token securely to the server via environment variable.
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

##### User Scope

Local scope is great when you're testing or only using the server in one project. User scope is better if you want it available everywhere.

If you'd like to make the server available globally (so you don't have to re-add it in each project), you can use the `user` scope:

```bash
claude mcp add -s user digitalocean-mcp-user-scope \
  -e DIGITALOCEAN_API_TOKEN=$DIGITALOCEAN_API_TOKEN \
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

---

### Claude Desktop

#### Remote MCP (Recommended)

**Before editing the config file**, ensure `DIGITALOCEAN_API_TOKEN` is set in your shell profile (see Step 1 — Option A). Claude Desktop reads environment variables from your shell profile at launch.

The config file is located at:
- **macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows:** `%APPDATA%\Claude\claude_desktop_config.json`
- **Linux:** `~/.config/Claude/claude_desktop_config.json`

Add the remote MCP servers to your config file. Reference the env var using `${DIGITALOCEAN_API_TOKEN}` — Claude Desktop will substitute it at runtime:

```json
{
  "mcpServers": {
    "digitalocean-apps": {
      "url": "https://apps.mcp.digitalocean.com/mcp",
      "headers": {
        "Authorization": "Bearer ${DIGITALOCEAN_API_TOKEN}"
      }
    },
    "digitalocean-databases": {
      "url": "https://databases.mcp.digitalocean.com/mcp",
      "headers": {
        "Authorization": "Bearer ${DIGITALOCEAN_API_TOKEN}"
      }
    }
  }
}
```

> **Important:** The `${DIGITALOCEAN_API_TOKEN}` syntax tells Claude Desktop to read the value from your system's environment variables. Your actual token is never written into this file. This file is safe to commit to version control.

You can add any of the endpoints listed in the [Available Services](#available-services) section.

#### Local Installation

Add the following to your `claude_desktop_config.json` file. The `env` block passes the environment variable directly to the MCP server process — no hardcoded token needed:

```json
{
  "mcpServers": {
    "digitalocean": {
      "command": "npx",
      "args": ["@digitalocean/mcp", "--services", "apps"],
      "env": {
        "DIGITALOCEAN_API_TOKEN": "${DIGITALOCEAN_API_TOKEN}"
      }
    }
  }
}
```

> **How this works:** The `"env"` block in Claude Desktop config passes environment variables to the spawned MCP server process. The `${DIGITALOCEAN_API_TOKEN}` value is resolved from your system environment at runtime. Your token is never stored in the config file itself.

After saving the file, **restart Claude Desktop** for the changes to take effect.

---

### Cursor

#### Remote MCP (Recommended)

**Before editing the config file**, ensure `DIGITALOCEAN_API_TOKEN` is set in your shell profile (see Step 1 — Option A).

The Cursor config file is located at:
- **macOS / Linux:** `~/.cursor/config.json`
- **Windows:** `%USERPROFILE%\.cursor\config.json`

Add the remote MCP servers to your Cursor settings file:

```json
{
  "mcpServers": {
    "digitalocean-apps": {
      "url": "https://apps.mcp.digitalocean.com/mcp",
      "headers": {
        "Authorization": "Bearer ${DIGITALOCEAN_API_TOKEN}"
      }
    },
    "digitalocean-databases": {
      "url": "https://databases.mcp.digitalocean.com/mcp",
      "headers": {
        "Authorization": "Bearer ${DIGITALOCEAN_API_TOKEN}"
      }
    }
  }
}
```

You can add any of the endpoints listed in the [Available Services](#available-services) section.

#### Local Installation

[![Install MCP Server](https://cursor.com/deeplink/mcp-install-dark.svg)](https://cursor.com/en/install-mcp?name=digitalocean&config=eyJjb21tYW5kIjoibnB4IEBkaWdpdGFsb2NlYW4vbWNwIC0tc2VydmljZXMgYXBwcyIsImVudiI6eyJESUdJVEFMT0NFQU5fQVBJX1RPS0VOIjoiWU9VUl9BUElfVE9LRU4ifX0%3D)

Add the following to your Cursor settings file located at `~/.cursor/config.json`:

```json
{
  "mcpServers": {
    "digitalocean": {
      "command": "npx",
      "args": ["@digitalocean/mcp", "--services", "apps"],
      "env": {
        "DIGITALOCEAN_API_TOKEN": "${DIGITALOCEAN_API_TOKEN}"
      }
    }
  }
}
```

> **How this works:** Cursor resolves `${DIGITALOCEAN_API_TOKEN}` from your system environment variables when it launches the MCP server. Make sure the variable is set in your shell profile before starting Cursor.

##### Verify Installation

1. Open Cursor and open Command Palette (`Shift + ⌘ + P` on Mac or `Ctrl + Shift + P` on Windows/Linux)
2. Search for "MCP" in the command palette search bar
3. Select "View: Open MCP Settings"
4. Select "Tools & Integrations" from the left sidebar
5. You should see "digitalocean" listed under Available MCP Servers
6. Click on "N tools enabled" (N is the number of tools currently enabled)

##### Debugging

To check MCP server logs and debug issues:
1. Open the Command Palette (`⌘+Shift+P` on Mac or `Ctrl+Shift+P` on Windows/Linux)
2. Type "Developer: Toggle Developer Tools" and press Enter
3. Navigate to the Console tab to view MCP server logs
4. You'll find MCP related logs as you interact with the MCP server

##### Testing the Connection

In Cursor's chat, try asking: "List all my DigitalOcean apps" — this should trigger the MCP server to fetch your apps if properly configured. If you are getting a 401 error or authentication related errors, verify that the `DIGITALOCEAN_API_TOKEN` variable is correctly set by running `echo $DIGITALOCEAN_API_TOKEN` in your terminal.

---

### VS Code

#### Remote MCP (Recommended)

**Before editing the config file**, ensure `DIGITALOCEAN_API_TOKEN` is set in your shell profile (see Step 1 — Option A).

The VS Code MCP config file is located at `.vscode/mcp.json` in your workspace root. Create this file if it does not exist. Add the remote MCP servers:

```json
{
  "inputs": [
    {
      "id": "digitalocean-token",
      "type": "promptString",
      "description": "DigitalOcean API Token",
      "password": true
    }
  ],
  "servers": {
    "digitalocean-apps": {
      "url": "https://apps.mcp.digitalocean.com/mcp",
      "headers": {
        "Authorization": "Bearer ${input:digitalocean-token}"
      }
    },
    "digitalocean-databases": {
      "url": "https://databases.mcp.digitalocean.com/mcp",
      "headers": {
        "Authorization": "Bearer ${input:digitalocean-token}"
      }
    }
  }
}
```

> **How VS Code inputs work:** The `inputs` block defines a named input (`digitalocean-token`) that VS Code will **prompt you to enter securely** the first time you use the server. Your token is stored in VS Code's secret storage — never written to disk in plain text. The `"password": true` flag ensures it is masked during input. This config file is completely safe to commit to GitHub.

Alternatively, if you prefer to use a system environment variable directly:

```json
{
  "inputs": [],
  "servers": {
    "digitalocean-apps": {
      "url": "https://apps.mcp.digitalocean.com/mcp",
      "headers": {
        "Authorization": "Bearer ${env:DIGITALOCEAN_API_TOKEN}"
      }
    }
  }
}
```

> **Note:** VS Code uses `${env:VARIABLE_NAME}` syntax to read from system environment variables.

#### Local Installation

Add the following to your `.vscode/mcp.json` file. Use VS Code's `inputs` feature to avoid hardcoding the token:

```json
{
  "inputs": [
    {
      "id": "digitalocean-token",
      "type": "promptString",
      "description": "DigitalOcean API Token",
      "password": true
    }
  ],
  "servers": {
    "mcpDigitalOcean": {
      "command": "npx",
      "args": [
        "@digitalocean/mcp",
        "--services",
        "apps"
      ],
      "env": {
        "DIGITALOCEAN_API_TOKEN": "${input:digitalocean-token}"
      }
    }
  }
}
```

Or, to use a system environment variable directly:

```json
{
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
        "DIGITALOCEAN_API_TOKEN": "${env:DIGITALOCEAN_API_TOKEN}"
      }
    }
  }
}
```

##### Verify Installation

1. Open VS Code and open Command Palette (`Shift + ⌘ + P` on Mac or `Ctrl + Shift + P` on Windows/Linux)
2. Search for "MCP" in the command palette search bar
3. Select "MCP: List Servers"
4. Verify that "mcpDigitalOcean" appears in the list of configured servers

##### Viewing Available Tools

To see what tools are available from the MCP server:
1. Open the Command Palette (`⌘+Shift+P` on Mac or `Ctrl+Shift+P` on Windows/Linux)
2. Select "Agent" mode in the chatbox
3. Click "Configure tools" on the right, and check for DigitalOcean related tools under `MCP Server: mcpDigitalocean`. You should be able to list available tools like `app-create`, `app-list`, `app-delete`, etc.

##### Debugging

To troubleshoot MCP connections:
1. Open the Command Palette (`⌘+Shift+P` on Mac or `Ctrl+Shift+P` on Windows/Linux)
2. Type "Developer: Toggle Developer Tools" and press Enter
3. Navigate to the Console tab to view MCP server logs
4. Check for connection status and error messages

If you are getting a 401 error or authentication related errors, verify that the `DIGITALOCEAN_API_TOKEN` variable is set by running `echo $DIGITALOCEAN_API_TOKEN` in your terminal before launching VS Code.

---

## Step 3: Protect Your Token — Checklist

Before pushing any code to GitHub, verify the following:

- [ ] Your `.env` file (if used) is listed in `.gitignore`
- [ ] No config file contains a raw token string like `dop_v1_...`
- [ ] Config files use `${DIGITALOCEAN_API_TOKEN}`, `${env:DIGITALOCEAN_API_TOKEN}`, or `${input:...}` syntax
- [ ] You have run `git diff` or `git status` to confirm no secrets are staged
- [ ] You have never committed a token — if you have, [rotate it immediately](https://cloud.digitalocean.com/account/api/tokens) and revoke the old one

> **If your token was already committed:** Go to the [DigitalOcean API Tokens page](https://cloud.digitalocean.com/account/api/tokens), delete the compromised token, and generate a new one. GitHub's secret scanning will detect and flag exposed tokens automatically.

---

## Prerequisites for Local Installation

If you're using the local installation method (not Remote MCP), you'll need:

- Node.js (v18 or later)
- NPM (v8 or later)

You can find installation guides at [https://nodejs.org/en/download](https://nodejs.org/en/download)

Verify your installation:
```bash
node --version
npm --version
```

### Quick Test

To verify the local MCP server works correctly, you can test it directly from the command line (make sure `DIGITALOCEAN_API_TOKEN` is set in your environment first):
```bash
npx @digitalocean/mcp --services apps
```

---

## Configuration

### Local Installation Configuration

When using the local installation, you use the `--services` flag to specify which service you want to enable. It is highly recommended to only enable the services you need to reduce context size and improve accuracy. See list of supported services below.

```bash
npx @digitalocean/mcp --services apps,droplets
```

## Documentation

Each service provides a detailed README describing all available tools, resources, arguments, and example queries. See the following files for full documentation:

- [Apps Service](pkg/registry/apps/README.md)
- [Droplet Service](pkg/registry/droplet/README.md)
- [Account Service](pkg/registry/account/README.md)
- [Networking Service](pkg/registry/networking/README.md)
- [Databases Service](pkg/registry/dbaas/README.md)
- [Insights Service](pkg/registry/insights/README.md)
- [Spaces Service](pkg/registry/spaces/README.md)
- [Marketplace Service](pkg/registry/marketplace/README.md)
- [DOKS Service](pkg/registry/doks/README.md)
- [DOCR Service](pkg/registry/docr/README.md)

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