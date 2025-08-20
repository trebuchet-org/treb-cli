package compatibility

import (
	"testing"
	// "github.com/trebuchet-org/treb-cli/test/helpers"
)

func TestGenCommand(t *testing.T) {
	tests := []CompatibilityTest{
		{
			Name: "gen_deploy_singleton",
			TestCmds: [][]string{
				{"gen", "deploy", "src/Counter.sol:Counter"},
			},
		},
		{
			Name: "gen_deploy_no_contract",
			TestCmds: [][]string{
				{"gen", "deploy", "DoesntExist"},
			},
			ExpectErr:  true,
			ExpectDiff: true,
		},
		{
			Name: "gen_deploy_multiple",
			TestCmds: [][]string{
				{"gen", "deploy", "Counter"},
			},
			ExpectErr:  true,
			ExpectDiff: true,
		},
	}

	RunCompatibilityTests(t, tests)
}
