package integration

import (
	"testing"
)

func TestNetworksCommand(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "networks list",
			TestCmds: [][]string{
				{"networks"},
			},
		},
	}

	RunIntegrationTests(t, tests)
}
