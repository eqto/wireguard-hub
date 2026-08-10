# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability, **please do not open a public issue**.

Instead, report it privately by emailing the maintainers. Include:

- A description of the vulnerability and its impact
- Steps to reproduce or a proof of concept
- Suggested fix (if any)

You will receive a response within 72 hours. If the vulnerability is confirmed, a fix will be prioritized and a security advisory will be published.

## Scope

- SSH credential handling and storage
- Server configuration data (`servers.yaml`, `local.yaml`)
- Wails IPC boundary between Go backend and webview frontend
- Command execution on remote servers (sudo handling, input sanitization)

## Out of Scope

- Vulnerabilities in third-party dependencies (report upstream)
- Social engineering attacks
- Physical access to an unlocked device
- WireGuard itself (report to the [WireGuard project](https://www.wireguard.com/))

## Security Notes

- Server credentials (passwords, private keys) are stored locally in `~/.config/wireguardhub/` with file permissions restricted to the user account.
- Commands executed on remote servers are constructed by the app and do not accept arbitrary user shell input.
- SSH connections use the Go `golang.org/x/crypto/ssh` library with standard host key verification.

**Note:** WireguardHub has not undergone a formal security audit. Use at your own risk.
