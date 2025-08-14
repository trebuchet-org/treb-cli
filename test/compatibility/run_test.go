package compatibility

import (
	"testing"

	"github.com/trebuchet-org/treb-cli/test/helpers"
)

func TestRunCommandCompatibility(t *testing.T) {
	tests := []CompatibilityTest{
		{
			Name: "simple",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
			},
			TestCmds: [][]string{
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
		},
		{
			Name: "colission",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
			},
			TestCmds: [][]string{
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
		},
		{
			Name: "env/label",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
			},
			TestCmds: [][]string{
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"run", "script/deploy/DeployCounter.s.sol", "-e", "label=test"},
			},
		},
		{
			Name: "chain",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
			},
			TestCmds: [][]string{
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"run", "script/deploy/DeployCounter.s.sol", "-n", "anvil-31338"},
			},
		},
		{
			Name: "namespace",
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
			},
			TestCmds: [][]string{
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"run", "script/deploy/DeployCounter.s.sol", "-s", "production"},
			},
		},
	}

	RunCompatibilityTests(t, tests)
}
