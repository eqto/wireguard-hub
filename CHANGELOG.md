# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0-rc.1] - 2026-08-10

### Added

- Visual point-and-click interface for managing WireGuard VPN servers.
- Manage multiple remote servers over SSH or the local machine from one window.
- Create, start, stop, restart, and delete WireGuard interfaces with automatic key generation.
- Enable/disable systemd service (`wg-quick@`) for auto-start on boot.
- Add, edit, and remove peers with automatic keypair generation and ready-to-paste config export.
- Live status monitoring with automatic polling: interface and peer online state, data transferred, last handshake times.
- View WireGuard config files directly within the app.
- One-click WireGuard installation on supported Linux distributions (Ubuntu, Debian, Fedora, RHEL, CentOS, Arch, and related).
- SSH jump server (bastion) support for reaching servers behind another host.
- Built-in terminal panel showing every command the app runs and its output.
- Dark and light theme toggle.
- Server mode (no GUI, HTTP server only) for headless/Docker deployments.
- Cross-platform desktop builds for Linux (`.rpm`/`.deb`), macOS (`.dmg`), and Windows (NSIS installer/portable `.exe`).
