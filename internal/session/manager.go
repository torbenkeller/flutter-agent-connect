package session

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/torben/flutter-agent-connect/internal/device"
	"github.com/torben/flutter-agent-connect/internal/flutter"
	"github.com/torben/flutter-agent-connect/pkg/models"
)

// Manager handles session lifecycle.
type Manager struct {
	mu         sync.RWMutex
	sessions   map[string]*Session
	agents     map[string]*models.Agent
	pool       *device.Pool
	flutterSDK string
}

// Session wraps the model with runtime state.
type Session struct {
	models.Session
	flutterProcess *flutter.RunProcess
}

func NewManager(pool *device.Pool, flutterSDK string) *Manager {
	return &Manager{
		sessions:   make(map[string]*Session),
		agents:     make(map[string]*models.Agent),
		pool:       pool,
		flutterSDK: flutterSDK,
	}
}

// RegisterAgent registers a new agent.
func (m *Manager) RegisterAgent(agentID string) *models.Agent {
	m.mu.Lock()
	defer m.mu.Unlock()

	if a, ok := m.agents[agentID]; ok {
		return a
	}

	a := &models.Agent{
		ID:        agentID,
		CreatedAt: time.Now(),
	}
	m.agents[agentID] = a
	return a
}

// CreateSession creates a new session with a fresh simulator for the agent.
func (m *Manager) CreateSession(agentID string, platform models.PlatformType, deviceType, name, workDir string) (*models.Session, error) {
	id := uuid.New().String()[:8]

	// Use session name for simulator name, or fall back to ID
	simName := name
	if simName == "" {
		simName = id
	}

	// Check for duplicate session name within agent
	m.mu.RLock()
	for _, s := range m.sessions {
		if s.AgentID == agentID && s.Name == name && name != "" && s.State != models.SessionStateDestroyed {
			m.mu.RUnlock()
			return nil, &models.ErrConflict{Message: fmt.Sprintf("Session name '%s' already in use", name)}
		}
	}
	m.mu.RUnlock()

	// Create a new simulator specifically for this session
	log.Info().
		Str("agent", agentID).
		Str("session", simName).
		Str("device", deviceType).
		Msg("Creating simulator")

	dev, err := m.pool.CreateDevice(agentID, simName, deviceType, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create simulator: %w", err)
	}

	// Boot the simulator
	log.Info().Str("udid", dev.UDID).Str("name", dev.Name).Msg("Booting simulator")
	if err := m.pool.BootDevice(dev.UDID); err != nil {
		m.pool.DeleteDevice(dev.UDID)
		return nil, fmt.Errorf("failed to boot simulator: %w", err)
	}
	dev.State = models.DeviceStateBooted

	session := &Session{
		Session: models.Session{
			ID:        id,
			AgentID:   agentID,
			Name:      name,
			Platform:  platform,
			State:     models.SessionStateCreated,
			Device:    dev,
			WorkDir:   workDir,
			CreatedAt: time.Now(),
		},
	}

	m.mu.Lock()
	m.sessions[id] = session
	m.mu.Unlock()

	return &session.Session, nil
}

// GetSession returns a session by ID, scoped to the agent.
func (m *Manager) GetSession(agentID, sessionID string) (*models.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, ok := m.sessions[sessionID]
	if !ok || s.AgentID != agentID {
		return nil, &models.ErrNotFound{Resource: "session", ID: sessionID}
	}
	return &s.Session, nil
}

// ListSessions returns all sessions for an agent.
func (m *Manager) ListSessions(agentID string) []*models.Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*models.Session
	for _, s := range m.sessions {
		if s.AgentID == agentID && s.State != models.SessionStateDestroyed {
			result = append(result, &s.Session)
		}
	}
	return result
}

// FindSession finds a session by ID or name for an agent.
func (m *Manager) FindSession(agentID, idOrName string) (*models.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, s := range m.sessions {
		if s.AgentID != agentID {
			continue
		}
		if s.ID == idOrName || s.Name == idOrName {
			return &s.Session, nil
		}
	}
	return nil, &models.ErrNotFound{Resource: "session", ID: idOrName}
}

// StartApp starts the Flutter app for a session.
func (m *Manager) StartApp(agentID, sessionID, target string) (*AppStartResult, error) {
	m.mu.Lock()
	s, ok := m.sessions[sessionID]
	if !ok || s.AgentID != agentID {
		m.mu.Unlock()
		return nil, &models.ErrNotFound{Resource: "session", ID: sessionID}
	}

	if s.flutterProcess != nil && s.flutterProcess.IsRunning() {
		m.mu.Unlock()
		return nil, &models.ErrConflict{Message: "App already running. Use 'fac reload' or 'fac app stop' first"}
	}

	if s.WorkDir == "" {
		m.mu.Unlock()
		return nil, &models.ErrValidation{Message: "No work directory set. Pass --work-dir when creating the session"}
	}

	s.State = models.SessionStateBuilding
	m.mu.Unlock()

	log.Info().
		Str("session", sessionID).
		Str("device", s.Device.UDID).
		Str("workDir", s.WorkDir).
		Str("target", target).
		Msg("Starting Flutter app")

	proc, err := flutter.Start(m.flutterSDK, s.WorkDir, s.Device.UDID, target)
	if err != nil {
		m.mu.Lock()
		s.State = models.SessionStateStopped
		m.mu.Unlock()
		return nil, fmt.Errorf("failed to start flutter run: %w", err)
	}

	m.mu.Lock()
	s.flutterProcess = proc
	m.mu.Unlock()

	// Wait for app to start
	select {
	case <-proc.Started:
		m.mu.Lock()
		s.State = models.SessionStateRunning
		m.mu.Unlock()

		return &AppStartResult{
			AppID:        proc.AppID(),
			State:        "running",
			VMServiceURI: proc.VMServiceURI(),
		}, nil

	case <-proc.Stopped:
		m.mu.Lock()
		s.State = models.SessionStateStopped
		s.flutterProcess = nil
		m.mu.Unlock()

		return nil, fmt.Errorf("flutter run exited before app started: %v", proc.Err)
	}
}

type AppStartResult struct {
	AppID        string `json:"app_id"`
	State        string `json:"state"`
	VMServiceURI string `json:"vm_service_uri,omitempty"`
}

// HotReload triggers a hot reload on the session's Flutter app.
func (m *Manager) HotReload(agentID, sessionID string) error {
	m.mu.RLock()
	s, ok := m.sessions[sessionID]
	if !ok || s.AgentID != agentID {
		m.mu.RUnlock()
		return &models.ErrNotFound{Resource: "session", ID: sessionID}
	}
	m.mu.RUnlock()

	if s.flutterProcess == nil || !s.flutterProcess.IsRunning() {
		return &models.ErrConflict{Message: "No running app. Use 'fac app start' first"}
	}

	return s.flutterProcess.HotReload()
}

// HotRestart triggers a full restart on the session's Flutter app.
func (m *Manager) HotRestart(agentID, sessionID string) error {
	m.mu.RLock()
	s, ok := m.sessions[sessionID]
	if !ok || s.AgentID != agentID {
		m.mu.RUnlock()
		return &models.ErrNotFound{Resource: "session", ID: sessionID}
	}
	m.mu.RUnlock()

	if s.flutterProcess == nil || !s.flutterProcess.IsRunning() {
		return &models.ErrConflict{Message: "No running app. Use 'fac app start' first"}
	}

	return s.flutterProcess.HotRestart()
}

// StopApp stops the Flutter app but keeps the session and simulator alive.
func (m *Manager) StopApp(agentID, sessionID string) error {
	m.mu.Lock()
	s, ok := m.sessions[sessionID]
	if !ok || s.AgentID != agentID {
		m.mu.Unlock()
		return &models.ErrNotFound{Resource: "session", ID: sessionID}
	}
	m.mu.Unlock()

	if s.flutterProcess == nil || !s.flutterProcess.IsRunning() {
		return &models.ErrConflict{Message: "No running app"}
	}

	if err := s.flutterProcess.Stop(); err != nil {
		s.flutterProcess.Kill()
	}

	m.mu.Lock()
	s.State = models.SessionStateStopped
	s.flutterProcess = nil
	m.mu.Unlock()

	return nil
}

// Screenshot takes a screenshot of the session's simulator.
func (m *Manager) Screenshot(agentID, sessionID string) ([]byte, error) {
	m.mu.RLock()
	s, ok := m.sessions[sessionID]
	if !ok || s.AgentID != agentID {
		m.mu.RUnlock()
		return nil, &models.ErrNotFound{Resource: "session", ID: sessionID}
	}
	m.mu.RUnlock()

	if s.Device == nil {
		return nil, &models.ErrValidation{Message: "No device assigned to session"}
	}

	return m.pool.Simulator().Screenshot(s.Device.UDID)
}

// DestroySession stops the app, deletes the simulator, and cleans up.
func (m *Manager) DestroySession(agentID, sessionID string) error {
	m.mu.Lock()
	s, ok := m.sessions[sessionID]
	if !ok || s.AgentID != agentID {
		m.mu.Unlock()
		return &models.ErrNotFound{Resource: "session", ID: sessionID}
	}
	m.mu.Unlock()

	// Stop flutter process
	if s.flutterProcess != nil && s.flutterProcess.IsRunning() {
		log.Info().Str("session", sessionID).Msg("Stopping flutter run")
		if err := s.flutterProcess.Stop(); err != nil {
			s.flutterProcess.Kill()
		}
	}

	// Delete the simulator (not just shutdown)
	if s.Device != nil {
		log.Info().Str("udid", s.Device.UDID).Str("name", s.Device.Name).Msg("Deleting simulator")
		if err := m.pool.DeleteDevice(s.Device.UDID); err != nil {
			log.Warn().Err(err).Msg("Failed to delete simulator")
		}
	}

	m.mu.Lock()
	s.State = models.SessionStateDestroyed
	m.mu.Unlock()

	return nil
}

// DestroyAll stops all sessions (for graceful shutdown).
func (m *Manager) DestroyAll() {
	m.mu.RLock()
	var toDestroy []struct{ agent, id string }
	for _, s := range m.sessions {
		if s.State != models.SessionStateDestroyed {
			toDestroy = append(toDestroy, struct{ agent, id string }{s.AgentID, s.ID})
		}
	}
	m.mu.RUnlock()

	for _, item := range toDestroy {
		_ = m.DestroySession(item.agent, item.id)
	}
}

// CommandResult holds the output of a flutter CLI command.
type CommandResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// FlutterClean runs `flutter clean` in the session's work directory.
func (m *Manager) FlutterClean(agentID, sessionID string) (*CommandResult, error) {
	m.mu.RLock()
	s, ok := m.sessions[sessionID]
	if !ok || s.AgentID != agentID {
		m.mu.RUnlock()
		return nil, &models.ErrNotFound{Resource: "session", ID: sessionID}
	}
	m.mu.RUnlock()

	if s.WorkDir == "" {
		return nil, &models.ErrValidation{Message: "No work directory set. Pass --work-dir when creating the session"}
	}

	log.Info().Str("session", sessionID).Str("workDir", s.WorkDir).Msg("Running flutter clean")

	cmd := exec.Command(m.flutterSDK, "clean")
	cmd.Dir = s.WorkDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("flutter clean failed: %s: %w", string(output), err)
	}

	return &CommandResult{
		Success: true,
		Message: "flutter clean completed",
	}, nil
}

// FlutterPubGet runs `flutter pub get` in the session's work directory.
func (m *Manager) FlutterPubGet(agentID, sessionID string) (*CommandResult, error) {
	m.mu.RLock()
	s, ok := m.sessions[sessionID]
	if !ok || s.AgentID != agentID {
		m.mu.RUnlock()
		return nil, &models.ErrNotFound{Resource: "session", ID: sessionID}
	}
	m.mu.RUnlock()

	if s.WorkDir == "" {
		return nil, &models.ErrValidation{Message: "No work directory set. Pass --work-dir when creating the session"}
	}

	log.Info().Str("session", sessionID).Str("workDir", s.WorkDir).Msg("Running flutter pub get")

	cmd := exec.Command(m.flutterSDK, "pub", "get")
	cmd.Dir = s.WorkDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("flutter pub get failed: %s: %w", string(output), err)
	}

	return &CommandResult{
		Success: true,
		Message: "flutter pub get completed",
	}, nil
}

// FlutterVersion runs `flutter --version --machine` and returns the parsed JSON.
func (m *Manager) FlutterVersion() (any, error) {
	cmd := exec.Command(m.flutterSDK, "--version", "--machine")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("flutter --version failed: %w", err)
	}

	var version any
	if err := json.Unmarshal(output, &version); err != nil {
		// If it's not JSON, return as string
		return map[string]string{"version": string(output)}, nil
	}

	return version, nil
}
