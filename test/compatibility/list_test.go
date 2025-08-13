package compatibility

import (
	"testing"

	"github.com/trebuchet-org/treb-cli/test/helpers"
)

func TestListCommandCompatibility(t *testing.T) {
	tests := []CompatibilityTest{
		{
			Name:     "list_empty",
			TestCmds: [][]string{{"list"}},
		},
		{
			Name:     "list_with_deployments",
			SetupCtx: helpers.NewTrebContext(t, helpers.BinaryV1),
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
			},
			TestCmds: [][]string{{"list"}},
		},
	}

	RunCompatibilityTests(t, tests)
}
