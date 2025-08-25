package integration

import (
	"testing"

	"github.com/trebuchet-org/treb-cli/test/helpers"
)

func TestPruneCommand(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "prune",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				if err := ctx.TakeSnapshots("before-deploy"); err != nil {
					t.Fatal(err)
				}
			},
			SetupCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
				{"run", "script/deploy/DeployCounter.s.sol"},
				{"run", "script/deploy/DeployCounter.s.sol", "--network", "anvil-31338"},
			},
			PostSetup: func(t *testing.T, ctx *helpers.TestContext) {
				if err := ctx.RevertSnapshots("before-deploy"); err != nil {
					t.Fatal(err)
				}
			},
			TestCmds: [][]string{
				{"prune", "--network", "anvil-31337"},
				{"prune", "--network", "anvil-31338"},
			},
		},
	}

	RunIntegrationTests(t, tests)
}
