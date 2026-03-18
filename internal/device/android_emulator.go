package device

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/torbenkeller/flutter-agent-connect/pkg/models"
)

const androidFacPrefix = "fac_" // AVD names can't have dashes

// AndroidEmulator wraps the Android SDK tools for emulator management.
type AndroidEmulator struct {
	sdkRoot string
}

// NewAndroidEmulator creates a new Android emulator manager, auto-detecting the SDK.
func NewAndroidEmulator() *AndroidEmulator {
	sdkRoot := os.Getenv("ANDROID_HOME")
	if sdkRoot == "" {
		sdkRoot = os.Getenv("ANDROID_SDK_ROOT")
	}
	if sdkRoot == "" {
		home, _ := os.UserHomeDir()
		sdkRoot = filepath.Join(home, "Library", "Android", "sdk")
	}
	return &AndroidEmulator{sdkRoot: sdkRoot}
}

func (a *AndroidEmulator) emulatorBin() string {
	return filepath.Join(a.sdkRoot, "emulator", "emulator")
}

func (a *AndroidEmulator) avdmanagerBin() string {
	return filepath.Join(a.sdkRoot, "cmdline-tools", "latest", "bin", "avdmanager")
}

func (a *AndroidEmulator) adbBin() string {
	path, err := exec.LookPath("adb")
	if err != nil {
		return filepath.Join(a.sdkRoot, "platform-tools", "adb")
	}
	return path
}

// ListAll returns all available Android emulators (AVDs).
func (a *AndroidEmulator) ListAll() ([]models.Device, error) {
	out, err := exec.Command(a.emulatorBin(), "-list-avds").Output()
	if err != nil {
		return nil, fmt.Errorf("emulator -list-avds failed: %w", err)
	}

	var devices []models.Device
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		name := strings.TrimSpace(scanner.Text())
		if name == "" {
			continue
		}
		devices = append(devices, models.Device{
			UDID:      name, // AVD name serves as identifier
			Name:      name,
			Platform:  models.PlatformAndroid,
			State:     models.DeviceStateShutdown,
			Available: true,
		})
	}

	// Check which are currently running
	running := a.listRunningEmulators()
	for i := range devices {
		if _, ok := running[devices[i].Name]; ok {
			devices[i].State = models.DeviceStateBooted
		}
	}

	return devices, nil
}

// ListForAgent returns AVDs belonging to a specific agent.
func (a *AndroidEmulator) ListForAgent(agentID string) ([]models.Device, error) {
	all, err := a.ListAll()
	if err != nil {
		return nil, err
	}

	prefix := androidFacPrefix + agentID + "_"
	var result []models.Device
	for _, d := range all {
		if strings.HasPrefix(d.Name, prefix) {
			result = append(result, d)
		}
	}
	return result, nil
}

// Create creates a new Android emulator AVD.
// Name format: fac_<agentID>_<sessionName>
func (a *AndroidEmulator) Create(agentID, sessionName, deviceType, apiLevel string) (*models.Device, error) {
	if deviceType == "" {
		deviceType = "pixel_7_pro"
	}
	if apiLevel == "" {
		apiLevel = "34"
	}

	avdName := androidFacPrefix + agentID + "_" + sessionName

	// Find system image
	systemImage := fmt.Sprintf("system-images;android-%s;google_apis;arm64-v8a", apiLevel)

	// Check if system image exists
	imgPath := filepath.Join(a.sdkRoot, "system-images", "android-"+apiLevel, "google_apis", "arm64-v8a")
	if _, err := os.Stat(imgPath); os.IsNotExist(err) {
		// Try google_apis_playstore
		systemImage = fmt.Sprintf("system-images;android-%s;google_apis_playstore;arm64-v8a", apiLevel)
		imgPath = filepath.Join(a.sdkRoot, "system-images", "android-"+apiLevel, "google_apis_playstore", "arm64-v8a")
		if _, err := os.Stat(imgPath); os.IsNotExist(err) {
			// Try default
			systemImage = fmt.Sprintf("system-images;android-%s;default;arm64-v8a", apiLevel)
		}
	}

	log.Info().Str("avd", avdName).Str("device", deviceType).Str("image", systemImage).Msg("Creating Android AVD")

	cmd := exec.Command(a.avdmanagerBin(), "create", "avd",
		"--name", avdName,
		"--package", systemImage,
		"--device", deviceType,
		"--force",
	)
	cmd.Stdin = strings.NewReader("no\n") // Don't create custom hardware profile
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("avdmanager create failed: %s: %w", string(output), err)
	}

	return &models.Device{
		UDID:      avdName,
		Name:      avdName,
		Platform:  models.PlatformAndroid,
		Runtime:   "android-" + apiLevel,
		State:     models.DeviceStateShutdown,
		Available: true,
	}, nil
}

// Boot starts an Android emulator in the background. Returns the serial (e.g., emulator-5554).
func (a *AndroidEmulator) Boot(avdName string) (string, error) {
	log.Info().Str("avd", avdName).Msg("Booting Android emulator")

	cmd := exec.Command(a.emulatorBin(),
		"-avd", avdName,
		"-no-window",
		"-no-audio",
		"-no-boot-anim",
		"-gpu", "swiftshader_indirect",
	)
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start emulator: %w", err)
	}

	// Wait for the emulator to appear in adb devices
	serial, err := a.waitForBoot(avdName, 120*time.Second)
	if err != nil {
		_ = cmd.Process.Kill()
		return "", err
	}

	log.Info().Str("avd", avdName).Str("serial", serial).Msg("Android emulator booted")
	return serial, nil
}

// waitForBoot waits for the emulator to be fully booted.
func (a *AndroidEmulator) waitForBoot(avdName string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Check adb devices for new emulator
		running := a.listRunningEmulators()
		for serial, name := range running {
			if name == avdName || strings.Contains(serial, "emulator") {
				// Check if boot completed
				out, err := exec.Command(a.adbBin(), "-s", serial, "shell", "getprop", "sys.boot_completed").Output()
				if err == nil && strings.TrimSpace(string(out)) == "1" {
					return serial, nil
				}
			}
		}
		time.Sleep(2 * time.Second)
	}

	return "", fmt.Errorf("emulator %s did not boot within %v", avdName, timeout)
}

// Shutdown stops an emulator by its serial.
func (a *AndroidEmulator) Shutdown(serial string) error {
	err := exec.Command(a.adbBin(), "-s", serial, "emu", "kill").Run()
	if err != nil {
		return fmt.Errorf("failed to shutdown emulator %s: %w", serial, err)
	}
	return nil
}

// Delete removes an AVD.
func (a *AndroidEmulator) Delete(avdName string) error {
	output, err := exec.Command(a.avdmanagerBin(), "delete", "avd", "--name", avdName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("avdmanager delete failed: %s: %w", string(output), err)
	}
	return nil
}

// Screenshot captures the emulator screen as PNG via adb.
func (a *AndroidEmulator) Screenshot(serial string) ([]byte, error) {
	out, err := exec.Command(a.adbBin(), "-s", serial, "exec-out", "screencap", "-p").Output()
	if err != nil {
		return nil, fmt.Errorf("screenshot failed: %w", err)
	}
	return out, nil
}

// Tap sends a tap event at coordinates.
func (a *AndroidEmulator) Tap(serial string, x, y int) error {
	return exec.Command(a.adbBin(), "-s", serial,
		"shell", "input", "tap", fmt.Sprintf("%d", x), fmt.Sprintf("%d", y)).Run()
}

// TypeText types text into the focused field.
func (a *AndroidEmulator) TypeText(serial, text string) error {
	// adb shell input text has issues with special chars, use base64 broadcast instead
	// For simple text, input text works
	escaped := strings.ReplaceAll(text, " ", "%s")
	escaped = strings.ReplaceAll(escaped, "'", "\\'")
	return exec.Command(a.adbBin(), "-s", serial, "shell", "input", "text", escaped).Run()
}

// Swipe performs a swipe gesture.
func (a *AndroidEmulator) Swipe(serial string, x1, y1, x2, y2, durationMs int) error {
	return exec.Command(a.adbBin(), "-s", serial,
		"shell", "input", "swipe",
		fmt.Sprintf("%d", x1), fmt.Sprintf("%d", y1),
		fmt.Sprintf("%d", x2), fmt.Sprintf("%d", y2),
		fmt.Sprintf("%d", durationMs),
	).Run()
}

// PressKey sends a key event (e.g., KEYCODE_ENTER = 66).
func (a *AndroidEmulator) PressKey(serial string, keycode int) error {
	return exec.Command(a.adbBin(), "-s", serial,
		"shell", "input", "keyevent", fmt.Sprintf("%d", keycode)).Run()
}

// listRunningEmulators returns running emulators: serial -> avd name.
func (a *AndroidEmulator) listRunningEmulators() map[string]string {
	result := make(map[string]string)

	out, err := exec.Command(a.adbBin(), "devices", "-l").Output()
	if err != nil {
		return result
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "emulator-") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 || parts[1] != "device" {
			continue
		}
		serial := parts[0]

		// Get AVD name via adb
		nameOut, err := exec.Command(a.adbBin(), "-s", serial, "emu", "avd", "name").Output()
		if err == nil {
			name := strings.TrimSpace(strings.Split(string(nameOut), "\n")[0])
			result[serial] = name
		} else {
			result[serial] = serial
		}
	}

	return result
}

// GetDeviceInfo returns screen resolution info for an emulator.
func (a *AndroidEmulator) GetDeviceInfo(serial string) (width, height int, err error) {
	out, err := exec.Command(a.adbBin(), "-s", serial, "shell", "wm", "size").Output()
	if err != nil {
		return 0, 0, err
	}
	// Output: "Physical size: 1080x2400"
	parts := strings.Split(strings.TrimSpace(string(out)), ": ")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected wm size output: %s", string(out))
	}
	var w, h int
	_, err = fmt.Sscanf(parts[1], "%dx%d", &w, &h)
	return w, h, err
}

// FlutterDeviceID returns the device ID that flutter uses (from adb devices).
func (a *AndroidEmulator) FlutterDeviceID(serial string) string {
	// Flutter uses the adb serial directly
	return serial
}

// IsAvailable checks if Android SDK tools are present.
func (a *AndroidEmulator) IsAvailable() bool {
	_, err := os.Stat(a.emulatorBin())
	return err == nil
}
