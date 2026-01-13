# @omdr/cli

Official command-line interface for the [Open MCP Directory](https://openmcpdirectory.com) - discover, install, and manage MCP servers.

Visit [openmcpdirectory.com](https://openmcpdirectory.com) to explore the directory.

## Installation

```bash
npm install -g @omdr/cli
```

## Usage

```bash
# Login to OMDR
omdr auth login

# Search for MCP servers
omdr search "stripe payments"

# Install a server
omdr install @stripe/payments

# Check your environment
omdr doctor
```

## Documentation

Full documentation available at [omdr.dev](https://docs.omdr.dev)

## License

MIT
