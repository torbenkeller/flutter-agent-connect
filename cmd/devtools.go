package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/torbenkeller/flutter-agent-connect/internal/client"
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
		tail, _ := cmd.Flags().GetInt("tail")
		session, _ := cmd.Flags().GetString("session")

		c, err := client.Load()
		if err != nil {
			return fmt.Errorf("not connected: %w", err)
		}

		logs, err := c.GetLogs(session, tail)
		if err != nil {
			return fmt.Errorf("failed to get logs: %w", err)
		}

		for _, line := range logs {
			fmt.Println(line)
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

	devtoolsLogsCmd.Flags().Int("tail", 0, "Show only the last N lines (0 = all)")
	devtoolsLogsCmd.Flags().String("session", "", "Session ID or name (default: active session)")
	devtoolsCmd.AddCommand(devtoolsLogsCmd)

	rootCmd.AddCommand(devtoolsCmd)
}
