package server

import (
	"fmt"
	"os/exec"
)

type Config struct {
	Port       int
	Host       string
	FlutterSDK string
}

func (c *Config) Validate() error {
	if c.FlutterSDK == "" {
		path, err := exec.LookPath("flutter")
		if err != nil {
			return fmt.Errorf("flutter not found in PATH. Install Flutter or use --flutter-sdk")
		}
		c.FlutterSDK = path
	}

	if _, err := exec.LookPath("xcrun"); err != nil {
		return fmt.Errorf("xcrun not found. Install Xcode Command Line Tools: xcode-select --install")
	}

	return nil
}
