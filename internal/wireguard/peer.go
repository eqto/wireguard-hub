package wireguard

import (
	"fmt"
	"strings"

	"wireguardhub/internal/models"
)

func (s *Service) AddPeer(req models.AddPeerRequest) (models.AddPeerResult, error) {
	client, err := s.serverSvc.GetClient(req.ServerID)
	if err != nil {
		return models.AddPeerResult{}, err
	}

	// Check if this is a client interface (no ListenPort in config).
	confPath := fmt.Sprintf("/etc/wireguard/%s.conf", req.Interface)
	confText, _, _ := client.ExecSilentF("sudo cat %s", confPath)
	isClientInterface := !configHasListenPort(confText)

	if isClientInterface {
		if req.Endpoint == "" {
			return models.AddPeerResult{}, fmt.Errorf("client interface requires peer endpoint")
		}
		peersOut, _, _ := client.ExecF("sudo wg show %s peers", req.Interface)
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
	existingConf, _, _ := client.ExecSilentF("sudo cat %s", confPath)

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
	_, stderr, err = client.ExecWithInputSilentF(updatedConf, "sudo tee %s > /dev/null", confPath)
	if err != nil {
		return models.AddPeerResult{}, fmt.Errorf("failed to update config file: %s: %w", stderr, err)
	}

	clientConfig := generateClientConfig(privKey, pubKey, req)

	return models.AddPeerResult{
		PublicKey: pubKey,
		Config:    clientConfig,
	}, nil
}

func generateClientConfig(privKey string, pubKey string, req models.AddPeerRequest) string {
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

	_, stderr, err := client.ExecF("sudo wg set %s peer %s remove", iface, publicKey)
	if err != nil {
		return false, fmt.Errorf("failed to remove peer: %s: %w", stderr, err)
	}

	s.SyncConfig(serverID, iface)

	// Remove the [Peer] section from the config file.
	confPath := fmt.Sprintf("/etc/wireguard/%s.conf", iface)
	existingConf, _, _ := client.ExecSilentF("sudo cat %s", confPath)
	updatedConf := removePeerSection(existingConf, publicKey)
	_, stderr, err = client.ExecWithInputSilentF(updatedConf, "sudo tee %s > /dev/null", confPath)
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

	// If not restarting, apply endpoint/allowedIPs changes via wg set on the live interface.
	if !req.Restart && (req.Endpoint != "" || len(req.AllowedIPs) > 0) {
		cmdParts := []string{
			fmt.Sprintf("sudo wg set %s peer %s", req.Interface, req.PublicKey),
		}
		if len(req.AllowedIPs) > 0 {
			cmdParts = append(cmdParts, fmt.Sprintf("allowed-ips %s", strings.Join(req.AllowedIPs, ",")))
		}
		if req.Endpoint != "" {
			cmdParts = append(cmdParts, fmt.Sprintf("endpoint %s", req.Endpoint))
		}
		cmd := strings.Join(cmdParts, " ")
		_, stderr, err := client.Exec(cmd)
		if err != nil {
			return false, fmt.Errorf("failed to update peer: %s: %w", stderr, err)
		}
	}

	confPath := fmt.Sprintf("/etc/wireguard/%s.conf", req.Interface)
	existingConf, stderr, err := client.ExecSilentF("sudo cat %s", confPath)
	if err != nil {
		return false, fmt.Errorf("failed to read config: %s: %w", stderr, err)
	}

	updatedConf := updatePeerMetaInConfig(existingConf, req.PublicKey, req.Name, req.Description)
	if updatedConf == "" {
		return false, fmt.Errorf("peer not found in config: %s", req.PublicKey)
	}

	// Also update Endpoint and AllowedIPs in the config file if provided.
	if req.Endpoint != "" {
		updatedConf = updatePeerFieldInConfig(updatedConf, req.PublicKey, "Endpoint", req.Endpoint)
	}
	if len(req.AllowedIPs) > 0 {
		updatedConf = updatePeerFieldInConfig(updatedConf, req.PublicKey, "AllowedIPs", strings.Join(req.AllowedIPs, ","))
	}

	if req.NewPublicKey != "" && req.NewPublicKey != req.PublicKey {
		updatedConf = updatePeerFieldInConfig(updatedConf, req.PublicKey, "PublicKey", req.NewPublicKey)
	}

	_, stderr, err = client.ExecWithInputSilentF(updatedConf, "sudo tee %s > /dev/null", confPath)
	if err != nil {
		return false, fmt.Errorf("failed to write config: %s: %w", stderr, err)
	}

	// If restart requested, bring interface down and back up to apply changes.
	if req.Restart {
		_, stderr, err := client.ExecF("sudo wg-quick down %s", req.Interface)
		if err != nil {
			return false, fmt.Errorf("failed to bring down interface: %s: %w", stderr, err)
		}
		_, stderr, err = client.ExecF("sudo wg-quick up %s", req.Interface)
		if err != nil {
			return false, fmt.Errorf("failed to bring up interface: %s: %w", stderr, err)
		}
	}

	return true, nil
}

// removePeerSection removes the [Peer] section matching the given public key from the config text.
func removePeerSection(configText string, publicKey string) string {
	cfg := parseWGConfig(configText)
	cfg.removePeer(publicKey)
	return cfg.serialize()
}

// updatePeerFieldInConfig updates a field (e.g. Endpoint, AllowedIPs) in the [Peer] section
// matching the given public key. Returns the original text if peer not found.
func updatePeerFieldInConfig(configText string, publicKey string, field string, value string) string {
	cfg := parseWGConfig(configText)
	peer := cfg.findPeer(publicKey)
	if peer == nil {
		return configText
	}
	peer.set(field, value)
	return cfg.serialize()
}

// updatePeerMetaInConfig adds or updates # Name and # Description comments in the
// [Peer] section matching the given public key. Returns empty string if peer not found.
func updatePeerMetaInConfig(configText string, publicKey string, name string, description string) string {
	cfg := parseWGConfig(configText)
	peer := cfg.findPeer(publicKey)
	if peer == nil {
		return ""
	}
	peer.setMetadata(name, description)
	return cfg.serialize()
}
