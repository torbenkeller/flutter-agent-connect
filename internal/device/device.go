package device

import "github.com/torbenkeller/flutter-agent-connect/pkg/models"

// Manager defines the interface for device lifecycle management.
type Manager interface {
	List() ([]models.Device, error)
	Boot(udid string) error
	Shutdown(udid string) error
	Screenshot(udid string) ([]byte, error)
}
