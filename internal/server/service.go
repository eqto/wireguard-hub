package server

import (
	"fmt"
	"strings"
	"sync"

	"wireguardadmin/internal/config"
	"wireguardadmin/internal/models"
	"wireguardadmin/internal/ssh"

	"github.com/google/uuid"
)

type Service struct {
	mu      sync.RWMutex
	servers []models.ServerConfig
	clients map[string]*ssh.Client
}

func NewService() *Service {
	s := &Service{
		clients: make(map[string]*ssh.Client),
	}
	s.load()
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

func (s *Service) GetServers() ([]models.ServerConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.servers, nil
}

func (s *Service) AddServer(server models.ServerConfig) (models.ServerConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

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
func (s *Service) resolveJump(server models.ServerConfig) (*ssh.Client, error) {
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

func (s *Service) GetClient(serverID string) (*ssh.Client, error) {
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

	s.mu.Lock()
	// Replace any client that may have been cached concurrently.
	if existing, ok := s.clients[serverID]; ok {
		existing.Close()
	}
	s.clients[serverID] = client
	s.mu.Unlock()
	return client, nil
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
