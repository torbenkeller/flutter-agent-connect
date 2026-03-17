package device

import (
	"sync"

	"github.com/torben/flutter-agent-connect/pkg/models"
)

// Pool tracks FAC-managed devices.
type Pool struct {
	mu      sync.RWMutex
	devices map[string]*models.Device // udid -> device
	sim     *IOSSimulator
}

func NewPool() *Pool {
	return &Pool{
		devices: make(map[string]*models.Device),
		sim:     &IOSSimulator{},
	}
}

// Discover scans for existing FAC-managed simulators (from previous runs).
func (p *Pool) Discover() (int, error) {
	// Count all available simulators for info logging
	all, err := p.sim.ListAll()
	if err != nil {
		return 0, err
	}
	return len(all), nil
}

// CreateDevice creates a new simulator for an agent's session.
func (p *Pool) CreateDevice(agentID, sessionName, deviceType, runtime string) (*models.Device, error) {
	dev, err := p.sim.Create(agentID, sessionName, deviceType, runtime)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	p.devices[dev.UDID] = dev
	p.mu.Unlock()

	return dev, nil
}

// BootDevice boots a simulator.
func (p *Pool) BootDevice(udid string) error {
	if err := p.sim.Boot(udid); err != nil {
		return err
	}

	p.mu.Lock()
	if dev, ok := p.devices[udid]; ok {
		dev.State = models.DeviceStateBooted
	}
	p.mu.Unlock()

	return nil
}

// DeleteDevice deletes a simulator permanently.
func (p *Pool) DeleteDevice(udid string) error {
	if err := p.sim.Delete(udid); err != nil {
		return err
	}

	p.mu.Lock()
	delete(p.devices, udid)
	p.mu.Unlock()

	return nil
}

// ListForAgent returns devices belonging to a specific agent.
func (p *Pool) ListForAgent(agentID string) ([]models.Device, error) {
	return p.sim.ListForAgent(agentID)
}

// List returns all known FAC-managed devices (for the /devices endpoint).
func (p *Pool) List() []models.Device {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]models.Device, 0, len(p.devices))
	for _, d := range p.devices {
		result = append(result, *d)
	}
	return result
}

// Simulator returns the underlying iOS simulator for direct access.
func (p *Pool) Simulator() *IOSSimulator {
	return p.sim
}
