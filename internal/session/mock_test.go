package session

import (
	"encoding/json"
	"fmt"

	"github.com/torbenkeller/flutter-agent-connect/internal/device"
	"github.com/torbenkeller/flutter-agent-connect/internal/flutter"
	"github.com/torbenkeller/flutter-agent-connect/pkg/models"
)

// --- Mock DevicePool ---

type mockDevicePool struct {
	devices     map[string]*device.ManagedDevice
	createErr   error
	bootErr     error
	deleteErr   error
	screenshot  []byte
	screenshotE error
	deviceInfoW int
	deviceInfoH int
	deviceInfoE error
	bootSerial  string // returned by BootDevice for Android
}

func newMockDevicePool() *mockDevicePool {
	return &mockDevicePool{devices: make(map[string]*device.ManagedDevice)}
}

func (p *mockDevicePool) CreateDevice(agentID, sessionName string, platform models.PlatformType, deviceType, runtime string) (*device.ManagedDevice, error) {
	if p.createErr != nil {
		return nil, p.createErr
	}
	d := &device.ManagedDevice{
		Device: models.Device{
			UDID:     fmt.Sprintf("mock-udid-%s-%s", agentID, sessionName),
			Name:     fmt.Sprintf("fac-%s-%s", agentID, sessionName),
			Platform: platform,
			State:    models.DeviceStateShutdown,
		},
	}
	p.devices[d.UDID] = d
	return d, nil
}

func (p *mockDevicePool) BootDevice(udid string, platform models.PlatformType) (string, error) {
	if p.bootErr != nil {
		return "", p.bootErr
	}
	if d, ok := p.devices[udid]; ok {
		d.State = models.DeviceStateBooted
		if platform == models.PlatformAndroid && p.bootSerial != "" {
			d.ADBSerial = p.bootSerial
			return p.bootSerial, nil
		}
	}
	return udid, nil
}

func (p *mockDevicePool) DeleteDevice(udid string, platform models.PlatformType) error {
	if p.deleteErr != nil {
		return p.deleteErr
	}
	delete(p.devices, udid)
	return nil
}

func (p *mockDevicePool) GetManaged(udid string) *device.ManagedDevice {
	return p.devices[udid]
}

func (p *mockDevicePool) Screenshot(udid string, platform models.PlatformType) ([]byte, error) {
	return p.screenshot, p.screenshotE
}

func (p *mockDevicePool) DeviceInfo(_ string) (width, height int, err error) {
	return p.deviceInfoW, p.deviceInfoH, p.deviceInfoE
}

func (p *mockDevicePool) List() []models.Device {
	result := make([]models.Device, 0, len(p.devices))
	for _, d := range p.devices {
		result = append(result, d.Device)
	}
	return result
}

// --- Mock Interactor ---

type mockInteractor struct {
	tapCalls   []tapCall
	typeCalls  []typeCall
	swipeCalls []swipeCall
	tapErr     error
	typeErr    error
	swipeErr   error
}

type tapCall struct {
	DeviceID string
	X, Y     int
}

type typeCall struct {
	DeviceID string
	Text     string
	Clear    bool
	Enter    bool
}

type swipeCall struct {
	DeviceID  string
	Direction string
}

func (i *mockInteractor) Tap(deviceID string, x, y int) error {
	i.tapCalls = append(i.tapCalls, tapCall{deviceID, x, y})
	return i.tapErr
}

func (i *mockInteractor) TypeText(deviceID, text string, clearField, enter bool) error {
	i.typeCalls = append(i.typeCalls, typeCall{deviceID, text, clearField, enter})
	return i.typeErr
}

func (i *mockInteractor) Swipe(deviceID, direction string, w, h, durationMs int) error {
	i.swipeCalls = append(i.swipeCalls, swipeCall{deviceID, direction})
	return i.swipeErr
}

// --- Mock FlutterProcess ---

type mockFlutterProcess struct {
	appID      string
	wsURI      string
	running    bool
	startedCh  chan struct{}
	stoppedCh  chan struct{}
	err        error
	logs       []flutter.LogEntry
	reloadErr  error
	restartErr error
	stopErr    error
}

func newMockFlutterProcess() *mockFlutterProcess {
	p := &mockFlutterProcess{
		appID:     "mock-app-id",
		wsURI:     "ws://127.0.0.1:12345/mock/ws",
		running:   true,
		startedCh: make(chan struct{}),
		stoppedCh: make(chan struct{}),
	}
	close(p.startedCh)
	return p
}

func (p *mockFlutterProcess) AppID() string                    { return p.appID }
func (p *mockFlutterProcess) VMServiceURI() string             { return p.wsURI }
func (p *mockFlutterProcess) IsRunning() bool                  { return p.running }
func (p *mockFlutterProcess) HotReload() error                 { return p.reloadErr }
func (p *mockFlutterProcess) HotRestart() error                { return p.restartErr }
func (p *mockFlutterProcess) Stop() error                      { return p.stopErr }
func (p *mockFlutterProcess) Kill()                            {}
func (p *mockFlutterProcess) Logs(last int) []flutter.LogEntry { return p.logs }
func (p *mockFlutterProcess) Started() <-chan struct{}         { return p.startedCh }
func (p *mockFlutterProcess) Stopped() <-chan struct{}         { return p.stoppedCh }
func (p *mockFlutterProcess) Err() error                       { return p.err }

// --- Mock VMService ---

type mockVMService struct {
	semanticsTree *flutter.SemanticsNode
	widgetTree    string
	renderTree    string
	toggleResult  bool
	callExtResult json.RawMessage
	callExtErr    error
	semanticsErr  error
	widgetsErr    error
	renderErr     error
	toggleErr     error
	closed        bool
}

func (v *mockVMService) CallExtension(method string, args map[string]any) (json.RawMessage, error) {
	return v.callExtResult, v.callExtErr
}

func (v *mockVMService) GetSemanticsTree() (*flutter.SemanticsNode, error) {
	return v.semanticsTree, v.semanticsErr
}

func (v *mockVMService) GetWidgetTree() (string, error) {
	return v.widgetTree, v.widgetsErr
}

func (v *mockVMService) GetRenderTree() (string, error) {
	return v.renderTree, v.renderErr
}

func (v *mockVMService) ToggleDebugFlag(extension string) (bool, error) {
	return v.toggleResult, v.toggleErr
}

func (v *mockVMService) Close() {
	v.closed = true
}
