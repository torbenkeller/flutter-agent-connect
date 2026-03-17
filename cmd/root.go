package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "fac",
	Short: "Flutter Agent Connect — Bridge between AI agents and Flutter simulators",
	Long: `FAC (Flutter Agent Connect) bridges AI agents in DevContainers
with Flutter simulators/emulators on a Mac.

Server mode (on the Mac):
  fac serve

Client mode (in the DevContainer):
  fac connect
  fac session create --platform ios
  fac flutter run
  fac flutter hot-reload
  fac device screenshot`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
