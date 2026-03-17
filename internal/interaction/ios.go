package interaction

import (
	"fmt"
	"os/exec"
	"strings"
)

// IOSInteraction handles input events for iOS simulators via Facebook's idb.
// All interactions go directly through the simulator process — no mouse movement,
// no window focus required, fully isolated for parallel use.
type IOSInteraction struct {
	CompanionPath string // path to idb_companion binary
}

// NewIOSInteraction creates a new interaction handler, auto-detecting the companion path.
func NewIOSInteraction() *IOSInteraction {
	path, _ := exec.LookPath("idb_companion")
	if path == "" {
		path = "/opt/homebrew/bin/idb_companion"
	}
	return &IOSInteraction{CompanionPath: path}
}

// Tap sends a tap at logical pixel coordinates directly to the simulator.
func (i *IOSInteraction) Tap(udid string, x, y int) error {
	return i.runIDB(udid, "ui", "tap", fmt.Sprintf("%d", x), fmt.Sprintf("%d", y))
}

// TypeText types text into the currently focused field.
func (i *IOSInteraction) TypeText(udid string, text string, clear bool, enter bool) error {
	if clear {
		// Select all + delete
		if err := i.runIDB(udid, "ui", "key-sequence", "40", "42"); err != nil {
			// Fallback: just continue with typing
		}
	}

	if err := i.runIDB(udid, "ui", "text", text); err != nil {
		return err
	}

	if enter {
		// Key code 40 = Return/Enter in HID
		if err := i.runIDB(udid, "ui", "key", "40"); err != nil {
			return err
		}
	}

	return nil
}

// Swipe performs a swipe gesture between two points.
func (i *IOSInteraction) Swipe(udid string, direction string, screenW, screenH int) error {
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

	return i.runIDB(udid, "ui", "swipe",
		fmt.Sprintf("%d", x1), fmt.Sprintf("%d", y1),
		fmt.Sprintf("%d", x2), fmt.Sprintf("%d", y2),
	)
}

// runIDB executes an idb command targeting a specific simulator by UDID.
func (i *IOSInteraction) runIDB(udid string, args ...string) error {
	fullArgs := []string{"--companion-path", i.CompanionPath}
	fullArgs = append(fullArgs, args...)
	fullArgs = append(fullArgs, "--udid", udid)

	cmd := exec.Command("idb", fullArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("idb %s failed: %s: %w", args[0], strings.TrimSpace(string(output)), err)
	}
	return nil
}
