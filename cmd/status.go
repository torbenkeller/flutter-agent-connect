package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/torbenkeller/flutter-agent-connect/internal/client"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show connection and session status",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.Load()
		if err != nil {
			fmt.Println("Server:  Not connected")
			fmt.Println("Run 'fac connect' to connect")
			return nil
		}

		health, err := c.Health()
		if err != nil {
			fmt.Printf("Server:  %s (unreachable)\n", c.ServerURL)
			return nil
		}

		fmt.Printf("Server:  %s (connected, v%s)\n", c.ServerURL, health.Version)
		fmt.Printf("Agent:   %s\n", c.AgentID)

		if c.ActiveSessionID == "" {
			fmt.Println("Session: None")
			fmt.Println("Run 'fac session create --platform ios' to create a session")
			return nil
		}

		session, err := c.GetSession(c.ActiveSessionID)
		if err != nil {
			fmt.Printf("Session: %s (error: %v)\n", c.ActiveSessionID[:8], err)
			return nil
		}

		deviceName := ""
		if session.Device != nil {
			deviceName = session.Device.Name
		}
		fmt.Printf("Session: %s (%s, %s)\n", session.Name, session.ID[:8], session.Platform)
		fmt.Printf("Device:  %s\n", deviceName)
		fmt.Printf("State:   %s\n", session.State)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
