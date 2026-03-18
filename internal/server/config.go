package server

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Config struct {
	Port       int
	Host       string
	FlutterSDK string
}

type dependency struct {
	name       string
	check      func() (string, bool)
	required   bool
	installMsg string
}

func (c *Config) Validate() error {
	fmt.Println("Checking dependencies...")
	fmt.Println()

	deps := []dependency{
		{
			name:     "Flutter SDK",
			required: true,
			check: func() (string, bool) {
				if c.FlutterSDK != "" {
					return c.FlutterSDK, true
				}
				path, err := exec.LookPath("flutter")
				if err != nil {
					return "", false
				}
				out, _ := exec.Command(path, "--version", "--machine").Output()
				version := "found"
				if len(out) > 0 {
					version = path
				}
				c.FlutterSDK = path
				return version, true
			},
			installMsg: "Install from https://docs.flutter.dev/get-started/install",
		},
		{
			name:     "Xcode CLI Tools",
			required: true,
			check: func() (string, bool) {
				_, err := exec.LookPath("xcrun")
				if err != nil {
					return "", false
				}
				out, _ := exec.Command("xcrun", "--version").Output()
				return strings.TrimSpace(string(out)), true
			},
			installMsg: "Run: xcode-select --install",
		},
		{
			name:     "iOS Simulator",
			required: false,
			check: func() (string, bool) {
				out, err := exec.Command("xcrun", "simctl", "list", "runtimes", "-j").Output()
				if err != nil {
					return "", false
				}
				if strings.Contains(string(out), "isAvailable") {
					return "available", true
				}
				return "", false
			},
			installMsg: "Install via Xcode → Settings → Platforms",
		},
		{
			name:     "idb (iOS interaction)",
			required: false,
			check: func() (string, bool) {
				path, err := exec.LookPath("idb")
				if err != nil {
					return "", false
				}
				return path, true
			},
			installMsg: "Run: brew install facebook/fb/idb-companion && pip3 install fb-idb",
		},
		{
			name:     "Android SDK",
			required: false,
			check: func() (string, bool) {
				sdkRoot := os.Getenv("ANDROID_HOME")
				if sdkRoot == "" {
					sdkRoot = os.Getenv("ANDROID_SDK_ROOT")
				}
				if sdkRoot == "" {
					home, _ := os.UserHomeDir()
					sdkRoot = filepath.Join(home, "Library", "Android", "sdk")
				}
				if _, err := os.Stat(filepath.Join(sdkRoot, "emulator", "emulator")); err == nil {
					return sdkRoot, true
				}
				return "", false
			},
			installMsg: "Install Android Studio or SDK from https://developer.android.com/studio",
		},
		{
			name:     "adb",
			required: false,
			check: func() (string, bool) {
				path, err := exec.LookPath("adb")
				if err != nil {
					home, _ := os.UserHomeDir()
					path = filepath.Join(home, "Library", "Android", "sdk", "platform-tools", "adb")
					if _, err := os.Stat(path); err != nil {
						return "", false
					}
				}
				return path, true
			},
			installMsg: "Included with Android SDK",
		},
	}

	allOK := true
	for _, dep := range deps {
		info, ok := dep.check()
		if ok {
			fmt.Printf("  ✓ %-20s %s\n", dep.name, info)
		} else if dep.required {
			fmt.Printf("  ✗ %-20s MISSING (required)\n", dep.name)
			fmt.Printf("    → %s\n", dep.installMsg)
			allOK = false
		} else {
			fmt.Printf("  - %-20s not found (optional)\n", dep.name)
			fmt.Printf("    → %s\n", dep.installMsg)
		}
	}

	fmt.Println()

	if !allOK {
		return fmt.Errorf("required dependencies missing — see above")
	}

	return nil
}
