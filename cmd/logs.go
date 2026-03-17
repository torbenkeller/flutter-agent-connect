package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/torben/flutter-agent-connect/internal/client"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show app logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		errors, _ := cmd.Flags().GetBool("errors")
		lines, _ := cmd.Flags().GetInt("lines")
		session, _ := cmd.Flags().GetString("session")

		c, err := client.Load()
		if err != nil {
			return fmt.Errorf("not connected: %w", err)
		}

		logs, err := c.GetLogs(session, errors, lines)
		if err != nil {
			return fmt.Errorf("failed to get logs: %w", err)
		}

		for _, entry := range logs {
			fmt.Printf("[%s] %s\n", entry.Timestamp, entry.Message)
		}
		return nil
	},
}

func init() {
	logsCmd.Flags().Bool("errors", false, "Show only errors and exceptions")
	logsCmd.Flags().Int("lines", 50, "Number of log lines to show")
	logsCmd.Flags().String("session", "", "Session ID or name (default: active session)")
	rootCmd.AddCommand(logsCmd)
}
