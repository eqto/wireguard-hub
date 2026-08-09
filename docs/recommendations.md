# WireguardHub — Recommendations

Prioritized recommendations based on a full codebase analysis.

---

## 1. Add Unit Tests

Start with pure functions that are easily testable:

- `parseWGDump` — parses `wg show all dump` output into `WGStatus`
- `parseInterfaceConfig` — parses `.conf` file content into `WGInterface`
- `removePeerSection` — removes a `[Peer]` section from config text
- `updatePeerMetaInConfig` — updates `# Name` / `# Description` comments in a peer section
- `updatePeerFieldInConfig` — updates a field (Endpoint, AllowedIPs) in a peer section
- `configHasListenPort` — checks if config text contains a `ListenPort` line
- `generateClientConfig` — generates client config string from peer request

## 2. Fix `ExecStreaming` Return Value (SSH Client)

In `internal/ssh/client.go`, `ExecStreaming` calls `session.Close()` at line 218 then returns `session` as the `io.Closer`. The returned closer is already closed, making `CancelInstall` ineffective for SSH clients. Return a meaningful `io.Closer` that can actually cancel the running command (the local client does this correctly with `processCloser`).

## 3. Sanitize Shell Inputs

User-supplied values (interface names, public keys, preshared keys, endpoints) are interpolated into shell commands via `fmt.Sprintf` without sanitization. For example:

```go
fmt.Sprintf("sudo wg set %s peer %s ...", req.Interface, pubKey)
```

Validate before interpolation:

- **Interface names**: `^[a-zA-Z0-9_-]+$`
- **Public keys**: WireGuard base64 (44 chars)
- **IP addresses**: valid IPv4/IPv6 CIDR notation
- **Endpoints**: `host:port` format

## 4. Implement SSH Host Key Verification

Currently uses `ssh.InsecureIgnoreHostKey()` (susceptible to MITM). At minimum, offer "trust on first use" (TOFU) with a persistent `known_hosts` file stored alongside the config.

## 5. Add Credential Encryption

Server passwords and private keys are stored in plaintext YAML (`servers.yaml`). Use OS keychain (libsecret on Linux, Keychain on macOS) or at minimum AES encryption with a master password.

## 6. Call `CloseAll()` on Shutdown

`server.Service.CloseAll()` exists but is never called from `main.go`. Register a cleanup handler so SSH sessions are properly closed on app exit:

```go
defer serverSvc.CloseAll()
```

## 7. ~~Implement Distro Strategy Pattern (or Update Docs)~~ — Resolved

The `Distro` interface and `internal/wireguard/distros/` package were removed in favor of runtime detection (`CommandExists` for package managers, `hasSystemd` for init system). The architecture and supported-distros docs have been updated to reflect this. No further action needed.

## 8. Update `build/config.yml` Metadata

Replace placeholder values:

- `companyName: "My Company"` → actual company name
- `productName: "My Product"` → "WireguardHub"
- `productIdentifier: "com.mycompany.myproduct"` → actual identifier
- `description`, `copyright`, `comments` → actual values

## 9. Invalidate `serverInfoCache`

Server info (hostname, OS, IP) is cached per server ID and never invalidated. If the server's IP or OS changes, the cache won't update. Add a TTL or manual refresh mechanism.

## 10. Add Input Validation

Interface names, port numbers, and IP addresses are not validated before being sent to the server. Add validation in both frontend (modal forms) and backend (request structs) to catch errors early.
