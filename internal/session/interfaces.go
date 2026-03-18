package session

import (
	"encoding/json"

	"github.com/torbenkeller/flutter-agent-connect/internal/device"
	"github.com/torbenkeller/flutter-agent-connect/internal/flutter"
	"github.com/torbenkeller/flutter-agent-connect/pkg/models"
)

// DevicePool abstracts device lifecycle management.
type DevicePool interface {
	CreateDevice(agentID, sessionName string, platform models.PlatformType, deviceType, runtime string) (*device.ManagedDevice, error)
	BootDevice(udid string, platform models.PlatformType) (string, error)
	DeleteDevice(udid string, platform models.PlatformType) error
	GetManaged(udid string) *device.ManagedDevice
	Screenshot(udid string, platform models.PlatformType) ([]byte, error)
	DeviceInfo(udid string) (width, height int, err error)
	List() []models.Device
}

// Interactor sends input events to a device.
type Interactor interface {
	Tap(deviceID string, x, y int) error
	TypeText(deviceID string, text string, clear, enter bool) error
	Swipe(deviceID string, direction string, screenW, screenH, durationMs int) error
}

// FlutterProcess abstracts a running flutter run --machine process.
type FlutterProcess interface {
	AppID() string
	VMServiceURI() string
	IsRunning() bool
	HotReload() error
	HotRestart() error
	Stop() error
	Kill()
	Logs(last int) []flutter.LogEntry
	Started() <-chan struct{}
	Stopped() <-chan struct{}
	Err() error
}

// FlutterStarter spawns a flutter run --machine process.
type FlutterStarter func(flutterBin, workDir, deviceID, target string, dartDefines []string) (FlutterProcess, error)

// VMService abstracts the Dart VM Service connection.
type VMService interface {
	CallExtension(method string, args map[string]any) (json.RawMessage, error)
	GetSemanticsTree() (*flutter.SemanticsNode, error)
	GetWidgetTree() (string, error)
	GetRenderTree() (string, error)
	ToggleDebugFlag(extension string) (bool, error)
	Close()
}

// VMServiceConnector creates a new VM Service connection.
type VMServiceConnector func(wsURI, adbSerial string) (VMService, error)
