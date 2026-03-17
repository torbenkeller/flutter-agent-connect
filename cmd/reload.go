package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/torben/flutter-agent-connect/internal/client"
)

var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Hot reload the running app",
	RunE: func(cmd *cobra.Command, args []string) error {
		session, _ := cmd.Flags().GetString("session")

		c, err := client.Load()
		if err != nil {
			return fmt.Errorf("not connected: %w", err)
		}

		result, err := c.Reload(session)
		if err != nil {
			return fmt.Errorf("hot reload failed: %w", err)
		}

		if result.Success {
			fmt.Printf("Hot reload successful (%dms)\n", result.DurationMs)
		} else {
			fmt.Printf("Hot reload failed: %s\nHint: Run 'fac restart' instead\n", result.Message)
		}
		return nil
	},
}

func init() {
	reloadCmd.Flags().String("session", "", "Session ID or name (default: active session)")
	rootCmd.AddCommand(reloadCmd)
}
