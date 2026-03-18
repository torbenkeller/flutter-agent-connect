package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/torbenkeller/flutter-agent-connect/internal/flutter"
	"github.com/torbenkeller/flutter-agent-connect/internal/session"
	"github.com/torbenkeller/flutter-agent-connect/pkg/models"
)

// --- Mock SessionService ---

type mockSessionService struct {
	// Return values for each method
	agent          *models.Agent
	session        *models.Session
	sessions       []*models.Session
	appStartResult *session.AppStartResult
	appStartErr    error
	commandResult  *session.CommandResult
	tapResult      *session.TapResult
	screenshot     []byte
	widgetTree     string
	renderTree     string
	semanticsTree  *flutter.SemanticsNode
	toggleResult   bool
	logs           []string
	forward        *session.PortForward
	forwards       []session.PortForward
	version        any
	err            error
}

func (m *mockSessionService) RegisterAgent(agentID string) *models.Agent {
	if m.agent != nil {
		return m.agent
	}
	return &models.Agent{ID: agentID}
}
func (m *mockSessionService) CreateSession(agentID string, platform models.PlatformType, deviceType, name, workDir string) (*models.Session, error) {
	return m.session, m.err
}
func (m *mockSessionService) ListSessions(agentID string) []*models.Session {
	return m.sessions
}
func (m *mockSessionService) GetSession(agentID, sessionID string) (*models.Session, error) {
	return m.session, m.err
}
func (m *mockSessionService) DestroySession(agentID, sessionID string) error {
	return m.err
}
func (m *mockSessionService) StartApp(agentID, sessionID, target string) (*session.AppStartResult, error) {
	return m.appStartResult, m.appStartErr
}
func (m *mockSessionService) StopApp(agentID, sessionID string) error {
	return m.err
}
func (m *mockSessionService) HotReload(agentID, sessionID string) error {
	return m.err
}
func (m *mockSessionService) HotRestart(agentID, sessionID string) error {
	return m.err
}
func (m *mockSessionService) FlutterClean(agentID, sessionID string) (*session.CommandResult, error) {
	return m.commandResult, m.err
}
func (m *mockSessionService) FlutterPubGet(agentID, sessionID string) (*session.CommandResult, error) {
	return m.commandResult, m.err
}
func (m *mockSessionService) FlutterVersion() (any, error) {
	return m.version, m.err
}
func (m *mockSessionService) Screenshot(agentID, sessionID string) ([]byte, error) {
	return m.screenshot, m.err
}
func (m *mockSessionService) DeviceTap(agentID, sessionID, label, key string, x, y float64, index int) (*session.TapResult, error) {
	return m.tapResult, m.err
}
func (m *mockSessionService) DeviceSwipe(agentID, sessionID, direction string, durationMs int) error {
	return m.err
}
func (m *mockSessionService) DeviceType(agentID, sessionID, text string, clear, enter bool) error {
	return m.err
}
func (m *mockSessionService) InspectWidgets(agentID, sessionID string) (string, error) {
	return m.widgetTree, m.err
}
func (m *mockSessionService) InspectRender(agentID, sessionID string) (string, error) {
	return m.renderTree, m.err
}
func (m *mockSessionService) InspectSemantics(agentID, sessionID string) (*flutter.SemanticsNode, error) {
	return m.semanticsTree, m.err
}
func (m *mockSessionService) ToggleDebugFlag(agentID, sessionID, flag string) (bool, error) {
	return m.toggleResult, m.err
}
func (m *mockSessionService) GetLogs(agentID, sessionID string, tail int) ([]string, error) {
	return m.logs, m.err
}
func (m *mockSessionService) AddForward(agentID, sessionID string, containerPort int, envName string) (*session.PortForward, error) {
	return m.forward, m.err
}
func (m *mockSessionService) ListForwards(agentID, sessionID string) ([]session.PortForward, error) {
	return m.forwards, m.err
}

// --- Mock DeviceLister ---

type mockDeviceLister struct {
	devices []models.Device
}

func (m *mockDeviceLister) List() []models.Device {
	return m.devices
}

// --- Helper ---

func setupRouter(svc *mockSessionService, devices *mockDeviceLister) *http.ServeMux {
	if devices == nil {
		devices = &mockDeviceLister{}
	}
	return NewRouter(svc, devices)
}

func doRequest(mux *http.ServeMux, method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-ID", "test-agent")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func decodeJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode JSON response: %v\nbody: %s", err, w.Body.String())
	}
	return result
}

// --- Tests ---

func TestHealthEndpoint(t *testing.T) {
	mux := setupRouter(&mockSessionService{}, nil)
	w := doRequest(mux, "GET", "/health", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	result := decodeJSON(t, w)
	if result["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%v'", result["status"])
	}
}

func TestListDevices(t *testing.T) {
	devices := &mockDeviceLister{
		devices: []models.Device{
			{UDID: "abc", Name: "iPhone 16", Platform: models.PlatformIOS},
		},
	}
	mux := setupRouter(&mockSessionService{}, devices)
	w := doRequest(mux, "GET", "/devices", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	result := decodeJSON(t, w)
	devList, ok := result["devices"].([]any)
	if !ok {
		t.Fatal("devices should be an array")
	}
	if len(devList) != 1 {
		t.Errorf("expected 1 device, got %d", len(devList))
	}
}

func TestRegisterAgent(t *testing.T) {
	mux := setupRouter(&mockSessionService{}, nil)
	w := doRequest(mux, "POST", "/agents", map[string]string{"id": "my-agent"})

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
}

func TestRegisterAgentMissingID(t *testing.T) {
	mux := setupRouter(&mockSessionService{}, nil)
	w := doRequest(mux, "POST", "/agents", map[string]string{})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateSession(t *testing.T) {
	svc := &mockSessionService{
		session: &models.Session{ID: "s1", Name: "test", Platform: models.PlatformIOS},
	}
	mux := setupRouter(svc, nil)
	w := doRequest(mux, "POST", "/sessions", map[string]string{
		"platform": "ios",
		"name":     "test",
		"work_dir": "/tmp/app",
	})

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateSessionInvalidPlatform(t *testing.T) {
	mux := setupRouter(&mockSessionService{}, nil)
	w := doRequest(mux, "POST", "/sessions", map[string]string{
		"platform": "invalid",
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetSession(t *testing.T) {
	svc := &mockSessionService{
		session: &models.Session{ID: "s1", Name: "test"},
	}
	mux := setupRouter(svc, nil)
	w := doRequest(mux, "GET", "/sessions/s1", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGetSessionNotFound(t *testing.T) {
	svc := &mockSessionService{
		err: &models.ErrNotFound{Resource: "session", ID: "s1"},
	}
	mux := setupRouter(svc, nil)
	w := doRequest(mux, "GET", "/sessions/s1", nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestListSessions(t *testing.T) {
	svc := &mockSessionService{
		sessions: []*models.Session{
			{ID: "s1", Name: "ios"},
			{ID: "s2", Name: "android"},
		},
	}
	mux := setupRouter(svc, nil)
	w := doRequest(mux, "GET", "/sessions", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	result := decodeJSON(t, w)
	sessions, ok := result["sessions"].([]any)
	if !ok {
		t.Fatal("sessions should be an array")
	}
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(sessions))
	}
}

func TestDeleteSession(t *testing.T) {
	mux := setupRouter(&mockSessionService{}, nil)
	w := doRequest(mux, "DELETE", "/sessions/s1", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestFlutterRun(t *testing.T) {
	svc := &mockSessionService{
		appStartResult: &session.AppStartResult{AppID: "app-1", State: "running"},
	}
	mux := setupRouter(svc, nil)
	w := doRequest(mux, "POST", "/sessions/s1/flutter/run", map[string]string{"target": "lib/main.dart"})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	result := decodeJSON(t, w)
	if result["state"] != "running" {
		t.Errorf("expected state 'running', got '%v'", result["state"])
	}
}

func TestFlutterRunBuildError(t *testing.T) {
	svc := &mockSessionService{
		appStartResult: &session.AppStartResult{State: "failed"},
		appStartErr: &session.BuildError{
			Err:         nil,
			BuildOutput: []string{"Error: Expected ';'"},
		},
	}
	mux := setupRouter(svc, nil)
	w := doRequest(mux, "POST", "/sessions/s1/flutter/run", nil)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}

	result := decodeJSON(t, w)
	if result["error"] != "build_error" {
		t.Errorf("expected error type 'build_error', got '%v'", result["error"])
	}

	buildOutput, ok := result["build_output"].([]any)
	if !ok || len(buildOutput) == 0 {
		t.Error("expected build_output in response")
	}
}

func TestFlutterStop(t *testing.T) {
	mux := setupRouter(&mockSessionService{}, nil)
	w := doRequest(mux, "POST", "/sessions/s1/flutter/stop", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestFlutterHotReload(t *testing.T) {
	mux := setupRouter(&mockSessionService{}, nil)
	w := doRequest(mux, "POST", "/sessions/s1/flutter/hot-reload", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestFlutterHotRestart(t *testing.T) {
	mux := setupRouter(&mockSessionService{}, nil)
	w := doRequest(mux, "POST", "/sessions/s1/flutter/hot-restart", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestScreenshot(t *testing.T) {
	svc := &mockSessionService{
		screenshot: []byte{0x89, 0x50, 0x4E, 0x47}, // PNG magic bytes
	}
	mux := setupRouter(svc, nil)
	w := doRequest(mux, "GET", "/sessions/s1/device/screenshot", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "image/png" {
		t.Errorf("expected Content-Type image/png, got %s", w.Header().Get("Content-Type"))
	}
	if len(w.Body.Bytes()) != 4 {
		t.Errorf("expected 4 bytes, got %d", len(w.Body.Bytes()))
	}
}

func TestDeviceTap(t *testing.T) {
	svc := &mockSessionService{
		tapResult: &session.TapResult{Success: true, X: 100, Y: 200, Element: "Login"},
	}
	mux := setupRouter(svc, nil)
	w := doRequest(mux, "POST", "/sessions/s1/device/tap", map[string]any{"label": "Login"})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	result := decodeJSON(t, w)
	if result["success"] != true {
		t.Error("expected success=true")
	}
	if result["element"] != "Login" {
		t.Errorf("expected element 'Login', got '%v'", result["element"])
	}
}

func TestDeviceSwipe(t *testing.T) {
	mux := setupRouter(&mockSessionService{}, nil)
	w := doRequest(mux, "POST", "/sessions/s1/device/swipe", map[string]any{"direction": "down"})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestDeviceType(t *testing.T) {
	mux := setupRouter(&mockSessionService{}, nil)
	w := doRequest(mux, "POST", "/sessions/s1/device/type", map[string]any{"text": "hello"})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestDeviceTypeMissingText(t *testing.T) {
	mux := setupRouter(&mockSessionService{}, nil)
	w := doRequest(mux, "POST", "/sessions/s1/device/type", map[string]any{})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestInspectWidgets(t *testing.T) {
	svc := &mockSessionService{widgetTree: "MyApp\n └Scaffold"}
	mux := setupRouter(svc, nil)
	w := doRequest(mux, "GET", "/sessions/s1/devtools/widgets", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	result := decodeJSON(t, w)
	if result["type"] != "widgets" {
		t.Errorf("expected type 'widgets', got '%v'", result["type"])
	}
	if result["data"] != "MyApp\n └Scaffold" {
		t.Errorf("unexpected data: %v", result["data"])
	}
}

func TestInspectSemantics(t *testing.T) {
	svc := &mockSessionService{
		semanticsTree: &flutter.SemanticsNode{ID: 0, Label: "root"},
	}
	mux := setupRouter(svc, nil)
	w := doRequest(mux, "GET", "/sessions/s1/devtools/semantics", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	result := decodeJSON(t, w)
	if result["type"] != "semantics" {
		t.Errorf("expected type 'semantics', got '%v'", result["type"])
	}
}

func TestToggleDebugPaint(t *testing.T) {
	svc := &mockSessionService{toggleResult: true}
	mux := setupRouter(svc, nil)
	w := doRequest(mux, "POST", "/sessions/s1/devtools/paint", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	result := decodeJSON(t, w)
	if result["enabled"] != true {
		t.Error("expected enabled=true")
	}
	if result["flag"] != "paint" {
		t.Errorf("expected flag 'paint', got '%v'", result["flag"])
	}
}

func TestGetLogs(t *testing.T) {
	svc := &mockSessionService{
		logs: []string{"line1", "line2", "line3"},
	}
	mux := setupRouter(svc, nil)

	req := httptest.NewRequest("GET", "/sessions/s1/devtools/logs?tail=2", nil)
	req.Header.Set("X-Agent-ID", "test-agent")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "text/plain" {
		t.Errorf("expected text/plain, got %s", w.Header().Get("Content-Type"))
	}
}

func TestMissingAgentIDHeader(t *testing.T) {
	mux := setupRouter(&mockSessionService{}, nil)

	req := httptest.NewRequest("GET", "/sessions", nil)
	// No X-Agent-ID header
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
