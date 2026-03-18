package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/torbenkeller/flutter-agent-connect/internal/client"
)

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage sessions",
	Long:  "Create, list, switch, and destroy simulator sessions.",
}

var sessionCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new session",
	RunE: func(cmd *cobra.Command, args []string) error {
		platform, _ := cmd.Flags().GetString("platform")
		device, _ := cmd.Flags().GetString("device")
		name, _ := cmd.Flags().GetString("name")
		workDir, _ := cmd.Flags().GetString("work-dir")

		// Resolve work-dir to host path (handles container→host translation)
		if workDir != "" {
			resolved, err := client.ResolveWorkDir(workDir)
			if err != nil {
				return fmt.Errorf("failed to resolve work directory: %w", err)
			}
			if resolved != workDir {
				fmt.Printf("Resolved work-dir: %s → %s\n", workDir, resolved)
			}
			workDir = resolved
		}

		c, err := client.Load()
		if err != nil {
			return fmt.Errorf("not connected. Run 'fac connect' first: %w", err)
		}

		session, err := c.CreateSession(platform, device, name, workDir)
		if err != nil {
			return fmt.Errorf("failed to create session: %w", err)
		}

		fmt.Printf("Session created: %s\n", session.ID)
		if session.Name != "" {
			fmt.Printf("Name: %s\n", session.Name)
		}
		fmt.Printf("Platform: %s, Device: %s\n", session.Platform, session.Device.Name)
		return nil
	},
}

var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List sessions for this agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.Load()
		if err != nil {
			return fmt.Errorf("not connected. Run 'fac connect' first: %w", err)
		}

		sessions, err := c.ListSessions()
		if err != nil {
			return fmt.Errorf("failed to list sessions: %w", err)
		}

		if len(sessions) == 0 {
			fmt.Println("No active sessions. Run 'fac session create --platform ios' to create one.")
			return nil
		}

		fmt.Printf("  %-10s %-15s %-10s %-20s %-10s\n", "ID", "Name", "Platform", "Device", "State")
		for _, s := range sessions {
			prefix := " "
			if s.ID == c.ActiveSessionID {
				prefix = "*"
			}
			deviceName := ""
			if s.Device != nil {
				deviceName = s.Device.Name
			}
			fmt.Printf("%s %-10s %-15s %-10s %-20s %-10s\n", prefix, s.ID[:8], s.Name, s.Platform, deviceName, s.State)
		}
		return nil
	},
}

var sessionUseCmd = &cobra.Command{
	Use:   "use <session-id-or-name>",
	Short: "Switch active session",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.Load()
		if err != nil {
			return fmt.Errorf("not connected. Run 'fac connect' first: %w", err)
		}

		session, err := c.UseSession(args[0])
		if err != nil {
			return fmt.Errorf("failed to switch session: %w", err)
		}

		fmt.Printf("Active session: %s (%s)\n", session.Name, session.ID[:8])
		deviceName := ""
		if session.Device != nil {
			deviceName = session.Device.Name
		}
		fmt.Printf("Platform: %s, Device: %s, State: %s\n", session.Platform, deviceName, session.State)
		return nil
	},
}

var sessionDestroyCmd = &cobra.Command{
	Use:   "destroy [session-id-or-name]",
	Short: "Destroy a session",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.Load()
		if err != nil {
			return fmt.Errorf("not connected. Run 'fac connect' first: %w", err)
		}

		target := ""
		if len(args) > 0 {
			target = args[0]
		}

		if err := c.DestroySession(target); err != nil {
			return fmt.Errorf("failed to destroy session: %w", err)
		}

		fmt.Println("Session destroyed")
		return nil
	},
}

func init() {
	sessionCreateCmd.Flags().String("platform", "ios", "Platform: ios, android, web, macos")
	sessionCreateCmd.Flags().String("device", "", "Device type (e.g. 'iPhone 16 Pro')")
	sessionCreateCmd.Flags().String("name", "", "Session name (e.g. 'ios-main')")
	sessionCreateCmd.Flags().String("work-dir", "", "Path to Flutter project on the Mac")

	sessionCmd.AddCommand(sessionCreateCmd)
	sessionCmd.AddCommand(sessionListCmd)
	sessionCmd.AddCommand(sessionUseCmd)
	sessionCmd.AddCommand(sessionDestroyCmd)
	rootCmd.AddCommand(sessionCmd)
}
