package device

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/torbenkeller/flutter-agent-connect/pkg/models"
)

const facPrefix = "fac-"

// IOSSimulator wraps xcrun simctl for iOS simulator management.
type IOSSimulator struct{}

// simctl JSON structures

type simctlListOutput struct {
	Devices map[string][]simctlDevice `json:"devices"`
}

type simctlDevice struct {
	UDID        string `json:"udid"`
	Name        string `json:"name"`
	State       string `json:"state"`
	IsAvailable bool   `json:"isAvailable"`
}

type simctlDeviceTypesOutput struct {
	DeviceTypes []simctlDeviceType `json:"devicetypes"`
}

type simctlDeviceType struct {
	Name       string `json:"name"`
	Identifier string `json:"identifier"`
}

type simctlRuntimesOutput struct {
	Runtimes []simctlRuntime `json:"runtimes"`
}

type simctlRuntime struct {
	Name       string `json:"name"`
	Identifier string `json:"identifier"`
	IsAvailable bool  `json:"isAvailable"`
}

// ListAll returns all available iOS simulators.
func (s *IOSSimulator) ListAll() ([]models.Device, error) {
	out, err := exec.Command("xcrun", "simctl", "list", "devices", "-j").Output()
	if err != nil {
		return nil, fmt.Errorf("xcrun simctl list failed: %w", err)
	}

	return parseSimctlJSON(out)
}

// ListForAgent returns only FAC-managed simulators belonging to a specific agent.
func (s *IOSSimulator) ListForAgent(agentID string) ([]models.Device, error) {
	all, err := s.ListAll()
	if err != nil {
		return nil, err
	}

	return filterByAgentPrefix(all, agentID), nil
}

// parseSimctlJSON parses the JSON output of `xcrun simctl list devices -j`.
func parseSimctlJSON(data []byte) ([]models.Device, error) {
	var result simctlListOutput
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse simctl output: %w", err)
	}

	var devices []models.Device
	for runtime, devs := range result.Devices {
		for _, d := range devs {
			if !d.IsAvailable {
				continue
			}
			devices = append(devices, models.Device{
				UDID:      d.UDID,
				Name:      d.Name,
				Platform:  models.PlatformIOS,
				Runtime:   runtime,
				State:     models.DeviceState(d.State),
				Available: true,
			})
		}
	}
	return devices, nil
}

// filterByAgentPrefix filters devices by the FAC agent naming convention.
func filterByAgentPrefix(devices []models.Device, agentID string) []models.Device {
	prefix := facPrefix + agentID + "-"
	var result []models.Device
	for _, d := range devices {
		if strings.HasPrefix(d.Name, prefix) {
			result = append(result, d)
		}
	}
	return result
}

// Create creates a new simulator with FAC naming convention.
// Name format: fac-<agentID>-<sessionName>
func (s *IOSSimulator) Create(agentID, sessionName, deviceTypeName, runtimeVersion string) (*models.Device, error) {
	// Resolve device type identifier
	deviceTypeID, err := s.resolveDeviceType(deviceTypeName)
	if err != nil {
		return nil, err
	}

	// Resolve runtime identifier
	runtimeID, err := s.resolveRuntime(runtimeVersion)
	if err != nil {
		return nil, err
	}

	simName := facPrefix + agentID + "-" + sessionName

	out, err := exec.Command("xcrun", "simctl", "create", simName, deviceTypeID, runtimeID).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to create simulator: %w", err)
	}

	udid := strings.TrimSpace(string(out))

	return &models.Device{
		UDID:      udid,
		Name:      simName,
		Platform:  models.PlatformIOS,
		Runtime:   runtimeID,
		State:     models.DeviceStateShutdown,
		Available: true,
	}, nil
}

// Boot starts a simulator.
func (s *IOSSimulator) Boot(udid string) error {
	if err := exec.Command("xcrun", "simctl", "boot", udid).Run(); err != nil {
		return fmt.Errorf("failed to boot simulator %s: %w", udid, err)
	}
	return nil
}

// Shutdown stops a simulator.
func (s *IOSSimulator) Shutdown(udid string) error {
	if err := exec.Command("xcrun", "simctl", "shutdown", udid).Run(); err != nil {
		return fmt.Errorf("failed to shutdown simulator %s: %w", udid, err)
	}
	return nil
}

// Delete removes a simulator permanently.
func (s *IOSSimulator) Delete(udid string) error {
	// Shutdown first (ignore error if already shut down)
	_ = s.Shutdown(udid)

	if err := exec.Command("xcrun", "simctl", "delete", udid).Run(); err != nil {
		return fmt.Errorf("failed to delete simulator %s: %w", udid, err)
	}
	return nil
}

// Screenshot captures the simulator screen as PNG.
func (s *IOSSimulator) Screenshot(udid string) ([]byte, error) {
	out, err := exec.Command("xcrun", "simctl", "io", udid, "screenshot", "--type=png", "-").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to take screenshot: %w", err)
	}
	return out, nil
}

// resolveDeviceType maps a human-readable name like "iPhone 16 Pro" to its identifier.
func (s *IOSSimulator) resolveDeviceType(name string) (string, error) {
	if name == "" {
		name = "iPhone 16 Pro" // default
	}

	// If it's already an identifier, use it directly
	if strings.HasPrefix(name, "com.apple.") {
		return name, nil
	}

	out, err := exec.Command("xcrun", "simctl", "list", "devicetypes", "-j").Output()
	if err != nil {
		return "", fmt.Errorf("failed to list device types: %w", err)
	}

	var result simctlDeviceTypesOutput
	if err := json.Unmarshal(out, &result); err != nil {
		return "", fmt.Errorf("failed to parse device types: %w", err)
	}

	// Exact match
	for _, dt := range result.DeviceTypes {
		if strings.EqualFold(dt.Name, name) {
			return dt.Identifier, nil
		}
	}

	// Substring match
	for _, dt := range result.DeviceTypes {
		if strings.Contains(strings.ToLower(dt.Name), strings.ToLower(name)) {
			return dt.Identifier, nil
		}
	}

	return "", fmt.Errorf("device type not found: %s", name)
}

// resolveRuntime maps a version like "18.6" to its identifier.
func (s *IOSSimulator) resolveRuntime(version string) (string, error) {
	out, err := exec.Command("xcrun", "simctl", "list", "runtimes", "-j").Output()
	if err != nil {
		return "", fmt.Errorf("failed to list runtimes: %w", err)
	}

	var result simctlRuntimesOutput
	if err := json.Unmarshal(out, &result); err != nil {
		return "", fmt.Errorf("failed to parse runtimes: %w", err)
	}

	// If already an identifier, use directly
	if strings.HasPrefix(version, "com.apple.") {
		return version, nil
	}

	// Find latest available iOS runtime, or match specific version
	var latest simctlRuntime
	for _, rt := range result.Runtimes {
		if !rt.IsAvailable {
			continue
		}
		if !strings.HasPrefix(rt.Name, "iOS") {
			continue
		}

		// If version specified, match it
		if version != "" && strings.Contains(rt.Name, version) {
			return rt.Identifier, nil
		}

		// Track latest
		latest = rt
	}

	if version != "" {
		return "", fmt.Errorf("runtime not found for version: %s", version)
	}

	if latest.Identifier == "" {
		return "", fmt.Errorf("no iOS runtime available")
	}

	return latest.Identifier, nil
}

// ListDeviceTypes returns available device types (for user reference).
func (s *IOSSimulator) ListDeviceTypes() ([]string, error) {
	out, err := exec.Command("xcrun", "simctl", "list", "devicetypes", "-j").Output()
	if err != nil {
		return nil, err
	}

	var result simctlDeviceTypesOutput
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}

	var names []string
	for _, dt := range result.DeviceTypes {
		if strings.Contains(dt.Name, "iPhone") || strings.Contains(dt.Name, "iPad") {
			names = append(names, dt.Name)
		}
	}
	return names, nil
}
