# Supported Linux Distributions

WireguardHub does **not** ship per-distro code or configuration. Instead it probes the host at runtime and adapts its commands accordingly. This document describes what is detected, how, and which distributions are covered as a result.

## What Is Detected

| Probe | How | Used For |
|-------|-----|----------|
| Package manager | `command -v apt-get` → `dnf` → `yum` → `pacman` (first match) | `InstallWireGuard`, reported in `WGStatus.PackageManager` |
| Init system | `command -v systemctl` **and** `test -d /run/systemd/system` | Whether `wg-quick@<iface>` service management is available (`WGStatus.HasSystemd`) |
| OS name | `PRETTY_NAME` from `/etc/os-release` | Display only (`WGStatus.OS`) |
| Hostname / IP | `hostname`, `hostname -I \| awk '{print $1}'` | Display only |

There is no `distroId` config field, no distro dropdown, and no `/etc/os-release` `ID`/`ID_LIKE` mapping. Detection happens on every `GetStatus` call (server info is cached per server ID afterward).

## Supported Package Managers

| Package Manager | Install Command | Example Distros |
|-----------------|-----------------|-----------------|
| `apt-get` | `sudo apt-get update && sudo apt-get install -y wireguard wireguard-tools` | Ubuntu, Debian, Linux Mint, Pop!_OS |
| `dnf` | `sudo dnf install -y wireguard-tools` | Fedora, RHEL, Rocky, AlmaLinux |
| `yum` | `sudo yum install -y epel-release && sudo yum install -y wireguard-tools` | CentOS, Amazon Linux |
| `pacman` | `sudo pacman -S --noconfirm wireguard-tools` | Arch Linux |

The `apt-get` path retries up to 6 times (5s apart) when the dpkg lock is held by another process.

If none of the above are present, `InstallWireGuard` returns an error: *"no supported package manager found (apt/dnf/yum/pacman)"*.

## Service Management

### systemd (when `HasSystemd` is true)

| Action | Command |
|--------|---------|
| Start interface | `sudo systemctl start wg-quick@<iface>` (via `systemctl stop; start` to reset oneshot state) |
| Stop interface | `sudo systemctl stop wg-quick@<iface>` |
| Restart interface | `sudo systemctl stop wg-quick@<iface>; systemctl start wg-quick@<iface>` |
| Enable on boot | `sudo systemctl enable --now wg-quick@<iface>` |
| Disable on boot | `sudo systemctl disable wg-quick@<iface>` |
| Delete interface | `sudo systemctl disable --stop wg-quick@<iface>` |

When the `wg-quick@<iface>` service is enabled, start/stop/restart/delete route through `systemctl` so systemd tracks the unit state. When it is not enabled, they fall back to direct `wg-quick up/down`. Enabling a service that is currently up via `wg-quick` performs an atomic transition (`wg-quick down && systemctl enable --now`) inside a single `sudo bash -c` so the sudo password is only requested once.

### No systemd

When `HasSystemd` is false, only direct `wg-quick up/down` is available. Service enable/disable and the "enable as service" option are hidden/disabled in the UI, and `EnableService`/`DisableService` return an error.

## Adding Support for a New Distro

In most cases **no code changes are needed** — if the distribution uses `apt`, `dnf`, `yum`, or `pacman` and is booted with systemd, it is already supported.

If the distribution uses a different package manager or init system:

1. **Package manager** — add a branch to `InstallWireGuard` in `internal/wireguard/service.go` keyed on `client.CommandExists("<pm>")`, and add it to the package-manager detection chain in `fillServerInfo`.
2. **Init system** — if it is not systemd, the systemd service-management methods will already gracefully fall back to `wg-quick up/down`. A new init system would require new service-control branches in `interface.go` (`BringUpInterface`, `BringDownInterface`, `RestartInterface`, `EnableService`, `DisableService`, `DeleteInterface`).
3. **Update this document** — add the package manager / init system to the tables above.

## Privilege Escalation

All privileged commands are run with `sudo`. The SSH/local clients rewrite `sudo <cmd>` into `sudo -S -p '' <cmd>` and pipe the stored password to stdin, so no interactive TTY is required. On the local machine, if the user is already root, sudo is still used but the password may be empty.
