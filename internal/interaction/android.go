package interaction

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// AndroidInteraction handles input events for Android emulators via adb.
// All interactions go directly through adb — fully programmatic, no GUI needed.
type AndroidInteraction struct {
	adbPath string
}

// NewAndroidInteraction creates a new Android interaction handler.
func NewAndroidInteraction() *AndroidInteraction {
	adbPath, _ := exec.LookPath("adb")
	if adbPath == "" {
		home, _ := os.UserHomeDir()
		adbPath = filepath.Join(home, "Library", "Android", "sdk", "platform-tools", "adb")
	}
	return &AndroidInteraction{adbPath: adbPath}
}

// Tap sends a tap at coordinates via adb shell input.
func (a *AndroidInteraction) Tap(serial string, x, y int) error {
	return a.runADB(serial, "shell", "input", "tap",
		fmt.Sprintf("%d", x), fmt.Sprintf("%d", y))
}

// TypeText types text into the focused field via adb.
func (a *AndroidInteraction) TypeText(serial, text string, clearField, enter bool) error {
	if clearField {
		// Select all (Ctrl+A) + delete
		_ = a.runADB(serial, "shell", "input", "keyevent", "29", "67") // KEYCODE_A with CTRL, then DEL
	}

	// adb shell input text has issues with special characters
	// Escape spaces and special chars
	escaped := strings.ReplaceAll(text, " ", "%s")
	escaped = strings.ReplaceAll(escaped, "&", "\\&")
	escaped = strings.ReplaceAll(escaped, "<", "\\<")
	escaped = strings.ReplaceAll(escaped, ">", "\\>")
	escaped = strings.ReplaceAll(escaped, "(", "\\(")
	escaped = strings.ReplaceAll(escaped, ")", "\\)")
	escaped = strings.ReplaceAll(escaped, "'", "\\'")
	escaped = strings.ReplaceAll(escaped, "\"", "\\\"")

	if err := a.runADB(serial, "shell", "input", "text", escaped); err != nil {
		return err
	}

	if enter {
		return a.runADB(serial, "shell", "input", "keyevent", "66") // KEYCODE_ENTER
	}

	return nil
}

// Swipe performs a swipe gesture via adb shell input.
func (a *AndroidInteraction) Swipe(serial, direction string, screenW, screenH, durationMs int) error {
	centerX := screenW / 2
	centerY := screenH / 2

	var x1, y1, x2, y2 int
	switch direction {
	case "up":
		x1, y1 = centerX, screenH*7/10
		x2, y2 = centerX, screenH*3/10
	case "down":
		x1, y1 = centerX, screenH*3/10
		x2, y2 = centerX, screenH*7/10
	case "left":
		x1, y1 = screenW*7/10, centerY
		x2, y2 = screenW*3/10, centerY
	case "right":
		x1, y1 = screenW*3/10, centerY
		x2, y2 = screenW*7/10, centerY
	default:
		return fmt.Errorf("unknown direction: %s", direction)
	}

	return a.runADB(serial, "shell", "input", "swipe",
		fmt.Sprintf("%d", x1), fmt.Sprintf("%d", y1),
		fmt.Sprintf("%d", x2), fmt.Sprintf("%d", y2),
		fmt.Sprintf("%d", durationMs))
}

func (a *AndroidInteraction) runADB(serial string, args ...string) error {
	fullArgs := append([]string{"-s", serial}, args...)
	cmd := exec.Command(a.adbPath, fullArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("adb %s failed: %s: %w", args[0], strings.TrimSpace(string(output)), err)
	}
	return nil
}
