package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/torben/flutter-agent-connect/internal/client"
)

var appCmd = &cobra.Command{
	Use:   "app",
	Short: "Manage the Flutter app lifecycle",
}

var appStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Flutter app on the simulator",
	RunE: func(cmd *cobra.Command, args []string) error {
		target, _ := cmd.Flags().GetString("target")
		session, _ := cmd.Flags().GetString("session")

		c, err := client.Load()
		if err != nil {
			return fmt.Errorf("not connected: %w", err)
		}

		fmt.Println("Building app... (this may take a moment)")
		if err := c.AppStart(session, target); err != nil {
			return fmt.Errorf("failed to start app: %w", err)
		}

		fmt.Println("App started")
		return nil
	},
}

var appStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running Flutter app",
	RunE: func(cmd *cobra.Command, args []string) error {
		session, _ := cmd.Flags().GetString("session")

		c, err := client.Load()
		if err != nil {
			return fmt.Errorf("not connected: %w", err)
		}

		if err := c.AppStop(session); err != nil {
			return fmt.Errorf("failed to stop app: %w", err)
		}

		fmt.Println("App stopped")
		return nil
	},
}

func init() {
	appStartCmd.Flags().String("target", "lib/main.dart", "Entry point")
	appStartCmd.Flags().String("session", "", "Session ID or name (default: active session)")

	appStopCmd.Flags().String("session", "", "Session ID or name (default: active session)")

	appCmd.AddCommand(appStartCmd)
	appCmd.AddCommand(appStopCmd)
	rootCmd.AddCommand(appCmd)
}
