package integration

import (
	"testing"
)

func TestDevCommand(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "dev_help",
			TestCmds: [][]string{
				{"dev", "--help"},
			},
		},
		{
			Name: "dev_build",
			TestCmds: [][]string{
				{"dev", "build"},
			},
		},
		{
			Name: "dev_anvil_status",
			TestCmds: [][]string{
				{"dev", "anvil", "status"},
			},
		},
		// Note: Not testing anvil start/stop to avoid conflicts with test infrastructure
	}

	RunIntegrationTests(t, tests)
}

