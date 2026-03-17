package interaction

import (
	"fmt"
	"os/exec"
	"strings"
)

// IOSInteraction handles input events for iOS simulators.
type IOSInteraction struct{}

// Tap sends a tap event at logical coordinates to the simulator.
// Uses AppleScript to click in the Simulator.app window.
func (i *IOSInteraction) Tap(udid string, x, y float64) error {
	// We use simctl's built-in functionality where possible.
	// For taps, we need to use AppleScript to interact with the Simulator window.
	//
	// The approach:
	// 1. Find the Simulator window for this device
	// 2. Get window bounds and content area
	// 3. Map logical coordinates to window coordinates
	// 4. Click at the computed position

	script := fmt.Sprintf(`
tell application "Simulator" to activate
delay 0.2
tell application "System Events"
	tell process "Simulator"
		set frontWindow to window 1
		set {winX, winY} to position of frontWindow
		set {winW, winH} to size of frontWindow

		-- Title bar is approximately 28px on macOS
		set titleBarHeight to 28

		-- The content area maps to the device screen
		set contentH to winH - titleBarHeight
		set contentW to winW

		-- Device logical size for coordinate mapping
		-- We receive coordinates in logical pixels, window shows at some scale
		-- The scale is: contentW / deviceLogicalWidth
		-- For now we pass normalized coordinates (0-1 range) from the caller
		set clickX to winX + (%f * contentW)
		set clickY to winY + titleBarHeight + (%f * contentH)

		click at {clickX, clickY}
	end tell
end tell
`, x, y)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tap failed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// TapAbsolute sends a tap at pixel coordinates, converting from device logical pixels.
func (i *IOSInteraction) TapAbsolute(udid string, logicalX, logicalY float64, screenWidth, screenHeight float64) error {
	// Convert logical coordinates to normalized 0-1 range
	normalizedX := logicalX / screenWidth
	normalizedY := logicalY / screenHeight

	return i.Tap(udid, normalizedX, normalizedY)
}

// TypeText types text into the currently focused field.
// Uses the simulator pasteboard for reliable text input.
func (i *IOSInteraction) TypeText(udid string, text string, clear bool, enter bool) error {
	if clear {
		// Select All (Cmd+A) then Delete
		if err := i.sendKeystroke(udid, "a", true); err != nil {
			return fmt.Errorf("clear failed: %w", err)
		}
		if err := i.sendKey(udid, "delete"); err != nil {
			return fmt.Errorf("clear failed: %w", err)
		}
	}

	// Copy text to simulator pasteboard and paste
	copyCmd := exec.Command("xcrun", "simctl", "pbcopy", udid)
	copyCmd.Stdin = strings.NewReader(text)
	if err := copyCmd.Run(); err != nil {
		return fmt.Errorf("pbcopy failed: %w", err)
	}

	// Paste via Cmd+V
	if err := i.sendKeystroke(udid, "v", true); err != nil {
		return fmt.Errorf("paste failed: %w", err)
	}

	if enter {
		if err := i.sendKey(udid, "return"); err != nil {
			return fmt.Errorf("enter failed: %w", err)
		}
	}

	return nil
}

// Swipe performs a swipe gesture on the simulator.
func (i *IOSInteraction) Swipe(udid string, direction string, durationMs int) error {
	// Map direction to normalized start/end coordinates
	var startX, startY, endX, endY float64

	switch direction {
	case "up":
		startX, startY = 0.5, 0.7
		endX, endY = 0.5, 0.3
	case "down":
		startX, startY = 0.5, 0.3
		endX, endY = 0.5, 0.7
	case "left":
		startX, startY = 0.7, 0.5
		endX, endY = 0.3, 0.5
	case "right":
		startX, startY = 0.3, 0.5
		endX, endY = 0.7, 0.5
	default:
		return fmt.Errorf("unknown swipe direction: %s", direction)
	}

	durationSec := float64(durationMs) / 1000.0

	script := fmt.Sprintf(`
tell application "Simulator" to activate
delay 0.2
tell application "System Events"
	tell process "Simulator"
		set frontWindow to window 1
		set {winX, winY} to position of frontWindow
		set {winW, winH} to size of frontWindow
		set titleBarHeight to 28
		set contentH to winH - titleBarHeight
		set contentW to winW

		set startClickX to winX + (%f * contentW)
		set startClickY to winY + titleBarHeight + (%f * contentH)
		set endClickX to winX + (%f * contentW)
		set endClickY to winY + titleBarHeight + (%f * contentH)

		-- Perform drag (simulates swipe)
		set startPoint to {startClickX, startClickY}
		set endPoint to {endClickX, endClickY}

		-- Mouse down, move, mouse up
		do shell script "cliclick dd:" & (round startClickX) & "," & (round startClickY) & " du:" & (round endClickX) & "," & (round endClickY)
	end tell
end tell
`, startX, startY, endX, endY)

	// Try with AppleScript first, fall back to simpler approach
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback: just use two taps (not a real swipe but better than nothing)
		_ = durationSec
		return fmt.Errorf("swipe failed (cliclick may not be installed): %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// sendKeystroke sends a keystroke with optional command modifier.
func (i *IOSInteraction) sendKeystroke(udid string, key string, withCmd bool) error {
	modifier := ""
	if withCmd {
		modifier = " using command down"
	}

	script := fmt.Sprintf(`
tell application "Simulator" to activate
delay 0.1
tell application "System Events"
	keystroke "%s"%s
end tell
`, key, modifier)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("keystroke failed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// sendKey sends a key code (like "return", "delete", "tab").
func (i *IOSInteraction) sendKey(udid string, keyName string) error {
	// Map key names to key codes
	keyCodes := map[string]int{
		"return": 36,
		"delete": 51,
		"tab":    48,
		"escape": 53,
	}

	code, ok := keyCodes[keyName]
	if !ok {
		return fmt.Errorf("unknown key: %s", keyName)
	}

	script := fmt.Sprintf(`
tell application "Simulator" to activate
delay 0.1
tell application "System Events"
	key code %d
end tell
`, code)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("key press failed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}
