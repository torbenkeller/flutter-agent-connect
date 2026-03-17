package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/torben/flutter-agent-connect/internal/client"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Inspect the running app (widget tree, render tree, semantics)",
}

func makeInspectSubcommand(name, short, treeType string) *cobra.Command {
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

func init() {
	widgets := makeInspectSubcommand("widgets", "Dump the widget tree", "widgets")
	render := makeInspectSubcommand("render", "Dump the render tree (layout, constraints)", "render")
	semantics := makeInspectSubcommand("semantics", "Dump the semantics tree (labels, rects, actions)", "semantics")

	for _, sub := range []*cobra.Command{widgets, render, semantics} {
		sub.Flags().String("session", "", "Session ID or name (default: active session)")
		inspectCmd.AddCommand(sub)
	}

	rootCmd.AddCommand(inspectCmd)
}
