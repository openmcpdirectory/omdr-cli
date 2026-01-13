# OMDR CLI

Official command-line interface for the [Open MCP Directory](https://openmcpdirectory.com) - discover, install, and manage MCP servers.

Visit [openmcpdirectory.com](https://openmcpdirectory.com) to explore the directory.

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

# Install a server (auto-detects Claude, Cursor, VS Code)
omdr install @stripe/payments

# Install to specific client
omdr install @stripe/payments --client vscode

# Install using custom config path
omdr install @stripe/payments --config-path ~/.config/Code/User/mcp.json

# List installed servers
omdr list

# Check your environment
omdr doctor
```

## Supported Clients

OMDR automatically detects and configures:

- **Claude Desktop** - `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Cursor** - `~/.cursor/mcp.json`
- **VS Code** - `~/.config/Code/User/mcp.json`

Or use `--config-path` to specify any custom MCP config file.

## Configuration

### Environment Variables

- `OMDR_API_URL` - Override API endpoint (default: `https://api.omdr.dev`)
- `OMDR_REGISTRY_URL` - Override registry endpoint (default: `https://registry.omdr.dev`)
- `OMDR_AUTH_TOKEN` - Set authentication token
- `OMDR_TIMEOUT` - HTTP timeout (e.g., `60s`)

### Config Files

- Global: `~/.omdr/config.yaml`
- Local: `omdr.yaml`

Priority: Environment variables > Local config > Global config > Defaults

## Author

Created by [Asman Mirza](mailto:asman@omdr.dev) and the OMDR Team.

## Links

- Website: [openmcpdirectory.com](https://openmcpdirectory.com)
- Documentation: [omdr.dev](https://docs.omdr.dev)
- GitHub: [github.com/openmcpdirectory/omdr-cli](https://github.com/openmcpdirectory/omdr-cli)

## License

MIT
