package config

import (
	"os"
	"path/filepath"

	"wireguardadmin/internal/models"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"
)

const (
	appConfigDir  = "wireguard-admin"
	configFile    = "servers.yaml"
)

func configPath() string {
	return filepath.Join(xdg.ConfigHome, appConfigDir, configFile)
}

func Load() ([]models.ServerConfig, error) {
	path := configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []models.ServerConfig{}, nil
		}
		return nil, err
	}

	var servers []models.ServerConfig
	if err := yaml.Unmarshal(data, &servers); err != nil {
		return nil, err
	}
	return servers, nil
}

func Save(servers []models.ServerConfig) error {
	path := configPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := yaml.Marshal(servers)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}
