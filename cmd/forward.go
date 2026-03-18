package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/torbenkeller/flutter-agent-connect/internal/client"
)

var forwardCmd = &cobra.Command{
	Use:   "forward <container-port>",
	Short: "Forward a port and register a dart-define variable",
	Long: `Forward a container port to the Mac host so simulators can reach it.
Optionally registers a dart-define variable that is automatically injected
when running the Flutter app.

Examples:
  fac forward 8080 -e BACKEND_URL
  fac forward 6379 -e REDIS_URL
  fac forward list`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		port, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid port: %s", args[0])
		}

		envName, _ := cmd.Flags().GetString("env")
		session, _ := cmd.Flags().GetString("session")

		c, err := client.Load()
		if err != nil {
			return fmt.Errorf("not connected: %w", err)
		}

		fwd, err := c.AddForward(session, port, envName)
		if err != nil {
			return fmt.Errorf("forward failed: %w", err)
		}

		fmt.Printf("Forwarding :%d → :%d", fwd.ContainerPort, fwd.HostPort)
		if fwd.EnvName != "" {
			fmt.Printf(" (%s)", fwd.EnvName)
		}
		fmt.Println()

		return nil
	},
}

var forwardListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active port forwards",
	RunE: func(cmd *cobra.Command, args []string) error {
		session, _ := cmd.Flags().GetString("session")

		c, err := client.Load()
		if err != nil {
			return fmt.Errorf("not connected: %w", err)
		}

		forwards, err := c.ListForwards(session)
		if err != nil {
			return fmt.Errorf("failed to list forwards: %w", err)
		}

		if len(forwards) == 0 {
			fmt.Println("No active port forwards.")
			return nil
		}

		fmt.Printf("%-12s %-8s %-18s %s\n", "CONTAINER", "HOST", "ENV", "URL")
		for _, f := range forwards {
			fmt.Printf(":%d%-8s :%d%-4s %-18s %s\n",
				f.ContainerPort, "",
				f.HostPort, "",
				f.EnvName,
				f.URLiOS)
		}
		return nil
	},
}

func init() {
	forwardCmd.Flags().StringP("env", "e", "", "Dart-define variable name (e.g. BACKEND_URL)")
	forwardCmd.Flags().String("session", "", "Session ID or name (default: active session)")

	forwardListCmd.Flags().String("session", "", "Session ID or name (default: active session)")
	forwardCmd.AddCommand(forwardListCmd)

	rootCmd.AddCommand(forwardCmd)
}
