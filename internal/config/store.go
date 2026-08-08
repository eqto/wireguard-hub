package config

import (
	"os"
	"path/filepath"

	"wireguardhub/internal/models"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"
)

const (
	appConfigDir  = "wireguardhub"
	configFile    = "servers.yaml"
	localConfFile = "local.yaml"
)

func configPath() string {
	return filepath.Join(xdg.ConfigHome, appConfigDir, configFile)
}

func localConfigPath() string {
	return filepath.Join(xdg.ConfigHome, appConfigDir, localConfFile)
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

func LoadLocalConfig() (*models.LocalConfig, error) {
	path := localConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &models.LocalConfig{}, nil
		}
		return nil, err
	}

	var cfg models.LocalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	cfg.Configured = cfg.Username != "" || cfg.Password != ""
	return &cfg, nil
}

func SaveLocalConfig(cfg *models.LocalConfig) error {
	path := localConfigPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}
