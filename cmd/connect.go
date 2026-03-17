package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/torben/flutter-agent-connect/internal/client"
)

var connectCmd = &cobra.Command{
	Use:   "connect [server-url]",
	Short: "Connect to a FAC server",
	Long:  "Connects to a FAC server and registers the agent. Saves connection info to ~/.fac/config.json.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverURL := "http://host.docker.internal:8420"
		if len(args) > 0 {
			serverURL = args[0]
		}

		agentID, _ := cmd.Flags().GetString("agent")

		cfg := client.ConnectConfig{
			ServerURL: serverURL,
			AgentID:   agentID,
		}

		c, err := client.Connect(cfg)
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}

		fmt.Printf("Connected to FAC Server at %s\n", c.ServerURL)
		fmt.Printf("Agent: %s\n", c.AgentID)
		return nil
	},
}

func init() {
	connectCmd.Flags().String("agent", "", "Agent name (default: auto-generated)")
	rootCmd.AddCommand(connectCmd)
}
