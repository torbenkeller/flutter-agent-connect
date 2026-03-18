package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/torbenkeller/flutter-agent-connect/internal/server"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the FAC server",
	Long:  "Starts the FAC HTTP server on the Mac. Manages simulators, Flutter processes, and exposes the REST API.",
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		host, _ := cmd.Flags().GetString("host")
		flutterSDK, _ := cmd.Flags().GetString("flutter-sdk")

		cfg := server.Config{
			Port:       port,
			Host:       host,
			FlutterSDK: flutterSDK,
		}

		srv, err := server.New(cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize server: %w", err)
		}

		return srv.Run()
	},
}

func init() {
	serveCmd.Flags().Int("port", 8420, "HTTP server port")
	serveCmd.Flags().String("host", "127.0.0.1", "Bind address (localhost only by default)")
	serveCmd.Flags().String("flutter-sdk", "", "Path to Flutter SDK (default: auto-detect from PATH)")
	rootCmd.AddCommand(serveCmd)
}
