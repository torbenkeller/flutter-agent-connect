package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/torben/flutter-agent-connect/pkg/models"
)

// Client is the FAC API client.
type Client struct {
	ServerURL       string
	AgentID         string
	ActiveSessionID string
	http            *http.Client
}

type ConnectConfig struct {
	ServerURL string
	AgentID   string
}

// Connect establishes a connection to the FAC server.
func Connect(cfg ConnectConfig) (*Client, error) {
	if cfg.AgentID == "" {
		cfg.AgentID = "agent-" + uuid.New().String()[:4]
	}

	c := &Client{
		ServerURL: cfg.ServerURL,
		AgentID:   cfg.AgentID,
		http:      &http.Client{Timeout: 30 * time.Second},
	}

	// Test connectivity
	if _, err := c.Health(); err != nil {
		return nil, fmt.Errorf("cannot reach server at %s: %w", cfg.ServerURL, err)
	}

	// Register agent
	body, _ := json.Marshal(map[string]string{"id": cfg.AgentID})
	resp, err := c.post("/agents", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to register agent: %w", err)
	}
	resp.Body.Close()

	// Save config
	if err := SaveConfig(&ClientConfig{
		ServerURL: c.ServerURL,
		AgentID:   c.AgentID,
	}); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	return c, nil
}

// Load creates a client from saved config.
func Load() (*Client, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	return &Client{
		ServerURL:       cfg.ServerURL,
		AgentID:         cfg.AgentID,
		ActiveSessionID: cfg.ActiveSessionID,
		http:            &http.Client{Timeout: 10 * time.Minute},
	}, nil
}

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

func (c *Client) Health() (*HealthResponse, error) {
	resp, err := c.get("/health")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var h HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&h); err != nil {
		return nil, err
	}
	return &h, nil
}

// Session operations

func (c *Client) CreateSession(platform, deviceType, name, workDir string) (*models.Session, error) {
	body, _ := json.Marshal(map[string]string{
		"platform":    platform,
		"device_type": deviceType,
		"name":        name,
		"work_dir":    workDir,
	})

	resp, err := c.post("/sessions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, readError(resp)
	}

	var session models.Session
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, err
	}

	// Save as active session
	c.ActiveSessionID = session.ID
	cfg, _ := LoadConfig()
	if cfg != nil {
		cfg.ActiveSessionID = session.ID
		SaveConfig(cfg)
	}

	return &session, nil
}

func (c *Client) ListSessions() ([]*models.Session, error) {
	resp, err := c.get("/sessions")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Sessions []*models.Session `json:"sessions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Sessions, nil
}

func (c *Client) GetSession(id string) (*models.Session, error) {
	resp, err := c.get("/sessions/" + id)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, readError(resp)
	}

	var session models.Session
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (c *Client) UseSession(idOrName string) (*models.Session, error) {
	// First try as direct ID
	sessions, err := c.ListSessions()
	if err != nil {
		return nil, err
	}

	for _, s := range sessions {
		if s.ID == idOrName || s.Name == idOrName {
			c.ActiveSessionID = s.ID
			cfg, _ := LoadConfig()
			if cfg != nil {
				cfg.ActiveSessionID = s.ID
				SaveConfig(cfg)
			}
			return s, nil
		}
	}

	return nil, fmt.Errorf("session not found: %s", idOrName)
}

func (c *Client) DestroySession(idOrName string) error {
	sessionID := c.resolveSession(idOrName)
	if sessionID == "" {
		return fmt.Errorf("no active session. Specify a session ID or name")
	}

	resp, err := c.delete("/sessions/" + sessionID)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return readError(resp)
	}

	// Clear active session if it was destroyed
	if sessionID == c.ActiveSessionID {
		cfg, _ := LoadConfig()
		if cfg != nil {
			cfg.ActiveSessionID = ""
			SaveConfig(cfg)
		}
	}

	return nil
}

// Flutter operations

type FlutterRunResult struct {
	AppID        string `json:"app_id"`
	State        string `json:"state"`
	DeviceID     string `json:"device_id,omitempty"`
	VMServiceURI string `json:"vm_service_uri,omitempty"`
}

func (c *Client) FlutterRun(session, target string) (*FlutterRunResult, error) {
	sessionID := c.resolveSession(session)
	if sessionID == "" {
		return nil, fmt.Errorf("no active session")
	}

	body, _ := json.Marshal(map[string]string{"target": target})
	resp, err := c.post("/sessions/"+sessionID+"/flutter/run", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, readError(resp)
	}

	var result FlutterRunResult
	json.NewDecoder(resp.Body).Decode(&result)
	return &result, nil
}

func (c *Client) FlutterStop(session string) error {
	sessionID := c.resolveSession(session)
	if sessionID == "" {
		return fmt.Errorf("no active session")
	}

	resp, err := c.post("/sessions/"+sessionID+"/flutter/stop", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return readError(resp)
	}
	return nil
}

type ReloadResult struct {
	Success    bool   `json:"success"`
	DurationMs int    `json:"reload_duration_ms"`
	Message    string `json:"message,omitempty"`
}

func (c *Client) FlutterHotReload(session string) (*ReloadResult, error) {
	sessionID := c.resolveSession(session)
	if sessionID == "" {
		return nil, fmt.Errorf("no active session")
	}

	resp, err := c.post("/sessions/"+sessionID+"/flutter/hot-reload", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, readError(resp)
	}

	var result ReloadResult
	json.NewDecoder(resp.Body).Decode(&result)
	return &result, nil
}

func (c *Client) FlutterHotRestart(session string) (*ReloadResult, error) {
	sessionID := c.resolveSession(session)
	if sessionID == "" {
		return nil, fmt.Errorf("no active session")
	}

	resp, err := c.post("/sessions/"+sessionID+"/flutter/hot-restart", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, readError(resp)
	}

	var result ReloadResult
	json.NewDecoder(resp.Body).Decode(&result)
	return &result, nil
}

type CommandResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func (c *Client) FlutterClean(session string) (*CommandResult, error) {
	sessionID := c.resolveSession(session)
	if sessionID == "" {
		return nil, fmt.Errorf("no active session")
	}

	resp, err := c.post("/sessions/"+sessionID+"/flutter/clean", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, readError(resp)
	}

	var result CommandResult
	json.NewDecoder(resp.Body).Decode(&result)
	return &result, nil
}

func (c *Client) FlutterPubGet(session string) (*CommandResult, error) {
	sessionID := c.resolveSession(session)
	if sessionID == "" {
		return nil, fmt.Errorf("no active session")
	}

	resp, err := c.post("/sessions/"+sessionID+"/flutter/pub-get", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, readError(resp)
	}

	var result CommandResult
	json.NewDecoder(resp.Body).Decode(&result)
	return &result, nil
}

func (c *Client) FlutterVersion() (string, error) {
	resp, err := c.get("/flutter/version")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", readError(resp)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// Device operations

func (c *Client) DeviceScreenshot(session string, deviceLevel bool) ([]byte, error) {
	sessionID := c.resolveSession(session)
	if sessionID == "" {
		return nil, fmt.Errorf("no active session")
	}

	resp, err := c.get("/sessions/" + sessionID + "/device/screenshot")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, readError(resp)
	}

	return io.ReadAll(resp.Body)
}

// Interaction
type TapResult struct {
	X int `json:"x"`
	Y int `json:"y"`
}

func (c *Client) TapByLabel(session, label string, index int) (*TapResult, error) {
	return nil, fmt.Errorf("not implemented")
}
func (c *Client) TapByKey(session, key string, index int) (*TapResult, error) {
	return nil, fmt.Errorf("not implemented")
}
func (c *Client) TapAtCoordinates(session string, x, y int) (*TapResult, error) {
	return nil, fmt.Errorf("not implemented")
}
func (c *Client) Swipe(session, direction string, durationMs int) error {
	return fmt.Errorf("not implemented")
}
func (c *Client) TypeText(session, text string, clear, enter bool) error {
	return fmt.Errorf("not implemented")
}

// Inspection
func (c *Client) Inspect(session, treeType string) (any, error) {
	return nil, fmt.Errorf("not implemented")
}

// Debug
func (c *Client) ToggleDebug(session, flag string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

// Logs
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

func (c *Client) GetLogs(session string, errorsOnly bool, lines int) ([]LogEntry, error) {
	return nil, fmt.Errorf("not implemented")
}

// Helpers

func (c *Client) resolveSession(idOrName string) string {
	if idOrName != "" {
		return idOrName
	}
	return c.ActiveSessionID
}

func (c *Client) get(path string) (*http.Response, error) {
	req, err := http.NewRequest("GET", c.ServerURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Agent-ID", c.AgentID)
	return c.http.Do(req)
}

func (c *Client) post(path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", c.ServerURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-ID", c.AgentID)
	return c.http.Do(req)
}

func (c *Client) delete(path string) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", c.ServerURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Agent-ID", c.AgentID)
	return c.http.Do(req)
}

func readError(resp *http.Response) error {
	var errResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		return fmt.Errorf("server error (status %d)", resp.StatusCode)
	}
	return fmt.Errorf("%s", errResp.Message)
}
