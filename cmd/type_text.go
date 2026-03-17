package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/torben/flutter-agent-connect/internal/client"
)

var typeCmd = &cobra.Command{
	Use:   "type <text>",
	Short: "Type text into the focused field",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		text := strings.Join(args, " ")
		clear, _ := cmd.Flags().GetBool("clear")
		enter, _ := cmd.Flags().GetBool("enter")
		session, _ := cmd.Flags().GetString("session")

		c, err := client.Load()
		if err != nil {
			return fmt.Errorf("not connected: %w", err)
		}

		if err := c.TypeText(session, text, clear, enter); err != nil {
			return fmt.Errorf("type failed: %w", err)
		}

		msg := fmt.Sprintf("Typed %q", text)
		if enter {
			msg += " + Enter"
		}
		fmt.Println(msg)
		return nil
	},
}

func init() {
	typeCmd.Flags().Bool("clear", false, "Clear field before typing")
	typeCmd.Flags().Bool("enter", false, "Press Enter after typing")
	typeCmd.Flags().String("session", "", "Session ID or name (default: active session)")
	rootCmd.AddCommand(typeCmd)
}
