package session

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/torbenkeller/flutter-agent-connect/internal/flutter"
	"github.com/torbenkeller/flutter-agent-connect/internal/interaction"
	"github.com/torbenkeller/flutter-agent-connect/pkg/models"
)

// Manager handles session lifecycle.
type Manager struct {
	mu           sync.RWMutex
	sessions     map[string]*Session
	agents       map[string]*models.Agent
	pool         DevicePool
	interactors  map[models.PlatformType]Interactor
	flutterSDK   string
	startFlutter FlutterStarter
	connectVM    VMServiceConnector
}

// Option configures a Manager.
type Option func(*Manager)

// WithFlutterStarter overrides the default flutter process starter.
func WithFlutterStarter(s FlutterStarter) Option {
	return func(m *Manager) { m.startFlutter = s }
}

// WithVMServiceConnector overrides the default VM Service connector.
func WithVMServiceConnector(c VMServiceConnector) Option {
	return func(m *Manager) { m.connectVM = c }
}

// WithInteractors overrides the default device interactors.
func WithInteractors(i map[models.PlatformType]Interactor) Option {
	return func(m *Manager) { m.interactors = i }
}

// PortForward represents a forwarded port with its dart-define mapping.
type PortForward struct {
	ContainerPort int    `json:"container_port"`
	HostPort      int    `json:"host_port"`
	EnvName       string `json:"env_name,omitempty"`
	URLiOS        string `json:"url_ios"`
	URLAndroid    string `json:"url_android"`
}

// Session wraps the model with runtime state.
type Session struct {
	models.Session
	flutterProcess  FlutterProcess
	vmServiceClient VMService
	flutterDeviceID string // UDID for iOS, adb serial for Android
	forwards        []PortForward
}

func NewManager(pool DevicePool, flutterSDK string, opts ...Option) *Manager {
	m := &Manager{
		sessions:   make(map[string]*Session),
		agents:     make(map[string]*models.Agent),
		pool:       pool,
		flutterSDK: flutterSDK,
		// Default: real implementations
		startFlutter: defaultFlutterStarter,
		connectVM:    defaultVMServiceConnector,
		interactors: map[models.PlatformType]Interactor{
			models.PlatformIOS:     interaction.NewIOSInteraction(),
			models.PlatformAndroid: interaction.NewAndroidInteraction(),
		},
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// defaultFlutterStarter wraps flutter.Start to satisfy the FlutterStarter type.
func defaultFlutterStarter(flutterBin, workDir, deviceID, target string, dartDefines []string) (FlutterProcess, error) {
	return flutter.Start(flutterBin, workDir, deviceID, target, dartDefines)
}

// defaultVMServiceConnector wraps flutter.ConnectVMService to satisfy VMServiceConnector.
func defaultVMServiceConnector(wsURI, adbSerial string) (VMService, error) {
	client, err := flutter.ConnectVMService(wsURI)
	if err != nil {
		return nil, err
	}
	client.ADBSerial = adbSerial
	return client, nil
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

	// Create a new device (simulator or emulator)
	log.Info().
		Str("agent", agentID).
		Str("session", simName).
		Str("platform", string(platform)).
		Str("device", deviceType).
		Msg("Creating device")

	managed, err := m.pool.CreateDevice(agentID, simName, platform, deviceType, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create device: %w", err)
	}

	// Boot the device
	log.Info().Str("udid", managed.UDID).Str("name", managed.Name).Msg("Booting device")
	flutterDeviceID, err := m.pool.BootDevice(managed.UDID, platform)
	if err != nil {
		_ = m.pool.DeleteDevice(managed.UDID, platform)
		return nil, fmt.Errorf("failed to boot device: %w", err)
	}
	managed.State = models.DeviceStateBooted

	session := &Session{
		Session: models.Session{
			ID:        id,
			AgentID:   agentID,
			Name:      name,
			Platform:  platform,
			State:     models.SessionStateCreated,
			Device:    &managed.Device,
			WorkDir:   workDir,
			CreatedAt: time.Now(),
		},
		flutterDeviceID: flutterDeviceID,
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
		Str("device", s.flutterDeviceID).
		Str("platform", string(s.Platform)).
		Str("workDir", s.WorkDir).
		Str("target", target).
		Msg("Starting Flutter app")

	// Build dart-defines from registered port forwards
	dartDefines := m.GetDartDefines(sessionID, s.Platform)
	if len(dartDefines) > 0 {
		log.Info().Strs("defines", dartDefines).Msg("Injecting dart-defines from port forwards")
	}

	proc, err := m.startFlutter(m.flutterSDK, s.WorkDir, s.flutterDeviceID, target, dartDefines)
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
	case <-proc.Started():
		m.mu.Lock()
		s.State = models.SessionStateRunning
		m.mu.Unlock()

		return &AppStartResult{
			AppID:        proc.AppID(),
			State:        "running",
			VMServiceURI: proc.VMServiceURI(),
		}, nil

	case <-proc.Stopped():
		// Collect build output before clearing the process
		logs := proc.Logs(0)
		var buildOutput []string
		for _, l := range logs {
			buildOutput = append(buildOutput, l.Message)
		}

		m.mu.Lock()
		s.State = models.SessionStateStopped
		s.flutterProcess = nil
		m.mu.Unlock()

		return &AppStartResult{
				State:       "failed",
				BuildOutput: buildOutput,
			}, &BuildError{
				Err:         proc.Err(),
				BuildOutput: buildOutput,
			}
	}
}

// BuildError is returned when flutter run fails during build.
// It carries the collected build output so callers can display it.
type BuildError struct {
	Err         error
	BuildOutput []string
}

func (e *BuildError) Error() string {
	return fmt.Sprintf("flutter run exited before app started: %v", e.Err)
}

func (e *BuildError) Unwrap() error {
	return e.Err
}

type AppStartResult struct {
	AppID        string   `json:"app_id"`
	State        string   `json:"state"`
	VMServiceURI string   `json:"vm_service_uri,omitempty"`
	BuildOutput  []string `json:"build_output,omitempty"`
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

// getVMServiceClient returns or lazily creates a VM Service client for the session.
// Reconnects if the old connection is stale (e.g., after hot restart).
func (m *Manager) getVMServiceClient(s *Session) (VMService, error) {
	if s.vmServiceClient != nil {
		// Test if still working with a quick ping
		_, err := s.vmServiceClient.CallExtension("ext.flutter.platformOverride", nil)
		if err == nil {
			return s.vmServiceClient, nil
		}
		// Connection stale, reconnect
		log.Info().Msg("VM Service connection stale, reconnecting")
		s.vmServiceClient.Close()
		s.vmServiceClient = nil
	}

	if s.flutterProcess == nil || !s.flutterProcess.IsRunning() {
		return nil, &models.ErrConflict{Message: "No running app. Use 'fac flutter run' first"}
	}

	wsURI := s.flutterProcess.VMServiceURI()
	if wsURI == "" {
		return nil, &models.ErrConflict{Message: "VM Service URI not available"}
	}

	// Determine ADB serial for Android (needed for EnsureSemantics)
	adbSerial := ""
	if s.Platform == models.PlatformAndroid {
		managed := m.pool.GetManaged(s.Device.UDID)
		if managed != nil {
			adbSerial = managed.ADBSerial
		}
	}

	log.Info().Str("wsUri", wsURI).Msg("Connecting to VM Service (lazy)")
	client, err := m.connectVM(wsURI, adbSerial)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to VM Service: %w", err)
	}

	m.mu.Lock()
	s.vmServiceClient = client
	m.mu.Unlock()

	return client, nil
}

// DeviceTap sends a tap event to the simulator.
// Supports tapping by label (via semantics tree), key, or raw coordinates.
func (m *Manager) DeviceTap(agentID, sessionID, label, key string, x, y float64, index int) (*TapResult, error) { //nolint:gocritic // splitting label,key from agentID,sessionID improves readability
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

	// Widget-based tap via semantics tree
	if label != "" || key != "" {
		vmClient, err := m.getVMServiceClient(s)
		if err != nil {
			return nil, err
		}

		tree, err := vmClient.GetSemanticsTree()
		if err != nil {
			return nil, fmt.Errorf("failed to get semantics tree: %w", err)
		}

		var node *flutter.SemanticsNode
		if label != "" {
			node = tree.FindByLabel(label, index)
		} else {
			node = tree.FindByKey(key, index)
		}

		if node == nil {
			availableLabels := tree.AllLabels()
			return nil, fmt.Errorf("element not found with label %q. Available: %v", label+key, availableLabels)
		}

		if node.Rect == nil {
			return nil, fmt.Errorf("element %q has no bounding rect", label+key)
		}

		cx, cy := node.Rect.Center()

		// On Android, semantics coordinates are in logical pixels (dp).
		// adb shell input tap expects physical pixels.
		// Multiply by device pixel ratio = physical_width / logical_width.
		if s.Platform == models.PlatformAndroid {
			// Get physical screen size
			w, _, devErr := m.pool.DeviceInfo(s.Device.UDID)
			if devErr == nil && w > 0 {
				// Get logical screen size from semantics tree root children
				logicalW := 411.4 // sensible default
				for _, child := range tree.Children {
					if child.Rect != nil && child.Rect.Right > 100 && child.Rect.Right < float64(w) {
						logicalW = child.Rect.Right
						break
					}
				}
				scale := float64(w) / logicalW
				log.Debug().Float64("scale", scale).Int("physicalW", w).Float64("logicalW", logicalW).Msg("Android coordinate scaling")
				cx *= scale
				cy *= scale
			}
		}

		log.Info().Str("label", label+key).Float64("x", cx).Float64("y", cy).Str("platform", string(s.Platform)).Msg("Found element, tapping")

		if err := m.platformTap(s, int(cx), int(cy)); err != nil {
			return nil, err
		}

		return &TapResult{Success: true, X: int(cx), Y: int(cy), Element: label + key}, nil
	}

	// Direct coordinate tap
	if err := m.platformTap(s, int(x), int(y)); err != nil {
		return nil, err
	}

	return &TapResult{Success: true, X: int(x), Y: int(y)}, nil
}

// TapResult represents the result of a tap operation.
type TapResult struct {
	Success bool   `json:"success"`
	X       int    `json:"x"`
	Y       int    `json:"y"`
	Element string `json:"element,omitempty"`
}

// platformTap sends a tap to the appropriate platform.
func (m *Manager) platformTap(s *Session, x, y int) error {
	deviceID, err := m.resolveDeviceID(s)
	if err != nil {
		return err
	}
	return m.interactors[s.Platform].Tap(deviceID, x, y)
}

// resolveDeviceID returns the device identifier for interactions (UDID for iOS, ADB serial for Android).
func (m *Manager) resolveDeviceID(s *Session) (string, error) {
	if s.Platform == models.PlatformAndroid {
		managed := m.pool.GetManaged(s.Device.UDID)
		if managed == nil || managed.ADBSerial == "" {
			return "", &models.ErrConflict{Message: "Android emulator not running"}
		}
		return managed.ADBSerial, nil
	}
	return s.Device.UDID, nil
}

// DeviceType types text into the simulator/emulator.
func (m *Manager) DeviceType(agentID, sessionID, text string, clearField, enter bool) error {
	m.mu.RLock()
	s, ok := m.sessions[sessionID]
	if !ok || s.AgentID != agentID {
		m.mu.RUnlock()
		return &models.ErrNotFound{Resource: "session", ID: sessionID}
	}
	m.mu.RUnlock()

	if s.Device == nil {
		return &models.ErrValidation{Message: "No device assigned to session"}
	}

	deviceID, err := m.resolveDeviceID(s)
	if err != nil {
		return err
	}
	return m.interactors[s.Platform].TypeText(deviceID, text, clearField, enter)
}

// DeviceSwipe sends a swipe gesture to the simulator/emulator.
func (m *Manager) DeviceSwipe(agentID, sessionID, direction string, durationMs int) error {
	m.mu.RLock()
	s, ok := m.sessions[sessionID]
	if !ok || s.AgentID != agentID {
		m.mu.RUnlock()
		return &models.ErrNotFound{Resource: "session", ID: sessionID}
	}
	m.mu.RUnlock()

	if s.Device == nil {
		return &models.ErrValidation{Message: "No device assigned to session"}
	}

	deviceID, err := m.resolveDeviceID(s)
	if err != nil {
		return err
	}

	// Default screen dimensions per platform
	screenW, screenH := 402, 874 // iOS logical pixels
	if s.Platform == models.PlatformAndroid {
		screenW, screenH = 1080, 2400 // Android physical pixels
	}
	return m.interactors[s.Platform].Swipe(deviceID, direction, screenW, screenH, durationMs)
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

	return m.pool.Screenshot(s.Device.UDID, s.Platform)
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

	// Close VM Service connection
	if s.vmServiceClient != nil {
		s.vmServiceClient.Close()
		s.vmServiceClient = nil
	}

	// Stop flutter process
	if s.flutterProcess != nil && s.flutterProcess.IsRunning() {
		log.Info().Str("session", sessionID).Msg("Stopping flutter run")
		if err := s.flutterProcess.Stop(); err != nil {
			s.flutterProcess.Kill()
		}
	}

	// Delete the device (simulator or emulator)
	if s.Device != nil {
		log.Info().Str("udid", s.Device.UDID).Str("name", s.Device.Name).Msg("Deleting device")
		if err := m.pool.DeleteDevice(s.Device.UDID, s.Platform); err != nil {
			log.Warn().Err(err).Msg("Failed to delete device")
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

// AddForward registers a port forward for a session.
// It discovers the host port via Docker and stores the mapping.
func (m *Manager) AddForward(agentID, sessionID string, containerPort int, envName string) (*PortForward, error) {
	m.mu.Lock()
	s, ok := m.sessions[sessionID]
	if !ok || s.AgentID != agentID {
		m.mu.Unlock()
		return nil, &models.ErrNotFound{Resource: "session", ID: sessionID}
	}
	m.mu.Unlock()

	// Discover host port via Docker
	hostPort, err := discoverDockerHostPort(containerPort)
	if err != nil {
		return nil, fmt.Errorf("could not discover host port for :%d: %w\nMake sure the port is exposed in your Docker/devcontainer config", containerPort, err)
	}

	fwd := PortForward{
		ContainerPort: containerPort,
		HostPort:      hostPort,
		EnvName:       envName,
		URLiOS:        fmt.Sprintf("http://localhost:%d", hostPort),
		URLAndroid:    fmt.Sprintf("http://10.0.2.2:%d", hostPort),
	}

	m.mu.Lock()
	s.forwards = append(s.forwards, fwd)
	m.mu.Unlock()

	log.Info().Int("container", containerPort).Int("host", hostPort).Str("env", envName).Msg("Port forward registered")
	return &fwd, nil
}

// ListForwards returns all registered forwards for a session.
func (m *Manager) ListForwards(agentID, sessionID string) ([]PortForward, error) {
	m.mu.RLock()
	s, ok := m.sessions[sessionID]
	if !ok || s.AgentID != agentID {
		m.mu.RUnlock()
		return nil, &models.ErrNotFound{Resource: "session", ID: sessionID}
	}
	m.mu.RUnlock()

	return s.forwards, nil
}

// GetDartDefines builds the --dart-define arguments for flutter run based on forwards.
func (m *Manager) GetDartDefines(sessionID string, platform models.PlatformType) []string {
	m.mu.RLock()
	s, ok := m.sessions[sessionID]
	m.mu.RUnlock()

	if !ok {
		return nil
	}

	defines := make([]string, 0, len(s.forwards))
	for _, fwd := range s.forwards {
		if fwd.EnvName == "" {
			continue
		}
		url := fwd.URLiOS
		if platform == models.PlatformAndroid {
			url = fwd.URLAndroid
		}
		defines = append(defines, fmt.Sprintf("%s=%s", fwd.EnvName, url))
	}
	return defines
}

// discoverDockerHostPort finds the host port mapped to a container port.
// Uses `docker port` on the current container, or falls back to checking
// if the port is directly reachable on localhost.
func discoverDockerHostPort(containerPort int) (int, error) {
	// Method 1: Try to find our container ID and use docker port
	containerID, err := detectContainerID()
	if err == nil && containerID != "" {
		out, err := exec.Command("docker", "port", containerID, fmt.Sprintf("%d", containerPort)).Output()
		if err == nil {
			// Output: "0.0.0.0:9001" or ":::9001"
			parts := strings.Split(strings.TrimSpace(string(out)), ":")
			if len(parts) >= 2 {
				port := 0
				_, _ = fmt.Sscanf(parts[len(parts)-1], "%d", &port)
				if port > 0 {
					return port, nil
				}
			}
		}
	}

	// Method 2: Check if the port is directly reachable on localhost
	// (works when running outside Docker or with host networking)
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", containerPort), 2*time.Second)
	if err == nil {
		conn.Close()
		return containerPort, nil
	}

	return 0, fmt.Errorf("port %d not reachable. Expose it in Docker config or ensure the service is running", containerPort)
}

// detectContainerID tries to detect if we're running inside a Docker container.
func detectContainerID() (string, error) {
	// Check /proc/self/cgroup for docker container ID
	data, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		// Not in a container (macOS), try to find containers via docker ps
		return detectContainerViaDocker()
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, "docker") {
			parts := strings.Split(line, "/")
			if len(parts) > 0 {
				id := parts[len(parts)-1]
				if len(id) >= 12 {
					return id[:12], nil
				}
			}
		}
	}

	return detectContainerViaDocker()
}

// detectContainerViaDocker lists running containers and finds one that has the port.
func detectContainerViaDocker() (string, error) {
	out, err := exec.Command("docker", "ps", "--format", "{{.ID}}").Output()
	if err != nil {
		return "", err
	}

	containers := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(containers) == 0 || containers[0] == "" {
		return "", fmt.Errorf("no running containers found")
	}

	// Return the first container (for Phase 1, usually there's only one dev container)
	return containers[0], nil
}

// InspectWidgets returns the widget tree dump via the VM Service.
func (m *Manager) InspectWidgets(agentID, sessionID string) (string, error) {
	m.mu.RLock()
	s, ok := m.sessions[sessionID]
	if !ok || s.AgentID != agentID {
		m.mu.RUnlock()
		return "", &models.ErrNotFound{Resource: "session", ID: sessionID}
	}
	m.mu.RUnlock()

	vmClient, err := m.getVMServiceClient(s)
	if err != nil {
		return "", err
	}

	return vmClient.GetWidgetTree()
}

// InspectRender returns the render tree dump via the VM Service.
func (m *Manager) InspectRender(agentID, sessionID string) (string, error) {
	m.mu.RLock()
	s, ok := m.sessions[sessionID]
	if !ok || s.AgentID != agentID {
		m.mu.RUnlock()
		return "", &models.ErrNotFound{Resource: "session", ID: sessionID}
	}
	m.mu.RUnlock()

	vmClient, err := m.getVMServiceClient(s)
	if err != nil {
		return "", err
	}

	return vmClient.GetRenderTree()
}

// InspectSemantics returns the semantics tree via the VM Service.
func (m *Manager) InspectSemantics(agentID, sessionID string) (*flutter.SemanticsNode, error) {
	m.mu.RLock()
	s, ok := m.sessions[sessionID]
	if !ok || s.AgentID != agentID {
		m.mu.RUnlock()
		return nil, &models.ErrNotFound{Resource: "session", ID: sessionID}
	}
	m.mu.RUnlock()

	vmClient, err := m.getVMServiceClient(s)
	if err != nil {
		return nil, err
	}

	return vmClient.GetSemanticsTree()
}

// ToggleDebugFlag toggles a debug flag via the VM Service.
func (m *Manager) ToggleDebugFlag(agentID, sessionID, flag string) (bool, error) {
	m.mu.RLock()
	s, ok := m.sessions[sessionID]
	if !ok || s.AgentID != agentID {
		m.mu.RUnlock()
		return false, &models.ErrNotFound{Resource: "session", ID: sessionID}
	}
	m.mu.RUnlock()

	vmClient, err := m.getVMServiceClient(s)
	if err != nil {
		return false, err
	}

	extensionMap := map[string]string{
		"paint":       "ext.flutter.debugPaint",
		"repaint":     "ext.flutter.repaintRainbow",
		"performance": "ext.flutter.showPerformanceOverlay",
	}

	ext, ok := extensionMap[flag]
	if !ok {
		return false, &models.ErrValidation{Message: fmt.Sprintf("unknown debug flag: %s", flag)}
	}

	return vmClient.ToggleDebugFlag(ext)
}

// GetLogs returns the collected logs for a session. If tail > 0, only the last N lines.
func (m *Manager) GetLogs(agentID, sessionID string, tail int) ([]string, error) {
	m.mu.RLock()
	s, ok := m.sessions[sessionID]
	if !ok || s.AgentID != agentID {
		m.mu.RUnlock()
		return nil, &models.ErrNotFound{Resource: "session", ID: sessionID}
	}
	m.mu.RUnlock()

	if s.flutterProcess == nil {
		return nil, &models.ErrConflict{Message: "No app running"}
	}

	entries := s.flutterProcess.Logs(tail)
	lines := make([]string, len(entries))
	for i, e := range entries {
		lines[i] = e.Message
	}
	return lines, nil
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
		return map[string]string{"version": string(output)}, err
	}

	return version, nil
}
