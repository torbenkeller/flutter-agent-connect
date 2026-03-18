package api

import (
	"github.com/torbenkeller/flutter-agent-connect/internal/flutter"
	"github.com/torbenkeller/flutter-agent-connect/internal/session"
	"github.com/torbenkeller/flutter-agent-connect/pkg/models"
)

// SessionService defines what the API handlers need from the session layer.
type SessionService interface {
	RegisterAgent(agentID string) *models.Agent
	CreateSession(agentID string, platform models.PlatformType, deviceType, name, workDir string) (*models.Session, error)
	ListSessions(agentID string) []*models.Session
	GetSession(agentID, sessionID string) (*models.Session, error)
	DestroySession(agentID, sessionID string) error

	StartApp(agentID, sessionID, target string) (*session.AppStartResult, error)
	StopApp(agentID, sessionID string) error
	HotReload(agentID, sessionID string) error
	HotRestart(agentID, sessionID string) error
	FlutterClean(agentID, sessionID string) (*session.CommandResult, error)
	FlutterPubGet(agentID, sessionID string) (*session.CommandResult, error)
	FlutterVersion() (any, error)

	Screenshot(agentID, sessionID string) ([]byte, error)
	DeviceTap(agentID, sessionID, label, key string, x, y float64, index int) (*session.TapResult, error)
	DeviceSwipe(agentID, sessionID, direction string, durationMs int) error
	DeviceType(agentID, sessionID, text string, clear, enter bool) error

	InspectWidgets(agentID, sessionID string) (string, error)
	InspectRender(agentID, sessionID string) (string, error)
	InspectSemantics(agentID, sessionID string) (*flutter.SemanticsNode, error)
	ToggleDebugFlag(agentID, sessionID, flag string) (bool, error)
	GetLogs(agentID, sessionID string, tail int) ([]string, error)

	AddForward(agentID, sessionID string, containerPort int, envName string) (*session.PortForward, error)
	ListForwards(agentID, sessionID string) ([]session.PortForward, error)
}

// DeviceLister lists known devices.
type DeviceLister interface {
	List() []models.Device
}
