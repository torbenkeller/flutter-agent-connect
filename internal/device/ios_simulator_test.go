package device

import (
	"testing"

	"github.com/torbenkeller/flutter-agent-connect/pkg/models"
)

// Test JSON fixture matching real `xcrun simctl list devices -j` output
const simctlFixture = `{
  "devices": {
    "com.apple.CoreSimulator.SimRuntime.iOS-18-6": [
      {
        "udid": "D59AF284-8940-435A-B184-2F9253F1F33C",
        "name": "iPhone 16 Pro",
        "state": "Shutdown",
        "isAvailable": true
      },
      {
        "udid": "AAAA-BBBB",
        "name": "iPhone 16 Pro Max",
        "state": "Booted",
        "isAvailable": true
      },
      {
        "udid": "CCCC-DDDD",
        "name": "unavailable device",
        "state": "Shutdown",
        "isAvailable": false
      }
    ],
    "com.apple.CoreSimulator.SimRuntime.iOS-17-5": [
      {
        "udid": "EEEE-FFFF",
        "name": "fac-agent1-ios-test",
        "state": "Booted",
        "isAvailable": true
      },
      {
        "udid": "1111-2222",
        "name": "fac-agent2-android",
        "state": "Shutdown",
        "isAvailable": true
      }
    ]
  }
}`

func TestParseSimctlDevices(t *testing.T) {
	devices, err := parseSimctlJSON([]byte(simctlFixture))
	if err != nil {
		t.Fatalf("parseSimctlJSON failed: %v", err)
	}

	// Should skip unavailable device
	if len(devices) != 4 {
		t.Fatalf("expected 4 devices, got %d", len(devices))
	}

	// Check first device
	found := false
	for _, d := range devices {
		if d.UDID == "D59AF284-8940-435A-B184-2F9253F1F33C" {
			found = true
			if d.Name != "iPhone 16 Pro" {
				t.Errorf("expected name 'iPhone 16 Pro', got '%s'", d.Name)
			}
			if d.State != models.DeviceStateShutdown {
				t.Errorf("expected state Shutdown, got '%s'", d.State)
			}
			if d.Platform != models.PlatformIOS {
				t.Errorf("expected platform ios, got '%s'", d.Platform)
			}
		}
	}
	if !found {
		t.Error("iPhone 16 Pro not found in parsed devices")
	}
}

func TestFilterForAgent(t *testing.T) {
	devices, _ := parseSimctlJSON([]byte(simctlFixture))

	agent1 := filterByAgentPrefix(devices, "agent1")
	if len(agent1) != 1 {
		t.Fatalf("expected 1 device for agent1, got %d", len(agent1))
	}
	if agent1[0].Name != "fac-agent1-ios-test" {
		t.Errorf("expected 'fac-agent1-ios-test', got '%s'", agent1[0].Name)
	}

	agent2 := filterByAgentPrefix(devices, "agent2")
	if len(agent2) != 1 {
		t.Fatalf("expected 1 device for agent2, got %d", len(agent2))
	}

	unknown := filterByAgentPrefix(devices, "unknown")
	if len(unknown) != 0 {
		t.Errorf("expected 0 devices for unknown agent, got %d", len(unknown))
	}
}
