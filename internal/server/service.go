package server

import (
	"fmt"
	"strings"
	"sync"

	"wireguardhub/internal/config"
	"wireguardhub/internal/local"
	"wireguardhub/internal/models"
	"wireguardhub/internal/ssh"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const LocalServerID = "local"

type Service struct {
	mu          sync.RWMutex
	servers     []models.ServerConfig
	clients     map[string]ssh.Executor
	localConfig *models.LocalConfig
	localClient *local.Client
}

func NewService() *Service {
	s := &Service{
		clients: make(map[string]ssh.Executor),
	}
	s.load()
	s.loadLocalConfig()
	return s
}

func (s *Service) load() error {
	servers, err := config.Load()
	if err != nil {
		return err
	}
	s.servers = servers
	return nil
}

func (s *Service) save() error {
	return config.Save(s.servers)
}

func (s *Service) loadLocalConfig() {
	cfg, err := config.LoadLocalConfig()
	if err != nil {
		return
	}
	s.localConfig = cfg
}

func (s *Service) GetServers() ([]models.ServerConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	localEntry := models.ServerConfig{
		ID:      LocalServerID,
		Name:    "Local",
		Host:    "localhost",
		IsLocal: true,
	}
	result := make([]models.ServerConfig, 0, len(s.servers)+1)
	result = append(result, localEntry)
	result = append(result, s.servers...)
	return result, nil
}

func (s *Service) AddServer(server models.ServerConfig) (models.ServerConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if server.ID == LocalServerID || server.IsLocal {
		return models.ServerConfig{}, fmt.Errorf("cannot add a server with reserved local ID")
	}
	if server.ID == "" {
		server.ID = uuid.New().String()
	}
	if server.Port == 0 {
		server.Port = 22
	}

	if err := s.validateViaServerLocked(server); err != nil {
		return models.ServerConfig{}, err
	}

	s.servers = append(s.servers, server)
	if err := s.save(); err != nil {
		return models.ServerConfig{}, err
	}
	return server, nil
}

func (s *Service) UpdateServer(server models.ServerConfig) (models.ServerConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if server.ID == LocalServerID || server.IsLocal {
		return models.ServerConfig{}, fmt.Errorf("local server cannot be edited")
	}
	if err := s.validateViaServerLocked(server); err != nil {
		return models.ServerConfig{}, err
	}

	found := false
	for i, srv := range s.servers {
		if srv.ID == server.ID {
			s.servers[i] = server
			found = true
			break
		}
	}
	if !found {
		return models.ServerConfig{}, fmt.Errorf("server not found: %s", server.ID)
	}

	s.disconnectClient(server.ID)

	if err := s.save(); err != nil {
		return models.ServerConfig{}, err
	}
	return server, nil
}

func (s *Service) DeleteServer(id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id == LocalServerID {
		return false, fmt.Errorf("local server cannot be deleted")
	}

	var dependents []string
	for _, srv := range s.servers {
		if srv.ViaServerID == id {
			dependents = append(dependents, srv.Name)
		}
	}
	if len(dependents) > 0 {
		return false, fmt.Errorf("cannot delete server: it is used as a jump host by: %s", strings.Join(dependents, ", "))
	}

	found := false
	for i, srv := range s.servers {
		if srv.ID == id {
			s.servers = append(s.servers[:i], s.servers[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return false, nil
	}

	s.disconnectClient(id)

	if err := s.save(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) TestConnection(server models.ServerConfig) (models.TestConnectionResult, error) {
	if server.ID == LocalServerID || server.IsLocal {
		return s.testLocalConnection()
	}

	jump, err := s.resolveJump(server)
	if err != nil {
		return models.TestConnectionResult{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	result, err := ssh.TestConnection(server, jump)
	if err != nil {
		return models.TestConnectionResult{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	return *result, nil
}

func (s *Service) testLocalConnection() (models.TestConnectionResult, error) {
	client := s.getOrCreateLocalClient()

	stdout, _, err := client.Exec("wg --version")
	if err != nil {
		return models.TestConnectionResult{
			Success: true,
			Message: "Local access works, but WireGuard may not be installed. " + stdout,
		}, nil
	}

	uidOut, _, _ := client.Exec("id -u")
	isRoot := strings.TrimSpace(uidOut) == "0"

	if !isRoot {
		_, stderr, err := client.Exec("sudo true")
		if err != nil {
			msg := strings.TrimSpace(stderr)
			return models.TestConnectionResult{
				Success: false,
				Message: "Sudo authentication failed: " + msg,
			}, nil
		}
	}

	return models.TestConnectionResult{
		Success: true,
		Message: "Connected locally. WireGuard is installed.",
	}, nil
}

// validateViaServerLocked checks ViaServerID constraints while holding s.mu.
// It does NOT establish a connection to the jump server.
func (s *Service) validateViaServerLocked(server models.ServerConfig) error {
	if server.ViaServerID == "" {
		return nil
	}
	if server.ViaServerID == server.ID {
		return fmt.Errorf("cannot use server as its own jump host")
	}
	var jump *models.ServerConfig
	for i := range s.servers {
		if s.servers[i].ID == server.ViaServerID {
			jump = &s.servers[i]
			break
		}
	}
	if jump == nil {
		return fmt.Errorf("jump server not found: %s", server.ViaServerID)
	}
	if jump.ViaServerID != "" {
		return fmt.Errorf("jump server %q must be a direct-connect server (single hop only)", jump.Name)
	}
	return nil
}

// resolveJump returns an SSH client for the jump server referenced by server.ViaServerID.
// Must be called WITHOUT holding s.mu (it calls GetClient which locks).
func (s *Service) resolveJump(server models.ServerConfig) (ssh.Executor, error) {
	if server.ViaServerID == "" {
		return nil, nil
	}
	if server.ViaServerID == server.ID {
		return nil, fmt.Errorf("cannot use server as its own jump host")
	}

	s.mu.RLock()
	var jump *models.ServerConfig
	for i := range s.servers {
		if s.servers[i].ID == server.ViaServerID {
			jump = &s.servers[i]
			break
		}
	}
	s.mu.RUnlock()

	if jump == nil {
		return nil, fmt.Errorf("jump server not found: %s", server.ViaServerID)
	}
	if jump.ViaServerID != "" {
		return nil, fmt.Errorf("jump server %q must be a direct-connect server (single hop only)", jump.Name)
	}

	return s.GetClient(jump.ID)
}

func (s *Service) GetClient(serverID string) (ssh.Executor, error) {
	if serverID == LocalServerID {
		return s.getOrCreateLocalClient(), nil
	}

	// Fast path: return cached client under read lock.
	s.mu.RLock()
	if client, ok := s.clients[serverID]; ok {
		if client.IsConnected() {
			s.mu.RUnlock()
			return client, nil
		}
	}
	s.mu.RUnlock()

	// Clean up any stale cached client and snapshot the server config under write lock.
	s.mu.Lock()
	if client, ok := s.clients[serverID]; ok {
		client.Close()
		delete(s.clients, serverID)
	}

	var server models.ServerConfig
	found := false
	for i := range s.servers {
		if s.servers[i].ID == serverID {
			server = s.servers[i]
			found = true
			break
		}
	}
	s.mu.Unlock()

	if !found {
		return nil, fmt.Errorf("server not found: %s", serverID)
	}

	// Resolve jump host outside the lock to avoid deadlock (jump's GetClient locks).
	jump, err := s.resolveJump(server)
	if err != nil {
		return nil, err
	}

	client, err := ssh.Connect(server, jump)
	if err != nil {
		return nil, err
	}

	client.ServerID = serverID
	client.OnExec = func(e ssh.ExecEvent) {
		if app := application.Get(); app != nil {
			app.Event.Emit("ssh-terminal", e)
		}
	}

	s.mu.Lock()
	// Replace any client that may have been cached concurrently.
	if existing, ok := s.clients[serverID]; ok {
		existing.Close()
	}
	s.clients[serverID] = client
	s.mu.Unlock()
	return client, nil
}

func (s *Service) getOrCreateLocalClient() *local.Client {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.localClient != nil {
		return s.localClient
	}

	password := ""
	if s.localConfig != nil {
		password = s.localConfig.Password
	}
	c := local.NewClient(password)
	c.ServerID = LocalServerID
	c.OnExec = func(e ssh.ExecEvent) {
		if app := application.Get(); app != nil {
			app.Event.Emit("ssh-terminal", e)
		}
	}
	s.localClient = c
	return c
}

func (s *Service) GetLocalConfig() (models.LocalConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.localConfig == nil {
		return models.LocalConfig{}, nil
	}
	cfg := *s.localConfig
	cfg.Password = ""
	cfg.Configured = s.localConfig.Password != ""
	return cfg, nil
}

func (s *Service) SaveLocalConfig(cfg models.LocalConfig) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	toSave := &models.LocalConfig{
		Username: cfg.Username,
		Password: cfg.Password,
	}

	if err := config.SaveLocalConfig(toSave); err != nil {
		return false, err
	}

	s.localConfig = toSave

	// Recreate the local client with updated credentials.
	if s.localClient != nil {
		s.localClient = nil
	}

	return true, nil
}

func (s *Service) SetLocalSessionCredentials(username, password string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.localConfig = &models.LocalConfig{
		Username: username,
		Password: password,
	}

	// Recreate the local client with updated credentials.
	s.localClient = nil

	return true, nil
}

func (s *Service) ClearLocalSessionCredentials() (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Reload from disk so only persisted config remains.
	s.loadLocalConfig()
	s.localClient = nil

	return true, nil
}

func (s *Service) disconnectClient(serverID string) {
	if client, ok := s.clients[serverID]; ok {
		client.Close()
		delete(s.clients, serverID)
	}
}

func (s *Service) CloseAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, client := range s.clients {
		client.Close()
		delete(s.clients, id)
	}
}
