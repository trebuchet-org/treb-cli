package compatibility

import (
	"github.com/trebuchet-org/treb-cli/test/helpers"
	"testing"
)

func TestPruneCommand(t *testing.T) {
	var snapshot *helpers.AnvilSnapshot
	tests := []CompatibilityTest{
		{
			Name:     "prune",
			SetupCtx: helpers.NewTrebContext(t, helpers.BinaryV1),
			PreSetup: func(t *testing.T, ctx *helpers.TrebContext) {
				var err error
				snapshot, err = helpers.GetAnvilManager().Snapshot()
				if err != nil {
					t.Fatal(err)
				}
			},
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"run", "script/deploy/DeployCounter.s.sol", "--network", "anvil-31338"},
			},
			PostSetup: func(t *testing.T, ctx *helpers.TrebContext) {
				helpers.GetAnvilManager().Revert(snapshot)
			},
			TestCmds: [][]string{
				{"prune", "--network", "anvil-31337"},
				{"prune", "--network", "anvil-31338"},
			},
		},
	}

	RunCompatibilityTests(t, tests)
}
