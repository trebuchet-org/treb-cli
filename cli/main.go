package main

import (
	"os"

	"github.com/trebuchet-org/treb-cli/cli/cmd"
)

// Version information - set at build time via ldflags
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	// Set version info in cmd package
	cmd.SetVersionInfo(version, commit, date)
	
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}