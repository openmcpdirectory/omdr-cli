# OMDR CLI

Official command-line interface for the [Open MCP Directory](https://openmcpdirectory.com) - discover, install, and manage MCP servers.

Visit [openmcpdirectory.com](https://openmcpdirectory.com) to explore the directory.

## What Does It Do?

The OMDR CLI simplifies MCP server management:

- **Discovery**: Search the registry for MCP servers
- **Installation**: Auto-configure Claude Desktop, Cursor, VS Code, Windsurf, Zed, Cline, Claude Code, and Codex
- **Local Servers**: Download and run servers as local subprocesses (100% free)
- **Hosted Servers**: Proxy to OMDR-hosted servers with usage-based billing
- **Publishing**: Publish your own MCP servers to the registry
- **Monetization**: Set pricing and receive payouts via Dodo Payments

## Deployment Models

### Local MCPs (Free)
Servers run as local subprocesses with direct stdio communication:

```bash
omdr install @namespace/server
```

**What happens:**
1. CLI downloads server manifest from registry
2. Checks runtime requirements (Node.js, Python, Docker)
3. Patches MCP client config with local command
4. Client launches server as subprocess

**Config example:**
```json
{
  "mcpServers": {
    "namespace/server": {
      "command": "node",
      "args": ["/path/to/server/index.js"],
      "env": {}
    }
  }
}
```

### Hosted MCPs (Paid)
Servers run on OMDR infrastructure with proxy-based billing:

```bash
omdr install --hosted @creator/premium-tool
```

**What happens:**
1. CLI configures local proxy command
2. Client launches `omdr proxy @creator/premium-tool`
3. Proxy forwards JSON-RPC to omdr-guard (cloud)
4. Guard validates API key, deducts credits
5. Guard forwards to omdr-runtime (cloud)
6. Runtime executes MCP server and returns response

**Config example:**
```json
{
  "mcpServers": {
    "creator-premium-tool": {
      "command": "/usr/local/bin/omdr",
      "args": ["proxy", "@creator/premium-tool"],
      "env": {
        "OMDR_API_KEY": "your_api_key_here"
      }
    }
  }
}
```

**Flow:**
```
Claude Desktop → omdr proxy (local) → omdr-guard (cloud) → omdr-runtime (cloud) → MCP Server
```

## Installation

### Homebrew (macOS/Linux)
```bash
brew install omdr/tap/omdr
```

### Scoop (Windows)
```bash
scoop bucket add omdr https://github.com/openmcpdirectory/scoop-bucket
scoop install omdr
```

### npm
```bash
npm install -g @omdr/cli
```

### Cargo
```bash
cargo install omdr-cli
```

### Go
```bash
go install github.com/openmcpdirectory/omdr-cli/cmd/omdr@latest
```

### Shell Script (Unix)
```bash
curl -fsSL https://raw.githubusercontent.com/openmcpdirectory/omdr-cli/main/distribution/installers/install.sh | sh
```

### PowerShell (Windows)
```powershell
irm https://raw.githubusercontent.com/openmcpdirectory/omdr-cli/main/distribution/installers/install.ps1 | iex
```

## Quick Start

```bash
# Login to OMDR (browser)
omdr auth login

# Login without a browser (device flow)
omdr auth login --device-flow

# Search for MCP servers
omdr search "stripe payments"

# Install a free local server (auto-detects Claude, Cursor, VS Code, Windsurf, Zed, Cline, Codex...)
omdr install @stripe/payments

# Install a paid hosted server
omdr install --hosted @creator/premium-tool

# Install to a specific client
omdr install @stripe/payments --client windsurf

# Install using a custom config path
omdr install @stripe/payments --config-path ~/.config/Code/User/mcp.json

# List installed servers
omdr list

# Check for updates
omdr update

# Check your environment
omdr doctor
```

## Commands

### Authentication
```bash
omdr auth login                 # Login via browser OAuth
omdr auth login --device-flow   # Login without a browser (device flow)
omdr auth logout                # Clear stored credentials
omdr auth status                # Show current auth state
```

### Discovery & Installation
```bash
omdr search <query>                     # Search the registry
omdr install <package>                  # Install a local server
omdr install --hosted <package>         # Install a hosted server
omdr install --client <type> <package>  # Target a specific client
omdr list                               # List installed servers
omdr uninstall <package>                # Remove a server
omdr update [server]                    # Check for and apply updates
```

Supported `--client` values: `claude`, `cursor`, `vscode`, `windsurf`, `zed`, `cline`, `claude-code`, `codex`

### Billing
```bash
omdr credits           # Show balance and recent transactions
omdr usage             # Show current billing period usage
omdr invoices          # List billing invoices
omdr pricing           # Show subscription tiers and pricing
```

### Publishing (Creators)
```bash
omdr init                                       # Interactive wizard — creates omdr.json or omdr.toml
omdr validate                                   # Validate local manifest (auto-detects file)
omdr validate ./omdr.json                       # Validate a specific path
omdr publish                                    # Publish from local manifest (free/hosted/self-hosted)
omdr publish --dry-run                          # Validate + preview — no submission
omdr publish --github-token ghp_xxxxx          # Pass GitHub token for private repo builds
omdr publish --namespace @yourname             # Override namespace
omdr earnings summary                           # View earnings summary
omdr earnings payouts                           # View payout history
```

### Utilities
```bash
omdr doctor                     # Check environment and dependencies
omdr version                    # Show CLI version
omdr version --json             # Output version as JSON
omdr help                       # Show help

# Global flags available on all commands:
#   --json        Output in JSON format
#   --no-banner   Suppress the banner
#   -v, --verbose Verbose output
```

## Supported Clients

OMDR automatically detects and configures:

| Client | Config location |
|--------|-----------------|
| **Claude Desktop** | `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) |
| **Cursor** | `~/.cursor/mcp.json` |
| **VS Code** | `~/.config/Code/User/mcp.json` (Linux/macOS) · `%APPDATA%\Code\User\mcp.json` (Windows) |
| **Windsurf** | `~/.codeium/windsurf/mcp_config.json` |
| **Zed** | `~/.config/zed/settings.json` (Linux/macOS) · `%APPDATA%\Zed\settings.json` (Windows) |
| **Cline** | VS Code globalStorage `saoudrizwan.claude-dev/cline_mcp_settings.json` |
| **Claude Code** | Managed via `claude mcp add` CLI |
| **Codex** | `~/.codex/config.toml` |

Or use `--config-path` to specify any custom MCP config file.

## Configuration

### Environment Variables

- `OMDR_API_URL` - Override API endpoint (default: `https://api.omdr.dev`)
- `OMDR_GUARD_URL` - Override guard endpoint (default: `https://guard.omdr.dev`)
- `OMDR_API_KEY` - Set authentication token (used by proxy command)
- `OMDR_TIMEOUT` - HTTP timeout (e.g., `60s`)

### Config Files

- Global: `~/.omdr/config.yaml`
- Local: `omdr.yaml` (in current directory)

Priority: Environment variables > Local config > Global config > Defaults

Example `config.yaml`:
```yaml
api_url: https://api.omdr.dev
guard_url: https://guard.omdr.dev
auth:
  token: your_token_here
```

## How It Works

### Local Installation Flow
1. User runs `omdr install @namespace/server`
2. CLI fetches manifest from registry API
3. CLI checks runtime requirements (Node.js, Python, Docker)
4. CLI detects installed MCP clients
5. CLI patches client configs with server command
6. User restarts MCP client
7. Client launches server as subprocess

### Hosted Installation Flow
1. User runs `omdr install --hosted @creator/premium-tool`
2. CLI fetches manifest from registry API
3. CLI gets user's API key from config
4. CLI patches client configs with proxy command: `omdr proxy @creator/premium-tool`
5. User restarts MCP client
6. Client launches proxy subprocess
7. Proxy reads JSON-RPC from stdin
8. Proxy forwards to omdr-guard with API key
9. Guard validates, deducts credits, forwards to runtime
10. Runtime executes server, returns response
11. Proxy writes JSON-RPC to stdout
12. Client receives response

### Proxy Command (Internal)
The `omdr proxy` command is hidden and used internally by MCP clients:

```bash
omdr proxy @creator/premium-tool
```

This command:
- Reads JSON-RPC 2.0 requests from stdin
- Forwards to omdr-guard via HTTP POST
- Handles authentication (OMDR_API_KEY env var)
- Converts HTTP errors to JSON-RPC errors
- Writes JSON-RPC 2.0 responses to stdout

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    MCP Client (Claude/Cursor)               │
│                                                             │
│  Local Server:  subprocess → stdio                          │
│  Hosted Server: omdr proxy → stdin/stdout                   │
└────────────────────────┬────────────────────────────────────┘
                         │
                         │ (hosted only)
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                      omdr-guard (cloud)                     │
│  - Validate API key                                         │
│  - Check credit balance                                     │
│  - Deduct credits                                           │
│  - Rate limiting                                            │
│  - Forward to runtime                                       │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                     omdr-runtime (cloud)                    │
│  - Execute MCP server (Docker/WASM)                         │
│  - Isolation & sandboxing                                   │
│  - Return response                                          │
└─────────────────────────────────────────────────────────────┘
```

## Development

### Building from Source

```bash
# Clone the repository
git clone https://github.com/openmcpdirectory/omdr-cli.git
cd omdr-cli

# Build
make build

# Run tests
make test

# Install locally
make install
```

### Project Structure

```
/omdr-cli
├── /cmd/omdr              # Main CLI entrypoint
├── /internal/cli
│   ├── /cmd               # Cobra commands (auth, install, search, list, uninstall, update,
│   │                      #   init, validate, publish, credits, usage, invoices, pricing, earnings, doctor, version)
│   ├── /client            # HTTP client for registry API
│   ├── /config            # Config file management
│   ├── /detector          # MCP client detection (8 clients)
│   ├── /installer         # Config patching (JSON / TOML / subprocess)
│   ├── /proxy             # MCP proxy server (stdio ↔ HTTP)
│   │   ├── server.go      # JSON-RPC stdio server
│   │   ├── guard_client.go # HTTP client for omdr-guard
│   │   └── protocol.go    # JSON-RPC 2.0 types
│   ├── /registry          # Local server registry (~/.omdr/servers.yaml)
│   ├── /runtime           # Runtime requirement checks
│   ├── /ui                # Output helpers (Success, Error, Table, JSON)
│   └── /logger            # Logging utilities
├── /pkg/mcp-spec          # MCP/OMDR manifest types (omdr.json + omdr.toml loader, OMDR extension, validation)
├── /distribution
│   ├── /docker            # Dockerfile.cli
│   ├── /installers        # Shell/PowerShell install scripts
│   ├── /npm               # @omdr/cli npm wrapper
│   └── /cargo             # omdr-cli cargo wrapper
└── /internal/entity       # Domain entities (Server, Version, etc.)
```

### For Creators

### Publishing Your MCP Server

#### Step 1: Create your manifest

Run the interactive wizard to scaffold an `omdr.json` (or `omdr.toml`) in your project:

```bash
omdr init
```

Or create it manually. The manifest is a superset of the [standard MCP manifest](https://spec.modelcontextprotocol.io/) — standard fields sit at the root, OMDR-specific config lives under the `omdr` key:

```json
{
  "name": "my-tool",
  "version": "1.0.0",
  "description": "My awesome MCP server",
  "runtime": {
    "type": "node",
    "command": "node",
    "args": ["index.js"]
  },
  "tools": [...],
  "omdr": {
    "version": "1",
    "deployment": "hosted",
    "hosting": {
      "artifact_type": "docker",
      "github_url": "https://github.com/yourname/mcp-server"
    },
    "pricing": {
      "model": "per_call",
      "per_call_cents": 10
    },
    "secrets": [
      { "name": "API_KEY", "description": "Your API key", "required": true, "managed_by": "user" }
    ],
    "categories": ["ai", "productivity"]
  }
}
```

**TOML equivalent** (`omdr.toml` is also supported):

```toml
name = "my-tool"
version = "1.0.0"
description = "My awesome MCP server"

[runtime]
type = "node"
command = "node"
args = ["index.js"]

[omdr]
version = "1"
deployment = "hosted"

[omdr.hosting]
artifact_type = "docker"
github_url = "https://github.com/yourname/mcp-server"

[omdr.pricing]
model = "per_call"
per_call_cents = 10

[[omdr.secrets]]
name = "API_KEY"
description = "Your API key"
required = true
managed_by = "user"
```

**Manifest file resolution order:** `omdr.json` → `omdr.toml` → `mcp.json`

#### Step 2: Validate

```bash
omdr validate          # auto-detects omdr.json / omdr.toml / mcp.json
omdr validate ./omdr.json  # validate a specific file
```

#### Step 3: Publish

```bash
omdr publish           # reads manifest, submits to registry
omdr publish --dry-run # validate + preview without submitting
```

For private GitHub repos add a token:

```bash
omdr publish --github-token ghp_xxxxx
```

Users install locally (free) or hosted (paid) based on the manifest's `omdr.deployment` value.

#### Deployment Modes

| `omdr.deployment` | Hosting | Build | Required fields |
|---|---|---|---|
| `local` (default) | User's machine | None | none |
| `hosted` | OMDR infrastructure | OMDR builds from `hosting.github_url` | `hosting.artifact_type` + `hosting.github_url` or `hosting.dockerfile` |
| `self_hosted` | Your infrastructure | You manage | `hosting.endpoint_url` |

#### Artifact Types (`hosting.artifact_type`)

| Value | Description |
|---|---|
| `docker` | Docker image built from `dockerfile` + `github_url` |
| `wasm` | WebAssembly module |
| `npm` | NPM package |
| `python` | Python package |

#### Pricing Models (`omdr.pricing.model`)

| Value | Fields |
|---|---|
| `free` | none |
| `per_call` | `per_call_cents` (> 0) |
| `subscription` | `monthly_cents` (> 0) |

#### Receiving Payouts

Earnings are processed via Dodo Payments. Complete onboarding via the creator dashboard at [openmcpdirectory.com](https://openmcpdirectory.com).

View earnings from the CLI:
```bash
omdr earnings summary   # Balance and totals
omdr earnings payouts   # Payout history
```

Payouts are processed weekly with a $10 minimum threshold.

## Troubleshooting

### "No MCP clients detected"
Install one of the supported clients (Claude Desktop, Cursor, VS Code, Windsurf, Zed, Cline, Claude Code, or Codex), or use `--config-path` to specify a custom config file.

### "Authentication required"
Run `omdr auth login` to authenticate with OMDR.

### "Insufficient credits"
For hosted servers, purchase credits via the dashboard at [openmcpdirectory.com](https://openmcpdirectory.com) or run `omdr credits` to check your balance.

### "Runtime check failed"
Install required runtime (Node.js, Python, Docker) based on server requirements.

### Proxy connection issues
Check `OMDR_GUARD_URL` environment variable and ensure you're authenticated.

## Author

Created by [Asman Mirza](mailto:asman@omdr.dev) and the OMDR Team.

## Links

- Website: [openmcpdirectory.com](https://openmcpdirectory.com)
- Documentation: [docs.omdr.dev](https://docs.omdr.dev)
- GitHub: [github.com/openmcpdirectory/omdr-cli](https://github.com/openmcpdirectory/omdr-cli)

## License

MIT
