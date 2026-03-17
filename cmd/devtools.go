package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/torben/flutter-agent-connect/internal/client"
)

var devtoolsCmd = &cobra.Command{
	Use:   "devtools",
	Short: "DevTools inspection and debugging",
}

func makeDevtoolsInspectSubcommand(name, short, treeType string) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			session, _ := cmd.Flags().GetString("session")

			c, err := client.Load()
			if err != nil {
				return fmt.Errorf("not connected: %w", err)
			}

			tree, err := c.Inspect(session, treeType)
			if err != nil {
				return fmt.Errorf("inspect failed: %w", err)
			}

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(tree)
		},
	}
}

func makeDevtoolsDebugSubcommand(name, short, flag string) *cobra.Command {
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

var devtoolsLogsCmd = &cobra.Command{
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
	widgets := makeDevtoolsInspectSubcommand("widgets", "Dump the widget tree", "widgets")
	render := makeDevtoolsInspectSubcommand("render", "Dump the render tree (layout, constraints)", "render")
	semantics := makeDevtoolsInspectSubcommand("semantics", "Dump the semantics tree (labels, rects, actions)", "semantics")

	paint := makeDevtoolsDebugSubcommand("paint", "Debug paint", "paint")
	repaint := makeDevtoolsDebugSubcommand("repaint", "Repaint rainbow", "repaint")
	perf := makeDevtoolsDebugSubcommand("performance", "Performance overlay", "performance")

	for _, sub := range []*cobra.Command{widgets, render, semantics, paint, repaint, perf} {
		sub.Flags().String("session", "", "Session ID or name (default: active session)")
		devtoolsCmd.AddCommand(sub)
	}

	devtoolsLogsCmd.Flags().Bool("errors", false, "Show only errors and exceptions")
	devtoolsLogsCmd.Flags().Int("lines", 50, "Number of log lines to show")
	devtoolsLogsCmd.Flags().String("session", "", "Session ID or name (default: active session)")
	devtoolsCmd.AddCommand(devtoolsLogsCmd)

	rootCmd.AddCommand(devtoolsCmd)
}
