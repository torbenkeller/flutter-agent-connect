package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/torbenkeller/flutter-agent-connect/internal/client"
)

var flutterCmd = &cobra.Command{
	Use:   "flutter",
	Short: "Manage the Flutter app lifecycle",
}

var flutterRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the Flutter app on the simulator",
	RunE: func(cmd *cobra.Command, args []string) error {
		target, _ := cmd.Flags().GetString("target")
		session, _ := cmd.Flags().GetString("session")

		c, err := client.Load()
		if err != nil {
			return fmt.Errorf("not connected: %w", err)
		}

		fmt.Println("Building app... (this may take a moment)")
		result, err := c.FlutterRun(session, target)
		if err != nil {
			if buildErr, ok := err.(*client.BuildError); ok {
				fmt.Println("Build failed:")
				for _, line := range buildErr.BuildOutput {
					fmt.Printf("  %s\n", line)
				}
				return fmt.Errorf("build failed")
			}
			return fmt.Errorf("failed to run app: %w", err)
		}

		fmt.Printf("App running on %s\n", result.DeviceID)
		return nil
	},
}

var flutterStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running Flutter app",
	RunE: func(cmd *cobra.Command, args []string) error {
		session, _ := cmd.Flags().GetString("session")

		c, err := client.Load()
		if err != nil {
			return fmt.Errorf("not connected: %w", err)
		}

		if err := c.FlutterStop(session); err != nil {
			return fmt.Errorf("failed to stop app: %w", err)
		}

		fmt.Println("App stopped")
		return nil
	},
}

var flutterHotReloadCmd = &cobra.Command{
	Use:   "hot-reload",
	Short: "Hot reload the running app",
	RunE: func(cmd *cobra.Command, args []string) error {
		session, _ := cmd.Flags().GetString("session")

		c, err := client.Load()
		if err != nil {
			return fmt.Errorf("not connected: %w", err)
		}

		result, err := c.FlutterHotReload(session)
		if err != nil {
			return fmt.Errorf("hot reload failed: %w", err)
		}

		if result.Success {
			fmt.Printf("Hot reload successful (%dms)\n", result.DurationMs)
		} else {
			fmt.Printf("Hot reload failed: %s\nHint: Run 'fac flutter hot-restart' instead\n", result.Message)
		}
		return nil
	},
}

var flutterHotRestartCmd = &cobra.Command{
	Use:   "hot-restart",
	Short: "Hot restart the running app",
	Long:  "Full restart of the app. State is lost but all code changes are applied.",
	RunE: func(cmd *cobra.Command, args []string) error {
		session, _ := cmd.Flags().GetString("session")

		c, err := client.Load()
		if err != nil {
			return fmt.Errorf("not connected: %w", err)
		}

		result, err := c.FlutterHotRestart(session)
		if err != nil {
			return fmt.Errorf("hot restart failed: %w", err)
		}

		fmt.Printf("Hot restart successful (%dms)\n", result.DurationMs)
		return nil
	},
}

var flutterCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Run flutter clean in the session's work directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		session, _ := cmd.Flags().GetString("session")

		c, err := client.Load()
		if err != nil {
			return fmt.Errorf("not connected: %w", err)
		}

		result, err := c.FlutterClean(session)
		if err != nil {
			return fmt.Errorf("flutter clean failed: %w", err)
		}

		fmt.Println(result.Message)
		return nil
	},
}

var flutterPubGetCmd = &cobra.Command{
	Use:   "pub-get",
	Short: "Run flutter pub get in the session's work directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		session, _ := cmd.Flags().GetString("session")

		c, err := client.Load()
		if err != nil {
			return fmt.Errorf("not connected: %w", err)
		}

		result, err := c.FlutterPubGet(session)
		if err != nil {
			return fmt.Errorf("flutter pub get failed: %w", err)
		}

		fmt.Println(result.Message)
		return nil
	},
}

var flutterVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the Flutter version from the server",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.Load()
		if err != nil {
			return fmt.Errorf("not connected: %w", err)
		}

		version, err := c.FlutterVersion()
		if err != nil {
			return fmt.Errorf("failed to get flutter version: %w", err)
		}

		fmt.Println(version)
		return nil
	},
}

func init() {
	flutterRunCmd.Flags().String("target", "lib/main.dart", "Entry point")
	flutterRunCmd.Flags().String("session", "", "Session ID or name (default: active session)")

	flutterStopCmd.Flags().String("session", "", "Session ID or name (default: active session)")
	flutterHotReloadCmd.Flags().String("session", "", "Session ID or name (default: active session)")
	flutterHotRestartCmd.Flags().String("session", "", "Session ID or name (default: active session)")
	flutterCleanCmd.Flags().String("session", "", "Session ID or name (default: active session)")
	flutterPubGetCmd.Flags().String("session", "", "Session ID or name (default: active session)")

	flutterCmd.AddCommand(flutterRunCmd)
	flutterCmd.AddCommand(flutterStopCmd)
	flutterCmd.AddCommand(flutterHotReloadCmd)
	flutterCmd.AddCommand(flutterHotRestartCmd)
	flutterCmd.AddCommand(flutterCleanCmd)
	flutterCmd.AddCommand(flutterPubGetCmd)
	flutterCmd.AddCommand(flutterVersionCmd)
	rootCmd.AddCommand(flutterCmd)
}
