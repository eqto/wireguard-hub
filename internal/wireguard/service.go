package wireguard

import (
	"fmt"
	"io"
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
	_, _, aptErr := client.Exec("command -v apt-get")
	if aptErr == nil {
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

	_, _, dnfErr := client.Exec("command -v dnf")
	if dnfErr == nil {
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

	_, _, yumErr := client.Exec("command -v yum")
	if yumErr == nil {
		session, err := client.ExecStreaming("sudo yum install -y epel-release 2>&1 && sudo yum install -y wireguard-tools 2>&1", emit)
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

	_, _, pacmanErr := client.Exec("command -v pacman")
	if pacmanErr == nil {
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

	stdout, stderr, err := client.Exec("sudo wg show all dump")
	if err != nil {
		// Check if wg binary exists on the server
		_, _, wgErr := client.Exec("command -v wg")
		if wgErr != nil {
			status := models.WGStatus{
				Interfaces:     []models.WGInterface{},
				WGNotInstalled: true,
			}
			s.fillServerInfo(serverID, client, &status)
			return status, nil
		}
		_ = stderr
		status := models.WGStatus{Interfaces: []models.WGInterface{}}
		s.fillServerInfo(serverID, client, &status)
		return status, nil
	}

	status := parseWGDump(stdout)

	// Merge peer metadata from config files on the server.
	for i := range status.Interfaces {
		confStdout, _, confErr := client.Exec(fmt.Sprintf("sudo cat /etc/wireguard/%s.conf", status.Interfaces[i].Name))
		if confErr != nil {
			continue
		}
		metaMap := parsePeerMeta(confStdout)
		for j := range status.Interfaces[i].Peers {
			if meta, ok := metaMap[status.Interfaces[i].Peers[j].PublicKey]; ok {
				status.Interfaces[i].Peers[j].Name = meta.Name
				status.Interfaces[i].Peers[j].Description = meta.Description
			}
		}
	}

	s.fillServerInfo(serverID, client, &status)

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

	if _, _, err := client.Exec("command -v apt-get"); err == nil {
		status.PackageManager = "apt"
	} else if _, _, err := client.Exec("command -v dnf"); err == nil {
		status.PackageManager = "dnf"
	} else if _, _, err := client.Exec("command -v yum"); err == nil {
		status.PackageManager = "yum"
	} else if _, _, err := client.Exec("command -v pacman"); err == nil {
		status.PackageManager = "pacman"
	}

	s.serverInfoMu.Lock()
	s.serverInfoCache[serverID] = serverInfo{
		Hostname:       status.Hostname,
		ServerIP:       status.ServerIP,
		OS:             status.OS,
		PackageManager: status.PackageManager,
	}
	s.serverInfoMu.Unlock()
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

// parsePeerMeta parses a WireGuard config file and extracts # Name and # Description
// comments from each [Peer] section, keyed by the peer's public key.
func parsePeerMeta(configText string) map[string]models.WGPeer {
	result := make(map[string]models.WGPeer)
	lines := strings.Split(configText, "\n")

	var currentPubKey string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "[Peer]" {
			currentPubKey = ""
			continue
		}

		if currentPubKey != "" || strings.HasPrefix(trimmed, "PublicKey") || strings.HasPrefix(trimmed, "# Name") || strings.HasPrefix(trimmed, "# Description") {
			// Check for PublicKey to associate metadata
			if strings.HasPrefix(trimmed, "PublicKey") {
				parts := strings.SplitN(trimmed, "=", 2)
				if len(parts) == 2 {
					currentPubKey = strings.TrimSpace(parts[1])
					if _, ok := result[currentPubKey]; !ok {
						result[currentPubKey] = models.WGPeer{PublicKey: currentPubKey}
					}
				}
				continue
			}

			if strings.HasPrefix(trimmed, "# Name") {
				parts := strings.SplitN(trimmed, "=", 2)
				if len(parts) == 2 && currentPubKey != "" {
					p := result[currentPubKey]
					p.Name = strings.TrimSpace(parts[1])
					result[currentPubKey] = p
				}
				continue
			}

			if strings.HasPrefix(trimmed, "# Description") {
				parts := strings.SplitN(trimmed, "=", 2)
				if len(parts) == 2 && currentPubKey != "" {
					p := result[currentPubKey]
					p.Description = strings.TrimSpace(parts[1])
					result[currentPubKey] = p
				}
				continue
			}
		}
	}

	return result
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

func parseTimestamp(s string) time.Time {
	ts, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(ts, 0)
}

func (s *Service) GenerateKeyPair(serverID string) (models.KeyPair, error) {
	client, err := s.serverSvc.GetClient(serverID)
	if err != nil {
		return models.KeyPair{}, err
	}

	privKey, stderr, err := client.Exec("wg genkey")
	if err != nil {
		return models.KeyPair{}, fmt.Errorf("failed to generate private key: %s: %w", stderr, err)
	}
	privKey = strings.TrimSpace(privKey)

	pubKey, stderr, err := client.ExecWithInput("wg pubkey", privKey)
	if err != nil {
		return models.KeyPair{}, fmt.Errorf("failed to derive public key: %s: %w", stderr, err)
	}
	pubKey = strings.TrimSpace(pubKey)

	return models.KeyPair{
		PublicKey:  pubKey,
		PrivateKey: privKey,
	}, nil
}

func (s *Service) CreateInterface(req models.CreateInterfaceRequest) (models.WGInterface, error) {
	client, err := s.serverSvc.GetClient(req.ServerID)
	if err != nil {
		return models.WGInterface{}, err
	}

	privKey := req.PrivateKey
	if privKey == "" {
		kp, err := s.GenerateKeyPair(req.ServerID)
		if err != nil {
			return models.WGInterface{}, err
		}
		privKey = kp.PrivateKey
	}

	pubKey, stderr, err := client.ExecWithInput("wg pubkey", privKey)
	if err != nil {
		return models.WGInterface{}, fmt.Errorf("failed to derive public key: %s: %w", stderr, err)
	}
	pubKey = strings.TrimSpace(pubKey)

	address := req.Address
	if address == "" {
		address = "10.0.0.1/24"
	}

	var conf strings.Builder
	conf.WriteString("[Interface]\n")
	conf.WriteString(fmt.Sprintf("PrivateKey = %s\n", privKey))
	conf.WriteString(fmt.Sprintf("Address = %s\n", address))
	if req.ListenPort > 0 {
		conf.WriteString(fmt.Sprintf("ListenPort = %d\n", req.ListenPort))
	}
	if req.Endpoint != "" {
		conf.WriteString(fmt.Sprintf("Endpoint = %s\n", req.Endpoint))
	}
	conf.WriteString("\n")

	confPath := fmt.Sprintf("/etc/wireguard/%s.conf", req.Name)

	_, stderr, err = client.ExecWithInput(fmt.Sprintf("sudo tee %s > /dev/null", confPath), conf.String())
	if err != nil {
		return models.WGInterface{}, fmt.Errorf("failed to write config file: %s: %w", stderr, err)
	}

	_, stderr, err = client.Exec(fmt.Sprintf("sudo wg-quick up %s", req.Name))
	if err != nil {
		return models.WGInterface{}, fmt.Errorf("failed to bring up interface: %s: %w", stderr, err)
	}

	return models.WGInterface{
		Name:       req.Name,
		PublicKey:  pubKey,
		PrivateKey: privKey,
		ListenPort: req.ListenPort,
		Endpoint:   req.Endpoint,
		Peers:      []models.WGPeer{},
	}, nil
}

func (s *Service) DeleteInterface(serverID string, name string) (bool, error) {
	client, err := s.serverSvc.GetClient(serverID)
	if err != nil {
		return false, err
	}

	_, stderr, err := client.Exec(fmt.Sprintf("sudo wg-quick down %s", name))
	if err != nil {
		return false, fmt.Errorf("failed to bring down interface: %s: %w", stderr, err)
	}

	_, stderr, err = client.Exec(fmt.Sprintf("sudo rm -f /etc/wireguard/%s.conf", name))
	if err != nil {
		return false, fmt.Errorf("failed to remove config file: %s: %w", stderr, err)
	}

	return true, nil
}

func (s *Service) GetInterfaceConfig(serverID string, name string) (string, error) {
	client, err := s.serverSvc.GetClient(serverID)
	if err != nil {
		return "", err
	}

	stdout, stderr, err := client.Exec(fmt.Sprintf("sudo cat /etc/wireguard/%s.conf", name))
	if err != nil {
		return "", fmt.Errorf("failed to read config: %s: %w", stderr, err)
	}

	return stdout, nil
}

func (s *Service) SyncConfig(serverID string, name string) (bool, error) {
	client, err := s.serverSvc.GetClient(serverID)
	if err != nil {
		return false, err
	}

	cmd := fmt.Sprintf("sudo wg syncconf %s <(sudo wg-quick strip %s)", name, name)
	_, stderr, err := client.Exec(cmd)
	if err != nil {
		return false, fmt.Errorf("failed to sync config: %s: %w", stderr, err)
	}

	return true, nil
}

func (s *Service) AddPeer(req models.AddPeerRequest) (models.AddPeerResult, error) {
	client, err := s.serverSvc.GetClient(req.ServerID)
	if err != nil {
		return models.AddPeerResult{}, err
	}

	// Check if this is a client interface (no ListenPort in config).
	confPath := fmt.Sprintf("/etc/wireguard/%s.conf", req.Interface)
	confText, _, _ := client.Exec(fmt.Sprintf("sudo cat %s", confPath))
	isClientInterface := !configHasListenPort(confText)

	if isClientInterface {
		if req.Endpoint == "" {
			return models.AddPeerResult{}, fmt.Errorf("client interface requires peer endpoint")
		}
		peersOut, _, _ := client.Exec(fmt.Sprintf("sudo wg show %s peers", req.Interface))
		if strings.TrimSpace(peersOut) != "" {
			return models.AddPeerResult{}, fmt.Errorf("client interface can only have one server peer")
		}
	}

	pubKey := req.PublicKey
	var privKey string

	if pubKey == "" {
		kp, err := s.GenerateKeyPair(req.ServerID)
		if err != nil {
			return models.AddPeerResult{}, err
		}
		pubKey = kp.PublicKey
		privKey = kp.PrivateKey
	}

	allowedIPs := strings.Join(req.AllowedIPs, ",")

	cmdParts := []string{
		fmt.Sprintf("sudo wg set %s peer %s", req.Interface, pubKey),
		fmt.Sprintf("allowed-ips %s", allowedIPs),
	}

	if req.PresharedKey != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("preshared-key <(echo '%s')", req.PresharedKey))
	}
	if req.Endpoint != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("endpoint %s", req.Endpoint))
	}
	if req.PersistentKeepalive > 0 {
		cmdParts = append(cmdParts, fmt.Sprintf("persistent-keepalive %d", req.PersistentKeepalive))
	}

	cmd := strings.Join(cmdParts, " ")
	_, stderr, err := client.Exec(cmd)
	if err != nil {
		return models.AddPeerResult{}, fmt.Errorf("failed to add peer: %s: %w", stderr, err)
	}

	s.SyncConfig(req.ServerID, req.Interface)

	// Append the [Peer] section to the config file for persistence across reboots.
	existingConf, _, _ := client.Exec(fmt.Sprintf("sudo cat %s", confPath))

	var peerSection strings.Builder
	peerSection.WriteString("\n[Peer]\n")
	peerSection.WriteString(fmt.Sprintf("PublicKey = %s\n", pubKey))
	if req.Name != "" {
		peerSection.WriteString(fmt.Sprintf("# Name = %s\n", req.Name))
	}
	if req.Description != "" {
		peerSection.WriteString(fmt.Sprintf("# Description = %s\n", req.Description))
	}
	peerSection.WriteString(fmt.Sprintf("AllowedIPs = %s\n", strings.Join(req.AllowedIPs, ",")))
	if req.PresharedKey != "" {
		peerSection.WriteString(fmt.Sprintf("PresharedKey = %s\n", req.PresharedKey))
	}
	if req.Endpoint != "" {
		peerSection.WriteString(fmt.Sprintf("Endpoint = %s\n", req.Endpoint))
	}
	if req.PersistentKeepalive > 0 {
		peerSection.WriteString(fmt.Sprintf("PersistentKeepalive = %d\n", req.PersistentKeepalive))
	}

	updatedConf := strings.TrimRight(existingConf, "\n") + "\n" + peerSection.String()
	_, stderr, err = client.ExecWithInput(fmt.Sprintf("sudo tee %s > /dev/null", confPath), updatedConf)
	if err != nil {
		return models.AddPeerResult{}, fmt.Errorf("failed to update config file: %s: %w", stderr, err)
	}

	serverIP, _, _ := client.Exec(fmt.Sprintf("sudo wg show %s listen-port | head -1 && hostname -I | awk '{print $1}'", req.Interface))

	clientConfig := generateClientConfig(privKey, pubKey, req, serverIP)

	return models.AddPeerResult{
		PublicKey: pubKey,
		Config:    clientConfig,
	}, nil
}

func generateClientConfig(privKey string, pubKey string, req models.AddPeerRequest, serverInfo string) string {
	var conf strings.Builder
	conf.WriteString("[Interface]\n")
	if privKey != "" {
		conf.WriteString(fmt.Sprintf("PrivateKey = %s\n", privKey))
	}
	if len(req.AllowedIPs) > 0 {
		conf.WriteString(fmt.Sprintf("Address = %s\n", req.AllowedIPs[0]))
	}
	conf.WriteString("\n[Peer]\n")
	conf.WriteString(fmt.Sprintf("PublicKey = %s\n", pubKey))
	if len(req.AllowedIPs) > 0 {
		conf.WriteString(fmt.Sprintf("AllowedIPs = %s\n", strings.Join(req.AllowedIPs, ", ")))
	}
	if req.Endpoint != "" {
		conf.WriteString(fmt.Sprintf("Endpoint = %s\n", req.Endpoint))
	}
	if req.PersistentKeepalive > 0 {
		conf.WriteString(fmt.Sprintf("PersistentKeepalive = %d\n", req.PersistentKeepalive))
	}
	return conf.String()
}

func (s *Service) RemovePeer(serverID string, iface string, publicKey string) (bool, error) {
	client, err := s.serverSvc.GetClient(serverID)
	if err != nil {
		return false, err
	}

	_, stderr, err := client.Exec(fmt.Sprintf("sudo wg set %s peer %s remove", iface, publicKey))
	if err != nil {
		return false, fmt.Errorf("failed to remove peer: %s: %w", stderr, err)
	}

	s.SyncConfig(serverID, iface)

	// Remove the [Peer] section from the config file.
	confPath := fmt.Sprintf("/etc/wireguard/%s.conf", iface)
	existingConf, _, _ := client.Exec(fmt.Sprintf("sudo cat %s", confPath))
	updatedConf := removePeerSection(existingConf, publicKey)
	_, stderr, err = client.ExecWithInput(fmt.Sprintf("sudo tee %s > /dev/null", confPath), updatedConf)
	if err != nil {
		return false, fmt.Errorf("failed to update config file: %s: %w", stderr, err)
	}

	return true, nil
}

func (s *Service) UpdatePeerMeta(req models.UpdatePeerMetaRequest) (bool, error) {
	client, err := s.serverSvc.GetClient(req.ServerID)
	if err != nil {
		return false, err
	}

	confPath := fmt.Sprintf("/etc/wireguard/%s.conf", req.Interface)
	existingConf, stderr, err := client.Exec(fmt.Sprintf("sudo cat %s", confPath))
	if err != nil {
		return false, fmt.Errorf("failed to read config: %s: %w", stderr, err)
	}

	updatedConf := updatePeerMetaInConfig(existingConf, req.PublicKey, req.Name, req.Description)
	if updatedConf == "" {
		return false, fmt.Errorf("peer not found in config: %s", req.PublicKey)
	}

	_, stderr, err = client.ExecWithInput(fmt.Sprintf("sudo tee %s > /dev/null", confPath), updatedConf)
	if err != nil {
		return false, fmt.Errorf("failed to write config: %s: %w", stderr, err)
	}

	return true, nil
}

// removePeerSection removes the [Peer] section matching the given public key from the config text.
func removePeerSection(configText string, publicKey string) string {
	lines := strings.Split(configText, "\n")
	var result []string

	inPeerSection := false
	skipSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "[Peer]" {
			inPeerSection = true
			skipSection = false
			// Look ahead: we need to buffer lines until we find the PublicKey
			// We'll add the [Peer] line tentatively and remove later if it matches
			result = append(result, line)
			continue
		}

		if trimmed == "[Interface]" {
			inPeerSection = false
			skipSection = false
			result = append(result, line)
			continue
		}

		if inPeerSection && strings.HasPrefix(trimmed, "PublicKey") {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 && strings.TrimSpace(parts[1]) == publicKey {
				// Remove the buffered [Peer] line and skip this entire section
				result = result[:len(result)-1] // remove the "[Peer]" line we added
				skipSection = true
				continue
			}
		}

		if skipSection {
			continue
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// updatePeerMetaInConfig adds or updates # Name and # Description comments in the
// [Peer] section matching the given public key. Returns empty string if peer not found.
func updatePeerMetaInConfig(configText string, publicKey string, name string, description string) string {
	lines := strings.Split(configText, "\n")

	inPeerSection := false
	foundPeer := false
	pubKeyLineIdx := -1

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "[Peer]" {
			inPeerSection = true
			foundPeer = false
			pubKeyLineIdx = -1
			continue
		}

		if trimmed == "[Interface]" {
			inPeerSection = false
			continue
		}

		if inPeerSection && strings.HasPrefix(trimmed, "PublicKey") {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 && strings.TrimSpace(parts[1]) == publicKey {
				foundPeer = true
				pubKeyLineIdx = i
			}
		}
	}

	if !foundPeer || pubKeyLineIdx == -1 {
		return ""
	}

	// Find the extent of this peer's metadata comments (Name/Description right after PublicKey)
	// and also find where the next non-comment, non-empty line is (end of metadata zone).
	hasName := false
	hasDescription := false
	nameLineIdx := -1
	descLineIdx := -1

	for i := pubKeyLineIdx + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "# Name") {
			hasName = true
			nameLineIdx = i
		} else if strings.HasPrefix(trimmed, "# Description") {
			hasDescription = true
			descLineIdx = i
		} else if strings.HasPrefix(trimmed, "#") || trimmed == "" {
			continue
		} else {
			break
		}
	}

	// Build replacement lines for metadata
	var metaLines []string
	if name != "" {
		metaLines = append(metaLines, fmt.Sprintf("# Name = %s", name))
	}
	if description != "" {
		metaLines = append(metaLines, fmt.Sprintf("# Description = %s", description))
	}

	// Determine the range to replace
	startIdx := pubKeyLineIdx + 1
	endIdx := pubKeyLineIdx // exclusive end, nothing to remove by default

	// Find the range of existing Name/Description comments
	if hasName && hasDescription {
		startIdx = min(nameLineIdx, descLineIdx)
		endIdx = max(nameLineIdx, descLineIdx) + 1
	} else if hasName {
		startIdx = nameLineIdx
		endIdx = nameLineIdx + 1
	} else if hasDescription {
		startIdx = descLineIdx
		endIdx = descLineIdx + 1
	}

	// Replace the range with new metadata lines
	var newLines []string
	newLines = append(newLines, lines[:startIdx]...)
	newLines = append(newLines, metaLines...)
	newLines = append(newLines, lines[endIdx:]...)

	return strings.Join(newLines, "\n")
}
