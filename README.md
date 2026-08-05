# WireGuard Admin

A cross-platform desktop application to manage multiple WireGuard VPN servers over SSH. Built with Wails 3 (Go + Svelte 5), it provides a full graphical interface for WireGuard interface lifecycle, peer management, and live status monitoring — no agent required on the server.

## Features

- **Multi-server management** — Add, edit, and delete SSH server profiles with password or private key authentication
- **Full interface lifecycle** — Create and delete WireGuard interfaces with automatic key generation and `wg-quick` bring-up
- **Peer management** — Add and remove peers with auto-generated keypairs, preshared keys, allowed IPs, endpoints, and persistent keepalive
- **Live status** — View interface and peer stats including latest handshake, transfer (RX/TX bytes), and endpoints
- **Config viewer** — Read WireGuard config files directly from the server
- **Distro-aware operations** — Automatically detects the server's Linux distribution and uses the correct package manager, service manager, and privilege escalation method
- **Install WireGuard** — One-click install of WireGuard on servers that don't have it yet
- **Jump server support** — Connect to servers through a bastion/jump host
- **Dark/light theme** — Toggle between dark and light UI themes

## Supported Linux Distributions

| Distro | Family | Package Manager | Init System | Privilege |
|--------|--------|----------------|-------------|-----------|
| Ubuntu | Debian | `apt` | systemd | `sudo` |
| Debian | Debian | `apt` | systemd | `sudo` |
| Fedora | RHEL | `dnf` | systemd | `sudo` |
| RHEL / Rocky / AlmaLinux | RHEL | `dnf`/`yum` | systemd | `sudo` |
| openSUSE | SUSE | `zypper` | systemd | `sudo` |
| Alpine | Alpine | `apk` | OpenRC | root |

The app auto-detects the distro on first connection via `/etc/os-release`. You can also manually select the distro when adding a server. See [docs/supported-distros.md](docs/supported-distros.md) for details.

## Prerequisites

- [Wails 3 CLI](https://v3.wails.io/) installed
- Go 1.21+
- Node.js 18+
- Linux: GTK4 + `webkitgtk-6.0` development headers
- WireGuard installed on target servers (or use the in-app install button)

## Getting Started

### Development

```bash
CGO_ENABLED=1 wails3 dev
```

This starts the app with hot-reloading for both frontend and backend.

### Build

```bash
CGO_ENABLED=1 wails3 build
```

The production binary is output to the `build/` directory.

## Configuration

Server profiles are stored as YAML at:

```
~/.config/wireguard-admin/servers.yaml
```

Each profile includes: name, host, port, username, auth method (password/key), optional passphrase, jump server reference, and optional distro override.

## Architecture

```
WireguardAdmin/
├── main.go                          # Entry point, Wails app + service registration
├── internal/
│   ├── models/models.go             # Data structures (ServerConfig, WGInterface, WGPeer, etc.)
│   ├── config/store.go              # YAML config load/save (~/.config/wireguard-admin/)
│   ├── ssh/client.go                # SSH connection manager (password/key auth, jump server)
│   ├── server/service.go            # ServerService — CRUD for server profiles, SSH session pooling
│   └── wireguard/
│       ├── service.go               # WireGuardService — WG operations via SSH
│       ├── distro.go                # Distro interface (strategy pattern)
│       ├── detect.go                # Auto-detection via /etc/os-release
│       └── distros/                 # Concrete distro implementations
│           ├── common.go            # Shared systemd + OpenRC base structs
│           ├── ubuntu.go            # Ubuntu/Debian
│           ├── fedora.go            # Fedora/RHEL
│           ├── opensuse.go          # openSUSE
│           └── alpine.go            # Alpine (OpenRC)
├── frontend/
│   ├── src/
│   │   ├── App.svelte               # Root layout: sidebar + main panel
│   │   ├── main.ts                  # App bootstrap
│   │   └── lib/
│   │       ├── components/          # UI components (Sidebar, Dashboard, modals, etc.)
│   │       ├── stores/servers.ts    # Svelte store wrapping Wails bindings
│   │       └── utils.ts             # Helpers
│   └── ...
├── docs/
│   ├── plan/architecture.md         # Architecture overview
│   └── supported-distros.md         # Distro support reference
└── build/                           # Build output
```

### Tech Stack

- **Wails 3** — Desktop app framework (Go backend + webview frontend)
- **Svelte 5 + TypeScript + Vite** — Frontend
- **TailwindCSS** — Styling
- **shadcn-svelte** — UI components
- **lucide-svelte** — Icons
- **golang.org/x/crypto/ssh** — SSH client
- **adrg/xdg** — Cross-platform config paths
- **gopkg.in/yaml.v3** — YAML config encoding

For the full architecture document, see [docs/plan/architecture.md](docs/plan/architecture.md).
