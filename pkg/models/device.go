package models

type DeviceState string

const (
	DeviceStateShutdown     DeviceState = "Shutdown"
	DeviceStateBooted       DeviceState = "Booted"
	DeviceStateShuttingDown DeviceState = "Shutting Down"
)

type Device struct {
	UDID      string       `json:"udid"`
	Name      string       `json:"name"`
	Platform  PlatformType `json:"platform"`
	Runtime   string       `json:"runtime"`
	State     DeviceState  `json:"state"`
	Available bool         `json:"available"`
}
