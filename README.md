# WireguardHub

<p align="center">
  <img src="docs/wireguardhub.svg" alt="WireguardHub logo" width="160">
</p>

WireguardHub is a desktop app that gives you a **visual point-and-click interface** for [WireGuard](https://www.wireguard.com/) VPNs. Instead of logging into each server and typing commands, you manage all your VPN servers from one window — no agent or extra software needs to be installed on the servers.

## What Can It Do?

- **Manage many servers in one place** — Keep a list of all your VPN servers and connect to any of them with a click.
- **Work on remote servers or this computer** — Manage WireGuard on remote servers over SSH, or on the machine you're sitting at directly.
- **Create and control VPN interfaces** — Set up a new VPN interface (with automatic key generation), then start, stop, or restart it whenever you want. You can also make a VPN start automatically when the server boots.
- **Add and edit peers** — Add devices (peers) to a VPN, with keys generated for you. Edit their settings later, and get a ready-to-use config file for each peer.
- **See live status** — Watch which VPNs and peers are online, how much data has been transferred, and when each peer last connected. The view updates automatically.
- **View config files** — Open and read the WireGuard config file on any server without leaving the app.
- **Install WireGuard with one click** — If a server doesn't have WireGuard yet, install it straight from the app (Ubuntu, Debian, Fedora, RHEL, CentOS, Arch, and related distros).
- **Connect through a jump server** — Reach servers that are only accessible through another (bastion) server.
- **See what's happening** — A built-in terminal panel shows every command the app runs and its output, so you always know what's going on.
- **Dark and light themes** — Switch between dark and light appearance.

## Who Is It For?

WireguardHub is for anyone who runs WireGuard VPN servers but doesn't want to memorize the command line — homelabbers, small-business admins, and developers who manage a handful of VPN endpoints.

## What Do I Need?

- **Your computer** — Linux (with GTK4 / WebKitGTK 6). macOS and Windows builds are possible via cross-compilation.
- **Your servers** — Any Linux server you can reach over SSH (or the computer you're running the app on). WireGuard doesn't need to be pre-installed — the app can install it for you.
- **Server login** — A username and password, or a private key, for each server. For most operations you'll also need `sudo` (admin) access on the server.

## Download

Pre-built binaries are available for each release on the [Releases](https://github.com/eqto/wireguard-hub/releases) page.

### Desktop App

| OS | Architecture | Download |
| --- | --- | --- |
| Linux | x86_64 | [`.rpm`](https://github.com/eqto/wireguard-hub/releases/download/v0.1.0-rc.1/wireguardhub-v0.1.0-rc.1-1.x86_64.rpm) · [`.deb`](https://github.com/eqto/wireguard-hub/releases/download/v0.1.0-rc.1/wireguardhub_v0.1.0-rc.1_amd64.deb) · [binary](https://github.com/eqto/wireguard-hub/releases/download/v0.1.0-rc.1/wireguardhub-linux-amd64) |
| macOS | Apple Silicon | [`.dmg`](https://github.com/eqto/wireguard-hub/releases/download/v0.1.0-rc.1/wireguardhub-darwin-arm64.dmg) · [binary](https://github.com/eqto/wireguard-hub/releases/download/v0.1.0-rc.1/wireguardhub-darwin-arm64) |
| macOS | Intel | [`.dmg`](https://github.com/eqto/wireguard-hub/releases/download/v0.1.0-rc.1/wireguardhub-darwin-amd64.dmg) · [binary](https://github.com/eqto/wireguard-hub/releases/download/v0.1.0-rc.1/wireguardhub-darwin-amd64) |
| Windows | x86_64 | [installer (`.exe`)](https://github.com/eqto/wireguard-hub/releases/download/v0.1.0-rc.1/wireguardhub-windows-amd64-setup.exe) · [portable](https://github.com/eqto/wireguard-hub/releases/download/v0.1.0-rc.1/wireguardhub-windows-amd64.exe) |
| Windows | ARM64 | [`.exe`](https://github.com/eqto/wireguard-hub/releases/download/v0.1.0-rc.1/wireguardhub-windows-arm64.exe) |

> Download links point to the latest release. Visit the [Releases](https://github.com/eqto/wireguard-hub/releases) page for all versions.

## Quick Start

1. **Open the app.** Your servers list starts empty (except for "Local", which is the computer you're on).

2. **Add a server.** Click **Add Server**, enter the server's address, your username, and your password or private key. Click **Test Connection** to check it works, then save.

3. **Select the server** in the sidebar. The app connects and shows the WireGuard status for that server. If WireGuard isn't installed yet, you'll see a button to install it.

4. **Create a VPN interface.** Click **Create Interface**, give it a name and a port, and the app generates the cryptographic keys for you and brings the interface up. Tick "enable as service" if you want it to start automatically on boot.

5. **Add peers (devices).** Click **Add Peer** on an interface, fill in the allowed IP range, and the app generates a keypair and gives you a ready-to-paste config file for that device.

6. **Monitor.** The dashboard refreshes automatically every few seconds, showing which interfaces and peers are online, data transferred, and last handshake times.

## Where Are My Settings Stored?

Your server list and login details are saved on your computer in:

```
~/.config/wireguardhub/servers.yaml
```

If you use local mode (managing WireGuard on this computer), those credentials are saved separately in:

```
~/.config/wireguardhub/local.yaml
```

Both files are private to your user account.

## Documentation

| Document | For |
|----------|-----|
| [Development guide](docs/development.md) | Building, running, and packaging the app from source |
| [Architecture & code flow](docs/plan/architecture.md) | How the codebase is organized and how it works internally |
| [Supported distros](docs/supported-distros.md) | Which Linux distributions are supported and how detection works |
| [Recommendations](docs/recommendations.md) | Known improvement opportunities for the codebase |

## Getting Help

- Found a bug or have a feature request? Please open an issue.
- For WireGuard itself (not this app), see the [official WireGuard project](https://www.wireguard.com/).
