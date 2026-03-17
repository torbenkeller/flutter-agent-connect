package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/torben/flutter-agent-connect/internal/client"
)

var screenshotCmd = &cobra.Command{
	Use:   "screenshot",
	Short: "Take a screenshot of the running app",
	Long: `Take a screenshot and save it to a file.
Outputs only the file path to stdout (machine-parseable for AI agents).

Examples:
  fac screenshot                    # saves to temp file, prints path
  fac screenshot -o screen.png      # saves to specific file, prints path
  path=$(fac screenshot)            # capture path in variable`,
	RunE: func(cmd *cobra.Command, args []string) error {
		output, _ := cmd.Flags().GetString("output")
		session, _ := cmd.Flags().GetString("session")

		c, err := client.Load()
		if err != nil {
			return fmt.Errorf("not connected: %w", err)
		}

		data, err := c.Screenshot(session, false)
		if err != nil {
			return fmt.Errorf("screenshot failed: %w", err)
		}

		// If no output specified, use a temp file
		if output == "" {
			f, err := os.CreateTemp("", "fac-screenshot-*.png")
			if err != nil {
				return fmt.Errorf("failed to create temp file: %w", err)
			}
			output = f.Name()
			f.Close()
		}

		if err := os.WriteFile(output, data, 0644); err != nil {
			return fmt.Errorf("failed to write screenshot: %w", err)
		}

		// Output only the file path (machine-parseable)
		fmt.Println(output)
		return nil
	},
}

func init() {
	screenshotCmd.Flags().StringP("output", "o", "", "Output file path (default: temp file)")
	screenshotCmd.Flags().String("session", "", "Session ID or name (default: active session)")
	rootCmd.AddCommand(screenshotCmd)
}
