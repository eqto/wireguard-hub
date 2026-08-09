# WireguardHub — Architecture

A cross-platform desktop app built with Wails 3 (Go + Svelte 5) to manage WireGuard on remote servers over SSH or on the local machine directly. Supports full interface lifecycle, peer management, systemd service integration, a live terminal panel, and dark/light themes.

## Tech Stack

- **Wails 3** — desktop app framework (Go backend + webview frontend)
- **Svelte 5 + TypeScript + Vite** — frontend
- **Bun** — frontend package manager / runtime
- **Custom SCSS + CSS variables** — styling (dark/light theme via CSS custom properties)
- **@lucide/svelte** — icons
- **golang.org/x/crypto/ssh** — SSH client in Go
- **adrg/xdg** — cross-platform config file paths
- **google/uuid** — server profile IDs
- **gopkg.in/yaml.v3** — YAML encoding/decoding

## Project Structure

```
WireguardHub/
├── main.go                              # Entry point, register services, window config
├── internal/
│   ├── models/models.go                 # Data structures (ServerConfig, WGInterface, WGPeer, …)
│   ├── config/store.go                  # YAML config store (servers.yaml + local.yaml)
│   ├── ssh/
│   │   ├── client.go                    # SSH connection manager (password/key auth, jump host)
│   │   └── executor.go                  # Executor interface (shared by SSH + local clients)
│   ├── local/client.go                  # Local os/exec client — implements Executor
│   ├── server/service.go                # ServerService — CRUD, session pooling, local config
│   └── wireguard/
│       ├── service.go                   # Status, install, parsing, server-info caching
│       ├── interface.go                 # Interface lifecycle + service management
│       └── peer.go                      # Peer add/remove/update + client config generation
├── frontend/
│   ├── src/
│   │   ├── main.ts                      # App bootstrap
│   │   ├── App.svelte                   # Root layout: sidebar + main panel + modals + terminal
│   │   ├── app.scss                     # Global styles (custom SCSS + CSS variables)
│   │   ├── lib/
│   │   │   ├── components/
│   │   │   │   ├── Sidebar.svelte         # Server list with status dots
│   │   │   │   ├── ServerGrid.svelte      # Empty-state server grid
│   │   │   │   ├── ServerDashboard.svelte # Main panel: WG status for selected server
│   │   │   │   ├── PeerTable.svelte       # Peer list with actions
│   │   │   │   ├── AddServerModal.svelte  # Add/edit server form
│   │   │   │   ├── AddPeerModal.svelte    # Add peer form (keygen or manual key)
│   │   │   │   ├── EditPeerModal.svelte   # Edit peer metadata/endpoint/allowed IPs
│   │   │   │   ├── InterfaceModal.svelte  # Create/edit WG interface
│   │   │   │   ├── ConfigViewer.svelte    # View config file content
│   │   │   │   ├── ConfirmDialog.svelte   # Generic confirmation dialog
│   │   │   │   ├── LocalSetupModal.svelte # Local-mode sudo credential setup
│   │   │   │   ├── Terminal.svelte        # Live command/output panel
│   │   │   │   ├── Toaster.svelte         # Toast notification renderer
│   │   │   │   ├── StatusBadge.svelte     # Connection status dot
│   │   │   │   └── ThemeToggle.svelte     # Dark/light toggle
│   │   │   ├── stores/
│   │   │   │   ├── servers.ts             # Server list, selection, theme state
│   │   │   │   ├── terminal.ts            # Per-server terminal entries
│   │   │   │   └── toast.ts              # Toast notification queue
│   │   │   └── utils.ts                   # cn() + unwrapResponse() helpers
│   │   └── bindings/                      # Auto-generated Wails Go→TS bindings
│   └── ...
├── docs/plan/
│   └── architecture.md                   # This document
└── build/                                # Wails build config, platform Taskfiles, Dockerfiles
```

## Go Backend Services

### Config Store (`internal/config/store.go`)
- Load/save server profiles to `~/.config/wireguardhub/servers.yaml`
- Load/save local-mode credentials to `~/.config/wireguardhub/local.yaml`
- Plaintext YAML for v1 (credentials included); both files written with `0600` permissions
- Uses `gopkg.in/yaml.v3` for encoding/decoding
- Uses `adrg/xdg` for cross-platform config path

### Executor Abstraction (`internal/ssh/executor.go`)
The `Executor` interface abstracts command execution so `wireguard.Service` works against either a remote SSH server or the local machine:

```go
type Executor interface {
    Exec(cmd string) (string, string, error)
    ExecWithInput(cmd, input string) (string, string, error)
    ExecStreaming(cmd string, onLine func(string)) (io.Closer, error)
    ExecSilent(cmd string) (string, string, error)            // no terminal events (sensitive data)
    ExecWithInputSilent(cmd, input string) (string, string, error)
    ExecF(format string, args ...any) (string, string, error) // printf-style wrappers
    ExecSilentF(format string, args ...any) (string, string, error)
    ExecWithInputF(input, format string, args ...any) (string, string, error)
    ExecWithInputSilentF(input, format string, args ...any) (string, string, error)
    CommandExists(name string) bool                           // `command -v <name>`
    IsConnected() bool
    Close() error
}
```

Two implementations:
- **`ssh.Client`** (`internal/ssh/client.go`) — connects via `golang.org/x/crypto/ssh`, supports password/private-key auth (with optional passphrase), jump hosts, sudo via `sudo -S` (password piped to stdin), and session pooling per server ID.
- **`local.Client`** (`internal/local/client.go`) — runs commands locally via `os/exec` (`bash -c`), handles sudo the same way (`sudo -S` with password via stdin). Used for local mode (server ID `"local"`).

Both emit `ssh.ExecEvent`s (`command`/`output`/`done`) that the backend forwards to the frontend via the Wails `ssh-terminal` event, driving the Terminal panel. `ExecSilent*` variants skip event emission and are used for commands that touch private keys or config files.

### ServerService (`internal/server/service.go`)
Exposed to frontend via Wails bindings:

| Method | Returns | Description |
|--------|---------|-------------|
| `GetServers()` | `[]ServerConfig` | List all profiles plus a synthetic `"local"` entry |
| `AddServer(config)` | `ServerConfig` | Add a new server profile (validates jump-host refs) |
| `UpdateServer(config)` | `ServerConfig` | Update an existing profile (drops cached SSH session) |
| `DeleteServer(id)` | `bool` | Delete a profile (refuses if used as a jump host) |
| `TestConnection(config)` | `TestConnectionResult` | SSH connect (or local `wg --version` + sudo check) |
| `GetLocalConfig()` | `LocalConfig` | Read local-mode config (password redacted) |
| `SaveLocalConfig(cfg)` | `bool` | Persist local-mode credentials to `local.yaml` |
| `SetLocalSessionCredentials(u, p)` | `bool` | Set session-only local credentials (not persisted) |
| `ClearLocalSessionCredentials()` | `bool` | Drop session credentials, reload from disk |
| `GetClient(serverID)` | `ssh.Executor` | Cached SSH client, or `local.Client` for `"local"` |

`GetClient` caches active SSH sessions per server ID and auto-reconnects if a session drops. Jump hosts are resolved via `ViaServerID` (single hop only — a jump host cannot itself use a jump host).

### WireGuardService (`internal/wireguard/`)
Split across three files but exposed as a single `Service` via Wails bindings:

**`service.go`** — status, install, parsing:

| Method | Returns | Description |
|--------|---------|-------------|
| `GetStatus(serverID)` | `WGStatus` | List `.conf` files, parse each, merge live `wg show all dump` stats; detects `wg` presence and sudo access; caches server info (hostname, OS, IP, package manager, HasSystemd) |
| `InstallWireGuard(serverID)` | `bool` | Streams `apt-get`/`dnf`/`yum`/`pacman` install (with dpkg-lock retry for apt); emits `wg-install-done` event |
| `CancelInstall()` | `bool` | Closes the running install session |

**`interface.go`** — interface lifecycle:

| Method | Returns | Description |
|--------|---------|-------------|
| `GenerateKeyPair(serverID)` | `KeyPair` | `wg genkey` + `wg pubkey` |
| `CreateInterface(req)` | `WGInterface` | Generate keys, write `/etc/wireguard/<name>.conf`, bring up via `wg-quick` or `systemctl enable --now wg-quick@<name>` |
| `BringUpInterface(serverID, name)` | `bool` | Start via systemctl (if service enabled) or `wg-quick up` |
| `BringDownInterface(serverID, name)` | `bool` | Stop via systemctl (if service enabled) or `wg-quick down` |
| `RestartInterface(serverID, name)` | `bool` | stop+start via systemctl or `wg-quick down && wg-quick up` |
| `DeleteInterface(serverID, name)` | `bool` | `systemctl disable --stop` (if enabled) or `wg-quick down`, then `rm -f` the config |
| `EnableService(serverID, name)` | `bool` | Enable `wg-quick@<name>` for boot; atomically transitions a `wg-quick up` interface to systemd management |
| `DisableService(serverID, name)` | `bool` | Disable `wg-quick@<name>` (does not change current run state) |
| `GetInterfaceConfig(serverID, name)` | `string` | Read `/etc/wireguard/<name>.conf` (silent — no terminal output) |
| `SyncConfig(serverID, name)` | `bool` | `wg-quick strip | wg syncconf … /dev/stdin` |

**`peer.go`** — peer operations:

| Method | Returns | Description |
|--------|---------|-------------|
| `AddPeer(req)` | `AddPeerResult` | `wg set … peer …`, sync, append `[Peer]` section to config; detects client interfaces (no `ListenPort`) and constrains to one server peer; returns generated client config |
| `RemovePeer(serverID, iface, pubKey)` | `bool` | `wg set … peer … remove`, sync, strip `[Peer]` section from config |
| `UpdatePeerMeta(req)` | `UpdatePeerMetaRequest` | Update peer name/description/endpoint/allowedIPs in config; optionally apply live via `wg set`; optional interface restart |

> Note: several `sudo` command chains are wrapped in a single `sudo bash -c '…'` so the sudo password (fed via stdin with `-S`) is only needed once — chaining two separate `sudo` calls with `&&` would leave the second without `-S` and no TTY, failing for non-root users.

## Runtime Distro Detection

There is no per-distro code or config. Instead the backend probes the host at runtime:

- **Package manager** — `client.CommandExists("apt-get"|"dnf"|"yum"|"pacman")` (first match wins). Used by `InstallWireGuard` and reported in `WGStatus.PackageManager`.
- **Init system** — `hasSystemd(client)` checks `command -v systemctl` **and** `test -d /run/systemd/system`. When true, `wg-quick@<iface>` service management (enable/disable/start/stop/restart) is available; otherwise only direct `wg-quick up/down` is used.
- **Server info** — `hostname`, `hostname -I`, and `PRETTY_NAME` from `/etc/os-release` are read once and cached per server ID in `serverInfoCache`.

See [docs/supported-distros.md](../supported-distros.md) for the supported package-manager/init combinations.

## Frontend UI

### Layout
- **Sidebar** (left): Server list with status dots (green=connected, red=offline, gray=untested), including a permanent "Local" entry. "Add Server" button and theme toggle at the bottom.
- **Main panel** (right): `ServerGrid` empty state, or `ServerDashboard` for the selected server.
- **Terminal panel** (bottom, collapsible): Live per-server command/output stream driven by `ssh-terminal` Wails events.

### Server Dashboard
- **Header**: Server name, host:port, connection status badge, refresh, edit/delete actions.
- **Interface cards**: Name, public key, listen port, RX/TX, online/offline dot, and service-enabled state. Actions: Add Peer, View Config, Sync Config, Start/Stop/Restart, Enable/Disable service, Delete.
- **Peer table**: Public key (truncated), name/description, endpoint, allowed IPs (chips), latest handshake (relative time), transfer stats, edit/remove actions.

### Modals
1. **Add/Edit Server Modal**: Name, host, port, username, auth method (password/key), password or private key, passphrase (optional), jump host selector. "Test Connection" before saving.
2. **Add Peer Modal**: Interface selector, public key (auto-generate or paste), allowed IPs (chip input), preshared key, endpoint, persistent keepalive, name/description. Shows generated client config after success.
3. **Edit Peer Modal**: Edit name/description/endpoint/allowed IPs, with optional interface restart.
4. **Create Interface Modal**: Interface name, listen port, private key (auto or paste), endpoint, address, default allowed IPs, "enable as service" toggle.
5. **Config Viewer Modal**: Read-only display of `/etc/wireguard/<name>.conf` with copy button.
6. **Local Setup Modal**: Username + sudo password for local-mode access (persisted to `local.yaml` or session-only).
7. **Confirm Dialog**: Generic confirmation (e.g. restart/delete).

### Theme
- Dark/light toggle stored in `localStorage` (`wg-admin-theme`).
- Implemented via CSS custom properties defined under `:root` (light) and `.dark` (dark) in `app.scss`; toggling adds/removes the `dark` class on `<html>`.
- Default: dark theme.

## Data Models

```go
type ServerConfig struct {
    ID, Name, Host, Username, AuthMethod string
    Port                                 int
    Password, PrivateKey, Passphrase     string // omitted from JSON if empty
    ViaServerID                          string // optional jump host (single hop)
    IsLocal                              bool   // synthetic "local" entry
}

type LocalConfig struct {
    Username   string
    Password   string // omitted from JSON if empty
    Configured bool   // derived, not serialized
}

type WGInterface struct {
    Name, PublicKey, PrivateKey, Endpoint string
    ListenPort                            int
    RxBytes, TxBytes                      int64
    Peers                                 []WGPeer
    Online                                bool
    ServiceEnabled                        bool // wg-quick@<name> enabled in systemd
}

type WGPeer struct {
    PublicKey, PresharedKey, Endpoint string
    AllowedIPs                        []string
    LatestHandshake                   time.Time
    RxBytes, TxBytes                  int64
    PersistentKeepalive               int
    Name, Description                 string // stored as # Name / # Description comments
}

type WGStatus struct {
    Interfaces     []WGInterface
    Hostname, ServerIP, OS, PackageManager string
    WGNotInstalled bool
    HasSystemd     bool
}
```

## Config Storage

- `~/.config/wireguardhub/servers.yaml` — YAML list of `ServerConfig` (remote profiles only; the `"local"` entry is synthetic and never persisted).
- `~/.config/wireguardhub/local.yaml` — `LocalConfig` (sudo credentials for local mode).
- Both files are plaintext for v1 (encryption can be added later) and written with `0600` permissions.

## SSH / Local Connection Flow

1. User selects a server in the sidebar (or the "Local" entry).
2. Frontend calls `WireGuardService.GetStatus(serverID)`.
3. `ServerService.GetClient(serverID)` returns a cached `ssh.Client` (reconnecting if dropped) or the `local.Client` for `"local"`.
4. `GetStatus` checks `wg` presence and sudo access, runs `wg show all dump` (silent), lists `/etc/wireguard/*.conf`, parses each, and merges live peer stats.
5. Every command (except `ExecSilent*`) emits `ssh-terminal` events that the Terminal panel renders live.
6. The SSH session stays cached for subsequent operations on the same server.

## Server Mode (Headless)

In addition to the desktop GUI, the app can be built as a pure HTTP server (`-tags server`) with no native GUI dependencies, suitable for headless/Docker deployment. Build/run tasks are defined in `Taskfile.yml` and `build/Taskfile.yml`:

- `task build:server` — build `bin/wireguardhub-server`
- `task run:server` — build + run a dev server
- `task build:docker` — build a minimal Docker image (`build/docker/Dockerfile.server`)
- `task run:docker` — build + run the Docker image (port 8080)

Cross-compilation to darwin/linux/windows (amd64/arm64) is supported via `build/docker/Dockerfile.cross` (`task setup:docker`).
