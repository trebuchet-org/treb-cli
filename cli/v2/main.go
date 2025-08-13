package main

import (
	"os"

	"github.com/trebuchet-org/treb-cli/internal/cli"
)

func main() {
	rootCmd := cli.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}