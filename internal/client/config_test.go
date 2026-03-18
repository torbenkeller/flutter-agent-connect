package client

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadConfig(t *testing.T) {
	// Use temp dir
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	cfg := &Config{
		ServerURL:       "http://localhost:8420",
		AgentID:         "test-agent",
		ActiveSessionID: "abc-123",
	}

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Verify file exists
	path := filepath.Join(tmpDir, ".fac", "config.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("config file not created")
	}

	// Load it back
	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if loaded.ServerURL != cfg.ServerURL {
		t.Errorf("ServerURL: expected '%s', got '%s'", cfg.ServerURL, loaded.ServerURL)
	}
	if loaded.AgentID != cfg.AgentID {
		t.Errorf("AgentID: expected '%s', got '%s'", cfg.AgentID, loaded.AgentID)
	}
	if loaded.ActiveSessionID != cfg.ActiveSessionID {
		t.Errorf("ActiveSessionID: expected '%s', got '%s'", cfg.ActiveSessionID, loaded.ActiveSessionID)
	}
}

func TestLoadConfigMissing(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	_, err := LoadConfig()
	if err == nil {
		t.Error("should error when config doesn't exist")
	}
}
