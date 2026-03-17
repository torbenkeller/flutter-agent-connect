package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/torben/flutter-agent-connect/internal/client"
)

var swipeCmd = &cobra.Command{
	Use:   "swipe",
	Short: "Swipe on the simulator screen",
	Long: `Swipe in a direction or between coordinates.

Examples:
  fac swipe --down                        # Scroll down
  fac swipe --up                          # Scroll up
  fac swipe --from 200,400 --to 200,100   # Custom swipe`,
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

func init() {
	swipeCmd.Flags().Bool("up", false, "Swipe up")
	swipeCmd.Flags().Bool("down", false, "Swipe down")
	swipeCmd.Flags().Bool("left", false, "Swipe left")
	swipeCmd.Flags().Bool("right", false, "Swipe right")
	swipeCmd.Flags().Int("duration", 300, "Swipe duration in milliseconds")
	swipeCmd.Flags().String("session", "", "Session ID or name (default: active session)")
	rootCmd.AddCommand(swipeCmd)
}
