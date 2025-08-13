package compatibility

import (
	"testing"
)

func TestNetworksCommand(t *testing.T) {
	tests := []CompatibilityTest{
		{
			Name: "networks list",
			TestCmds: [][]string{
				{"networks"},
			},
		},
	}

	RunCompatibilityTests(t, tests)
}