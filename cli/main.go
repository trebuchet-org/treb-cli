package main

import (
	"os"

	"github.com/trebuchet-org/treb-cli/cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}