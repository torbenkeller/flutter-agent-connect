package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/torben/flutter-agent-connect/internal/client"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Hot restart the running app",
	Long:  "Full restart of the app. State is lost but all code changes are applied.",
	RunE: func(cmd *cobra.Command, args []string) error {
		session, _ := cmd.Flags().GetString("session")

		c, err := client.Load()
		if err != nil {
			return fmt.Errorf("not connected: %w", err)
		}

		result, err := c.Restart(session)
		if err != nil {
			return fmt.Errorf("hot restart failed: %w", err)
		}

		fmt.Printf("Hot restart successful (%dms)\n", result.DurationMs)
		return nil
	},
}

func init() {
	restartCmd.Flags().String("session", "", "Session ID or name (default: active session)")
	rootCmd.AddCommand(restartCmd)
}
