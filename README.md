# OMDR CLI

> ⚠️ **Under Active Development** - not ready for use yet. Core functionality is still being built.

Official command-line interface for the [Open MCP Directory](https://openmcpdirectory.com) - discover, install, and manage MCP servers.

Visit [openmcpdirectory.com](https://openmcpdirectory.com) to explore the directory.

## What Does It Do?

The OMDR CLI simplifies MCP server management:

- **Discovery**: Search the registry for MCP servers
- **Installation**: Auto-configure Claude Desktop, Cursor, VS Code
- **Local Servers**: Download and run servers as local subprocesses (100% free)
- **Hosted Servers**: Proxy to OMDR-hosted servers with usage-based billing
- **Publishing**: Publish your own MCP servers to the registry
- **Monetization**: Set pricing and receive payouts via Stripe Connect

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
# Login to OMDR
omdr auth login

# Search for MCP servers
omdr search "stripe payments"

# Install a free local server (auto-detects Claude, Cursor, VS Code)
omdr install @stripe/payments

# Install a paid hosted server
omdr install --hosted @creator/premium-tool

# Install to specific client
omdr install @stripe/payments --client vscode

# Install using custom config path
omdr install @stripe/payments --config-path ~/.config/Code/User/mcp.json

# List installed servers
omdr list

# Check your environment
omdr doctor
```

## Commands

### Authentication
```bash
omdr auth login          # Login with browser OAuth
omdr auth logout         # Clear stored credentials
omdr auth status         # Check authentication status
```

### Discovery & Installation
```bash
omdr search <query>                    # Search registry
omdr install <package>                 # Install local server
omdr install --hosted <package>        # Install hosted server
omdr install --client <type> <package> # Target specific client
omdr list                              # List installed servers
omdr uninstall <package>               # Remove server
```

### Billing (Hosted Servers)
```bash
omdr subscribe <tier>           # Subscribe to Pro/Enterprise
omdr credits buy <amount>       # Purchase credits ($1 = 100 credits)
omdr credits balance            # Check credit balance
omdr usage history              # View usage history
omdr invoices list              # List invoices
```

### Publishing (Creators)
```bash
omdr publish                                    # Publish local server (free)
omdr publish --deployment hosted_omdr \
  --artifact ./server.wasm \
  --pricing per_call --price-per-call 10        # Publish hosted WASM ($0.10/call)
omdr publish --github https://github.com/user/repo  # Publish from GitHub repo
omdr publish --self-hosted https://api.example.com  # Publish self-hosted endpoint
omdr pricing set <package>                      # Update pricing model
omdr payouts setup                              # Complete Stripe Connect onboarding
omdr earnings                                   # View earnings and payouts
```

### Utilities
```bash
omdr doctor                     # Check environment and dependencies
omdr version                    # Show CLI version
omdr help                       # Show help
```

## Supported Clients

OMDR automatically detects and configures:

- **Claude Desktop** - `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS)
- **Cursor** - `~/.cursor/mcp.json`
- **VS Code** - `~/.config/Code/User/mcp.json` (Linux/macOS), `%APPDATA%\Code\User\mcp.json` (Windows)
- **Zed** - `~/.config/zed/mcp.json`

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
│   ├── /cmd               # Cobra commands (install, search, auth, etc.)
│   ├── /client            # HTTP client for registry API
│   ├── /config            # Config file management
│   ├── /detector          # MCP client detection
│   ├── /installer         # Config patching logic
│   ├── /proxy             # MCP proxy server (stdio ↔ HTTP)
│   │   ├── server.go      # JSON-RPC stdio server
│   │   ├── guard_client.go # HTTP client for omdr-guard
│   │   └── protocol.go    # JSON-RPC 2.0 types
│   ├── /runtime           # Runtime requirement checks
│   └── /logger            # Logging utilities
├── /pkg/mcp-spec          # MCP manifest types (shared with main repo)
├── /distribution
│   ├── /installers        # Shell/PowerShell install scripts
│   └── /packages          # Homebrew/Scoop package definitions
└── /internal/entity       # Domain entities (Server, Version, etc.)
```

## For Creators

### Publishing Your MCP Server

#### Option 1: Local Server (Free)

1. Create `mcp.json` manifest:
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
  "resources": [...],
  "prompts": [...]
}
```

2. Publish to registry:
```bash
omdr publish
```

Users install locally (free):
```bash
omdr install @yourname/my-tool
```

#### Option 2: OMDR-Hosted Server (Paid) 🚧 Beta

**Upload WASM artifact:**
```bash
omdr publish --deployment hosted_omdr \
  --artifact ./server.wasm \
  --pricing per_call --price-per-call 10
```

**Build from GitHub:**
```bash
omdr publish --deployment hosted_omdr \
  --github https://github.com/yourname/mcp-server \
  --pricing per_call --price-per-call 10
```

**Supported artifact types:**
- `.wasm` - WebAssembly modules
- `.tar`, `.tar.gz` - Docker images (coming soon)

Users install hosted version:
```bash
omdr install --hosted @yourname/my-tool
```

#### Option 3: Self-Hosted Server (Paid)

Host on your infrastructure, use OMDR for billing:

```bash
omdr publish --deployment self_hosted \
  --self-hosted https://api.yourserver.com \
  --pricing per_call --price-per-call 10
```

OMDR forwards requests to your endpoint after billing.

### Deployment Model Comparison

| Model | Hosting | Build | Pricing | Revenue Split |
|-------|---------|-------|---------|---------------|
| **Local** | User's machine | N/A | Free | N/A |
| **OMDR-Hosted** | OMDR infrastructure | OMDR builds from GitHub or uploaded artifact | Per-call or subscription | Creator 90%, OMDR 10% |
| **Self-Hosted** | Your infrastructure | You manage | Per-call or subscription | Creator 95%, OMDR 5% |
| **Enterprise** | Enterprise infrastructure | Custom | Negotiated | Custom |

### Tier Requirements

- **Free tier**: Can only publish local servers
- **Pro tier ($20/mo)**: Can publish local, OMDR-hosted, and self-hosted
- **Enterprise**: All deployment models including private hosting

### Monetization Options

- **Free**: No pricing, users install locally
- **Per-call**: Charge per tool invocation (e.g., $0.10/call)
- **Subscription**: Monthly fee for unlimited access (coming soon)
- **Hybrid**: Base subscription + per-call overage (coming soon)

### Complete Stripe Connect Onboarding

To receive payouts:
```bash
omdr payouts setup
```

View earnings:
```bash
omdr earnings
```

Payouts are processed weekly with a $10 minimum threshold.

## Troubleshooting

### "No MCP clients detected"
Install Claude Desktop, Cursor, or VS Code with MCP extension, or use `--config-path` to specify a custom config file.

### "Authentication required"
Run `omdr auth login` to authenticate with OMDR.

### "Insufficient credits"
For hosted servers, purchase credits: `omdr credits buy 50` ($50 = 5000 credits)

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
