# Development

This document covers how to build, run, and package WireguardHub from source. It is intended for developers and contributors. If you just want to use the app, see the [README](../README.md).

## Prerequisites

- [Wails 3 CLI](https://v3.wails.io/) installed
- Go 1.25+
- [Bun](https://bun.sh/) (frontend package manager)
- Linux: GTK4 + `webkitgtk-6.0` development headers
- WireGuard installed on target servers (or use the in-app install button)

## Development

```bash
CGO_ENABLED=1 wails3 dev
```

This starts the app with hot-reloading for both frontend and backend.

## Build

```bash
CGO_ENABLED=1 wails3 build
```

The production binary is output to the `bin/` directory.

## Server Mode (No GUI)

WireguardHub can also run as a pure HTTP server without native GUI dependencies, useful for headless/Docker deployments:

```bash
task build:server      # build the server binary
task run:server        # build + run a dev server
task build:docker      # build a minimal Docker image
task run:docker        # build + run the Docker image (port 8080)
```

## Cross-Compilation

Cross-compile to darwin/linux/windows (amd64/arm64) via the bundled Docker image:

```bash
task setup:docker                    # build the wails-cross Docker image (~800MB download)
docker run --rm -v $(pwd):/app wails-cross linux amd64
```

See `build/docker/Dockerfile.cross` for the available targets.

## Configuration Files

| File | Purpose |
|------|---------|
| `~/.config/wireguardhub/servers.yaml` | Remote server profiles (name, host, port, credentials, jump host) |
| `~/.config/wireguardhub/local.yaml` | Local-mode sudo credentials for the host machine |

Both files are written with `0600` permissions. The `"local"` server entry shown in the app is synthetic and is never persisted to `servers.yaml`.

## Further Reading

- [Architecture & code flow](plan/architecture.md)
- [Supported distros & runtime detection](supported-distros.md)
- [Codebase recommendations](recommendations.md)
