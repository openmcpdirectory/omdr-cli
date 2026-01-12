# @omdr/cli

Official npm wrapper for the OMDR CLI - the package manager for MCP (Model Context Protocol) servers.

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

Full documentation available at [github.com/openmcpdirectory/omdr-cli](https://github.com/openmcpdirectory/omdr-cli)

## License

MIT
