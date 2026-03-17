package client

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type ClientConfig struct {
	ServerURL       string `json:"server_url"`
	AgentID         string `json:"agent_id"`
	ActiveSessionID string `json:"active_session_id,omitempty"`
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".fac", "config.json")
}

func LoadConfig() (*ClientConfig, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return nil, fmt.Errorf("no config found. Run 'fac connect' first")
	}

	var cfg ClientConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return &cfg, nil
}

func SaveConfig(cfg *ClientConfig) error {
	path := configPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
