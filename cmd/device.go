package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/torbenkeller/flutter-agent-connect/internal/client"
)

var deviceCmd = &cobra.Command{
	Use:   "device",
	Short: "Interact with the simulator device",
}

var deviceScreenshotCmd = &cobra.Command{
	Use:   "screenshot",
	Short: "Take a screenshot of the running app",
	Long: `Take a screenshot and save it to a file.
Outputs only the file path to stdout (machine-parseable for AI agents).

Examples:
  fac device screenshot                    # saves to temp file, prints path
  fac device screenshot -o screen.png      # saves to specific file, prints path
  path=$(fac device screenshot)            # capture path in variable`,
	RunE: func(cmd *cobra.Command, args []string) error {
		output, _ := cmd.Flags().GetString("output")
		session, _ := cmd.Flags().GetString("session")

		c, err := client.Load()
		if err != nil {
			return fmt.Errorf("not connected: %w", err)
		}

		data, err := c.DeviceScreenshot(session, false)
		if err != nil {
			return fmt.Errorf("screenshot failed: %w", err)
		}

		// If no output specified, use a temp file
		if output == "" {
			f, err := os.CreateTemp("", "fac-screenshot-*.png")
			if err != nil {
				return fmt.Errorf("failed to create temp file: %w", err)
			}
			output = f.Name()
			f.Close()
		}

		if err := os.WriteFile(output, data, 0644); err != nil {
			return fmt.Errorf("failed to write screenshot: %w", err)
		}

		// Output only the file path (machine-parseable)
		fmt.Println(output)
		return nil
	},
}

var deviceTapCmd = &cobra.Command{
	Use:   "tap [x y]",
	Short: "Tap on the simulator screen",
	Long: `Tap on an element by semantics label, widget key, or pixel coordinates.

Examples:
  fac device tap --label "Login"          # Tap by semantics label
  fac device tap --key "submitButton"     # Tap by widget key
  fac device tap 195 400                  # Tap by pixel coordinates`,
	RunE: func(cmd *cobra.Command, args []string) error {
		label, _ := cmd.Flags().GetString("label")
		key, _ := cmd.Flags().GetString("key")
		index, _ := cmd.Flags().GetInt("index")
		session, _ := cmd.Flags().GetString("session")

		c, err := client.Load()
		if err != nil {
			return fmt.Errorf("not connected: %w", err)
		}

		var result *client.TapResult

		switch {
		case label != "":
			result, err = c.TapByLabel(session, label, index)
		case key != "":
			result, err = c.TapByKey(session, key, index)
		case len(args) == 2:
			x, errX := strconv.Atoi(args[0])
			y, errY := strconv.Atoi(args[1])
			if errX != nil || errY != nil {
				return fmt.Errorf("invalid coordinates: %s %s", args[0], args[1])
			}
			result, err = c.TapAtCoordinates(session, x, y)
		default:
			return fmt.Errorf("specify --label, --key, or x y coordinates")
		}

		if err != nil {
			return fmt.Errorf("tap failed: %w", err)
		}

		fmt.Printf("Tapped at (%d, %d)\n", result.X, result.Y)
		return nil
	},
}

var deviceSwipeCmd = &cobra.Command{
	Use:   "swipe",
	Short: "Swipe on the simulator screen",
	Long: `Swipe in a direction or between coordinates.

Examples:
  fac device swipe --down                        # Scroll down
  fac device swipe --up                          # Scroll up
  fac device swipe --from 200,400 --to 200,100   # Custom swipe`,
	RunE: func(cmd *cobra.Command, args []string) error {
		up, _ := cmd.Flags().GetBool("up")
		down, _ := cmd.Flags().GetBool("down")
		left, _ := cmd.Flags().GetBool("left")
		right, _ := cmd.Flags().GetBool("right")
		duration, _ := cmd.Flags().GetInt("duration")
		session, _ := cmd.Flags().GetString("session")

		c, err := client.Load()
		if err != nil {
			return fmt.Errorf("not connected: %w", err)
		}

		var direction string
		switch {
		case up:
			direction = "up"
		case down:
			direction = "down"
		case left:
			direction = "left"
		case right:
			direction = "right"
		default:
			return fmt.Errorf("specify --up, --down, --left, or --right")
		}

		if err := c.Swipe(session, direction, duration); err != nil {
			return fmt.Errorf("swipe failed: %w", err)
		}

		fmt.Printf("Swiped %s\n", direction)
		return nil
	},
}

var deviceTypeCmd = &cobra.Command{
	Use:   "type <text>",
	Short: "Type text into the focused field",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		text := strings.Join(args, " ")
		clear, _ := cmd.Flags().GetBool("clear")
		enter, _ := cmd.Flags().GetBool("enter")
		session, _ := cmd.Flags().GetString("session")

		c, err := client.Load()
		if err != nil {
			return fmt.Errorf("not connected: %w", err)
		}

		if err := c.TypeText(session, text, clear, enter); err != nil {
			return fmt.Errorf("type failed: %w", err)
		}

		msg := fmt.Sprintf("Typed %q", text)
		if enter {
			msg += " + Enter"
		}
		fmt.Println(msg)
		return nil
	},
}

func init() {
	deviceScreenshotCmd.Flags().StringP("output", "o", "", "Output file path (default: temp file)")
	deviceScreenshotCmd.Flags().String("session", "", "Session ID or name (default: active session)")

	deviceTapCmd.Flags().String("label", "", "Semantics label of the element")
	deviceTapCmd.Flags().String("key", "", "Widget key")
	deviceTapCmd.Flags().Int("index", 0, "Match index (0-based) when multiple elements match")
	deviceTapCmd.Flags().String("session", "", "Session ID or name (default: active session)")

	deviceSwipeCmd.Flags().Bool("up", false, "Swipe up")
	deviceSwipeCmd.Flags().Bool("down", false, "Swipe down")
	deviceSwipeCmd.Flags().Bool("left", false, "Swipe left")
	deviceSwipeCmd.Flags().Bool("right", false, "Swipe right")
	deviceSwipeCmd.Flags().Int("duration", 300, "Swipe duration in milliseconds")
	deviceSwipeCmd.Flags().String("session", "", "Session ID or name (default: active session)")

	deviceTypeCmd.Flags().Bool("clear", false, "Clear field before typing")
	deviceTypeCmd.Flags().Bool("enter", false, "Press Enter after typing")
	deviceTypeCmd.Flags().String("session", "", "Session ID or name (default: active session)")

	deviceCmd.AddCommand(deviceScreenshotCmd)
	deviceCmd.AddCommand(deviceTapCmd)
	deviceCmd.AddCommand(deviceSwipeCmd)
	deviceCmd.AddCommand(deviceTypeCmd)
	rootCmd.AddCommand(deviceCmd)
}
