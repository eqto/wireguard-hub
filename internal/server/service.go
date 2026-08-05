package server

import (
	"fmt"
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

	s.servers = append(s.servers, server)
	if err := s.save(); err != nil {
		return models.ServerConfig{}, err
	}
	return server, nil
}

func (s *Service) UpdateServer(server models.ServerConfig) (models.ServerConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	result, err := ssh.TestConnection(server)
	if err != nil {
		return models.TestConnectionResult{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	return *result, nil
}

func (s *Service) GetClient(serverID string) (*ssh.Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if client, ok := s.clients[serverID]; ok {
		if client.IsConnected() {
			return client, nil
		}
		client.Close()
		delete(s.clients, serverID)
	}

	var server *models.ServerConfig
	for i := range s.servers {
		if s.servers[i].ID == serverID {
			server = &s.servers[i]
			break
		}
	}
	if server == nil {
		return nil, fmt.Errorf("server not found: %s", serverID)
	}

	client, err := ssh.Connect(*server)
	if err != nil {
		return nil, err
	}

	s.clients[serverID] = client
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
