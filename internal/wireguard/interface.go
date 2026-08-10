package wireguard

import (
	"fmt"
	"strings"

	"wireguardhub/internal/models"
	"wireguardhub/internal/ssh"
)

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

	_, stderr, err = client.ExecWithInputSilentF(conf.String(), "sudo tee %s > /dev/null", confPath)
	if err != nil {
		return models.WGInterface{}, fmt.Errorf("failed to write config file: %s: %w", stderr, err)
	}

	// Bring up the interface. When the user requested "run as service" and
	// systemd is available, enable + start the wg-quick@<name> unit so it
	// auto-starts on boot. Otherwise fall back to wg-quick up directly.
	serviceEnabled := false
	if req.EnableService && hasSystemd(client) {
		_, stderr, err = client.ExecF("sudo systemctl enable --now wg-quick@%s", req.Name)
		if err != nil {
			return models.WGInterface{}, fmt.Errorf("failed to enable wg-quick@%s service: %s: %w", req.Name, stderr, err)
		}
		serviceEnabled = true
	} else {
		_, stderr, err = client.ExecF("sudo wg-quick up %s", req.Name)
		if err != nil {
			return models.WGInterface{}, fmt.Errorf("failed to bring up interface: %s: %w", stderr, err)
		}
	}

	return models.WGInterface{
		Name:           req.Name,
		PublicKey:      pubKey,
		PrivateKey:     privKey,
		ListenPort:     req.ListenPort,
		Endpoint:       req.Endpoint,
		Peers:          []models.WGPeer{},
		ServiceEnabled: serviceEnabled,
	}, nil
}

// EnableService enables the wg-quick@<name> systemd unit so the interface
// auto-starts on boot. If the interface is currently up via wg-quick, it is
// transitioned to systemd management atomically in a single command
// (wg-quick down && systemctl enable --now) to avoid leaving the interface
// down with the service not started.
func (s *Service) EnableService(serverID string, name string) (bool, error) {
	client, err := s.serverSvc.GetClient(serverID)
	if err != nil {
		return false, err
	}
	if !hasSystemd(client) {
		return false, fmt.Errorf("systemd not available on this server")
	}

	// Check if the interface is currently up via wg-quick (not via systemctl).
	_, _, showErr := client.ExecSilentF("sudo wg show %s", name)
	ifaceOnline := showErr == nil
	alreadyEnabled := isServiceEnabled(client, name)

	if ifaceOnline && !alreadyEnabled {
		// Atomic transition: tear down wg-quick, then enable+start the unit.
		// Wrap both commands in a single `sudo bash -c` so the sudo password
		// (fed via stdin with -S) is only needed once. Chaining two separate
		// `sudo` calls with && would leave the second one without -S and no TTY
		// to prompt for the password, causing it to fail for non-root users.
		_, stderr, err := client.ExecF("sudo bash -c 'wg-quick down %s && systemctl enable --now wg-quick@%s'", name, name)
		if err != nil {
			return false, fmt.Errorf("failed to enable service: %s: %w", stderr, err)
		}
		return true, nil
	}

	// Either already enabled, or interface is down. If already enabled, this
	// is a no-op. Otherwise enable AND start the unit now (--now) so the
	// interface comes up immediately, matching the create-with-service flow.
	if alreadyEnabled {
		return true, nil
	}
	_, stderr, err := client.ExecF("sudo systemctl enable --now wg-quick@%s", name)
	if err != nil {
		return false, fmt.Errorf("failed to enable service: %s: %w", stderr, err)
	}
	return true, nil
}

// DisableService disables the wg-quick@<name> systemd unit so the interface
// no longer auto-starts on boot. This does not change the current run state
// of the interface.
func (s *Service) DisableService(serverID string, name string) (bool, error) {
	client, err := s.serverSvc.GetClient(serverID)
	if err != nil {
		return false, err
	}
	if !hasSystemd(client) {
		return false, fmt.Errorf("systemd not available on this server")
	}

	_, stderr, err := client.ExecF("sudo systemctl disable wg-quick@%s", name)
	if err != nil {
		return false, fmt.Errorf("failed to disable service: %s: %w", stderr, err)
	}
	return true, nil
}

// runInterfaceCmd runs either the systemctl or wg-quick command for an
// interface, depending on whether systemd is available and the wg-quick@
// unit is enabled. systemctlAction and wgQuickAction are used in the error
// message for the respective branch.
func runInterfaceCmd(client ssh.Executor, name, systemctlAction, wgQuickAction, systemctlFmt, wgQuickFmt string) error {
	if hasSystemd(client) && isServiceEnabled(client, name) {
		_, stderr, err := client.ExecF(systemctlFmt, name, name)
		if err != nil {
			return fmt.Errorf("failed to %s: %s: %w", systemctlAction, stderr, err)
		}
		return nil
	}
	_, stderr, err := client.ExecF(wgQuickFmt, name, name)
	if err != nil {
		return fmt.Errorf("failed to %s: %s: %w", wgQuickAction, stderr, err)
	}
	return nil
}

func (s *Service) BringUpInterface(serverID string, name string) (bool, error) {
	client, err := s.serverSvc.GetClient(serverID)
	if err != nil {
		return false, err
	}

	// When the wg-quick@<name> service is enabled, start via systemctl so
	// systemd tracks the unit state. Otherwise fall back to wg-quick up.
	// wg-quick@.service is a oneshot with RemainAfterExit=yes. After the
	// interface is brought up, the service stays "active (exited)" even if
	// the interface later goes down (manually, crash, etc.). In that state
	// `systemctl start` is a silent no-op — the interface never comes up.
	// Fix: stop first (reset the unit state, ignoring errors if the
	// interface is already down), then start. Wrapped in a single
	// `sudo bash -c` so the sudo password is only needed once.
	if err := runInterfaceCmd(client, name,
		"start service", "bring up interface",
		"sudo bash -c 'systemctl stop wg-quick@%s 2>/dev/null; systemctl start wg-quick@%s'",
		"sudo wg-quick up %s"); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) BringDownInterface(serverID string, name string) (bool, error) {
	client, err := s.serverSvc.GetClient(serverID)
	if err != nil {
		return false, err
	}

	// When the wg-quick@<name> service is enabled, stop via systemctl so
	// systemd tracks the unit state. Otherwise fall back to wg-quick down.
	if err := runInterfaceCmd(client, name,
		"stop service", "bring down interface",
		"sudo systemctl stop wg-quick@%s",
		"sudo wg-quick down %s"); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) RestartInterface(serverID string, name string) (bool, error) {
	client, err := s.serverSvc.GetClient(serverID)
	if err != nil {
		return false, err
	}

	// When the wg-quick@<name> service is enabled, restart via systemctl so
	// systemd tracks the unit state. Use stop+start (with ';') instead of
	// restart, because restart may abort if the stop step fails when the
	// interface is already down (see BringUpInterface for details).
	// Wrap in a single `sudo bash -c` so the sudo password is only needed once
	// (see EnableService for details on the chained-sudo password issue).
	if err := runInterfaceCmd(client, name,
		"restart service", "restart interface",
		"sudo bash -c 'systemctl stop wg-quick@%s 2>/dev/null; systemctl start wg-quick@%s'",
		"sudo bash -c 'wg-quick down %s && wg-quick up %s'"); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) DeleteInterface(serverID string, name string) (bool, error) {
	client, err := s.serverSvc.GetClient(serverID)
	if err != nil {
		return false, err
	}

	// If the wg-quick@<name> service is enabled, disable + stop it in one
	// step so it won't auto-start on boot and the unit is torn down cleanly.
	if err := runInterfaceCmd(client, name,
		"disable service", "bring down interface",
		"sudo systemctl disable --stop wg-quick@%s",
		"sudo wg-quick down %s"); err != nil {
		return false, err
	}

	_, stderr, err := client.ExecF("sudo rm -f /etc/wireguard/%s.conf", name)
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

	stdout, stderr, err := client.ExecSilentF("sudo cat /etc/wireguard/%s.conf", name)
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

	// Pipe the stripped config into wg syncconf via /dev/stdin. Avoid process
	// substitution <(...) because sudo closes non-standard file descriptors, so
	// the /dev/fd/N path created by <(...) is not accessible to wg, causing
	// "fopen: No such file or directory". /dev/stdin (fd 0) is always available.
	_, stderr, err := client.ExecF("sudo bash -c 'wg-quick strip %s | wg syncconf %s /dev/stdin'", name, name)
	if err != nil {
		return false, fmt.Errorf("failed to sync config: %s: %w", stderr, err)
	}

	return true, nil
}
