package session

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/torbenkeller/flutter-agent-connect/internal/device"
	"github.com/torbenkeller/flutter-agent-connect/internal/flutter"
	"github.com/torbenkeller/flutter-agent-connect/pkg/models"
)

func newTestManager(opts ...Option) *Manager {
	pool := newMockDevicePool()
	return NewManager(pool, "flutter", opts...)
}

func newTestManagerWithPool(pool *mockDevicePool, opts ...Option) *Manager {
	return NewManager(pool, "flutter", opts...)
}

// --- Agent Tests ---

func TestRegisterAgent(t *testing.T) {
	m := newTestManager()

	a1 := m.RegisterAgent("agent-1")
	if a1.ID != "agent-1" {
		t.Errorf("expected agent ID 'agent-1', got '%s'", a1.ID)
	}

	// Registering same agent again returns same instance
	a2 := m.RegisterAgent("agent-1")
	if a1 != a2 {
		t.Error("expected same agent instance on re-register")
	}

	// Different agent
	a3 := m.RegisterAgent("agent-2")
	if a3.ID != "agent-2" {
		t.Errorf("expected agent ID 'agent-2', got '%s'", a3.ID)
	}
}

// --- Session Scoping Tests ---

func TestListSessionsScoping(t *testing.T) {
	m := newTestManager()
	m.RegisterAgent("agent-1")
	m.RegisterAgent("agent-2")

	m.mu.Lock()
	m.sessions["s1"] = &Session{
		Session: models.Session{ID: "s1", AgentID: "agent-1", Name: "ios", State: models.SessionStateCreated},
	}
	m.sessions["s2"] = &Session{
		Session: models.Session{ID: "s2", AgentID: "agent-1", Name: "android", State: models.SessionStateRunning},
	}
	m.sessions["s3"] = &Session{
		Session: models.Session{ID: "s3", AgentID: "agent-2", Name: "ios", State: models.SessionStateCreated},
	}
	m.sessions["s4"] = &Session{
		Session: models.Session{ID: "s4", AgentID: "agent-1", Name: "old", State: models.SessionStateDestroyed},
	}
	m.mu.Unlock()

	list1 := m.ListSessions("agent-1")
	if len(list1) != 2 {
		t.Errorf("agent-1: expected 2 sessions, got %d", len(list1))
	}

	list2 := m.ListSessions("agent-2")
	if len(list2) != 1 {
		t.Errorf("agent-2: expected 1 session, got %d", len(list2))
	}

	list3 := m.ListSessions("unknown")
	if len(list3) != 0 {
		t.Errorf("unknown: expected 0 sessions, got %d", len(list3))
	}
}

func TestGetSessionScoping(t *testing.T) {
	m := newTestManager()

	m.mu.Lock()
	m.sessions["s1"] = &Session{
		Session: models.Session{ID: "s1", AgentID: "agent-1", Name: "ios"},
	}
	m.mu.Unlock()

	s, err := m.GetSession("agent-1", "s1")
	if err != nil {
		t.Fatalf("agent-1 should access s1: %v", err)
	}
	if s.Name != "ios" {
		t.Errorf("expected name 'ios', got '%s'", s.Name)
	}

	_, err = m.GetSession("agent-2", "s1")
	if err == nil {
		t.Error("agent-2 should NOT be able to access agent-1's session")
	}

	_, err = m.GetSession("agent-1", "nonexistent")
	if err == nil {
		t.Error("should error on non-existent session")
	}
}

func TestFindSessionByName(t *testing.T) {
	m := newTestManager()

	m.mu.Lock()
	m.sessions["abc123"] = &Session{
		Session: models.Session{ID: "abc123", AgentID: "agent-1", Name: "ios-main"},
	}
	m.sessions["def456"] = &Session{
		Session: models.Session{ID: "def456", AgentID: "agent-1", Name: "android"},
	}
	m.mu.Unlock()

	s, err := m.FindSession("agent-1", "ios-main")
	if err != nil {
		t.Fatalf("should find by name: %v", err)
	}
	if s.ID != "abc123" {
		t.Errorf("expected ID 'abc123', got '%s'", s.ID)
	}

	s, err = m.FindSession("agent-1", "def456")
	if err != nil {
		t.Fatalf("should find by ID: %v", err)
	}
	if s.Name != "android" {
		t.Errorf("expected name 'android', got '%s'", s.Name)
	}

	_, err = m.FindSession("agent-2", "ios-main")
	if err == nil {
		t.Error("agent-2 should not find agent-1's session")
	}
}

// --- CreateSession Tests ---

func TestCreateSession(t *testing.T) {
	pool := newMockDevicePool()
	m := newTestManagerWithPool(pool)

	s, err := m.CreateSession("agent-1", models.PlatformIOS, "", "test", "/tmp/app")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	if s.AgentID != "agent-1" {
		t.Errorf("expected agent-1, got %s", s.AgentID)
	}
	if s.Name != "test" {
		t.Errorf("expected name 'test', got '%s'", s.Name)
	}
	if s.Platform != models.PlatformIOS {
		t.Errorf("expected ios, got %s", s.Platform)
	}
	if s.State != models.SessionStateCreated {
		t.Errorf("expected state created, got %s", s.State)
	}
	if s.WorkDir != "/tmp/app" {
		t.Errorf("expected work dir '/tmp/app', got '%s'", s.WorkDir)
	}
	if s.Device == nil {
		t.Fatal("expected device to be assigned")
	}
}

func TestCreateSessionDuplicateName(t *testing.T) {
	pool := newMockDevicePool()
	m := newTestManagerWithPool(pool)

	_, err := m.CreateSession("agent-1", models.PlatformIOS, "", "test", "/tmp/app")
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	_, err = m.CreateSession("agent-1", models.PlatformIOS, "", "test", "/tmp/app")
	if err == nil {
		t.Error("should reject duplicate session name")
	}
}

func TestCreateSessionDeviceFailure(t *testing.T) {
	pool := newMockDevicePool()
	pool.createErr = fmt.Errorf("simctl failed")
	m := newTestManagerWithPool(pool)

	_, err := m.CreateSession("agent-1", models.PlatformIOS, "", "test", "/tmp/app")
	if err == nil {
		t.Error("should propagate device creation error")
	}
}

func TestCreateSessionBootFailure(t *testing.T) {
	pool := newMockDevicePool()
	pool.bootErr = fmt.Errorf("boot failed")
	m := newTestManagerWithPool(pool)

	_, err := m.CreateSession("agent-1", models.PlatformIOS, "", "test", "/tmp/app")
	if err == nil {
		t.Error("should propagate boot error")
	}

	// Device should have been cleaned up
	if len(pool.devices) != 0 {
		t.Error("device should be deleted after boot failure")
	}
}

// --- StartApp Tests ---

func TestStartAppSuccess(t *testing.T) {
	pool := newMockDevicePool()
	mockProc := newMockFlutterProcess(true)

	starter := func(bin, dir, device, target string, defines []string) (FlutterProcess, error) {
		return mockProc, nil
	}

	m := newTestManagerWithPool(pool, WithFlutterStarter(starter))

	m.mu.Lock()
	m.sessions["s1"] = &Session{
		Session:         models.Session{ID: "s1", AgentID: "a1", WorkDir: "/tmp/app", State: models.SessionStateCreated},
		flutterDeviceID: "mock-udid",
	}
	m.mu.Unlock()

	result, err := m.StartApp("a1", "s1", "lib/main.dart")
	if err != nil {
		t.Fatalf("StartApp failed: %v", err)
	}

	if result.State != "running" {
		t.Errorf("expected state 'running', got '%s'", result.State)
	}
	if result.AppID != "mock-app-id" {
		t.Errorf("expected app ID 'mock-app-id', got '%s'", result.AppID)
	}
}

func TestStartAppBuildFailure(t *testing.T) {
	pool := newMockDevicePool()
	mockProc := &mockFlutterProcess{
		startedCh: make(chan struct{}),
		stoppedCh: make(chan struct{}),
		err:       fmt.Errorf("exit status 1"),
		logs: []flutter.LogEntry{
			{Message: "lib/main.dart:4:23: Error: Expected ';' after this."},
			{Message: "  runApp(const MyApp())"},
		},
	}
	close(mockProc.stoppedCh) // process exited immediately

	starter := func(bin, dir, device, target string, defines []string) (FlutterProcess, error) {
		return mockProc, nil
	}

	m := newTestManagerWithPool(pool, WithFlutterStarter(starter))

	m.mu.Lock()
	m.sessions["s1"] = &Session{
		Session:         models.Session{ID: "s1", AgentID: "a1", WorkDir: "/tmp/app", State: models.SessionStateCreated},
		flutterDeviceID: "mock-udid",
	}
	m.mu.Unlock()

	result, err := m.StartApp("a1", "s1", "lib/main.dart")
	if err == nil {
		t.Fatal("expected build error")
	}

	buildErr, ok := err.(*BuildError)
	if !ok {
		t.Fatalf("expected *BuildError, got %T: %v", err, err)
	}

	if len(buildErr.BuildOutput) != 2 {
		t.Errorf("expected 2 build output lines, got %d", len(buildErr.BuildOutput))
	}

	// Result should also contain build output
	if result == nil {
		t.Fatal("expected result even on failure")
	}
	if result.State != "failed" {
		t.Errorf("expected state 'failed', got '%s'", result.State)
	}
}

func TestStartAppNoWorkDir(t *testing.T) {
	m := newTestManager()

	m.mu.Lock()
	m.sessions["s1"] = &Session{
		Session: models.Session{ID: "s1", AgentID: "agent-1", WorkDir: ""},
	}
	m.mu.Unlock()

	_, err := m.StartApp("agent-1", "s1", "lib/main.dart")
	if err == nil {
		t.Error("should error when work_dir is empty")
	}
}

func TestStartAppAlreadyRunning(t *testing.T) {
	m := newTestManager()
	mockProc := newMockFlutterProcess(true)

	m.mu.Lock()
	m.sessions["s1"] = &Session{
		Session:        models.Session{ID: "s1", AgentID: "a1", WorkDir: "/tmp/app"},
		flutterProcess: mockProc,
	}
	m.mu.Unlock()

	_, err := m.StartApp("a1", "s1", "lib/main.dart")
	if err == nil {
		t.Error("should reject start when app is already running")
	}
}

// --- HotReload / HotRestart Tests ---

func TestHotReload(t *testing.T) {
	m := newTestManager()
	mockProc := newMockFlutterProcess(true)

	m.mu.Lock()
	m.sessions["s1"] = &Session{
		Session:        models.Session{ID: "s1", AgentID: "a1"},
		flutterProcess: mockProc,
	}
	m.mu.Unlock()

	if err := m.HotReload("a1", "s1"); err != nil {
		t.Fatalf("HotReload failed: %v", err)
	}
}

func TestHotReloadNoApp(t *testing.T) {
	m := newTestManager()

	m.mu.Lock()
	m.sessions["s1"] = &Session{
		Session: models.Session{ID: "s1", AgentID: "a1"},
	}
	m.mu.Unlock()

	err := m.HotReload("a1", "s1")
	if err == nil {
		t.Error("should error when no app running")
	}
}

func TestHotRestart(t *testing.T) {
	m := newTestManager()
	mockProc := newMockFlutterProcess(true)

	m.mu.Lock()
	m.sessions["s1"] = &Session{
		Session:        models.Session{ID: "s1", AgentID: "a1"},
		flutterProcess: mockProc,
	}
	m.mu.Unlock()

	if err := m.HotRestart("a1", "s1"); err != nil {
		t.Fatalf("HotRestart failed: %v", err)
	}
}

// --- StopApp Tests ---

func TestStopApp(t *testing.T) {
	m := newTestManager()
	mockProc := newMockFlutterProcess(true)

	m.mu.Lock()
	m.sessions["s1"] = &Session{
		Session:        models.Session{ID: "s1", AgentID: "a1", State: models.SessionStateRunning},
		flutterProcess: mockProc,
	}
	m.mu.Unlock()

	if err := m.StopApp("a1", "s1"); err != nil {
		t.Fatalf("StopApp failed: %v", err)
	}

	s, _ := m.GetSession("a1", "s1")
	if s.State != models.SessionStateStopped {
		t.Errorf("expected state stopped, got %s", s.State)
	}
}

func TestStopAppNoApp(t *testing.T) {
	m := newTestManager()

	m.mu.Lock()
	m.sessions["s1"] = &Session{
		Session: models.Session{ID: "s1", AgentID: "a1"},
	}
	m.mu.Unlock()

	err := m.StopApp("a1", "s1")
	if err == nil {
		t.Error("should error when no app running")
	}
}

// --- Screenshot Tests ---

func TestScreenshot(t *testing.T) {
	pool := newMockDevicePool()
	pool.screenshot = []byte("fake-png-data")

	m := newTestManagerWithPool(pool)

	m.mu.Lock()
	m.sessions["s1"] = &Session{
		Session: models.Session{ID: "s1", AgentID: "a1", Device: &models.Device{UDID: "test-udid"}, Platform: models.PlatformIOS},
	}
	m.mu.Unlock()

	data, err := m.Screenshot("a1", "s1")
	if err != nil {
		t.Fatalf("Screenshot failed: %v", err)
	}

	if string(data) != "fake-png-data" {
		t.Errorf("unexpected screenshot data: %s", string(data))
	}
}

func TestScreenshotNoDevice(t *testing.T) {
	m := newTestManager()

	m.mu.Lock()
	m.sessions["s1"] = &Session{
		Session: models.Session{ID: "s1", AgentID: "a1"},
	}
	m.mu.Unlock()

	_, err := m.Screenshot("a1", "s1")
	if err == nil {
		t.Error("should error when no device")
	}
}

// --- DeviceTap Tests ---

func TestDeviceTapByCoordinates(t *testing.T) {
	pool := newMockDevicePool()
	pool.devices["test-udid"] = &device.ManagedDevice{
		Device: models.Device{UDID: "test-udid", Platform: models.PlatformIOS},
	}

	ios := &mockInteractor{}
	m := newTestManagerWithPool(pool, WithInteractors(map[models.PlatformType]Interactor{
		models.PlatformIOS: ios,
	}))

	m.mu.Lock()
	m.sessions["s1"] = &Session{
		Session: models.Session{ID: "s1", AgentID: "a1", Device: &models.Device{UDID: "test-udid"}, Platform: models.PlatformIOS},
	}
	m.mu.Unlock()

	result, err := m.DeviceTap("a1", "s1", "", "", 100, 200, 0)
	if err != nil {
		t.Fatalf("DeviceTap failed: %v", err)
	}

	if !result.Success {
		t.Error("expected success")
	}
	if result.X != 100 || result.Y != 200 {
		t.Errorf("expected (100, 200), got (%d, %d)", result.X, result.Y)
	}

	if len(ios.tapCalls) != 1 {
		t.Fatalf("expected 1 tap call, got %d", len(ios.tapCalls))
	}
	if ios.tapCalls[0].X != 100 || ios.tapCalls[0].Y != 200 {
		t.Errorf("tap called with wrong coords: (%d, %d)", ios.tapCalls[0].X, ios.tapCalls[0].Y)
	}
}

func TestDeviceTapByLabel(t *testing.T) {
	pool := newMockDevicePool()
	pool.devices["test-udid"] = &device.ManagedDevice{
		Device: models.Device{UDID: "test-udid", Platform: models.PlatformIOS},
	}

	ios := &mockInteractor{}
	mockVM := &mockVMService{
		callExtResult: json.RawMessage(`{}`),
		semanticsTree: &flutter.SemanticsNode{
			ID: 0,
			Children: []*flutter.SemanticsNode{
				{
					ID:    1,
					Label: "Login",
					Rect:  &flutter.Rect{Left: 100, Top: 200, Right: 200, Bottom: 250},
				},
			},
		},
	}

	m := newTestManagerWithPool(pool, WithInteractors(map[models.PlatformType]Interactor{
		models.PlatformIOS: ios,
	}))

	mockProc := newMockFlutterProcess(true)

	m.mu.Lock()
	m.sessions["s1"] = &Session{
		Session:         models.Session{ID: "s1", AgentID: "a1", Device: &models.Device{UDID: "test-udid"}, Platform: models.PlatformIOS},
		flutterProcess:  mockProc,
		vmServiceClient: mockVM,
	}
	m.mu.Unlock()

	result, err := m.DeviceTap("a1", "s1", "Login", "", 0, 0, 0)
	if err != nil {
		t.Fatalf("DeviceTap by label failed: %v", err)
	}

	if result.Element != "Login" {
		t.Errorf("expected element 'Login', got '%s'", result.Element)
	}

	// Center of (100, 200, 200, 250) = (150, 225)
	if result.X != 150 || result.Y != 225 {
		t.Errorf("expected (150, 225), got (%d, %d)", result.X, result.Y)
	}
}

func TestDeviceTapLabelNotFound(t *testing.T) {
	pool := newMockDevicePool()
	pool.devices["test-udid"] = &device.ManagedDevice{
		Device: models.Device{UDID: "test-udid", Platform: models.PlatformIOS},
	}

	mockVM := &mockVMService{
		callExtResult: json.RawMessage(`{}`),
		semanticsTree: &flutter.SemanticsNode{ID: 0},
	}

	m := newTestManagerWithPool(pool)
	mockProc := newMockFlutterProcess(true)

	m.mu.Lock()
	m.sessions["s1"] = &Session{
		Session:         models.Session{ID: "s1", AgentID: "a1", Device: &models.Device{UDID: "test-udid"}, Platform: models.PlatformIOS},
		flutterProcess:  mockProc,
		vmServiceClient: mockVM,
	}
	m.mu.Unlock()

	_, err := m.DeviceTap("a1", "s1", "NonExistent", "", 0, 0, 0)
	if err == nil {
		t.Error("should error when label not found")
	}
}

// --- DeviceType Tests ---

func TestDeviceType(t *testing.T) {
	pool := newMockDevicePool()
	pool.devices["test-udid"] = &device.ManagedDevice{
		Device: models.Device{UDID: "test-udid", Platform: models.PlatformIOS},
	}

	ios := &mockInteractor{}
	m := newTestManagerWithPool(pool, WithInteractors(map[models.PlatformType]Interactor{
		models.PlatformIOS: ios,
	}))

	m.mu.Lock()
	m.sessions["s1"] = &Session{
		Session: models.Session{ID: "s1", AgentID: "a1", Device: &models.Device{UDID: "test-udid"}, Platform: models.PlatformIOS},
	}
	m.mu.Unlock()

	if err := m.DeviceType("a1", "s1", "hello@test.com", true, true); err != nil {
		t.Fatalf("DeviceType failed: %v", err)
	}

	if len(ios.typeCalls) != 1 {
		t.Fatalf("expected 1 type call, got %d", len(ios.typeCalls))
	}
	call := ios.typeCalls[0]
	if call.Text != "hello@test.com" || !call.Clear || !call.Enter {
		t.Errorf("unexpected type call: %+v", call)
	}
}

// --- DeviceSwipe Tests ---

func TestDeviceSwipe(t *testing.T) {
	pool := newMockDevicePool()
	pool.devices["test-udid"] = &device.ManagedDevice{
		Device: models.Device{UDID: "test-udid", Platform: models.PlatformIOS},
	}

	ios := &mockInteractor{}
	m := newTestManagerWithPool(pool, WithInteractors(map[models.PlatformType]Interactor{
		models.PlatformIOS: ios,
	}))

	m.mu.Lock()
	m.sessions["s1"] = &Session{
		Session: models.Session{ID: "s1", AgentID: "a1", Device: &models.Device{UDID: "test-udid"}, Platform: models.PlatformIOS},
	}
	m.mu.Unlock()

	if err := m.DeviceSwipe("a1", "s1", "down", 300); err != nil {
		t.Fatalf("DeviceSwipe failed: %v", err)
	}

	if len(ios.swipeCalls) != 1 {
		t.Fatalf("expected 1 swipe call, got %d", len(ios.swipeCalls))
	}
	if ios.swipeCalls[0].Direction != "down" {
		t.Errorf("expected direction 'down', got '%s'", ios.swipeCalls[0].Direction)
	}
}

// --- Inspect Tests ---

func TestInspectWidgets(t *testing.T) {
	m := newTestManager()
	mockVM := &mockVMService{
		callExtResult: json.RawMessage(`{}`),
		widgetTree:    "MyApp\n └MaterialApp\n  └Scaffold",
	}
	mockProc := newMockFlutterProcess(true)

	m.mu.Lock()
	m.sessions["s1"] = &Session{
		Session:         models.Session{ID: "s1", AgentID: "a1"},
		flutterProcess:  mockProc,
		vmServiceClient: mockVM,
	}
	m.mu.Unlock()

	tree, err := m.InspectWidgets("a1", "s1")
	if err != nil {
		t.Fatalf("InspectWidgets failed: %v", err)
	}

	if tree != "MyApp\n └MaterialApp\n  └Scaffold" {
		t.Errorf("unexpected widget tree: %s", tree)
	}
}

func TestInspectRender(t *testing.T) {
	m := newTestManager()
	mockVM := &mockVMService{
		callExtResult: json.RawMessage(`{}`),
		renderTree:    "RenderView#abc\n └RenderSemanticsAnnotations",
	}
	mockProc := newMockFlutterProcess(true)

	m.mu.Lock()
	m.sessions["s1"] = &Session{
		Session:         models.Session{ID: "s1", AgentID: "a1"},
		flutterProcess:  mockProc,
		vmServiceClient: mockVM,
	}
	m.mu.Unlock()

	tree, err := m.InspectRender("a1", "s1")
	if err != nil {
		t.Fatalf("InspectRender failed: %v", err)
	}

	if tree != "RenderView#abc\n └RenderSemanticsAnnotations" {
		t.Errorf("unexpected render tree: %s", tree)
	}
}

func TestInspectSemantics(t *testing.T) {
	m := newTestManager()
	expected := &flutter.SemanticsNode{ID: 0, Label: "root"}
	mockVM := &mockVMService{
		callExtResult: json.RawMessage(`{}`),
		semanticsTree: expected,
	}
	mockProc := newMockFlutterProcess(true)

	m.mu.Lock()
	m.sessions["s1"] = &Session{
		Session:         models.Session{ID: "s1", AgentID: "a1"},
		flutterProcess:  mockProc,
		vmServiceClient: mockVM,
	}
	m.mu.Unlock()

	tree, err := m.InspectSemantics("a1", "s1")
	if err != nil {
		t.Fatalf("InspectSemantics failed: %v", err)
	}

	if tree.Label != "root" {
		t.Errorf("expected label 'root', got '%s'", tree.Label)
	}
}

// --- ToggleDebugFlag Tests ---

func TestToggleDebugFlag(t *testing.T) {
	m := newTestManager()
	mockVM := &mockVMService{
		callExtResult: json.RawMessage(`{}`),
		toggleResult:  true,
	}
	mockProc := newMockFlutterProcess(true)

	m.mu.Lock()
	m.sessions["s1"] = &Session{
		Session:         models.Session{ID: "s1", AgentID: "a1"},
		flutterProcess:  mockProc,
		vmServiceClient: mockVM,
	}
	m.mu.Unlock()

	enabled, err := m.ToggleDebugFlag("a1", "s1", "paint")
	if err != nil {
		t.Fatalf("ToggleDebugFlag failed: %v", err)
	}

	if !enabled {
		t.Error("expected enabled=true")
	}
}

func TestToggleDebugFlagUnknown(t *testing.T) {
	m := newTestManager()
	mockVM := &mockVMService{callExtResult: json.RawMessage(`{}`)}
	mockProc := newMockFlutterProcess(true)

	m.mu.Lock()
	m.sessions["s1"] = &Session{
		Session:         models.Session{ID: "s1", AgentID: "a1"},
		flutterProcess:  mockProc,
		vmServiceClient: mockVM,
	}
	m.mu.Unlock()

	_, err := m.ToggleDebugFlag("a1", "s1", "unknown_flag")
	if err == nil {
		t.Error("should reject unknown debug flag")
	}
}

// --- GetLogs Tests ---

func TestGetLogs(t *testing.T) {
	m := newTestManager()
	mockProc := newMockFlutterProcess(true)
	mockProc.logs = []flutter.LogEntry{
		{Message: "flutter: Starting app"},
		{Message: "flutter: Hello World"},
		{Message: "flutter: Error happened"},
	}

	m.mu.Lock()
	m.sessions["s1"] = &Session{
		Session:        models.Session{ID: "s1", AgentID: "a1"},
		flutterProcess: mockProc,
	}
	m.mu.Unlock()

	logs, err := m.GetLogs("a1", "s1", 0)
	if err != nil {
		t.Fatalf("GetLogs failed: %v", err)
	}

	if len(logs) != 3 {
		t.Errorf("expected 3 logs, got %d", len(logs))
	}
}

func TestGetLogsNoApp(t *testing.T) {
	m := newTestManager()

	m.mu.Lock()
	m.sessions["s1"] = &Session{
		Session: models.Session{ID: "s1", AgentID: "a1"},
	}
	m.mu.Unlock()

	_, err := m.GetLogs("a1", "s1", 0)
	if err == nil {
		t.Error("should error when no app running")
	}
}

// --- DestroySession Tests ---

func TestDestroySession(t *testing.T) {
	pool := newMockDevicePool()
	pool.devices["test-udid"] = &device.ManagedDevice{
		Device: models.Device{UDID: "test-udid"},
	}

	m := newTestManagerWithPool(pool)

	m.mu.Lock()
	m.sessions["s1"] = &Session{
		Session: models.Session{ID: "s1", AgentID: "a1", Device: &models.Device{UDID: "test-udid"}, Platform: models.PlatformIOS},
	}
	m.mu.Unlock()

	if err := m.DestroySession("a1", "s1"); err != nil {
		t.Fatalf("DestroySession failed: %v", err)
	}

	s, _ := m.GetSession("a1", "s1")
	if s.State != models.SessionStateDestroyed {
		t.Errorf("expected state destroyed, got %s", s.State)
	}

	// Device should be removed from pool
	if len(pool.devices) != 0 {
		t.Error("device should be deleted from pool")
	}
}

func TestDestroySessionStopsApp(t *testing.T) {
	pool := newMockDevicePool()
	pool.devices["test-udid"] = &device.ManagedDevice{
		Device: models.Device{UDID: "test-udid"},
	}
	mockProc := newMockFlutterProcess(true)

	m := newTestManagerWithPool(pool)

	m.mu.Lock()
	m.sessions["s1"] = &Session{
		Session:        models.Session{ID: "s1", AgentID: "a1", Device: &models.Device{UDID: "test-udid"}, Platform: models.PlatformIOS},
		flutterProcess: mockProc,
	}
	m.mu.Unlock()

	if err := m.DestroySession("a1", "s1"); err != nil {
		t.Fatalf("DestroySession failed: %v", err)
	}
}

// --- Port Forward Tests ---

func TestDartDefines(t *testing.T) {
	m := newTestManager()

	m.mu.Lock()
	m.sessions["s1"] = &Session{
		Session: models.Session{ID: "s1", AgentID: "a1", Platform: models.PlatformIOS},
		forwards: []PortForward{
			{ContainerPort: 8080, HostPort: 9001, EnvName: "API_URL", URLiOS: "http://localhost:9001", URLAndroid: "http://10.0.2.2:9001"},
			{ContainerPort: 5432, HostPort: 5432}, // no env name
		},
	}
	m.mu.Unlock()

	defines := m.GetDartDefines("s1", models.PlatformIOS)
	if len(defines) != 1 {
		t.Fatalf("expected 1 define, got %d", len(defines))
	}
	if defines[0] != "API_URL=http://localhost:9001" {
		t.Errorf("unexpected define: %s", defines[0])
	}

	// Android should use different URL
	definesAndroid := m.GetDartDefines("s1", models.PlatformAndroid)
	if len(definesAndroid) != 1 {
		t.Fatalf("expected 1 define, got %d", len(definesAndroid))
	}
	if definesAndroid[0] != "API_URL=http://10.0.2.2:9001" {
		t.Errorf("unexpected android define: %s", definesAndroid[0])
	}
}

// --- VMService Reconnect Tests ---

func TestVMServiceReconnectsOnStale(t *testing.T) {
	pool := newMockDevicePool()
	staleVM := &mockVMService{
		callExtErr: fmt.Errorf("connection closed"),
	}
	freshVM := &mockVMService{
		callExtResult: json.RawMessage(`{}`),
		widgetTree:    "fresh tree",
	}

	connectorCalls := 0
	connector := func(wsURI, adbSerial string) (VMService, error) {
		connectorCalls++
		return freshVM, nil
	}

	m := newTestManagerWithPool(pool, WithVMServiceConnector(connector))
	mockProc := newMockFlutterProcess(true)

	m.mu.Lock()
	m.sessions["s1"] = &Session{
		Session:         models.Session{ID: "s1", AgentID: "a1"},
		flutterProcess:  mockProc,
		vmServiceClient: staleVM,
	}
	m.mu.Unlock()

	tree, err := m.InspectWidgets("a1", "s1")
	if err != nil {
		t.Fatalf("InspectWidgets failed: %v", err)
	}

	if tree != "fresh tree" {
		t.Errorf("expected fresh tree, got '%s'", tree)
	}
	if connectorCalls != 1 {
		t.Errorf("expected 1 reconnect, got %d", connectorCalls)
	}
	if !staleVM.closed {
		t.Error("stale VM service should have been closed")
	}
}
