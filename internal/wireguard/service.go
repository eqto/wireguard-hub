package wireguard

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"wireguardadmin/internal/models"
	"wireguardadmin/internal/server"
)

type Service struct {
	serverSvc *server.Service
}

func NewService(serverSvc *server.Service) *Service {
	return &Service{serverSvc: serverSvc}
}

func (s *Service) GetStatus(serverID string) (models.WGStatus, error) {
	client, err := s.serverSvc.GetClient(serverID)
	if err != nil {
		return models.WGStatus{}, err
	}

	stdout, stderr, err := client.Exec("sudo wg show all dump")
	if err != nil {
		return models.WGStatus{}, fmt.Errorf("failed to get wg status: %s: %w", stderr, err)
	}

	status := parseWGDump(stdout)
	return status, nil
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
		if len(fields) < 8 {
			continue
		}

		if len(fields) == 8 {
			ifaceName := fields[0]
			if _, ok := ifaceMap[ifaceName]; !ok {
				ifaceMap[ifaceName] = &models.WGInterface{
					Name:   ifaceName,
					Peers:  []models.WGPeer{},
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
				PublicKey:  fields[1],
				PresharedKey: fields[2],
				Endpoint:   fields[3],
				AllowedIPs: []string{},
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
	conf.WriteString(fmt.Sprintf("ListenPort = %d\n", req.ListenPort))
	if req.Endpoint != "" {
		conf.WriteString(fmt.Sprintf("Endpoint = %s\n", req.Endpoint))
	}
	conf.WriteString("\n")

	confPath := fmt.Sprintf("/etc/wireguard/%s.conf", req.Name)

	_, stderr, err = client.Exec(fmt.Sprintf("sudo tee %s > /dev/null << 'WGCONF'\n%s\nWGCONF", confPath, conf.String()))
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

	return true, nil
}
