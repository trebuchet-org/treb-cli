package main

import (
	"os"

	"github.com/trebuchet-org/treb-cli/internal/cli"
	"github.com/trebuchet-org/treb-cli/internal/config"
)

var version, commit, date, trebSolCommit string

func main() {
	config.SetBuildFlags(version, commit, date, trebSolCommit)
	rootCmd := cli.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
