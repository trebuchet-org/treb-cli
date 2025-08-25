package integration

import (
	"testing"
)

func TestVersionCommand(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "version_basic",
			TestCmds: [][]string{
				{"version"},
			},
		},
	}

	RunIntegrationTests(t, tests)
}

