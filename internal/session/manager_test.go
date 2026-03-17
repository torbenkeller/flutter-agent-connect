package session

import (
	"testing"

	"github.com/torben/flutter-agent-connect/internal/device"
	"github.com/torben/flutter-agent-connect/pkg/models"
)

func newTestManager() *Manager {
	pool := device.NewPool()
	return NewManager(pool, "flutter")
}

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

func TestListSessionsScoping(t *testing.T) {
	m := newTestManager()
	m.RegisterAgent("agent-1")
	m.RegisterAgent("agent-2")

	// Manually add sessions (bypassing device creation which needs xcrun)
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

	// Agent-1 should see 2 sessions (not destroyed, not agent-2's)
	list1 := m.ListSessions("agent-1")
	if len(list1) != 2 {
		t.Errorf("agent-1: expected 2 sessions, got %d", len(list1))
	}

	// Agent-2 should see 1 session
	list2 := m.ListSessions("agent-2")
	if len(list2) != 1 {
		t.Errorf("agent-2: expected 1 session, got %d", len(list2))
	}

	// Unknown agent should see 0
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

	// Agent-1 can access their session
	s, err := m.GetSession("agent-1", "s1")
	if err != nil {
		t.Fatalf("agent-1 should access s1: %v", err)
	}
	if s.Name != "ios" {
		t.Errorf("expected name 'ios', got '%s'", s.Name)
	}

	// Agent-2 cannot access agent-1's session
	_, err = m.GetSession("agent-2", "s1")
	if err == nil {
		t.Error("agent-2 should NOT be able to access agent-1's session")
	}

	// Non-existent session
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

	// Find by name
	s, err := m.FindSession("agent-1", "ios-main")
	if err != nil {
		t.Fatalf("should find by name: %v", err)
	}
	if s.ID != "abc123" {
		t.Errorf("expected ID 'abc123', got '%s'", s.ID)
	}

	// Find by ID
	s, err = m.FindSession("agent-1", "def456")
	if err != nil {
		t.Fatalf("should find by ID: %v", err)
	}
	if s.Name != "android" {
		t.Errorf("expected name 'android', got '%s'", s.Name)
	}

	// Wrong agent can't find it
	_, err = m.FindSession("agent-2", "ios-main")
	if err == nil {
		t.Error("agent-2 should not find agent-1's session")
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
