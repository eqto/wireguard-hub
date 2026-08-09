package wireguard

import (
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"wireguardhub/internal/models"
	"wireguardhub/internal/server"
	"wireguardhub/internal/ssh"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type Service struct {
	serverSvc *server.Service
	mu        sync.Mutex
	session   io.Closer

	serverInfoMu    sync.Mutex
	serverInfoCache map[string]serverInfo
}

type serverInfo struct {
	Hostname       string
	ServerIP       string
	OS             string
	PackageManager string
	HasSystemd     bool
}

func NewService(serverSvc *server.Service) *Service {
	return &Service{serverSvc: serverSvc, serverInfoCache: make(map[string]serverInfo)}
}

func (s *Service) InstallWireGuard(serverID string) (bool, error) {
	client, err := s.serverSvc.GetClient(serverID)
	if err != nil {
		application.Get().Event.Emit("wg-install-done", map[string]interface{}{"success": false, "error": err.Error()})
		return false, err
	}

	emit := func(line string) {
		application.Get().Event.Emit("ssh-terminal", map[string]interface{}{
			"serverId": serverID,
			"kind":     "output",
			"line":     line,
		})
	}

	// Detect package manager and install
	if client.CommandExists("apt-get") {
		// Step 1: apt-get update
		session, err := client.ExecStreaming("sudo env DEBIAN_FRONTEND=noninteractive DEBCONF_NONINTERACTIVE_SEEN=true apt-get update -y 2>&1", emit)
		s.mu.Lock()
		s.session = session
		s.mu.Unlock()
		if err != nil {
			application.Get().Event.Emit("wg-install-done", map[string]interface{}{"success": false, "error": err.Error()})
			return false, err
		}
		s.mu.Lock()
		s.session = nil
		s.mu.Unlock()

		// Step 2: apt-get install with retry for dpkg lock
		installCmd := "sudo env DEBIAN_FRONTEND=noninteractive DEBCONF_NONINTERACTIVE_SEEN=true apt-get install -y wireguard wireguard-tools 2>&1"
		var installErr error
		for attempt := 1; attempt <= 6; attempt++ {
			session, err = client.ExecStreaming(installCmd, emit)
			s.mu.Lock()
			s.session = session
			s.mu.Unlock()
			if err == nil {
				s.mu.Lock()
				s.session = nil
				s.mu.Unlock()
				installErr = nil
				break
			}
			s.mu.Lock()
			s.session = nil
			s.mu.Unlock()
			installErr = err
			errMsg := err.Error()
			if strings.Contains(errMsg, "lock") || strings.Contains(errMsg, "held by process") {
				if attempt < 6 {
					emit(fmt.Sprintf("Waiting for package manager lock... (attempt %d/6)", attempt+1))
					time.Sleep(5 * time.Second)
					continue
				}
			}
			break
		}
		if installErr != nil {
			application.Get().Event.Emit("wg-install-done", map[string]interface{}{"success": false, "error": installErr.Error()})
			return false, installErr
		}
		application.Get().Event.Emit("wg-install-done", map[string]interface{}{"success": true})
		return true, nil
	}

	if client.CommandExists("dnf") {
		session, err := client.ExecStreaming("sudo dnf install -y wireguard-tools 2>&1", emit)
		s.mu.Lock()
		s.session = session
		s.mu.Unlock()
		if err != nil {
			application.Get().Event.Emit("wg-install-done", map[string]interface{}{"success": false, "error": err.Error()})
			return false, err
		}
		s.mu.Lock()
		s.session = nil
		s.mu.Unlock()
		application.Get().Event.Emit("wg-install-done", map[string]interface{}{"success": true})
		return true, nil
	}

	if client.CommandExists("yum") {
		session, err := client.ExecStreaming("sudo bash -c 'yum install -y epel-release 2>&1 && yum install -y wireguard-tools 2>&1'", emit)
		s.mu.Lock()
		s.session = session
		s.mu.Unlock()
		if err != nil {
			application.Get().Event.Emit("wg-install-done", map[string]interface{}{"success": false, "error": err.Error()})
			return false, err
		}
		s.mu.Lock()
		s.session = nil
		s.mu.Unlock()
		application.Get().Event.Emit("wg-install-done", map[string]interface{}{"success": true})
		return true, nil
	}

	if client.CommandExists("pacman") {
		session, err := client.ExecStreaming("sudo pacman -S --noconfirm wireguard-tools 2>&1", emit)
		s.mu.Lock()
		s.session = session
		s.mu.Unlock()
		if err != nil {
			application.Get().Event.Emit("wg-install-done", map[string]interface{}{"success": false, "error": err.Error()})
			return false, err
		}
		s.mu.Lock()
		s.session = nil
		s.mu.Unlock()
		application.Get().Event.Emit("wg-install-done", map[string]interface{}{"success": true})
		return true, nil
	}

	application.Get().Event.Emit("wg-install-done", map[string]interface{}{"success": false, "error": "no supported package manager found"})
	return false, fmt.Errorf("no supported package manager found (apt/dnf/yum/pacman)")
}

func (s *Service) CancelInstall() (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session != nil {
		s.session.Close()
		s.session = nil
	}
	return true, nil
}

func (s *Service) GetStatus(serverID string) (models.WGStatus, error) {
	client, err := s.serverSvc.GetClient(serverID)
	if err != nil {
		return models.WGStatus{}, err
	}

	// Check if wg binary exists on the server.
	if !client.CommandExists("wg") {
		status := models.WGStatus{
			Interfaces:     []models.WGInterface{},
			WGNotInstalled: true,
		}
		s.fillServerInfo(serverID, client, &status)
		return status, nil
	}

	// Verify sudo access before running sudo commands.
	_, sudoStderr, sudoErr := client.ExecSilent("sudo true")
	if sudoErr != nil {
		msg := strings.TrimSpace(sudoStderr)
		if strings.Contains(msg, "incorrect password attempt") {
			return models.WGStatus{}, fmt.Errorf("Incorrect password")
		}
		return models.WGStatus{}, fmt.Errorf("sudo authentication failed: %s", msg)
	}

	// 1. Get running interface names and live stats from wg show all dump.
	runningStats := map[string]models.WGInterface{}
	stdout, _, _ := client.ExecSilent("sudo wg show all dump")
	if strings.TrimSpace(stdout) != "" {
		liveStatus := parseWGDump(stdout)
		for _, iface := range liveStatus.Interfaces {
			runningStats[iface.Name] = iface
		}
	}

	// 2. List all .conf files and parse each one.
	status := models.WGStatus{Interfaces: []models.WGInterface{}}
	// Populate server info (incl. HasSystemd) up front so we can decide
	// whether to query systemctl per interface below.
	s.fillServerInfo(serverID, client, &status)

	confListOut, _, _ := client.ExecSilent("sudo bash -c 'ls /etc/wireguard/*.conf 2>/dev/null'")
	for _, confPath := range strings.Split(strings.TrimSpace(confListOut), "\n") {
		confPath = strings.TrimSpace(confPath)
		if confPath == "" {
			continue
		}
		base := filepath.Base(confPath)
		ifaceName := strings.TrimSuffix(base, ".conf")

		confText, _, confErr := client.ExecSilentF("sudo cat %s", confPath)
		if confErr != nil {
			continue
		}

		iface := parseInterfaceConfig(confText, ifaceName)

		// 3. Determine online from wg dump.
		if live, ok := runningStats[ifaceName]; ok {
			iface.Online = true
			iface.PublicKey = live.PublicKey
			iface.RxBytes = live.RxBytes
			iface.TxBytes = live.TxBytes
			// Merge live peer stats (handshake, rx/tx) with config peer data.
			for j := range iface.Peers {
				for k := range live.Peers {
					if iface.Peers[j].PublicKey == live.Peers[k].PublicKey {
						iface.Peers[j].RxBytes = live.Peers[k].RxBytes
						iface.Peers[j].TxBytes = live.Peers[k].TxBytes
						iface.Peers[j].LatestHandshake = live.Peers[k].LatestHandshake
						if live.Peers[k].Endpoint != "" {
							iface.Peers[j].Endpoint = live.Peers[k].Endpoint
						}
						break
					}
				}
			}
		} else if iface.PrivateKey != "" {
			// Interface is offline: derive public key from the config file's private key.
			pubKey, _, err := client.ExecWithInputSilent("wg pubkey", iface.PrivateKey)
			if err == nil {
				iface.PublicKey = strings.TrimSpace(pubKey)
			}
		}

		// 4. Detect systemd service enabled state when systemd is available.
		if status.HasSystemd {
			iface.ServiceEnabled = isServiceEnabled(client, ifaceName)
		}

		status.Interfaces = append(status.Interfaces, iface)
	}

	return status, nil
}

func (s *Service) fillServerInfo(serverID string, client ssh.Executor, status *models.WGStatus) {
	s.serverInfoMu.Lock()
	cached, ok := s.serverInfoCache[serverID]
	s.serverInfoMu.Unlock()
	if ok {
		status.Hostname = cached.Hostname
		status.ServerIP = cached.ServerIP
		status.OS = cached.OS
		status.PackageManager = cached.PackageManager
		status.HasSystemd = cached.HasSystemd
		return
	}

	hostname, _, _ := client.Exec("hostname")
	status.Hostname = strings.TrimSpace(hostname)

	serverIP, _, _ := client.Exec("hostname -I | awk '{print $1}'")
	status.ServerIP = strings.TrimSpace(serverIP)

	osRelease, _, _ := client.Exec("cat /etc/os-release")
	for _, line := range strings.Split(osRelease, "\n") {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			status.OS = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
			break
		}
	}

	if client.CommandExists("apt-get") {
		status.PackageManager = "apt"
	} else if client.CommandExists("dnf") {
		status.PackageManager = "dnf"
	} else if client.CommandExists("yum") {
		status.PackageManager = "yum"
	} else if client.CommandExists("pacman") {
		status.PackageManager = "pacman"
	}

	status.HasSystemd = hasSystemd(client)

	s.serverInfoMu.Lock()
	s.serverInfoCache[serverID] = serverInfo{
		Hostname:       status.Hostname,
		ServerIP:       status.ServerIP,
		OS:             status.OS,
		PackageManager: status.PackageManager,
		HasSystemd:     status.HasSystemd,
	}
	s.serverInfoMu.Unlock()
}

// hasSystemd reports whether the host is booted with systemd and has
// systemctl available. Used to decide whether wg-quick@<iface> service
// management is possible.
func hasSystemd(client ssh.Executor) bool {
	if !client.CommandExists("systemctl") {
		return false
	}
	if _, _, err := client.ExecSilent("test -d /run/systemd/system"); err != nil {
		return false
	}
	return true
}

// isServiceEnabled reports whether wg-quick@<iface> is enabled in systemd.
// Only returns true when `systemctl is-enabled` prints exactly "enabled".
func isServiceEnabled(client ssh.Executor, iface string) bool {
	out, _, err := client.ExecSilentF("systemctl is-enabled wg-quick@%s 2>/dev/null", iface)
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) == "enabled"
}

func parseWGDump(output string) models.WGStatus {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	status := models.WGStatus{Interfaces: []models.WGInterface{}}

	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return status
	}

	ifaceMap := make(map[string]*models.WGInterface)
	var ifaceOrder []string

	for _, line := range lines {
		fields := strings.Split(line, "\t")

		// Interface line: interface, private-key, public-key, listen-port, fwmark (5 fields).
		// Peer line: interface, public-key, preshared-key, endpoint, allowed-ips,
		//            latest-handshake, transfer-rx, transfer-tx, persistent-keepalive (9 fields).
		if len(fields) <= 5 {
			ifaceName := fields[0]
			if _, ok := ifaceMap[ifaceName]; !ok {
				ifaceMap[ifaceName] = &models.WGInterface{
					Name:  ifaceName,
					Peers: []models.WGPeer{},
				}
				ifaceOrder = append(ifaceOrder, ifaceName)
			}
			iface := ifaceMap[ifaceName]
			iface.PrivateKey = fields[1]
			iface.PublicKey = fields[2]
			iface.ListenPort, _ = strconv.Atoi(fields[3])
			// fields[4] is fwmark (unused)
			continue
		}

		if len(fields) >= 8 {
			ifaceName := fields[0]
			iface, ok := ifaceMap[ifaceName]
			if !ok {
				continue
			}

			peer := models.WGPeer{
				PublicKey:    fields[1],
				PresharedKey: fields[2],
				Endpoint:     fields[3],
				AllowedIPs:   []string{},
			}
			if peer.Endpoint == "(none)" {
				peer.Endpoint = ""
			}

			allowedIPs := strings.Split(fields[4], ",")
			for _, ip := range allowedIPs {
				ip = strings.TrimSpace(ip)
				if ip != "" {
					peer.AllowedIPs = append(peer.AllowedIPs, ip)
				}
			}

			if fields[5] != "(none)" && fields[5] != "" {
				peer.LatestHandshake = parseTimestamp(fields[5])
			}

			peer.RxBytes, _ = strconv.ParseInt(fields[6], 10, 64)
			peer.TxBytes, _ = strconv.ParseInt(fields[7], 10, 64)

			if len(fields) > 8 {
				peer.PersistentKeepalive, _ = strconv.Atoi(fields[8])
			}

			iface.Peers = append(iface.Peers, peer)
			iface.RxBytes += peer.RxBytes
			iface.TxBytes += peer.TxBytes
		}
	}

	for _, name := range ifaceOrder {
		status.Interfaces = append(status.Interfaces, *ifaceMap[name])
	}

	return status
}

func configHasListenPort(configText string) bool {
	for _, line := range strings.Split(configText, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "ListenPort") {
			return true
		}
	}
	return false
}

func parseInterfaceConfig(configText, ifaceName string) models.WGInterface {
	iface := models.WGInterface{
		Name:   ifaceName,
		Online: false,
		Peers:  []models.WGPeer{},
	}

	inPeerSection := false
	var currentPeer *models.WGPeer

	for _, line := range strings.Split(configText, "\n") {
		trimmed := strings.TrimSpace(line)

		if trimmed == "[Peer]" {
			if currentPeer != nil {
				iface.Peers = append(iface.Peers, *currentPeer)
			}
			currentPeer = &models.WGPeer{AllowedIPs: []string{}}
			inPeerSection = true
			continue
		}

		if trimmed == "[Interface]" {
			if currentPeer != nil {
				iface.Peers = append(iface.Peers, *currentPeer)
				currentPeer = nil
			}
			inPeerSection = false
			continue
		}

		if trimmed == "" || strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(strings.ToLower(trimmed), "# name") && !strings.HasPrefix(strings.ToLower(trimmed), "# description") {
			continue
		}

		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		val := strings.TrimSpace(parts[1])

		if !inPeerSection {
			switch key {
			case "privatekey":
				iface.PrivateKey = val
			case "listenport":
				iface.ListenPort, _ = strconv.Atoi(val)
			case "endpoint":
				iface.Endpoint = val
			}
		} else if currentPeer != nil {
			switch key {
			case "publickey":
				currentPeer.PublicKey = val
			case "endpoint":
				currentPeer.Endpoint = val
			case "allowedips":
				for _, ip := range strings.Split(val, ",") {
					ip = strings.TrimSpace(ip)
					if ip != "" {
						currentPeer.AllowedIPs = append(currentPeer.AllowedIPs, ip)
					}
				}
			case "presharedkey":
				currentPeer.PresharedKey = val
			case "persistentkeepalive":
				currentPeer.PersistentKeepalive, _ = strconv.Atoi(val)
			case "# name":
				currentPeer.Name = val
			case "# description":
				currentPeer.Description = val
			}
		}
	}

	if currentPeer != nil {
		iface.Peers = append(iface.Peers, *currentPeer)
	}

	return iface
}

func parseTimestamp(s string) time.Time {
	ts, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(ts, 0)
}
