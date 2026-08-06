# WireguardHub — Desktop App Plan

A cross-platform desktop app built with Wails 3 (Go + Svelte) to manage multiple WireGuard servers over SSH, supporting full interface lifecycle, peer management, and dark/light themes.

## Tech Stack

- **Wails 3** — desktop app framework (Go backend + webview frontend)
- **Svelte 5 + TypeScript + Vite** — frontend
- **TailwindCSS** — styling
- **lucide-svelte** — icons
- **shadcn-svelte** — UI component library
- **golang.org/x/crypto/ssh** — SSH client in Go
- **adrg/xdg** (already a dep) — config file paths
- **gopkg.in/yaml.v3** — YAML encoding/decoding

## Project Structure

```
WireguardHub/
├── main.go                              # Entry point, register services, window config
├── internal/
│   ├── models/models.go                 # Data structures
│   ├── ssh/client.go                    # SSH connection manager
│   ├── server/service.go                # ServerService — CRUD for server profiles
│   ├── wireguard/service.go             # WireGuardService — full WG management via SSH
│   └── config/store.go                  # YAML config store (load/save server profiles)
├── frontend/
│   ├── src/
│   │   ├── main.ts                      # App bootstrap
│   │   ├── App.svelte                   # Root layout: sidebar + main panel + theme provider
│   │   ├── lib/
│   │   │   ├── components/
│   │   │   │   ├── Sidebar.svelte       # Server list with status indicators
│   │   │   │   ├── ServerDashboard.svelte  # Main panel: WG status for selected server
│   │   │   │   ├── AddServerModal.svelte   # Add/edit server form
│   │   │   │   ├── PeerTable.svelte        # Peer list with actions
│   │   │   │   ├── AddPeerModal.svelte     # Add peer form (keygen or manual key)
│   │   │   │   ├── InterfaceModal.svelte   # Create/edit WG interface
│   │   │   │   ├── ConfigViewer.svelte     # View config file content
│   │   │   │   ├── StatusBadge.svelte      # Connection status dot
│   │   │   │   ├── ThemeToggle.svelte      # Dark/light toggle
│   │   │   │   └── ui/                     # shadcn-svelte components
│   │   │   ├── stores/
│   │   │   │   └── servers.ts             # Svelte store wrapping Wails bindings
│   │   │   └── utils.ts                   # cn() helper
│   │   └── app.css                        # TailwindCSS imports
│   └── ...
├── docs/plan/
│   └── architecture.md                   # This plan
```

## Go Backend Services

### Config Store (`internal/config/store.go`)
- Load/save server profiles to `~/.config/wireguardhub/servers.yaml`
- Plaintext YAML for v1 (credentials included)
- Uses `gopkg.in/yaml.v3` for encoding/decoding
- Uses `adrg/xdg` for cross-platform config path

### SSH Client (`internal/ssh/client.go`)
- Connect using password or private key auth (`golang.org/x/crypto/ssh`)
- Support optional passphrase for encrypted private keys
- Execute commands, return stdout/stderr
- Connection pooling: cache active SSH sessions per server ID
- Auto-reconnect if session dropped

### ServerService (`internal/server/service.go`)
Exposed to frontend via Wails bindings:

| Method | Returns | Description |
|--------|---------|-------------|
| `GetServers()` | `[]ServerConfig` | List all saved server profiles |
| `AddServer(config)` | `ServerConfig` | Add a new server profile |
| `UpdateServer(config)` | `ServerConfig` | Update an existing profile |
| `DeleteServer(id)` | `bool` | Delete a server profile |
| `TestConnection(config)` | `TestConnectionResult` | Try SSH connect, return success/fail with message |

### WireGuardService (`internal/wireguard/service.go`)
Exposed to frontend via Wails bindings:

| Method | Returns | Description |
|--------|---------|-------------|
| `GetStatus(serverID)` | `WGStatus` | Run `wg show all dump`, parse into structured data |
| `CreateInterface(serverID, req)` | `WGInterface` | Generate keys, create config file, bring up interface with `wg-quick` |
| `DeleteInterface(serverID, name)` | `bool` | Take down interface (`wg-quick down`), remove config file |
| `GetInterfaceConfig(serverID, name)` | `string` | Return contents of `/etc/wireguard/<name>.conf` |
| `SyncConfig(serverID, name)` | `bool` | Run `wg syncconf` to apply config changes without restart |
| `AddPeer(req)` | `AddPeerResult` | Generate keypair via `wg genkey/pubkey`, add peer to interface, return client config |
| `RemovePeer(serverID, iface, publicKey)` | `bool` | Remove peer from interface |
| `GenerateKeyPair(serverID)` | `{public, private}` | Run `wg genkey` + `wg pubkey` on server |

## Frontend UI

### Layout
- **Sidebar** (left, 280px): Server list with status dots (green=connected, red=offline, gray=untested). "Add Server" button at bottom. Theme toggle at very bottom.
- **Main panel** (right, fills remaining): Dashboard for selected server, or empty state when no server selected.

### Server Dashboard
- **Header**: Server name, host:port, SSH status badge, refresh button, edit/delete actions
- **Interface section**: Cards for each WG interface showing name, public key, listen port, RX/TX bytes. Each card has actions: Add Peer, View Config, Sync Config, Delete Interface
- **Peer table**: Public key (truncated), endpoint, allowed IPs, latest handshake (relative time), transfer stats, remove button
- **"Create Interface" button**: Opens InterfaceModal for full lifecycle creation

### Modals
1. **Add/Edit Server Modal**: Name, host, port, username, auth method (password/key), password or private key textarea, passphrase (optional). "Test Connection" button before saving.
2. **Add Peer Modal**: Interface selector, public key (auto-generate or paste), allowed IPs input (tag-style), preshared key (optional), endpoint (optional), persistent keepalive (optional). Shows generated client config after success.
3. **Create Interface Modal**: Interface name, listen port, private key (auto-generate or paste), endpoint address, default allowed IPs. Creates interface and brings it up.
4. **Config Viewer Modal**: Read-only display of `/etc/wireguard/<name>.conf` with copy button.

### Theme
- Dark/light toggle stored in localStorage
- TailwindCSS `dark:` variant for dark mode
- Default: dark theme

## Data Models

```go
type ServerConfig struct {
    ID, Name, Host, Username, AuthMethod string
    Port                                 int
    Password, PrivateKey, Passphrase     string // omitted from JSON if empty
}

type WGInterface struct {
    Name, PublicKey, PrivateKey, Endpoint string
    ListenPort                            int
    RxBytes, TxBytes                      int64
    Peers                                 []WGPeer
}

type WGPeer struct {
    PublicKey, PresharedKey, Endpoint string
    AllowedIPs                        []string
    LatestHandshake                   time.Time
    RxBytes, TxBytes                  int64
    PersistentKeepalive               int
}
```

## Implementation Order

1. **Re-init project**: Replace frontend with Svelte template (`wails3 init -t svelte`)
2. **Go backend**: models → config store → SSH client → server service → wireguard service
3. **main.go**: register services, configure window (1200x800, dark bg)
4. **Frontend setup**: install TailwindCSS, lucide-svelte, shadcn-svelte, configure theme system
5. **Frontend components**: Sidebar → ServerDashboard → modals → peer table → theme toggle
6. **Wire up**: generate Wails bindings, connect Svelte stores to backend services
7. **Build & test**: `wails3 dev` for dev, `wails3 build` for production

## Config Storage

- Location: `$XDG_CONFIG_HOME/wireguardhub/servers.yaml` (e.g. `~/.config/wireguardhub/servers.yaml`)
- Format: YAML list of `ServerConfig`
- Plaintext for v1 — encryption can be added later

## SSH Connection Flow

1. User selects a server in sidebar
2. Frontend calls `WireGuardService.GetStatus(serverID)`
3. Backend checks if cached SSH session exists for serverID
4. If not, connects using stored credentials, caches session
5. Runs `wg show all dump` over SSH, parses output
6. Returns structured `WGStatus` to frontend
7. Session stays cached for subsequent operations

## Distro Abstraction

WireGuard operations differ across Linux distributions — package managers, init systems, and privilege escalation methods vary. The app uses a **strategy pattern** to abstract these differences behind a `Distro` interface.

### Interface (`internal/wireguard/distro.go`)

```go
type Distro interface {
    ID() string
    DisplayName() string
    InstallWireGuard() string
    StartInterface(name string) string
    StopInterface(name string) string
    EnableInterface(name string) string
    DisableInterface(name string) string
    WriteConfig(name, content string) string
    ReadConfig(name string) string
    RemoveConfig(name string) string
    SyncConfig(name string) string
}
```

### Concrete Implementations (`internal/wireguard/distros/`)

Two shared base structs handle the common patterns:

- **`systemdDistro`** — shared by Ubuntu, Fedora, openSUSE (systemctl + sudo + `/etc/wireguard/`)
- **`openrcDistro`** — shared by Alpine (rc-service + no sudo + `/etc/wireguard/`)

Each distro embeds the appropriate base and overrides `InstallWireGuard()` with its package manager command. Future distros using the same init system can reuse these bases.

### Auto-Detection (`internal/wireguard/detect.go`)

- Runs `cat /etc/os-release` over SSH on first connection
- Parses `ID=` and `ID_LIKE=` fields
- Maps to the closest supported distro
- Falls back to a generic systemd distro if unknown
- Result is cached for the session

### Manual Override

The `ServerConfig.DistroID` field allows manual selection. If set, auto-detection is skipped. The frontend exposes this as a dropdown in the Add/Edit Server modal.

For the full list of supported distros and their specific commands, see [supported-distros.md](../supported-distros.md).

## WireGuard CLI Commands Used

- `wg show all dump` — get full status
- `wg show <iface> dump` — get single interface status
- `wg genkey` / `wg pubkey` — generate keypairs
- `wg set <iface> peer <pubkey> allowed-ips <ips> [preshared-key <psk>] [endpoint <addr>] [persistent-keepalive <n>]` — add/modify peer
- `wg set <iface> peer <pubkey> remove` — remove peer
- `wg syncconf <iface> <(wg-quick strip <iface>)` — sync config without restart
- `wg-quick up/down <iface>` — bring interface up/down
- `cat /etc/wireguard/<iface>.conf` — read config file

Note: The exact commands (including privilege escalation prefix and service management) are determined by the active `Distro` implementation. The commands above are the base WireGuard operations; the distro layer wraps them with the appropriate `sudo`/root prefix and service manager calls.
