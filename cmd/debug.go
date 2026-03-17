package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/torben/flutter-agent-connect/internal/client"
)

var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Toggle debug flags (paint, repaint, performance)",
}

func makeDebugSubcommand(name, short, flag string) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			session, _ := cmd.Flags().GetString("session")

			c, err := client.Load()
			if err != nil {
				return fmt.Errorf("not connected: %w", err)
			}

			enabled, err := c.ToggleDebug(session, flag)
			if err != nil {
				return fmt.Errorf("debug toggle failed: %w", err)
			}

			state := "disabled"
			if enabled {
				state = "enabled"
			}
			fmt.Printf("%s: %s\n", short, state)
			return nil
		},
	}
}

func init() {
	paint := makeDebugSubcommand("paint", "Debug paint", "paint")
	repaint := makeDebugSubcommand("repaint", "Repaint rainbow", "repaint")
	perf := makeDebugSubcommand("performance", "Performance overlay", "performance")

	for _, sub := range []*cobra.Command{paint, repaint, perf} {
		sub.Flags().String("session", "", "Session ID or name (default: active session)")
		debugCmd.AddCommand(sub)
	}

	rootCmd.AddCommand(debugCmd)
}
