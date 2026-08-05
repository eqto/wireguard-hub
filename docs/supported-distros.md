# Supported Linux Distributions

WireGuard Admin supports multiple Linux distributions by abstracting distro-specific differences behind a `Distro` interface. This document covers what's supported, how detection works, and how to add new distros.

## Supported Distros

| Distro ID | Display Name | Package Manager | Install Command | Init System | Privilege | Config Path |
|-----------|-------------|----------------|-----------------|-------------|-----------|-------------|
| `ubuntu` | Ubuntu / Debian | `apt` | `apt install -y wireguard wireguard-tools` | systemd | `sudo` | `/etc/wireguard/<iface>.conf` |
| `fedora` | Fedora / RHEL | `dnf` | `dnf install -y wireguard-tools` | systemd | `sudo` | `/etc/wireguard/<iface>.conf` |
| `opensuse` | openSUSE | `zypper` | `zypper install -y wireguard-tools` | systemd | `sudo` | `/etc/wireguard/<iface>.conf` |
| `alpine` | Alpine | `apk` | `apk add wireguard-tools` | OpenRC | root (no sudo) | `/etc/wireguard/<iface>.conf` |

### Family Coverage

Each distro implementation covers its broader family:

- **`ubuntu`** — Ubuntu, Debian, Linux Mint, Pop!_OS, and other Debian derivatives
- **`fedora`** — Fedora, RHEL, CentOS, Rocky Linux, AlmaLinux, Amazon Linux
- **`opensuse`** — openSUSE Leap, openSUSE Tumbleweed, SUSE Linux Enterprise
- **`alpine`** — Alpine Linux (commonly used in containers and lightweight VPS)

## Service Management Commands

### systemd (Ubuntu, Fedora, openSUSE)

| Action | Command |
|--------|---------|
| Start interface | `sudo systemctl start wg-quick@<iface>` |
| Stop interface | `sudo systemctl stop wg-quick@<iface>` |
| Enable on boot | `sudo systemctl enable wg-quick@<iface>` |
| Disable on boot | `sudo systemctl disable wg-quick@<iface>` |

### OpenRC (Alpine)

| Action | Command |
|--------|---------|
| Start interface | `rc-service wg-quick.<iface> start` |
| Stop interface | `rc-service wg-quick.<iface> stop` |
| Enable on boot | `rc-update add wg-quick.<iface> default` |
| Disable on boot | `rc-update del wg-quick.<iface> default` |

Key differences in Alpine:
- Uses **dot notation** (`wg-quick.<iface>`) instead of systemd's **template units** (`wg-quick@<iface>`)
- No `sudo` prefix — the SSH user is typically root
- `rc-service` / `rc-update` instead of `systemctl`

## Auto-Detection

When a server's `distroId` is not set (empty string), the app auto-detects the distro on first SSH connection:

1. Runs `cat /etc/os-release` over SSH
2. Parses the `ID=` and `ID_LIKE=` fields
3. Maps the result to the closest supported distro:

| `ID` / `ID_LIKE` value | Detected Distro |
|------------------------|----------------|
| `ubuntu`, `debian`, `linuxmint`, `pop` | `ubuntu` |
| `fedora`, `rhel`, `centos`, `rocky`, `almalinux`, `amzn` | `fedora` |
| `opensuse`, `opensuse-leap`, `opensuse-tumbleweed`, `sles` | `opensuse` |
| `alpine` | `alpine` |

4. If no match is found, falls back to a **generic systemd distro** (assumes `sudo` + `systemctl` + `/etc/wireguard/`)

The detected distro is cached for the session. To persist it, set `distroId` in the server config or select it from the dropdown in the Add/Edit Server modal.

## Manual Override

You can override auto-detection in two ways:

1. **UI**: In the Add/Edit Server modal, select a distro from the "Linux Distro" dropdown (defaults to "Auto-detect")
2. **Config file**: Set the `distroId` field in `~/.config/wireguard-admin/servers.yaml`:

```yaml
- id: "abc-123"
  name: "My Server"
  host: "10.0.0.1"
  port: 22
  username: "root"
  authMethod: "key"
  distroId: "alpine"
```

## Architecture

The distro abstraction uses the **strategy pattern**:

```
internal/wireguard/
├── distro.go              # Distro interface
├── detect.go              # Auto-detection logic
└── distros/
    ├── common.go          # systemdDistro + openrcDistro base structs
    ├── ubuntu.go          # UbuntuDistro (embeds systemdDistro)
    ├── fedora.go          # FedoraDistro (embeds systemdDistro)
    ├── opensuse.go        # OpenSusedistro (embeds systemdDistro)
    └── alpine.go          # AlpineDistro (embeds openrcDistro)
```

The `Distro` interface defines methods for all distro-specific operations:

- `InstallWireGuard()` — package install command
- `StartInterface(name)` / `StopInterface(name)` — service control
- `EnableInterface(name)` / `DisableInterface(name)` — boot persistence
- `WriteConfig(name, content)` / `ReadConfig(name)` / `RemoveConfig(name)` — config file operations
- `SyncConfig(name)` — `wg syncconf` command

Two shared base structs reduce duplication:
- **`systemdDistro`** — shared by Ubuntu, Fedora, openSUSE (systemctl + sudo + `/etc/wireguard/`)
- **`openrcDistro`** — shared by Alpine (rc-service + no sudo + `/etc/wireguard/`)

Each concrete distro embeds the appropriate base and only overrides `InstallWireGuard()`.

## Adding a New Distro

To support a new distribution:

1. **Create a distro implementation** in `internal/wireguard/distros/<name>.go`:
   - Embed `systemdDistro` or `openrcDistro` (or implement all methods if it uses a different init system)
   - Override `ID()`, `DisplayName()`, and `InstallWireGuard()` at minimum

2. **Register the distro** in the distro registry (`internal/wireguard/distro.go`)

3. **Add detection mapping** in `internal/wireguard/detect.go` — map the distro's `ID` / `ID_LIKE` value from `/etc/os-release` to your new implementation

4. **Update the frontend** — add the new distro as an option in the Add/Edit Server modal dropdown

5. **Update this document** — add the distro to the supported distros table
