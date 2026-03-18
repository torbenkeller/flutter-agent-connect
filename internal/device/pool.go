package device

import (
	"fmt"
	"sync"

	"github.com/torbenkeller/flutter-agent-connect/pkg/models"
)

// Pool tracks FAC-managed devices across iOS and Android.
type Pool struct {
	mu      sync.RWMutex
	devices map[string]*ManagedDevice // udid/avdName -> managed device
	sim     *IOSSimulator
	emu     *AndroidEmulator
}

// ManagedDevice wraps a device model with platform-specific runtime info.
type ManagedDevice struct {
	models.Device
	// Android-specific: the adb serial (e.g., "emulator-5554")
	ADBSerial string `json:"adb_serial,omitempty"`
}

func NewPool() *Pool {
	return &Pool{
		devices: make(map[string]*ManagedDevice),
		sim:     &IOSSimulator{},
		emu:     NewAndroidEmulator(),
	}
}

// Discover scans for available simulators/emulators (for info logging).
func (p *Pool) Discover() (int, error) {
	count := 0

	iosDevices, err := p.sim.ListAll()
	if err == nil {
		count += len(iosDevices)
	}

	if p.emu.IsAvailable() {
		androidDevices, err := p.emu.ListAll()
		if err == nil {
			count += len(androidDevices)
		}
	}

	return count, nil
}

// CreateDevice creates a new device (simulator or emulator) for an agent's session.
func (p *Pool) CreateDevice(agentID, sessionName string, platform models.PlatformType, deviceType, runtime string) (*ManagedDevice, error) {
	switch platform {
	case models.PlatformIOS:
		dev, err := p.sim.Create(agentID, sessionName, deviceType, runtime)
		if err != nil {
			return nil, err
		}
		managed := &ManagedDevice{Device: *dev}
		p.mu.Lock()
		p.devices[dev.UDID] = managed
		p.mu.Unlock()
		return managed, nil

	case models.PlatformAndroid:
		if !p.emu.IsAvailable() {
			return nil, fmt.Errorf("Android SDK not found. Set ANDROID_HOME or install Android SDK")
		}
		dev, err := p.emu.Create(agentID, sessionName, deviceType, runtime)
		if err != nil {
			return nil, err
		}
		managed := &ManagedDevice{Device: *dev}
		p.mu.Lock()
		p.devices[dev.UDID] = managed
		p.mu.Unlock()
		return managed, nil

	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
}

// BootDevice boots a device (simulator or emulator).
func (p *Pool) BootDevice(udid string, platform models.PlatformType) (string, error) {
	switch platform {
	case models.PlatformIOS:
		if err := p.sim.Boot(udid); err != nil {
			return "", err
		}
		p.mu.Lock()
		if dev, ok := p.devices[udid]; ok {
			dev.State = models.DeviceStateBooted
		}
		p.mu.Unlock()
		return udid, nil

	case models.PlatformAndroid:
		serial, err := p.emu.Boot(udid) // udid is the AVD name
		if err != nil {
			return "", err
		}
		p.mu.Lock()
		if dev, ok := p.devices[udid]; ok {
			dev.State = models.DeviceStateBooted
			dev.ADBSerial = serial
		}
		p.mu.Unlock()
		return serial, nil

	default:
		return "", fmt.Errorf("unsupported platform: %s", platform)
	}
}

// DeleteDevice deletes a device permanently.
func (p *Pool) DeleteDevice(udid string, platform models.PlatformType) error {
	p.mu.RLock()
	managed, ok := p.devices[udid]
	p.mu.RUnlock()

	switch platform {
	case models.PlatformIOS:
		if err := p.sim.Delete(udid); err != nil {
			return err
		}
	case models.PlatformAndroid:
		if ok && managed.ADBSerial != "" {
			_ = p.emu.Shutdown(managed.ADBSerial)
		}
		if err := p.emu.Delete(udid); err != nil {
			return err
		}
	}

	p.mu.Lock()
	delete(p.devices, udid)
	p.mu.Unlock()
	return nil
}

// GetManaged returns the managed device for a UDID.
func (p *Pool) GetManaged(udid string) *ManagedDevice {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.devices[udid]
}

// List returns all known FAC-managed devices.
func (p *Pool) List() []models.Device {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]models.Device, 0, len(p.devices))
	for _, d := range p.devices {
		result = append(result, d.Device)
	}
	return result
}

// Screenshot takes a screenshot of a device, dispatching by platform.
func (p *Pool) Screenshot(udid string, platform models.PlatformType) ([]byte, error) {
	switch platform {
	case models.PlatformAndroid:
		p.mu.RLock()
		managed, ok := p.devices[udid]
		p.mu.RUnlock()
		if !ok || managed.ADBSerial == "" {
			return nil, fmt.Errorf("Android device not found or not booted: %s", udid)
		}
		return p.emu.Screenshot(managed.ADBSerial)
	default:
		return p.sim.Screenshot(udid)
	}
}

// DeviceInfo returns the screen dimensions for a device.
func (p *Pool) DeviceInfo(udid string) (width, height int, err error) {
	p.mu.RLock()
	managed, ok := p.devices[udid]
	p.mu.RUnlock()
	if !ok || managed.ADBSerial == "" {
		return 0, 0, fmt.Errorf("device not found or not booted: %s", udid)
	}
	return p.emu.GetDeviceInfo(managed.ADBSerial)
}
