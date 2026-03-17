package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/torben/flutter-agent-connect/internal/client"
)

var tapCmd = &cobra.Command{
	Use:   "tap [x y]",
	Short: "Tap on the simulator screen",
	Long: `Tap on an element by semantics label, widget key, or pixel coordinates.

Examples:
  fac tap --label "Login"          # Tap by semantics label
  fac tap --key "submitButton"     # Tap by widget key
  fac tap 195 400                  # Tap by pixel coordinates`,
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

func init() {
	tapCmd.Flags().String("label", "", "Semantics label of the element")
	tapCmd.Flags().String("key", "", "Widget key")
	tapCmd.Flags().Int("index", 0, "Match index (0-based) when multiple elements match")
	tapCmd.Flags().String("session", "", "Session ID or name (default: active session)")
	rootCmd.AddCommand(tapCmd)
}
